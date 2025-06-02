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
	defaultModel   string
	openaiBaseURL  string
	dmrBaseURL     string
	ollamaBaseURL  string
	apiKey         string
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
func NewAIService(apiKey string, productService *ProductService, cartService *CartService, dmrBaseURL string, ollamaBaseURL string) *AIService {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithMiddleware(Logger),
	)

	return &AIService{
		client:         &client,
		productService: productService,
		cartService:    cartService,
		defaultModel:   "gpt-4",
		openaiBaseURL:  "https://api.openai.com/v1",
		dmrBaseURL:     dmrBaseURL,
		ollamaBaseURL:  ollamaBaseURL,
		apiKey:         apiKey,
	}
}

// ProcessChatMessage processes a chat message and returns a response
func (ai *AIService) ProcessChatMessage(ctx context.Context, sessionID string, session *models.ChatSession, settings *models.Settings) (*models.ChatResponse, error) {
	// Define the tools available to the AI
	tools := ai.getToolDefinitions()

	// Build messages including conversation history
	messages := ai.buildMessagesFromSession(session)

	var cartSummary *models.CartSummary
	var toolResults []models.ToolCallResult
	var responseMessage string

	// Maximum number of tool call iterations
	maxIterations := 5
	currentIteration := 0

	// Use provided settings or defaults
	model := ai.defaultModel
	baseURL := ai.openaiBaseURL
	if settings != nil {
		if settings.Model != "" {
			model = settings.Model
		}
		if settings.Provider != "" {
			provider := settings.Provider
			if provider == "dmr" {
				baseURL = ai.dmrBaseURL
			} else if provider == "ollama" {
				baseURL = ai.ollamaBaseURL
			}

			// Create a new client with the custom base URL
			client := openai.NewClient(
				option.WithAPIKey(ai.apiKey),
				option.WithBaseURL(baseURL),
				option.WithMiddleware(Logger),
			)
			ai.client = &client
		}
	}

	for currentIteration < maxIterations {
		// Create the chat completion request
		completion, err := ai.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       model,
			Messages:    messages,
			Tools:       tools,
			Temperature: param.Opt[float64]{Value: 0.00000000000001},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get AI response: %w", err)
		}

		// Process the response
		choice := completion.Choices[0]
		responseMessage = choice.Message.Content

		// If no tool calls, we're done
		if len(choice.Message.ToolCalls) == 0 {
			break
		}

		// Add the model's function call message to the conversation
		messages = append(messages, choice.Message.ToParam())

		// Execute tool calls
		iterationResults, err := ai.executeToolCalls(ctx, choice.Message.ToolCalls, sessionID)
		if err != nil {
			log.Printf("Error executing tool calls: %v", err)
		}

		// Add results to our collection
		toolResults = append(toolResults, iterationResults...)

		// Add tool results to the conversation as function call outputs
		for _, result := range iterationResults {
			// Convert the result to JSON string
			resultJSON, err := json.Marshal(result.Result)
			if err != nil {
				log.Printf("Error marshaling tool result: %v", err)
				continue
			}

			// Add the function call output message
			messages = append(messages, openai.ToolMessage(string(resultJSON), result.CallID))
		}

		currentIteration++
	}

	// If we hit the maximum iterations, add a warning message
	if currentIteration >= maxIterations {
		responseMessage = "I've reached the maximum number of operations I can perform. Let me know if you need anything else!"
	}

	// Get the final cart summary after all tool executions
	cartSummary = ai.cartService.GetCartSummary(sessionID)

	return &models.ChatResponse{
		Message:     responseMessage,
		SessionID:   sessionID,
		CartSummary: cartSummary,
		Timestamp:   time.Now(),
		ToolCalls:   toolResults,
	}, nil
}

