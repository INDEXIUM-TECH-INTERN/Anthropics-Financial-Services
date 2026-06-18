package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/domain/interfaces"
)

// GeminiProvider implements the LLMProvider interface for Gemini
type GeminiProvider struct {
	config     *GeminiConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// GeminiConfig represents Gemini-specific configuration
type GeminiConfig struct {
	APIKey          string
	Model           string
	Temperature     float64
	MaxTokens       int
	RequestTimeout  time.Duration
	StreamTimeout   time.Duration
	RateLimit       RateLimitConfig
	SafetySettings  []SafetySetting
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize        int
	Window           time.Duration
}

// SafetySetting represents Gemini safety settings
type SafetySetting struct {
	Category string `json:"category"`
	Threshold string `json:"threshold"`
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config *GeminiConfig) *GeminiProvider {
	return &GeminiProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// Generate generates a response from Gemini
func (p *GeminiProvider) Generate(ctx context.Context, req *interfaces.LLMRequest) (*interfaces.LLMResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Prepare request
	geminiReq, err := p.prepareGeminiRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %w", err)
	}

	// Execute request
	resp, err := p.executeRequest(ctx, geminiReq, false)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// GenerateStream generates a streaming response from Gemini
func (p *GeminiProvider) GenerateStream(ctx context.Context, req *interfaces.LLMRequest, onChunk func(interfaces.StreamChunk)) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Prepare request
	geminiReq, err := p.prepareGeminiRequest(req)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %w", err)
	}

	// Execute streaming request
	return p.executeStreamRequest(ctx, geminiReq, onChunk)
}

// GetModelInfo returns information about the current model
func (p *GeminiProvider) GetModelInfo() interfaces.ModelInfo {
	return interfaces.ModelInfo{
		ID:          p.config.Model,
		Name:        p.config.Model,
		Description: "Google Gemini AI model",
		Capabilities: []string{
			"text-generation",
			"function-calling",
			"streaming",
			"context-aware",
		},
		MaxTokens: p.config.MaxTokens,
	}
}

// SetModel sets the model to use
func (p *GeminiProvider) SetModel(model string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.Model = model
	return nil
}

// GetAvailableModels returns available models
func (p *GeminiProvider) GetAvailableModels() []interfaces.ModelInfo {
	return []interfaces.ModelInfo{
		{
			ID:          "gemini-3.1-flash-lite",
			Name:        "Gemini 3.1 Flash Lite",
			Description: "Fast and efficient model for most tasks",
			Capabilities: []string{"text-generation", "function-calling", "streaming"},
			MaxTokens:   8192,
		},
		{
			ID:          "gemini-3.5-pro",
			Name:        "Gemini 3.5 Pro",
			Description: "High performance model for complex tasks",
			Capabilities: []string{"text-generation", "function-calling", "streaming", "long-context"},
			MaxTokens:   32768,
		},
	}
}

// IsHealthy checks if the provider is healthy
func (p *GeminiProvider) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Quick health check with simple request
	req := &interfaces.LLMRequest{
		History: []entities.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 10,
	}

	_, err := p.Generate(ctx, req)
	return err == nil
}

// prepareGeminiRequest converts the LLM request to Gemini format
func (p *GeminiProvider) prepareGeminiRequest(req *interfaces.LLMRequest) (*GeminiRequest, error) {
	var contents []GeminiContent
	var systemInstruction *GeminiContent

	// Process messages
	for _, msg := range req.History {
		if msg.Role == "system" {
			systemInstruction = &GeminiContent{
				Role: "user",
				Parts: []GeminiPart{
					{Text: msg.Content},
				},
			}
			continue
		}

		geminiRole := "user"
		if msg.Role == "assistant" {
			geminiRole = "model"
		}

		// Handle message content and attachments
		parts := []GeminiPart{}
		if msg.Content != "" {
			parts = append(parts, GeminiPart{Text: msg.Content})
		}

		// Handle tool calls and responses
		for _, tc := range msg.ToolCalls {
			parts = append(parts, GeminiPart{
				FunctionCall: &FunctionCall{
					Name: tc.Name,
					Args: tc.Args,
				},
			})
		}

		for _, tr := range msg.ToolResults {
			parts = append(parts, GeminiPart{
				FunctionResponse: &FunctionResponse{
					Name:     tr.Name,
					Response: tr.Result,
				},
			})
		}

		contents = append(contents, GeminiContent{
			Role:  geminiRole,
			Parts: parts,
		})
	}

	// Prepare tool declarations
	var functionDeclarations []FunctionDeclaration
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			params := tool.Parameters
			if params == nil {
				params = map[string]interface{}{}
			}
			functionDeclarations = append(functionDeclarations, FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			})
		}
	}

	// Build Gemini request
	geminiReq := &GeminiRequest{
		Contents: contents,
	}

	if systemInstruction != nil {
		geminiReq.SystemInstruction = systemInstruction
	}

	if len(functionDeclarations) > 0 {
		geminiReq.Tools = []Tool{
			{
				FunctionDeclarations: functionDeclarations,
			},
		}
	}

	geminiReq.SafetySettings = p.config.SafetySettings

	return geminiReq, nil
}

