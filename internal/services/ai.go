package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"chat2cart/internal/models"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// AIService handles AI interactions with OpenAI
type AIService struct {
	client         *openai.Client
	productService *ProductService
	cartService    *CartService
}

type loggingReadCloser struct {
	io.ReadCloser
	buf *bytes.Buffer
}

func (lrc *loggingReadCloser) Read(p []byte) (int, error) {
	n, err := lrc.ReadCloser.Read(p)
	if n > 0 {
		lrc.buf.Write(p[:n])
	}
	return n, err
}

func Logger(req *http.Request, next option.MiddlewareNext) (res *http.Response, err error) {
	start := time.Now()

	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	log.Printf("Request: %s %s %s", req.Method, req.URL.Path, string(bodyBytes))

	res, err = next(req)

	end := time.Now()

	if res != nil && res.Body != nil && strings.Contains(res.Header.Get("Content-Type"), "text/event-stream") {
		// Don't buffer, just log as it streams
		buf := new(bytes.Buffer)
		res.Body = &loggingReadCloser{ReadCloser: res.Body, buf: buf}

		go func() {
			// Wait until the body is fully consumed
			<-req.Context().Done()
			log.Printf("Response: %s %s %s\nBody: %s", res.Status, req.URL.Path, end.Sub(start), buf.String())
		}()
	} else if res != nil && res.Body != nil {
		responseBody, _ := io.ReadAll(res.Body)
		res.Body = io.NopCloser(bytes.NewBuffer(responseBody))
		log.Printf("Response: %s %s %s\nBody: %s", res.Status, req.URL.Path, end.Sub(start), string(responseBody))
	}

	return res, err
}

// NewAIService creates a new AI service
func NewAIService(apiKey string, productService *ProductService, cartService *CartService) *AIService {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithMiddleware(Logger),
	)

	return &AIService{
		client:         &client,
		productService: productService,
		cartService:    cartService,
	}
}

// ProcessChatMessage processes a chat message and returns a response
func (ai *AIService) ProcessChatMessage(ctx context.Context, message, sessionID string, session *models.ChatSession) (*models.ChatResponse, error) {
	// Define the tools available to the AI
	tools := ai.getToolDefinitions()

	// Build messages including conversation history
	messages := ai.buildMessagesFromSession(session)

	// Create the chat completion request
	completion, err := ai.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       openai.ChatModelGPT4o,
		Messages:    messages,
		Tools:       tools,
		Temperature: param.Opt[float64]{Value: 0.00000000000001},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get AI response: %w", err)
	}

	// Process the response
	choice := completion.Choices[0]
	responseMessage := choice.Message.Content

	// Handle tool calls if any
	var cartSummary *models.CartSummary
	var toolResults []models.ToolCallResult
	if len(choice.Message.ToolCalls) > 0 {
		var err error
		toolResults, err = ai.executeToolCalls(ctx, choice.Message.ToolCalls, sessionID)
		if err != nil {
			log.Printf("Error executing tool calls: %v", err)
		}

		// Get updated cart summary after tool execution
		cartSummary = ai.cartService.GetCartSummary(sessionID)

		// If we have tool results, create a follow-up completion to generate a natural response
		if len(toolResults) > 0 {
			followUpCompletion, err := ai.createFollowUpResponse(ctx, message, toolResults, cartSummary)
			if err != nil {
				log.Printf("Error creating follow-up response: %v", err)
			} else {
				responseMessage = followUpCompletion
			}
		}
	}

	return &models.ChatResponse{
		Message:     responseMessage,
		SessionID:   sessionID,
		CartSummary: cartSummary,
		Timestamp:   time.Unix(completion.Created, 0),
		ToolCalls:   toolResults,
	}, nil
}

