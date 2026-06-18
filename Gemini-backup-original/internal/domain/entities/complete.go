package entities

import (
	"context"
	"strings"
	"time"
)

var maps = struct{}{}

// Conversation represents a complete conversation with all entities
type Conversation struct {
	ID        string                 `json:"id"`
	Context   *ContextWindow         `json:"context_window"`
	Agent     *Agent                `json:"agent"`
	Session   *Session              `json:"session"`
	Metadata  map[string]any        `json:"metadata"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID          string                    `json:"id"`
	UserID      string                    `json:"user_id"`
	ChatID      string                    `json:"chat_id"`
	LastActive  time.Time                 `json:"last_active"`
	Preferences SessionPreferences        `json:"preferences"`
	Context     map[string]any            `json:"context"`
}

// SessionPreferences represents user preferences
type SessionPreferences struct {
	Theme        string            `json:"theme"`
	Language     string            `json:"language"`
	Agent        string            `json:"agent"`
	MaxTokens    int               `json:"max_tokens"`
	Temperature  float64           `json:"temperature"`
	ToolsEnabled map[string]bool   `json:"tools_enabled"`
}

// NewConversation creates a new conversation
func NewConversation(id string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        id,
		Context:   NewContextWindow(100, 92000, 50),
		Agent:     NewAgent("default", "Default Agent", "General purpose AI assistant", []string{"general"}),
		Session: &Session{
			ID:          id,
			UserID:      "anonymous",
			ChatID:      "",
			LastActive:  now,
			Preferences: SessionPreferences{
				Theme:        "auto",
				Language:     "vi",
				Agent:        "default",
				MaxTokens:    8192,
				Temperature:  0.7,
				ToolsEnabled: make(map[string]bool),
			},
			Context: make(map[string]any),
		},
		Metadata: make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetLastMessages returns the last N messages
func (c *Conversation) GetLastMessages(n int) []Message {
	return c.Context.GetRecentMessages(n)
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(msg Message) {
	c.Context.AddMessage(msg)
	c.UpdatedAt = time.Now()
}

// GetTotalTokens returns total token count
func (c *Conversation) GetTotalTokens() int {
	return c.Context.GetTotalTokens()
}

// ShouldSummarize checks if conversation needs summarization
func (c *Conversation) ShouldSummarize() bool {
	return c.Context.ShouldPrune()
}

// GetHistory returns conversation history
func (c *Conversation) GetHistory() []Message {
	return c.Context.GetHistory()
}

// GetSystemPrompt returns the system prompt for this conversation
func (c *Conversation) GetSystemPrompt() string {
	if c.Agent == nil {
		return "Bạn là một trợ lý AI hữu ích."
	}

	var prompt strings.Builder
	prompt.WriteString(c.Agent.Description)
	prompt.WriteString("\n\nKhả năng của bạn bao gồm:\n")
	for _, cap := range c.Agent.Capabilities {
		prompt.WriteString("- ")
		prompt.WriteString(cap)
		prompt.WriteString("\n")
	}

	return prompt.String()
}

// UpdateSession updates session information
func (c *Conversation) UpdateSession(ctx context.Context, update map[string]any) {
	newContext := make(map[string]any, len(c.Session.Context))
	for k, v := range c.Session.Context {
		newContext[k] = v
	}
	for k, v := range update {
		newContext[k] = v
	}
	c.Session.Context = newContext
	c.Session.LastActive = time.Now()
	c.UpdatedAt = time.Now()
}

// GetAgentCapabilities returns agent capabilities
func (c *Conversation) GetAgentCapabilities() []string {
	if c.Agent == nil {
		return []string{"general"}
	}
	return c.Agent.Capabilities
}