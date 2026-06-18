package entities

import (
	"context"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Generate generates a response from the LLM
	Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error)

	// GenerateStream generates a streaming response
	GenerateStream(ctx context.Context, req *LLMRequest, onChunk func(StreamChunk)) error

	// GetModelInfo returns information about the current model
	GetModelInfo() ModelInfo

	// SetModel sets the model to use
	SetModel(model string) error

	// GetAvailableModels returns available models
	GetAvailableModels() []ModelInfo

	// IsHealthy checks if the provider is healthy
	IsHealthy() bool

	// GenerateText generates text from a simple prompt (for summarization)
	GenerateText(systemPrompt, userPrompt string) (string, error)

	// GetTools returns available tools
	GetTools() []ToolSchema
}

// LLMRequest represents a request to the LLM
type LLMRequest struct {
	History      []Message    `json:"history"`
	Tools        []ToolSchema `json:"tools"`
	SystemPrompt string       `json:"system_prompt"`
	MaxTokens    int          `json:"max_tokens,omitempty"`
	Temperature  float64      `json:"temperature,omitempty"`
	Stop         []string     `json:"stop,omitempty"`
}

// LLMResponse represents a response from the LLM
type LLMResponse struct {
	Message      Message    `json:"message"`
	Model        string     `json:"model"`
	FinishReason string     `json:"finish_reason"`
	TokenUsage   TokenUsage `json:"token_usage"`
}

// StreamChunk represents a streaming chunk
type StreamChunk struct {
	Text         string     `json:"text"`
	Done         bool       `json:"done"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	MaxTokens    int      `json:"max_tokens"`
}

// TokenUsage represents token consumption
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
