package main

import (
	"log"

	"chat2cart/config"
	"chat2cart/internal/handlers"
	"chat2cart/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize services
	productService := services.NewProductService()
	cartService := services.NewCartService(productService)
	aiService := services.NewAIService(cfg.OpenAIAPIKey, productService, cartService, cfg.DMR_BASE_URL, cfg.OLLAMA_BASE_URL)

	// Initialize handlers
	chatHandler := handlers.NewChatHandler(aiService, productService, cartService)
	staticHandler := handlers.NewStaticHandler()

	// Setup Gin router
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Load HTML templates
	router.LoadHTMLGlob("web/templates/*")

	// Serve static files
	router.Static("/static", "./web/static")

	// Routes
	setupRoutes(router, chatHandler, staticHandler)

	// Start server
	log.Printf("Starting chat2cart server on port %s", cfg.Port)
	log.Printf("Environment: %s", cfg.Environment)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(router *gin.Engine, chatHandler *handlers.ChatHandler, staticHandler *handlers.StaticHandler) {
	// Health check
	router.GET("/health", staticHandler.HealthCheck)

	// Main chat interface
	router.GET("/", staticHandler.ServeIndex)

	// API routes
	api := router.Group("/api/v1")
	{
		// Chat endpoints
		api.POST("/chat/message", chatHandler.PostMessage)
		api.GET("/chat/session/:sessionId", chatHandler.GetSession)

		// Cart endpoints
		api.GET("/cart/:sessionId", chatHandler.GetCart)
		api.POST("/cart/:sessionId/add", chatHandler.AddToCart)
		api.DELETE("/cart/:sessionId/item/:productName", chatHandler.RemoveFromCart)
		api.PUT("/cart/:sessionId/item/:productName", chatHandler.UpdateQuantity)
		api.POST("/cart/:sessionId/checkout", chatHandler.Checkout)

		// Product endpoints
		api.GET("/products/search", chatHandler.SearchProducts)

		// AI Model endpoints
		api.GET("/models/dmr", chatHandler.GetDMRModels)
	}
}
