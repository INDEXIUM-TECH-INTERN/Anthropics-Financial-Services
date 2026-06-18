package providers

import (
	"sync"

	"gemini-cli/internal/domain/entities"
)

// GeminiConfig represents Gemini-specific configuration
type GeminiConfig struct {
	APIKey   string
	Model    string
	Temperature float64
	MaxTokens int
}

// GeminiProvider implements the LLMProvider interface for Gemini
type GeminiProvider struct {
	apiKey  string
	model  string
	mu      sync.RWMutex
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	return &GeminiProvider{
		apiKey:  config.APIKey,
		model:   config.Model,
		mu:      sync.RWMutex{},
	}
}

// GetModelInfo returns information about the current model
func (p *GeminiProvider) GetModelInfo() entities.ModelInfo {
	return entities.ModelInfo{
		ID:          p.model,
		Name:        p.model,
		Description: "Google Gemini AI model",
		Capabilities: []string{"text-generation", "function-calling", "streaming"},
		MaxTokens:   8192,
	}
}

// GetAvailableModels returns available models
func (p *GeminiProvider) GetAvailableModels() []entities.ModelInfo {
	return []entities.ModelInfo{
		{
			ID:          "gemini-3.1-flash-lite",
			Name:        "Gemini 3.1 Flash Lite",
			Description: "Fast and efficient model for most tasks",
			Capabilities: []string{"text-generation", "function-calling", "streaming"},
			MaxTokens:   8192,
		},
	}
}

// SetModel sets the model to use
func (p *GeminiProvider) SetModel(model string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model = model
	return nil
}

// IsHealthy checks if the provider is healthy
func (p *GeminiProvider) IsHealthy() bool {
	return p.apiKey != "" && p.model != ""
}

// GenerateText generates text from a simple prompt (for summarization)
func (p *GeminiProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	// TODO: Implement GenerateText using Gemini API
	return "Summary placeholder", nil
}

// GetTools returns available tools
func (p *GeminiProvider) GetTools() []entities.ToolSchema {
	return []entities.ToolSchema{}
}
