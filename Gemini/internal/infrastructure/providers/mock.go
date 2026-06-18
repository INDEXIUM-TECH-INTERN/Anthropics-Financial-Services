package providers

import (
	"context"
	"fmt"
	"strings"

	"gemini-cli/internal/models/messaging"
)

// MockProvider is a test double for Provider interface.
type MockProvider struct {
	Responses        []messaging.Message
	Texts            []string
	StreamChunks     []StreamChunk
	CallCount        int
	GenerateErr      error
	GenerateTextErr  error
	GenerateStreamErr error
}

func (m *MockProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	if m.GenerateTextErr != nil {
		return "", m.GenerateTextErr
	}
	idx := m.CallCount
	if idx < len(m.Texts) {
		m.CallCount++
		return m.Texts[idx], nil
	}
	return "mock response", nil
}

func (m *MockProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	if m.GenerateErr != nil {
		return messaging.Message{}, m.GenerateErr
	}
	idx := m.CallCount
	m.CallCount++
	if idx < len(m.Responses) {
		return m.Responses[idx], nil
	}
	return messaging.Message{
		Role:    messaging.RoleAssistant,
		Content: "mock response",
	}, nil
}

func (m *MockProvider) GenerateStream(ctx context.Context, req messaging.Request, onChunk func(StreamChunk)) error {
	if m.GenerateStreamErr != nil {
		return m.GenerateStreamErr
	}
	for _, chunk := range m.StreamChunks {
		onChunk(chunk)
	}
	return nil
}

// NewMockProviderWithText creates a MockProvider that returns the given text responses.
func NewMockProviderWithText(texts ...string) *MockProvider {
	return &MockProvider{Texts: texts}
}

// NewMockProviderWithTools creates a MockProvider that returns a tool call response.
func NewMockProviderWithTools(toolName string, args map[string]any, finalText string) *MockProvider {
	toolID := fmt.Sprintf("call_%s_0", toolName)
	return &MockProvider{
		Responses: []messaging.Message{
			{
				Role: messaging.RoleAssistant,
				ToolCalls: []messaging.ToolCall{
					{ID: toolID, Name: toolName, Args: args},
				},
			},
			{
				Role:    messaging.RoleAssistant,
				Content: finalText,
			},
		},
		Texts: []string{finalText},
	}
}

// ErrIsProviderFailure checks if an error is a provider failure.
func ErrIsProviderFailure(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Tat ca cac dich vu AI")
}
