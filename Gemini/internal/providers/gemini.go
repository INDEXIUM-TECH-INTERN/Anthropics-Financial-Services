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
	"time"

	"gemini-cli/internal/models"
	"gemini-cli/internal/models/messaging"
)

type GeminiProvider struct {
	APIKey string
	Model  string
}

func (p *GeminiProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "gemini-2.0-flash" // Updated default for 2026
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	// Use v1beta as it supports systemInstruction and tools fields in the JSON payload.
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent", modelPath)

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
			var toolOutput map[string]interface{}
			// Try to unmarshal the content as JSON, if not, treat as text.
			if err := json.Unmarshal([]byte(tr.Content), &toolOutput); err != nil {
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
	httpReq.Header.Set("x-goog-api-key", p.APIKey)

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
		model = "gemini-2.0-flash"
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:streamGenerateContent?alt=sse", modelPath)

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
			parts = append(parts, models.GeminiPart{Text: msg.Content})
		}
		for _, tc := range msg.ToolCalls {
			parts = append(parts, models.GeminiPart{
				FunctionCall: &models.GeminiFunctionCall{Name: tc.Name, Args: tc.Args},
			})
		}
		for _, tr := range msg.ToolResponses {
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
	httpReq.Header.Set("x-goog-api-key", p.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0}
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
			if part.Text != "" {
				fullText.WriteString(part.Text)
				onChunk(StreamChunk{Text: part.Text})
			}
		}
	}

	onChunk(StreamChunk{Done: true, Text: fullText.String()})
	return nil
}
