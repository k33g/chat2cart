package models

import (
	"time"
)

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
}

// ChatRole represents the role of a chat message
type ChatRole string

const (
	RoleUser      ChatRole = "user"
	RoleAssistant ChatRole = "assistant"
	RoleSystem    ChatRole = "system"
)

// NewChatMessage creates a new chat message
func NewChatMessage(role ChatRole, content, sessionID string) *ChatMessage {
	return &ChatMessage{
		ID:        generateID(),
		Role:      string(role),
		Content:   content,
		Timestamp: time.Now(),
		SessionID: sessionID,
	}
}

// ChatSession represents a chat session
type ChatSession struct {
	ID        string         `json:"id"`
	Messages  []*ChatMessage `json:"messages"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	CartID    string         `json:"cart_id"`
}

// NewChatSession creates a new chat session
func NewChatSession(id string) *ChatSession {
	now := time.Now()
	return &ChatSession{
		ID:        id,
		Messages:  make([]*ChatMessage, 0),
		CreatedAt: now,
		UpdatedAt: now,
		CartID:    id, // Use same ID for cart
	}
}

// AddMessage adds a message to the chat session
func (cs *ChatSession) AddMessage(message *ChatMessage) {
	cs.Messages = append(cs.Messages, message)
	cs.UpdatedAt = time.Now()
}

// GetLastMessage returns the last message in the session
func (cs *ChatSession) GetLastMessage() *ChatMessage {
	if len(cs.Messages) == 0 {
		return nil
	}
	return cs.Messages[len(cs.Messages)-1]
}

// GetMessageCount returns the number of messages in the session
func (cs *ChatSession) GetMessageCount() int {
	return len(cs.Messages)
}

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Message   string `json:"message" binding:"required"`
	SessionID string `json:"session_id"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Message     string       `json:"message"`
	SessionID   string       `json:"session_id"`
	CartSummary *CartSummary `json:"cart_summary,omitempty"`
	Timestamp   time.Time    `json:"timestamp"`
}

// ToolCallResult represents the result of a tool call
type ToolCallResult struct {
	ToolName string      `json:"tool_name"`
	Success  bool        `json:"success"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// generateID generates a simple ID (in a real app, you'd use UUID)
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}
