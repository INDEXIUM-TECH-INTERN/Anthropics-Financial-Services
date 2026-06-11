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

// Free-tier model fallback chain for OpenRouter
// When primary model fails, we cycle through these alternatives.
var openRouterFreeModels = []string{
	"google/gemini-2.0-flash:free",
	"google/gemini-flash-1.5:free",
	"mistralai/mistral-7b-instruct:free",
	"qwen/qwen-2.5-72b-instruct:free",
	"meta-llama/llama-3.3-70b-instruct:free",
	"nvidia/nemotron-3-super-120b-a12b:free",
}

type OpenRouterProvider struct {
	APIKey string
	Model  string
}

func (p *OpenRouterProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = openRouterFreeModels[0]
	}

	// Try the configured model first, then fallback chain
	modelsToTry := []string{model}
	for _, m := range openRouterFreeModels {
		if m != model {
			modelsToTry = append(modelsToTry, m)
		}
	}

	var lastErr error
	for i, m := range modelsToTry {
		msg, err := p.generateWithModel(ctx, req, m)
		if err == nil {
			if i > 0 {
				fmt.Printf("🔀 [OpenRouter] Succeeded with fallback model: %s\n", m)
			}
			return msg, nil
		}
		lastErr = err
		errStr := err.Error()
		// Try next model on transient errors, rate limits, and quota exhaustion
		if strings.Contains(errStr, "Provider returned error") ||
			strings.Contains(errStr, "model not found") ||
			strings.Contains(errStr, "overloaded") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "502") ||
			strings.Contains(errStr, "429") ||
			strings.Contains(errStr, "rate limit") ||
			strings.Contains(errStr, "quota") ||
			strings.Contains(errStr, "exceeded") {
			fmt.Printf("⚠️ [OpenRouter] Model %s failed (%v), trying next...\n", m, err)
			continue
		}
		// For auth errors, stop immediately
		if strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "invalid api key") {
			break
		}
		// Unknown error — try next model anyway
		fmt.Printf("⚠️ [OpenRouter] Model %s failed (%v), trying next...\n", m, err)
		continue
	}
	return messaging.Message{}, lastErr
}

