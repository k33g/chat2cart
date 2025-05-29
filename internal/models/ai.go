package models

// AIModel represents an AI model available for use
type AIModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Object  string `json:"object,omitempty"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}
