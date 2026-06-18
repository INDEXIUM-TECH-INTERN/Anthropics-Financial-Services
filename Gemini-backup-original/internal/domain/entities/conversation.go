package entities

import (
	"sync"
	"time"
)

// Message represents a conversation message
type Message struct {
	ID           string                 `json:"id"`
	Role        string                 `json:"role"` // "user", "assistant", "system", "tool"
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
	Data     string `json:"data"` // Base64 encoded
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

// ContextWindow manages conversation history with smart pruning
type ContextWindow struct {
	History      []Message          `json:"history"`
	MaxMessages  int                `json:"max_messages"`
	MaxTokens    int                `json:"max_tokens"`
	WindowSize   int                `json:"window_size"`
	mu           sync.RWMutex       `json:"-"`
}

// NewContextWindow creates a new context window
func NewContextWindow(maxMessages, maxTokens, windowSize int) *ContextWindow {
	return &ContextWindow{
		History:     make([]Message, 0),
		MaxMessages: maxMessages,
		MaxTokens:   maxTokens,
		WindowSize:  windowSize,
	}
}

// AddMessage adds a message to the context window
func (cw *ContextWindow) AddMessage(msg Message) {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	msg.Timestamp = time.Now()
	cw.History = append(cw.History, msg)

	// Prune history if needed
	cw.pruneHistory()
}

// GetHistory returns a copy of the conversation history
func (cw *ContextWindow) GetHistory() []Message {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	history := make([]Message, len(cw.History))
	copy(history, cw.History)
	return history
}

// ClearHistory clears all messages
func (cw *ContextWindow) ClearHistory() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.History = make([]Message, 0)
}

// GetRecentMessages returns the most recent messages up to the limit
func (cw *ContextWindow) GetRecentMessages(limit int) []Message {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	if len(cw.History) <= limit {
		return cw.getCopy()
	}

	return cw.History[len(cw.History)-limit:]
}

// GetTotalTokens estimates the total token count in the context
func (cw *ContextWindow) GetTotalTokens() int {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	total := 0
	for _, msg := range cw.History {
		total += len(msg.Content) / 4 // Rough token estimation
		for _, toolCall := range msg.ToolCalls {
			total += len(toolCall.Name) + len(toolCall.ID)
			for _, arg := range toolCall.Args {
				total += len(arg.(string)) / 4
			}
		}
		for _, toolResult := range msg.ToolResults {
			total += len(toolResult.Name) + len(toolResult.CallID)
		}
	}
	return total
}

// ShouldPrune checks if history needs pruning
func (cw *ContextWindow) ShouldPrune() bool {
	return cw.GetTotalTokens() > cw.MaxTokens || len(cw.History) > cw.MaxMessages
}

// PruneHistory prunes the oldest messages to maintain window size
func (cw *ContextWindow) pruneHistory() {
	if len(cw.History) <= cw.WindowSize {
		return
	}

	cw.History = cw.History[len(cw.History)-cw.WindowSize:]
}

// getCopy returns a copy of the history
func (cw *ContextWindow) getCopy() []Message {
	history := make([]Message, len(cw.History))
	copy(history, cw.History)
	return history
}