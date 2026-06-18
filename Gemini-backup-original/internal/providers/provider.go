package providers

import (
	"context"

	"gemini-cli/internal/models/messaging"
)

// StreamChunk represents a single streaming token chunk from the provider.
type StreamChunk struct {
	Text      string
	Done      bool
	Metrics   *StreamMetrics
	ToolCalls []messaging.ToolCall
}

// StreamMetrics holds token usage info returned at the end of a stream.
type StreamMetrics struct {
	TokenIn  int `json:"token_in"`
	TokenOut int `json:"token_out"`
}

type Provider interface {
	GenerateText(systemPrompt, userPrompt string) (string, error)
	Generate(ctx context.Context, req messaging.Request) (messaging.Message, error)
	GenerateStream(ctx context.Context, req messaging.Request, onChunk func(StreamChunk)) error
}
