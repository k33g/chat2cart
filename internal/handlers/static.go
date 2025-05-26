package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StaticHandler handles static file serving
type StaticHandler struct{}

// NewStaticHandler creates a new static handler
func NewStaticHandler() *StaticHandler {
	return &StaticHandler{}
}

// ServeIndex serves the main chat interface
func (h *StaticHandler) ServeIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "chat2cart - AI Shopping Assistant",
	})
}

// HealthCheck provides a simple health check endpoint
func (h *StaticHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "chat2cart",
		"version": "1.0.0",
	})
}
