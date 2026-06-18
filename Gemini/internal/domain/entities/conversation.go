package entities

import (
	"sync"
	"time"
)

// Message represents a conversation message
type Message struct {
	ID           string                 `json:"id"`
	Role        string                 `json:"role"`
	Content     string                 `json:"content"`
	Attachments []Attachment           `json:"attachments,omitempty"`
	ToolCalls   []ToolCall             `json:"tool_calls,omitempty"`
	ToolResults []ToolResponse         `json:"tool_results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Attachment represents a file attachment
type Attachment struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Data     string `json:"data"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type,omitempty"`
}

// ToolCall represents a tool/function call
type ToolCall struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Args   map[string]any    `json:"args"`
}

// ToolResponse represents a tool execution result
type ToolResponse struct {
	CallID  string      `json:"call_id"`
	Name    string      `json:"name"`
	Result  interface{} `json:"result"`
	Error   string      `json:"error,omitempty"`
}

// ContextWindow manages conversation history
type ContextWindow struct {
	History      []Message          `json:"history"`
	MaxMessages  int                `json:"max_messages"`
	MaxTokens    int                `json:"max_tokens"`
	WindowSize   int                `json:"window_size"`
	UpdatedAt    time.Time          `json:"updated_at"`
	mu           sync.RWMutex       `json:"-"`
}

// NewContextWindow creates a new context window
func NewContextWindow(maxMessages, maxTokens, windowSize int) *ContextWindow {
	now := time.Now()
	return &ContextWindow{
		History:     make([]Message, 0),
		MaxMessages: maxMessages,
		MaxTokens:   maxTokens,
		WindowSize:  windowSize,
		UpdatedAt:   now,
	}
}

// GetHistory returns conversation history
func (cw *ContextWindow) GetHistory() []Message {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	history := make([]Message, len(cw.History))
	copy(history, cw.History)
	return history
}

// GetTotalTokens estimates total tokens
func (cw *ContextWindow) GetTotalTokens() int {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	total := 0
	for _, msg := range cw.History {
		total += len(msg.Content) / 4
	}
	return total
}
