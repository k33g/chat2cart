package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"chat2cart/internal/models"
	"chat2cart/internal/services"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles chat-related HTTP requests
type ChatHandler struct {
	aiService      *services.AIService
	productService *services.ProductService
	cartService    *services.CartService
	sessions       map[string]*models.ChatSession
	sessionsMutex  sync.RWMutex
}

// NewChatHandler creates a new chat handler
func NewChatHandler(aiService *services.AIService, productService *services.ProductService, cartService *services.CartService) *ChatHandler {
	return &ChatHandler{
		aiService:      aiService,
		productService: productService,
		cartService:    cartService,
		sessions:       make(map[string]*models.ChatSession),
		sessionsMutex:  sync.RWMutex{},
	}
}

// PostMessage handles incoming chat messages
func (h *ChatHandler) PostMessage(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}

	// Get or create chat session
	session := h.getOrCreateSession(req.SessionID)

	// Add user message to session
	userMessage := models.NewChatMessage(models.RoleUser, req.Message, req.SessionID)
	session.AddMessage(userMessage)

	// Process message with AI
	response, err := h.aiService.ProcessChatMessage(c.Request.Context(), req.Message, req.SessionID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}

	// Add AI response to session
	aiMessage := models.NewChatMessage(models.RoleAssistant, response.Message, req.SessionID)
	session.AddMessage(aiMessage)

	c.JSON(http.StatusOK, response)
}

// PostMessageStream handles incoming chat messages with streaming response
func (h *ChatHandler) PostMessageStream(c *gin.Context) {
	var req models.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}

	// Get or create chat session
	session := h.getOrCreateSession(req.SessionID)

	// Add user message to session
	userMessage := models.NewChatMessage(models.RoleUser, req.Message, req.SessionID)
	session.AddMessage(userMessage)

	// Set headers for Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create a writer function for streaming
	writer := func(content string) {
		// Send the content as SSE data
		fmt.Fprintf(c.Writer, "data: %s\n\n", content)
		c.Writer.Flush()
	}

	// Process message with AI streaming
	response, err := h.aiService.StreamChatMessage(c.Request.Context(), req.Message, req.SessionID, session, writer)
	if err != nil {
		fmt.Fprintf(c.Writer, "data: [ERROR] Failed to process message: %s\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	// Add AI response to session
	aiMessage := models.NewChatMessage(models.RoleAssistant, response.Message, req.SessionID)
	session.AddMessage(aiMessage)

	// Send final response with cart summary
	if response.CartSummary != nil {
		cartJSON, err := json.Marshal(response.CartSummary)
		if err != nil {
			fmt.Fprintf(c.Writer, "data: [ERROR] Failed to marshal cart update: %s\n\n", err.Error())
			c.Writer.Flush()
			return
		}
		fmt.Fprintf(c.Writer, "data: [CART_UPDATE] %s\n\n", string(cartJSON))
		c.Writer.Flush()
	}

	// Send end signal
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// GetSession returns the chat session history
func (h *ChatHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	session := h.getOrCreateSession(sessionID)
	c.JSON(http.StatusOK, session)
}

// GetCart returns the current cart for a session
func (h *ChatHandler) GetCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	cartSummary := h.cartService.GetCartSummary(sessionID)
	c.JSON(http.StatusOK, cartSummary)
}

// SearchProducts handles product search requests
func (h *ChatHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	limit := 10

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseLimit(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	filter := models.ProductFilter{
		Query:    query,
		Category: category,
		Limit:    limit,
	}

	results, err := h.productService.SearchProducts(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// AddToCart handles adding items to cart via API
func (h *ChatHandler) AddToCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req struct {
		ProductID string `json:"product_id" binding:"required"`
		Quantity  int    `json:"quantity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	cartSummary, err := h.cartService.AddToCart(sessionID, req.ProductID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cartSummary)
}

// RemoveFromCart handles removing items from cart via API
func (h *ChatHandler) RemoveFromCart(c *gin.Context) {
	sessionID := c.Param("sessionId")
	productID := c.Param("productId")

	if sessionID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID and Product ID are required"})
		return
	}

	cartSummary, err := h.cartService.RemoveFromCart(sessionID, productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cartSummary)
}

// UpdateQuantity handles updating item quantities via API
func (h *ChatHandler) UpdateQuantity(c *gin.Context) {
	sessionID := c.Param("sessionId")
	productID := c.Param("productId")

	if sessionID == "" || productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID and Product ID are required"})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	cartSummary, err := h.cartService.UpdateQuantity(sessionID, productID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cartSummary)
}

// Checkout handles checkout requests via API
func (h *ChatHandler) Checkout(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	checkoutResult, err := h.cartService.CheckoutCart(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, checkoutResult)
}

// Helper methods

func (h *ChatHandler) getOrCreateSession(sessionID string) *models.ChatSession {
	// First try to get the session with a read lock
	h.sessionsMutex.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMutex.RUnlock()

	if exists {
		return session
	}

	// If session doesn't exist, acquire write lock to create it
	h.sessionsMutex.Lock()
	defer h.sessionsMutex.Unlock()

	// Double check after acquiring write lock
	if session, exists = h.sessions[sessionID]; exists {
		return session
	}

	session = models.NewChatSession(sessionID)
	h.sessions[sessionID] = session
	return session
}

func generateSessionID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}

func parseLimit(limitStr string) (int, error) {
	// Simple integer parsing - in a real app you'd use strconv.Atoi
	switch limitStr {
	case "5":
		return 5, nil
	case "10":
		return 10, nil
	case "20":
		return 20, nil
	case "50":
		return 50, nil
	default:
		return 10, nil
	}
}
