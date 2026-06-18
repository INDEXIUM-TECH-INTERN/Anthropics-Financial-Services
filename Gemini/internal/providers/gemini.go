package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gemini-cli/internal/models"
	"gemini-cli/internal/models/messaging"
)
type GeminiProvider struct {
	APIKey  string
	Keys    []string
	Current int
	Model   string
}

func (p *GeminiProvider) getAPIKey() string {
	if p.APIKey != "" {
		return p.APIKey
	}
	if len(p.Keys) == 0 {
		return ""
	}
	// Round-robin rotation of API keys
	key := p.Keys[p.Current]
	p.Current = (p.Current + 1) % len(p.Keys)
	return key
}

func (p *GeminiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "gemini-3.1-flash-lite" // Updated default for 2026
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	apiKey := p.getAPIKey()
	// Use v1beta as it supports systemInstruction and tools fields in the JSON payload.
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent?key=%s", modelPath, apiKey)

	// Translate messaging.Request to models.GeminiRequest
	var geminiContents []models.GeminiContent
	var systemInstruction *models.GeminiContent

	for _, msg := range req.History {
		if msg.Role == messaging.RoleSystem {
			if systemInstruction == nil {
				systemInstruction = &models.GeminiContent{Parts: []models.GeminiPart{}}
			}
			systemInstruction.Parts = append(systemInstruction.Parts, models.GeminiPart{Text: msg.Content})
			continue
		}

		parts := []models.GeminiPart{}
		if msg.Content != "" {
			if msg.ThoughtSignature != "" {
				parts = append(parts, models.GeminiPart{
					Text:             msg.Content,
					ThoughtSignature: msg.ThoughtSignature,
				})
			} else {
				parts = append(parts, models.GeminiPart{Text: msg.Content})
			}
		}

		// Handle attachments (R6.1)
		for _, att := range msg.Attachments {
			if isGeminiSupportedMime(att.Type) {
				parts = append(parts, models.GeminiPart{
					InlineData: &models.GeminiInlineData{
						MimeType: att.Type,
						Data:     att.Data,
					},
				})
			} else {
				// For unsupported types, add a text note so the Agent knows
				parts = append(parts, models.GeminiPart{
					Text: fmt.Sprintf("\n[Hệ thống: Đã đính kèm tệp tin '%s' (loại: %s). Hiện tại tôi chưa thể đọc trực tiếp nội dung loại tệp này qua InlineData. Nếu đây là tệp văn bản hoặc Excel, hãy thử dùng công cụ tìm kiếm/đọc tệp hoặc yêu cầu người dùng copy nội dung.]", att.Name, att.Type),
				})
			}
		}

		for _, tc := range msg.ToolCalls {
			parts = append(parts, models.GeminiPart{
				FunctionCall: &models.GeminiFunctionCall{
					Name: tc.Name,
					Args: tc.Args,
				},
				ThoughtSignature: msg.ThoughtSignature,
			})
		}
		if msg.Content == "" && len(msg.ToolCalls) == 0 && msg.ThoughtSignature != "" {
			parts = append(parts, models.GeminiPart{
				ThoughtSignature: msg.ThoughtSignature,
			})
		}
		for _, tr := range msg.ToolResponses {
			// Tool responses are already normalized to valid JSON by the dispatcher.
			var toolOutput map[string]interface{}
			if err := json.Unmarshal([]byte(tr.Content), &toolOutput); err != nil {
				// Fallback: shouldn't happen since dispatcher normalizes, but handle gracefully
				toolOutput = map[string]interface{}{"content": tr.Content}
			}
			parts = append(parts, models.GeminiPart{
				FunctionResponse: &models.GeminiFunctionResponse{
					Name:     tr.Name,
					Response: toolOutput,
				},
			})
		}

		geminiRole := "user"
		switch msg.Role {
		case messaging.RoleAssistant:
			geminiRole = "model"
		case messaging.RoleTool:
			geminiRole = "tool"
		}

		geminiContents = append(geminiContents, models.GeminiContent{
			Role:  geminiRole,
			Parts: parts,
		})
	}

	var functionDeclarations []models.GeminiFunctionDeclaration
	if len(req.Tools) > 0 {
		for _, t := range req.Tools {
			var params map[string]interface{}
			if t.Parameters != nil {
				params = t.Parameters
			}
			functionDeclarations = append(functionDeclarations, models.GeminiFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			})
		}
	}

	geminiReq := models.GeminiRequest{
		Contents:          geminiContents,
		SystemInstruction: systemInstruction,
		SafetySettings: []models.SafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
		},
	}
	if len(functionDeclarations) > 0 {
		geminiReq.Tools = []models.GeminiTool{{FunctionDeclarations: functionDeclarations}}
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error marshalling gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error creating http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// TODO: Implement key rotation for pool

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error performing http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error reading response body: %w", err)
	}

	var gResp models.GeminiResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		// Before returning error, try to see if it's a simple text error from the API
		var apiError struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiError) == nil && apiError.Error.Message != "" {
			return messaging.Message{}, fmt.Errorf("gemini API error: %s", apiError.Error.Message)
		}
		return messaging.Message{}, fmt.Errorf("error unmarshalling gemini response: %s", string(body))
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if gResp.Error.Message != "" {
			return messaging.Message{}, fmt.Errorf("gemini API error: %s", gResp.Error.Message)
		}
		return messaging.Message{}, fmt.Errorf("gemini API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if len(gResp.Candidates) == 0 {
		// It's possible to get a 200 OK with no candidates if content is blocked.
		// Check for promptFeedback.
		// For now, just return an error.
		return messaging.Message{}, fmt.Errorf("no candidates returned from Gemini")
	}

	// Translate models.GeminiResponse to messaging.Message
	geminiContent := gResp.Candidates[0].Content
	responseMsg := messaging.Message{
		Role: messaging.RoleAssistant, // Gemini response is always from the assistant
	}

	var textContent strings.Builder
	for _, part := range geminiContent.Parts {
		if part.Text != "" {
			textContent.WriteString(part.Text)
		}
		if part.ThoughtSignature != "" {
			responseMsg.ThoughtSignature = part.ThoughtSignature
		}
		if part.FunctionCall != nil {
			// Per Gemini docs, ID is not returned, so we must create it.
			callID := fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(responseMsg.ToolCalls))
			responseMsg.ToolCalls = append(responseMsg.ToolCalls, messaging.ToolCall{
				ID:   callID,
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			})
		}
	}
	responseMsg.Content = textContent.String()

	return responseMsg, nil
}

