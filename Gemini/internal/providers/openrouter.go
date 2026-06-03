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

type OpenRouterProvider struct {
	APIKey string
	Model  string
}

func (p *OpenRouterProvider) Generate(ctx context.Context, req messaging.Request) (messaging.Message, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "google/gemini-1.5-flash"
	}

	// 1. Translate tools from messaging.ToolSchema to models.OpenRouterTool
	var orTools []models.OpenRouterTool
	for _, t := range req.Tools {
		var params models.Parameters
		// Convert map[string]interface{} to the concrete struct via marshaling
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
	httpReq.Header.Set("HTTP-Referer", "https://github.com/kichsatst/Gemini-CLI")
	httpReq.Header.Set("X-Title", "Gemini CLI")

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

	var orResp models.OpenRouterResponse
	if err := json.Unmarshal(body, &orResp); err != nil {
		return messaging.Message{}, fmt.Errorf("error unmarshalling openrouter response: %s", string(body))
	}

	if orResp.Error.Message != "" {
		return messaging.Message{}, fmt.Errorf("openrouter API error: %s", orResp.Error.Message)
	}
	if len(orResp.Choices) == 0 {
		return messaging.Message{}, fmt.Errorf("no choices returned from OpenRouter")
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
			// In case of malformed arguments, we skip this tool call but don't fail the whole request
			// A log would be appropriate here in a real application
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