// executeRequest executes a non-streaming request
func (p *GeminiProvider) executeRequest(ctx context.Context, req *GeminiRequest, stream bool) (*interfaces.LLMResponse, error) {
	modelPath := p.getModelPath()
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent", modelPath)

	if stream {
		url += "?alt=sse"
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.config.APIKey)

	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	client := p.getHttpClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error (%d): %s", resp.StatusCode, string(body))
	}

	if stream {
		return nil, fmt.Errorf("streaming not supported in executeRequest")
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini")
	}

	return p.convertGeminiResponse(&geminiResp), nil
}

// executeStreamRequest executes a streaming request
func (p *GeminiProvider) executeStreamRequest(ctx context.Context, req *GeminiRequest, onChunk func(interfaces.StreamChunk)) error {
	modelPath := p.getModelPath()
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:streamGenerateContent?alt=sse", modelPath)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.config.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := p.getHttpClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gemini API error (%d): %s", resp.StatusCode, string(body))
	}

	// Process SSE stream
	return p.processSSEStream(resp.Body, onChunk)
}

// processSSEStream processes SSE stream from Gemini
func (p *GeminiProvider) processSSEStream(body io.ReadCloser, onChunk func(interfaces.StreamChunk)) error {
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var geminiResp GeminiResponse
		if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
			continue
		}

		if len(geminiResp.Candidates) == 0 {
			continue
		}

		// Convert to chunk and notify
		chunk := interfaces.StreamChunk{}
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			if part.Text != "" {
				chunk.Text = part.Text
				onChunk(chunk)
			}
			if part.FunctionCall != nil {
				chunk.ToolCalls = append(chunk.ToolCalls, entities.ToolCall{
					ID:   fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(chunk.ToolCalls)),
					Name: part.FunctionCall.Name,
					Args: part.FunctionCall.Args,
				})
			}
		}
	}

	return scanner.Err()
}

// convertGeminiResponse converts Gemini response to LLMResponse
func (p *GeminiProvider) convertGeminiResponse(geminiResp *GeminiResponse) *interfaces.LLMResponse {
	resp := &interfaces.LLMResponse{
		Model: geminiResp.Candidates[0].Content.Role,
	}

	// Extract text content and tool calls
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			resp.Message.Content = part.Text
		}
		if part.FunctionCall != nil {
			resp.Message.ToolCalls = append(resp.Message.ToolCalls, entities.ToolCall{
				ID:   fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(resp.Message.ToolCalls)),
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			})
		}
	}

	return resp
}

// getModelPath returns the model path for API calls
func (p *GeminiProvider) getModelPath() string {
	model := strings.TrimSpace(p.config.Model)
	if !strings.HasPrefix(model, "models/") {
		return "models/" + model
	}
	return model
}

// getHttpClient returns the HTTP client
func (p *GeminiProvider) getHttpClient() *http.Client {
	return p.httpClient
}

// GeminiRequest represents Gemini API request format
type GeminiRequest struct {
	Contents          []GeminiContent     `json:"contents"`
	SystemInstruction *GeminiContent      `json:"systemInstruction,omitempty"`
	Tools             []Tool              `json:"tools,omitempty"`
	SafetySettings    []SafetySetting     `json:"safetySettings"`
}

// GeminiContent represents Gemini content
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents Gemini content part
type GeminiPart struct {
	Text            string         `json:"text,omitempty"`
	FunctionCall    *FunctionCall  `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// FunctionCall represents Gemini function call
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// FunctionResponse represents Gemini function response
type FunctionResponse struct {
	Name    string      `json:"name"`
	Response interface{} `json:"response"`
}

// Tool represents Gemini tool
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations"`
}

// FunctionDeclaration represents Gemini function declaration
type FunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GeminiResponse represents Gemini API response format
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// GeminiCandidate represents Gemini candidate
type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}