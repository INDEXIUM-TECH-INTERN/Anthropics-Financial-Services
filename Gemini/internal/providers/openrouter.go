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
	req.Header.Set("HTTP-Referer", "https://github.com/anthropics/financial-services-simulation") // Required by OpenRouter
	req.Header.Set("X-Title", "Gemini CLI Financial Simulation")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Printf("❌ [OpenRouter Error] Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return "", fmt.Errorf("OpenRouter API error: %d", resp.StatusCode)
	}


	var oResp models.OpenRouterResponse
	json.Unmarshal(body, &oResp)


	if oResp.Error.Message != "" {
		return "", fmt.Errorf(oResp.Error.Message)
	}
	if len(oResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenRouter")
	}

	return strings.TrimSpace(oResp.Choices[0].Message.Content), nil
}

func (p *OpenRouterProvider) Call(systemPrompt string, history []models.GeminiContent, tools ...models.Parameters) (models.GeminiContent, error) {
	fmt.Println("🔄 [Hệ thống] Đang gửi yêu cầu đến OpenRouter...")
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
			Function: models.OpenRouterFunctionDeclaration{
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

		for _, part := range h.Parts {
			if part.Text != "" {
				msg.Content = part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				msg.ToolCalls = append(msg.ToolCalls, models.OpenRouterToolCall{
					ID:   part.FunctionCall.ID,
					Type: "function",
					Function: models.OpenRouterFunction{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		messages = append(messages, msg)
	}

	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "google/gemini-2.0-flash-001"
	}

	reqBody := models.OpenRouterRequest{
		Model:    model,
		Messages: messages,
		Tools:    orTools,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/anthropics/financial-services-simulation")
	req.Header.Set("X-Title", "Gemini CLI Financial Simulation")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.GeminiContent{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		fmt.Printf("❌ [OpenRouter Error] Status: %d, Body: %s\n", resp.StatusCode, string(body))
		return models.GeminiContent{}, fmt.Errorf("OpenRouter API error: %d", resp.StatusCode)
	}

	var oResp models.OpenRouterResponse
	json.Unmarshal(body, &oResp)


	if oResp.Error.Message != "" {
		return models.GeminiContent{}, fmt.Errorf(oResp.Error.Message)
	}
	if len(oResp.Choices) == 0 {
		return models.GeminiContent{}, fmt.Errorf("no choices returned from OpenRouter")
	}

	oMsg := oResp.Choices[0].Message
	res := models.GeminiContent{Role: "model"}
	if oMsg.Content != "" {
		res.Parts = append(res.Parts, models.GeminiPart{Text: oMsg.Content})
	}
	for _, tc := range oMsg.ToolCalls {
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
