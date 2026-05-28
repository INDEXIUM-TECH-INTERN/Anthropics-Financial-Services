package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"gemini-cli/internal/models"
)

type OpenRouterProvider struct {
	APIKey string
	Model  string
}

func (p *OpenRouterProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "google/gemini-2.0-flash-001"
	}

	reqBody := models.OpenRouterRequest{
		Model: model,
		Messages: []models.OpenRouterMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/kichsatst/Gemini-CLI")
	req.Header.Set("X-Title", "Gemini CLI")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var orResp models.OpenRouterResponse
	json.Unmarshal(body, &orResp)

	if orResp.Error.Message != "" {
		return "", fmt.Errorf("%s", orResp.Error.Message)
	}
	if len(orResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return strings.TrimSpace(orResp.Choices[0].Message.Content), nil
}

func (p *OpenRouterProvider) Call(systemPrompt string, history []models.GeminiContent, tools ...models.Parameters) (models.GeminiContent, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	var orTools []models.OpenRouterTool
	for _, t := range tools {
		cleanParams := models.Parameters{
			Type:       t.Type,
			Properties: t.Properties,
			Required:   t.Required,
		}
		orTools = append(orTools, models.OpenRouterTool{
			Type: "function",
			Function: models.OpenRouterFunctionDeclare{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  cleanParams,
			},
		})
	}

	var messages []models.OpenRouterMessage
	messages = append(messages, models.OpenRouterMessage{Role: "system", Content: systemPrompt})

	for _, h := range history {
		msg := models.OpenRouterMessage{Role: h.Role}
		if h.Role == "model" {
			msg.Role = "assistant"
		} else if h.Role == "function" {
			msg.Role = "tool"
			msg.ToolCallID = h.Parts[0].FunctionResponse.ID
			msg.Content = fmt.Sprintf("%v", h.Parts[0].FunctionResponse.Response["content"])
		}

		for _, p := range h.Parts {
			if p.Text != "" {
				msg.Content = p.Text
			}
			if p.FunctionCall != nil {
				argsJSON, _ := json.Marshal(p.FunctionCall.Args)
				msg.ToolCalls = append(msg.ToolCalls, models.OpenRouterToolCall{
					ID:   p.FunctionCall.ID,
					Type: "function",
					Function: models.OpenRouterFunction{
						Name:      p.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		messages = append(messages, msg)
	}

	reqBody := models.OpenRouterRequest{
		Model:       p.Model,
		Messages:    messages,
		Tools:       orTools,
		MaxTokens:   4000,
		Temperature: 0.7,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/kichsatst/Gemini-CLI")
	req.Header.Set("X-Title", "Gemini CLI")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.GeminiContent{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var orResp models.OpenRouterResponse
	json.Unmarshal(body, &orResp)

	if orResp.Error.Message != "" {
		return models.GeminiContent{}, fmt.Errorf("%s", orResp.Error.Message)
	}
	if len(orResp.Choices) == 0 {
		return models.GeminiContent{}, fmt.Errorf("no choices returned from OpenRouter")
	}

	orMsg := orResp.Choices[0].Message
	res := models.GeminiContent{Role: "model"}
	if orMsg.Content != "" {
		res.Parts = append(res.Parts, models.GeminiPart{Text: orMsg.Content})
	}
	for _, tc := range orMsg.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		res.Parts = append(res.Parts, models.GeminiPart{
			FunctionCall: &models.GeminiFunctionCall{
				Name: tc.Function.Name,
				Args: args,
				ID:   tc.ID,
			},
		})
	}

	return res, nil
}