// StreamChatMessage processes a chat message and streams the response
func (ai *AIService) StreamChatMessage(ctx context.Context, message, sessionID string, session *models.ChatSession, writer func(string)) (*models.ChatResponse, error) {
	// Define the tools available to the AI
	tools := ai.getToolDefinitions()

	// Build messages including conversation history
	messages := ai.buildMessagesFromSession(session)

	// Create the streaming chat completion request with tools
	stream := ai.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:       openai.ChatModelGPT4o,
		Messages:    messages,
		Tools:       tools,
		Temperature: param.Opt[float64]{Value: 0.00000000000001},
	})

	var responseMessage strings.Builder
	var created int64
	var toolCalls []openai.ChatCompletionMessageToolCall
	var toolCallID string

	// Process the stream
	for stream.Next() {
		chunk := stream.Current()
		created = chunk.Created

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Handle content streaming
			if choice.Delta.Content != "" {
				content := choice.Delta.Content
				responseMessage.WriteString(content)
				// Write each token immediately
				writer(content)
			}

			// Handle tool calls
			if len(choice.Delta.ToolCalls) > 0 {
				for _, toolCall := range choice.Delta.ToolCalls {
					if toolCallID == "" {
						toolCallID = toolCall.ID
						toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCall{
							ID: toolCall.ID,
							Function: openai.ChatCompletionMessageToolCallFunction{
								Name:      toolCall.Function.Name,
								Arguments: toolCall.Function.Arguments,
							},
						})
					} else {
						// Append to existing tool call
						for i, existingCall := range toolCalls {
							if existingCall.ID == toolCall.ID {
								toolCalls[i].Function.Arguments += toolCall.Function.Arguments
								break
							}
						}
					}
				}
			}
		}
	}

	if stream.Err() != nil {
		return nil, fmt.Errorf("streaming error: %w", stream.Err())
	}

	finalMessage := responseMessage.String()
	var toolResults []models.ToolCallResult
	var cartSummary *models.CartSummary

	// Execute tool calls if any were received
	if len(toolCalls) > 0 {
		writer("\n\n🔧 Processing tools...")
		toolResults, err := ai.executeToolCalls(ctx, toolCalls, sessionID)
		if err != nil {
			log.Printf("Error executing tool calls: %v", err)
		} else {
			cartSummary = ai.cartService.GetCartSummary(sessionID)

			if len(toolResults) > 0 {
				writer("\n\n💭 Generating response...")
				followUpCompletion, err := ai.createFollowUpResponse(ctx, message, toolResults, cartSummary)
				if err != nil {
					log.Printf("Error creating follow-up response: %v", err)
				} else {
					writer("\n\n" + followUpCompletion)
					finalMessage = finalMessage + "\n\n" + followUpCompletion
				}
			}
		}
	} else {
		// If no tool calls, get current cart summary
		cartSummary = ai.cartService.GetCartSummary(sessionID)
	}

	return &models.ChatResponse{
		Message:     finalMessage,
		SessionID:   sessionID,
		CartSummary: cartSummary,
		Timestamp:   time.Unix(created, 0),
		ToolCalls:   toolResults,
	}, nil
}

// getSystemPrompt returns the system prompt for the AI
func (ai *AIService) getSystemPrompt() string {
	return `You are a helpful shopping assistant for chat2cart, an AI-powered shopping platform. Your role is to help users discover products, manage their shopping cart, and complete purchases through natural conversation.

Key capabilities:
- Search for products by name, description, or category
- Add products to the shopping cart
- Remove products from the cart
- Update quantities in the cart
- Show cart contents and totals
- Process checkout

Guidelines:
- Be friendly, helpful, and conversational
- Ask clarifying questions when needed (e.g., quantity, specific product details)
- Provide product recommendations when appropriate
- Always confirm actions like adding/removing items
- Help users understand their cart contents and totals
- Guide users through the checkout process

Available product categories: electronics, clothing, books, home, sports

When users ask about products, use the search_products tool to find relevant items. When they want to add items to their cart, use the appropriate cart management tools.`
}