// getSystemPrompt returns the system prompt for the AI
func (ai *AIService) getSystemPrompt() string {
	return `You are a helpful shopping assistant for chat2cart, an AI-powered shopping platform. 
	Your role is to help users discover products, manage their shopping cart, and complete purchases through natural conversation.
	`
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
						"product_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the product to remove",
						},
					},
					"required": []string{"product_name"},
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
						"product_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the product to update",
						},
						"quantity": map[string]interface{}{
							"type":        "integer",
							"description": "New quantity (use 0 to remove)",
						},
					},
					"required": []string{"product_name", "quantity"},
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
	toolCallID := toolCall.ID

	switch functionName {
	case "search_products":
		return ai.handleSearchProducts(arguments, toolCallID)
	case "add_to_cart":
		return ai.handleAddToCart(arguments, sessionID, toolCallID)
	case "remove_from_cart":
		return ai.handleRemoveFromCart(arguments, sessionID, toolCallID)
	case "view_cart":
		return ai.handleViewCart(sessionID, toolCallID)
	case "update_quantity":
		return ai.handleUpdateQuantity(arguments, sessionID, toolCallID)
	case "checkout":
		return ai.handleCheckout(sessionID, toolCallID)
	default:
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  functionName,
			Success:   false,
			Error:     fmt.Sprintf("Unknown tool: %s", functionName),
			Arguments: arguments,
		}
	}
}

// Tool call handlers
func (ai *AIService) handleSearchProducts(arguments string, toolCallID string) models.ToolCallResult {
	var args struct {
		Query    string `json:"query"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
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
			CallID:    toolCallID,
			ToolName:  "search_products",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		CallID:    toolCallID,
		ToolName:  "search_products",
		Success:   true,
		Result:    results,
		Arguments: arguments,
	}
}

func (ai *AIService) handleAddToCart(arguments string, sessionID string, toolCallID string) models.ToolCallResult {
	var args struct {
		ProductName string `json:"product_name"`
		Quantity    int    `json:"quantity"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "add_to_cart",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	if args.Quantity == 0 {
		args.Quantity = 1
	}

	cartSummary, err := ai.cartService.AddToCart(sessionID, args.ProductName, args.Quantity)
	if err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "add_to_cart",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		CallID:    toolCallID,
		ToolName:  "add_to_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleRemoveFromCart(arguments string, sessionID string, toolCallID string) models.ToolCallResult {
	var args struct {
		ProductName string `json:"product_name"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "remove_from_cart",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	cartSummary, err := ai.cartService.RemoveFromCart(sessionID, args.ProductName)
	if err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "remove_from_cart",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		CallID:    toolCallID,
		ToolName:  "remove_from_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleViewCart(sessionID string, toolCallID string) models.ToolCallResult {
	cartSummary := ai.cartService.GetCartSummary(sessionID)
	return models.ToolCallResult{
		CallID:    toolCallID,
		ToolName:  "view_cart",
		Success:   true,
		Result:    cartSummary,
		Arguments: "{}",
	}
}

func (ai *AIService) handleUpdateQuantity(arguments string, sessionID string, toolCallID string) models.ToolCallResult {
	var args struct {
		ProductName string `json:"product_name"`
		Quantity    int    `json:"quantity"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return models.ToolCallResult{
			ToolName:  "update_quantity",
			Success:   false,
			Error:     "Invalid arguments",
			Arguments: arguments,
		}
	}

	cartSummary, err := ai.cartService.UpdateQuantity(sessionID, args.ProductName, args.Quantity)
	if err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "update_quantity",
			Success:   false,
			Error:     err.Error(),
			Arguments: arguments,
		}
	}

	return models.ToolCallResult{
		CallID:    toolCallID,
		ToolName:  "update_quantity",
		Success:   true,
		Result:    cartSummary,
		Arguments: arguments,
	}
}

func (ai *AIService) handleCheckout(sessionID string, toolCallID string) models.ToolCallResult {
	checkoutResult, err := ai.cartService.CheckoutCart(sessionID)
	if err != nil {
		return models.ToolCallResult{
			CallID:    toolCallID,
			ToolName:  "checkout",
			Success:   false,
			Error:     err.Error(),
			Arguments: "{}",
		}
	}

	return models.ToolCallResult{
		CallID:    toolCallID,
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

// GetDMRModels fetches available models from the DMR API
func (ai *AIService) GetDMRModels(ctx context.Context) ([]models.AIModel, error) {
	// Create a new HTTP client
	client := &http.Client{}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", ai.dmrBaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DMR models: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DMR API returned non-200 status code: %d", resp.StatusCode)
	}

	// Parse the response
	var result struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode DMR models response: %w", err)
	}

	// Convert to our model format
	aiModels := make([]models.AIModel, len(result.Data))
	for i, model := range result.Data {
		aiModels[i] = models.AIModel{
			ID:      model.ID,
			Name:    model.ID, // Using ID as name since that's what we want to display
			Object:  model.Object,
			Created: model.Created,
			OwnedBy: model.OwnedBy,
		}
	}

	return aiModels, nil
}
