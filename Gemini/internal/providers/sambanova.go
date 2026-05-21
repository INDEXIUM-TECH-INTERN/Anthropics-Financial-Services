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

type SambaNovaProvider struct {
	APIKey string
	Model  string
}

func (p *SambaNovaProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	url := "https://api.sambanova.ai/v1/chat/completions"

	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "Meta-Llama-3.1-70B-Instruct"
	}

	reqBody := models.GroqRequest{ // SambaNova uses OpenAI-compatible schema
		Model: model,
		Messages: []models.GroqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("SambaNova API error: %d - %s", resp.StatusCode, string(body))
	}

	var qResp models.GroqResponse
	json.Unmarshal(body, &qResp)

	if len(qResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from SambaNova")
	}

	return strings.TrimSpace(qResp.Choices[0].Message.Content), nil
}

func (p *SambaNovaProvider) Call(systemPrompt string, history []models.GeminiContent, tools ...models.Parameters) (models.GeminiContent, error) {
	fmt.Println("🔄 [Hệ thống] Đang gửi yêu cầu đến SambaNova...")
	url := "https://api.sambanova.ai/v1/chat/completions"

	// SambaNova supports tool calls with standard OpenAI format
	var snTools []models.GroqTool
	for _, t := range tools {
		cleanParams := models.Parameters{
			Type:       t.Type,
			Properties: t.Properties,
			Required:   t.Required,
		}
		snTools = append(snTools, models.GroqTool{
			Type: "function",
			Function: models.GroqFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  cleanParams,
			},
		})
	}

	var messages []models.GroqMessage
	messages = append(messages, models.GroqMessage{Role: "system", Content: systemPrompt})

	for _, h := range history {
		msg := models.GroqMessage{Role: h.Role}
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
				msg.ToolCalls = append(msg.ToolCalls, models.GroqToolCall{
					ID:   part.FunctionCall.ID,
					Type: "function",
					Function: models.GroqFunction{
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
		model = "Meta-Llama-3.1-70B-Instruct"
	}

	reqBody := models.GroqRequest{
		Model:    model,
		Messages: messages,
		Tools:    snTools,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.GeminiContent{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return models.GeminiContent{}, fmt.Errorf("SambaNova API error: %d - %s", resp.StatusCode, string(body))
	}

	var qResp models.GroqResponse
	json.Unmarshal(body, &qResp)

	if len(qResp.Choices) == 0 {
		return models.GeminiContent{}, fmt.Errorf("no choices returned from SambaNova")
	}

	qMsg := qResp.Choices[0].Message
	res := models.GeminiContent{Role: "model"}
	if qMsg.Content != "" {
		res.Parts = append(res.Parts, models.GeminiPart{Text: qMsg.Content})
	}
	for _, tc := range qMsg.ToolCalls {
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