// getToolDefinitions returns the tool definitions for OpenAI function calling
func (ai *AIService) getToolDefinitions() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "search_products",
				Description: openai.String("Search for products by query, category, or price range"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query for product name or description",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Product category (electronics, clothing, books, home, sports, beauty, toys, food)",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of results to return (default: 10)",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "add_to_cart",
				Description: openai.String("Add a product to the shopping cart"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"product_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the product to add",
						},
						"quantity": map[string]interface{}{
							"type":        "integer",
							"description": "Quantity to add (default: 1)",
						},
					},
					"required": []string{"product_name"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "remove_from_cart",
				Description: openai.String("Remove a product from the shopping cart"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"product_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the product to remove",
						},
					},
					"required": []string{"product_id"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "view_cart",
				Description: openai.String("View the current shopping cart contents and totals"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "update_quantity",
				Description: openai.String("Update the quantity of a product in the cart"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"product_id": map[string]interface{}{
							"type":        "string",
							"description": "The ID of the product to update",
						},
						"quantity": map[string]interface{}{
							"type":        "integer",
							"description": "New quantity (use 0 to remove)",
						},
					},
					"required": []string{"product_id", "quantity"},
				},
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        "checkout",
				Description: openai.String("Process checkout for the current cart"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

// executeToolCalls executes the tool calls from OpenAI
func (ai *AIService) executeToolCalls(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCall, sessionID string) ([]models.ToolCallResult, error) {
	var results []models.ToolCallResult

	for _, toolCall := range toolCalls {
		result := ai.executeToolCall(ctx, toolCall, sessionID)
		results = append(results, result)
	}

	return results, nil
}

// executeToolCall executes a single tool call
func (ai *AIService) executeToolCall(ctx context.Context, toolCall openai.ChatCompletionMessageToolCall, sessionID string) models.ToolCallResult {
	functionName := toolCall.Function.Name
	arguments := toolCall.Function.Arguments

	switch functionName {
	case "search_products":
		return ai.handleSearchProducts(arguments)
	case "add_to_cart":
		return ai.handleAddToCart(arguments, sessionID)
	case "remove_from_cart":
		return ai.handleRemoveFromCart(arguments, sessionID)
	case "view_cart":
		return ai.handleViewCart(sessionID)
	case "update_quantity":
		return ai.handleUpdateQuantity(arguments, sessionID)
	case "checkout":
		return ai.handleCheckout(sessionID)
	default:
		return models.ToolCallResult{
			ToolName:  functionName,
			Success:   false,
			Error:     fmt.Sprintf("Unknown tool: %s", functionName),
			Arguments: arguments,
		}
	}
}

// Tool call handlers
func (ai *AIService) handleSearchProducts(arguments string) models.ToolCallResult {
	var args struct {
		Query    string `json:"query"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			ToolName:  "search_products",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	if args.Limit == 0 {
		args.Limit = 10
	}

	filter := models.ProductFilter{
		Query:    args.Query,
		Category: args.Category,
		Limit:    args.Limit,
	}

	results, err := ai.productService.SearchProducts(filter)
	if err != nil {
		return models.ToolCallResult{
			ToolName:  "search_products",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		ToolName:  "search_products",
		Success:   true,
		Result:    results,
		Arguments: arguments,
	}
}

func (ai *AIService) handleAddToCart(arguments string, sessionID string) models.ToolCallResult {
	var args struct {
		ProductName string `json:"product_name"`
		Quantity    int    `json:"quantity"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			ToolName:  "add_to_cart",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	if args.Quantity == 0 {
		args.Quantity = 1
	}

	cartSummary, err := ai.cartService.AddProductByName(sessionID, args.ProductName, args.Quantity)
	if err != nil {
		return models.ToolCallResult{
			ToolName:  "add_to_cart",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		ToolName:  "add_to_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleRemoveFromCart(arguments string, sessionID string) models.ToolCallResult {
	var args struct {
		ProductID string `json:"product_id"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			ToolName:  "remove_from_cart",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	cartSummary, err := ai.cartService.RemoveFromCart(sessionID, args.ProductID)
	if err != nil {
		return models.ToolCallResult{
			ToolName:  "remove_from_cart",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		ToolName:  "remove_from_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleViewCart(sessionID string) models.ToolCallResult {
	cartSummary := ai.cartService.GetCartSummary(sessionID)
	return models.ToolCallResult{
		ToolName:  "view_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: "{}",
	}
}

func (ai *AIService) handleUpdateQuantity(arguments string, sessionID string) models.ToolCallResult {
	var args struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			ToolName:  "update_quantity",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	cartSummary, err := ai.cartService.UpdateQuantity(sessionID, args.ProductID, args.Quantity)
	if err != nil {
		return models.ToolCallResult{
			ToolName:  "update_quantity",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		ToolName:  "update_quantity",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleCheckout(sessionID string) models.ToolCallResult {
	checkoutResult, err := ai.cartService.CheckoutCart(sessionID)
	if err != nil {
		return models.ToolCallResult{
			ToolName:  "checkout",
			Success:   false,
			Error:     err.Error(),
			Arguments: "{}",
		}
	}

	return models.ToolCallResult{
		ToolName:  "checkout",
		Success:   true,
		Result:    checkoutResult,
		Arguments: "{}",
	}
}

// buildMessagesFromSession converts chat session messages to OpenAI format
func (ai *AIService) buildMessagesFromSession(session *models.ChatSession) []openai.ChatCompletionMessageParamUnion {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(ai.getSystemPrompt()),
	}

	// Add previous messages from the session (excluding the current message which will be added separately)
	for _, msg := range session.Messages {
		switch models.ChatRole(msg.Role) {
		case models.RoleUser:
			messages = append(messages, openai.UserMessage(msg.Content))
		case models.RoleAssistant:
			messages = append(messages, openai.AssistantMessage(msg.Content))
		case models.RoleSystem:
			// Skip system messages as we already have one
			continue
		}
	}

	return messages
}

// createFollowUpResponse creates a natural language response based on tool results
func (ai *AIService) createFollowUpResponse(ctx context.Context, originalMessage string, toolResults []models.ToolCallResult, cartSummary *models.CartSummary) (string, error) {
	// Create a summary of what happened
	var actionSummary strings.Builder
	actionSummary.WriteString("Actions performed:\n")

	for _, result := range toolResults {
		if result.Success {
			switch result.ToolName {
			case "search_products":
				if results, ok := result.Result.([]*models.ProductSearchResult); ok {
					actionSummary.WriteString(fmt.Sprintf("- Found %d products:\n", len(results)))
					for _, result := range results {
						actionSummary.WriteString(fmt.Sprintf("  * %s - $%.2f\n", result.Product.Name, result.Product.Price))
					}
				}
			case "add_to_cart":
				if summary, ok := result.Result.(*models.CartSummary); ok {
					actionSummary.WriteString("- Added items to cart:\n")
					for _, item := range summary.Items {
						actionSummary.WriteString(fmt.Sprintf("  * %s x%d - $%.2f\n", item.Product.Name, item.Quantity, item.GetSubtotal()))
					}
				}
			case "remove_from_cart":
				if summary, ok := result.Result.(*models.CartSummary); ok {
					actionSummary.WriteString("- Removed items from cart:\n")
					for _, item := range summary.Items {
						actionSummary.WriteString(fmt.Sprintf("  * %s x%d - $%.2f\n", item.Product.Name, item.Quantity, item.GetSubtotal()))
					}
				}
			case "update_quantity":
				if summary, ok := result.Result.(*models.CartSummary); ok {
					actionSummary.WriteString("- Updated quantities in cart:\n")
					for _, item := range summary.Items {
						actionSummary.WriteString(fmt.Sprintf("  * %s x%d - $%.2f\n", item.Product.Name, item.Quantity, item.GetSubtotal()))
					}
				}
			case "checkout":
				if checkoutResult, ok := result.Result.(*CheckoutResult); ok {
					actionSummary.WriteString("- Processed checkout:\n")
					actionSummary.WriteString(fmt.Sprintf("  Order ID: %s\n", checkoutResult.OrderID))
					actionSummary.WriteString(fmt.Sprintf("  Status: %s\n", checkoutResult.Status))
					actionSummary.WriteString(fmt.Sprintf("  Message: %s\n", checkoutResult.Message))
					actionSummary.WriteString("  Items purchased:\n")
					for _, item := range checkoutResult.CartSummary.Items {
						actionSummary.WriteString(fmt.Sprintf("  * %s x%d - $%.2f\n", item.Product.Name, item.Quantity, item.GetSubtotal()))
					}
				}
			}
		} else {
			actionSummary.WriteString(fmt.Sprintf("- Error with %s: %s\n", result.ToolName, result.Error))
		}
	}

	// Create a follow-up completion to generate a natural response
	completion, err := ai.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful shopping assistant. Based on the user's request and the actions that were performed, provide a natural, conversational response. Be friendly and helpful."),
			openai.UserMessage(fmt.Sprintf("User said: %s\n\n%s\n\nCurrent cart total: $%.2f with %d items",
				originalMessage, actionSummary.String(), cartSummary.Total, cartSummary.ItemCount)),
		},
	})

	if err != nil {
		return "I've processed your request successfully!", nil
	}

	return completion.Choices[0].Message.Content, nil
}