func (p *GeminiProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	req := messaging.Request{
		History: []messaging.Message{
			{Role: messaging.RoleSystem, Content: systemPrompt},
			{Role: messaging.RoleUser, Content: userPrompt},
		},
	}
	resp, err := p.Generate(context.Background(), req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (p *GeminiProvider) GenerateStream(ctx context.Context, req messaging.Request, onChunk func(StreamChunk)) error {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "gemini-3.1-flash-lite"
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	apiKey := p.getAPIKey()
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:streamGenerateContent?alt=sse&key=%s", modelPath, apiKey)

	// Build Gemini request (same as Generate)
	var geminiContents []models.GeminiContent
	var systemInstruction *models.GeminiContent
	for _, msg := range req.History {
		if msg.Role == messaging.RoleSystem {
			if systemInstruction == nil {
				systemInstruction = &models.GeminiContent{Parts: []models.GeminiPart{}}
			}
			systemInstruction.Parts = append(systemInstruction.Parts, models.GeminiPart{Text: msg.Content})
			continue
		}
		parts := []models.GeminiPart{}
		if msg.Content != "" {
			if msg.ThoughtSignature != "" {
				parts = append(parts, models.GeminiPart{
					Text:             msg.Content,
					ThoughtSignature: msg.ThoughtSignature,
				})
			} else {
				parts = append(parts, models.GeminiPart{Text: msg.Content})
			}
		}
		// Handle attachments (R6.1)
		for _, att := range msg.Attachments {
			if isGeminiSupportedMime(att.Type) {
				parts = append(parts, models.GeminiPart{
					InlineData: &models.GeminiInlineData{
						MimeType: att.Type,
						Data:     att.Data,
					},
				})
			} else {
				parts = append(parts, models.GeminiPart{
					Text: fmt.Sprintf("\n[Hệ thống: Đã đính kèm tệp tin '%s' (loại: %s). Hiện tại tôi chưa thể đọc trực tiếp nội dung loại tệp này qua InlineData.]", att.Name, att.Type),
				})
			}
		}
		for _, tc := range msg.ToolCalls {
			parts = append(parts, models.GeminiPart{
				FunctionCall: &models.GeminiFunctionCall{
					Name: tc.Name,
					Args: tc.Args,
				},
				ThoughtSignature: msg.ThoughtSignature,
			})
		}
		if msg.Content == "" && len(msg.ToolCalls) == 0 && msg.ThoughtSignature != "" {
			parts = append(parts, models.GeminiPart{
				ThoughtSignature: msg.ThoughtSignature,
			})
		}
		for _, tr := range msg.ToolResponses {
			// Tool responses are already normalized to valid JSON by the dispatcher.
			var toolOutput map[string]interface{}
			if err := json.Unmarshal([]byte(tr.Content), &toolOutput); err != nil {
				toolOutput = map[string]interface{}{"content": tr.Content}
			}
			parts = append(parts, models.GeminiPart{
				FunctionResponse: &models.GeminiFunctionResponse{Name: tr.Name, Response: toolOutput},
			})
		}
		geminiRole := "user"
		switch msg.Role {
		case messaging.RoleAssistant:
			geminiRole = "model"
		case messaging.RoleTool:
			geminiRole = "tool"
		}
		geminiContents = append(geminiContents, models.GeminiContent{Role: geminiRole, Parts: parts})
	}

	var functionDeclarations []models.GeminiFunctionDeclaration
	if len(req.Tools) > 0 {
		for _, t := range req.Tools {
			var params map[string]interface{}
			if t.Parameters != nil {
				params = t.Parameters
			}
			functionDeclarations = append(functionDeclarations, models.GeminiFunctionDeclaration{
				Name: t.Name, Description: t.Description, Parameters: params,
			})
		}
	}

	geminiReq := models.GeminiRequest{
		Contents:          geminiContents,
		SystemInstruction: systemInstruction,
		SafetySettings: []models.SafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
		},
	}
	if len(functionDeclarations) > 0 {
		geminiReq.Tools = []models.GeminiTool{{FunctionDeclarations: functionDeclarations}}
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return fmt.Errorf("error marshalling gemini stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating gemini stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// TODO: Implement key rotation for pool
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error performing gemini stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini stream HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Gemini SSE streaming: each line is "data: {json}"
	scanner := bufio.NewScanner(resp.Body)
	var fullText strings.Builder
	var accumulatedToolCalls []messaging.ToolCall
	var accumulatedThoughtSignature string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var gResp models.GeminiResponse
		if err := json.Unmarshal([]byte(data), &gResp); err != nil {
			// Silently skip malformed SSE frames
			continue
		}
		if len(gResp.Candidates) == 0 {
			continue
		}
		for _, part := range gResp.Candidates[0].Content.Parts {
			if part.ThoughtSignature != "" {
				accumulatedThoughtSignature = part.ThoughtSignature
			}
			if part.Text != "" {
				fullText.WriteString(part.Text)
				onChunk(StreamChunk{Text: part.Text, ThoughtSignature: part.ThoughtSignature})
			}
			if part.FunctionCall != nil {
				callID := fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, len(accumulatedToolCalls))
				accumulatedToolCalls = append(accumulatedToolCalls, messaging.ToolCall{
					ID:   callID,
					Name: part.FunctionCall.Name,
					Args: part.FunctionCall.Args,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("gemini stream scanner error: %w", err)
	}

	onChunk(StreamChunk{
		Done:             true,
		Text:             fullText.String(),
		ToolCalls:        accumulatedToolCalls,
		ThoughtSignature: accumulatedThoughtSignature,
	})
	return nil
}

func isGeminiSupportedMime(mime string) bool {
	m := strings.ToLower(mime)
	// Images
	if strings.HasPrefix(m, "image/") {
		return true
	}
	// Video
	if strings.HasPrefix(m, "video/") {
		return true
	}
	// Audio
	if strings.HasPrefix(m, "audio/") {
		return true
	}
	// Document: PDF
	if m == "application/pdf" {
		return true
	}
	// Text formats
	textMimes := []string{
		"text/plain", "text/html", "text/css", "text/javascript",
		"text/x-typescript", "text/csv", "text/markdown", "text/x-python",
		"text/x-go", "application/json", "application/xml",
	}
	for _, tm := range textMimes {
		if m == tm {
			return true
		}
	}
	return false
}

// NewGeminiProvider creates a Gemini provider with the given API key and model
func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		APIKey: apiKey,
		Model:  normalizeGeminiModel(os.Getenv("GEMINI_MODEL")),
	}
}

// NewGeminiProviderWithPool creates a Gemini provider that uses a key pool
func NewGeminiProviderWithPool(keys []string) *GeminiProvider {
	return &GeminiProvider{
		APIKey:  "",
		Keys:    keys,
		Current: 0,
		Model:   normalizeGeminiModel(os.Getenv("GEMINI_MODEL")),
	}
}

// normalizeGeminiModel ensures the model name starts with "models/" prefix
func normalizeGeminiModel(model string) string {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return "gemini-3.1-flash-lite" // Updated default for 2026
	}
	if !strings.HasPrefix(normalized, "models/") {
		return "models/" + normalized
	}
	return normalized
}
