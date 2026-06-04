package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gemini-cli/internal/models"
	"gemini-cli/internal/models/messaging"
)

// Free-tier model fallback chain for OpenRouter
// When primary model fails, we cycle through these alternatives.
var openRouterFreeModels = []string{
	"nvidia/nemotron-3-super-120b-a12b:free",
	"google/gemini-flash-1.5:free",
	"meta-llama/llama-3.3-70b-instruct:free",
	"mistralai/mistral-7b-instruct:free",
	"qwen/qwen-2.5-72b-instruct:free",
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
		// Only try next model on specific transient / model-unavailable errors
		if strings.Contains(errStr, "Provider returned error") ||
			strings.Contains(errStr, "model not found") ||
			strings.Contains(errStr, "overloaded") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "502") {
			fmt.Printf("⚠️ [OpenRouter] Model %s failed (%v), trying next...\n", m, err)
			continue
		}
		// For other errors (auth, quota, bad request) stop immediately
		break
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
		msg := models.OpenRouterMessage{
			Role:    string(h.Role),
			Content: h.Content,
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

	client := &http.Client{Timeout: 100 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return messaging.Message{}, fmt.Errorf("error performing http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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
