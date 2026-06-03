package providers

import (
	"context"

	"gemini-cli/internal/models/messaging"
)

type Provider interface {
	GenerateText(systemPrompt, userPrompt string) (string, error)
	Generate(ctx context.Context, req messaging.Request) (messaging.Message, error)
}
