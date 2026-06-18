package interfaces

import (
	"context"

	"gemini-cli/internal/domain/entities"
)

// AgentService defines the interface for agent operations
type AgentService interface {
	// Process processes a user message and returns AI response
	Process(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error)

	// ProcessMessageLegacy maintains backward compatibility
	ProcessMessage(ctx context.Context, userInput string, atts []entities.Attachment) (string, error)

	// ProcessMessageStreamLegacy maintains backward compatibility
	ProcessMessageStream(ctx context.Context, userInput string, atts []entities.Attachment, onChunk func(string, bool)) error

	// GetAgent returns the current agent
	GetAgent() *entities.Agent

	// GetHistory returns conversation history
	GetHistory() []entities.Message

	// LoadHistory loads conversation history
	LoadHistory(msgs []entities.Message)

	// Reset clears conversation history
	Reset()

	// GetCapabilities returns agent capabilities
	GetCapabilities() []string
}

// ProcessRequest represents a message processing request
type ProcessRequest struct {
	Message      string                  `json:"message"`
	ChatID       string                  `json:"chat_id,omitempty"`
	Attachments  []entities.Attachment   `json:"attachments,omitempty"`
	Stream       bool                    `json:"stream,omitempty"`
	MaxTokens    int                     `json:"max_tokens,omitempty"`
	Temperature  float64                 `json:"temperature,omitempty"`
	ForceAgent   string                  `json:"force_agent,omitempty"` // Force specific agent
}

// ProcessResponse represents a message processing response
type ProcessResponse struct {
	Reply        string                  `json:"reply"`
	History      []entities.Message      `json:"history"`
	UsedAgent    string                  `json:"used_agent"`
	TokenUsage   entities.TokenUsage     `json:"token_usage"`
	ExecTime     int64                   `json:"exec_time_ms"`
	ToolCalls    []entities.ToolCall     `json:"tool_calls,omitempty"`
	Error        string                  `json:"error,omitempty"`
}