func (p *OpenRouterProvider) generateWithModel(ctx context.Context, req messaging.Request, model string) (messaging.Message, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	// 1. Translate tools from messaging.ToolSchema to models.OpenRouterTool
	var orTools []models.OpenRouterTool
	for _, t := range req.Tools {
		var params models.Parameters
		paramBytes, err := json.Marshal(t.Parameters)
		if err != nil {
			return messaging.Message{}, fmt.Errorf("error marshalling tool parameters for '%s': %w", t.Name, err)
		}
		if err := json.Unmarshal(paramBytes, &params); err != nil {
			return messaging.Message{}, fmt.Errorf("error unmarshalling tool parameters for '%s': %w", t.Name, err)
		}

		orTools = append(orTools, models.OpenRouterTool{
			Type: "function",
			Function: models.OpenRouterFunctionDeclare{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	// 2. Translate history from []messaging.Message to []models.OpenRouterMessage
	var messages []models.OpenRouterMessage
	for _, h := range req.History {
		content := h.Content

		// Handle attachments by appending to content (as many OR models are text-only or have varied multi-modal APIs)
		if len(h.Attachments) > 0 {
			var attNotes []string
			for _, att := range h.Attachments {
				attNotes = append(attNotes, fmt.Sprintf("[Đính kèm: %s (%s)]", att.Name, att.Type))
			}
			content += "\n" + strings.Join(attNotes, "\n")
		}

		msg := models.OpenRouterMessage{
			Role:    string(h.Role),
			Content: content,
		}

		// OpenRouter uses "assistant" for model responses
		if h.Role == messaging.RoleAssistant {
			msg.Role = "assistant"

			// Handle tool calls initiated by the assistant
			if len(h.ToolCalls) > 0 {
				msg.Content = "" // Content should be empty if there are tool calls
				for _, tc := range h.ToolCalls {
					argsJSON, err := json.Marshal(tc.Args)
					if err != nil {
						return messaging.Message{}, fmt.Errorf("error marshalling tool call arguments for '%s': %w", tc.Name, err)
					}
					msg.ToolCalls = append(msg.ToolCalls, models.OpenRouterToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: models.OpenRouterFunction{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}

		// Handle tool responses from the user/runtime
		if h.Role == messaging.RoleTool {
			msg.Role = "tool"
			if len(h.ToolResponses) > 0 {
				tr := h.ToolResponses[0] // OpenRouter expects one tool response per message
				msg.ToolCallID = tr.CallID
				msg.Content = tr.Content
			}
		}

		messages = append(messages, msg)
	}

	reqBody := models.OpenRouterRequest{
		Model:       model,
		Messages:    messages,
		Tools:       orTools,
		MaxTokens:   4000,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error marshalling openrouter request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error creating http request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://github.com/INDEXIUM-TECH-INTERN/Anthropics-Financial-Services")
	httpReq.Header.Set("X-Title", "Indexium Financial AI Agent")

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

	// Log raw response for debugging when status >= 400
	if resp.StatusCode >= 400 {
		fmt.Printf("⚠️ [OpenRouter] HTTP %d from model %s — Body: %s\n", resp.StatusCode, model, string(body))
	}

	var orResp models.OpenRouterResponse
	if err := json.Unmarshal(body, &orResp); err != nil {
		return messaging.Message{}, fmt.Errorf("error unmarshalling openrouter response (model=%s): body=%s", model, string(body))
	}

	if orResp.Error.Message != "" {
		return messaging.Message{}, fmt.Errorf("openrouter API error (model=%s): %s", model, orResp.Error.Message)
	}
	if len(orResp.Choices) == 0 {
		return messaging.Message{}, fmt.Errorf("no choices returned from OpenRouter (model=%s)", model)
	}

	// 3. Translate response from models.OpenRouterMessage to messaging.Message
	orMsg := orResp.Choices[0].Message
	responseMsg := messaging.Message{
		Role:    messaging.RoleAssistant,
		Content: orMsg.Content,
	}

	for _, tc := range orMsg.ToolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			continue
		}
		responseMsg.ToolCalls = append(responseMsg.ToolCalls, messaging.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}

	return responseMsg, nil
}

func (p *OpenRouterProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
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

func (p *OpenRouterProvider) GenerateStream(ctx context.Context, req messaging.Request, onChunk func(StreamChunk)) error {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = openRouterFreeModels[0]
	}

	modelsToTry := []string{model}
	for _, m := range openRouterFreeModels {
		if m != model {
			modelsToTry = append(modelsToTry, m)
		}
	}

	var lastErr error
	for _, m := range modelsToTry {
		err := p.streamWithModel(ctx, req, m, onChunk)
		if err == nil {
			return nil
		}
		lastErr = err
		errStr := err.Error()
		if strings.Contains(errStr, "Provider returned error") ||
			strings.Contains(errStr, "model not found") ||
			strings.Contains(errStr, "overloaded") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "502") ||
			strings.Contains(errStr, "429") ||
			strings.Contains(errStr, "rate limit") ||
			strings.Contains(errStr, "quota") ||
			strings.Contains(errStr, "exceeded") {
			fmt.Printf("⚠️ [OpenRouter] Stream model %s failed (%v), trying next...\n", m, err)
			continue
		}
		if strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "invalid api key") {
			break
		}
		fmt.Printf("⚠️ [OpenRouter] Stream model %s failed (%v), trying next...\n", m, err)
		continue
	}
	return lastErr
}

func (p *OpenRouterProvider) streamWithModel(ctx context.Context, req messaging.Request, model string, onChunk func(StreamChunk)) error {
	url := "https://openrouter.ai/api/v1/chat/completions"

	var orTools []models.OpenRouterTool
	for _, t := range req.Tools {
		var params models.Parameters
		paramBytes, _ := json.Marshal(t.Parameters)
		json.Unmarshal(paramBytes, &params)
		orTools = append(orTools, models.OpenRouterTool{
			Type: "function",
			Function: models.OpenRouterFunctionDeclare{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	var messages []models.OpenRouterMessage
	for _, h := range req.History {
		content := h.Content
		if len(h.Attachments) > 0 {
			var attNotes []string
			for _, att := range h.Attachments {
				attNotes = append(attNotes, fmt.Sprintf("[Đính kèm: %s (%s)]", att.Name, att.Type))
			}
			content += "\n" + strings.Join(attNotes, "\n")
		}
		msg := models.OpenRouterMessage{Role: string(h.Role), Content: content}
		if h.Role == messaging.RoleAssistant {
			msg.Role = "assistant"
			if len(h.ToolCalls) > 0 {
				msg.Content = ""
				for _, tc := range h.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Args)
					msg.ToolCalls = append(msg.ToolCalls, models.OpenRouterToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: models.OpenRouterFunction{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}
		}
		if h.Role == messaging.RoleTool {
			msg.Role = "tool"
			if len(h.ToolResponses) > 0 {
				msg.ToolCallID = h.ToolResponses[0].CallID
				msg.Content = h.ToolResponses[0].Content
			}
		}
		messages = append(messages, msg)
	}

	reqBody := models.OpenRouterRequest{
		Model:       model,
		Messages:    messages,
		Tools:       orTools,
		MaxTokens:   4000,
		Temperature: 0.7,
		Stream:      true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshalling openrouter stream request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating stream request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("HTTP-Referer", "https://github.com/INDEXIUM-TECH-INTERN/Anthropics-Financial-Services")
	httpReq.Header.Set("X-Title", "Indexium Financial AI Agent")
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0} // no timeout for streaming
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error performing stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openrouter stream HTTP %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var fullText strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk models.OpenRouterStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Silently skip malformed SSE frames
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" || delta == "null" {
			delta = chunk.Choices[0].Delta.Reasoning
		}
		if delta != "" {
			fullText.WriteString(delta)
			onChunk(StreamChunk{Text: delta})
		}
	}

	onChunk(StreamChunk{Done: true, Text: fullText.String()})
	return nil
}
