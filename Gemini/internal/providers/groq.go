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

type GroqProvider struct {
	APIKey string
	Model  string
}

func (p *GroqProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	reqBody := models.GroqRequest{
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
	var qResp models.GroqResponse
	json.Unmarshal(body, &qResp)

	if qResp.Error.Message != "" {
		return "", fmt.Errorf(qResp.Error.Message)
	}
	if len(qResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return strings.TrimSpace(qResp.Choices[0].Message.Content), nil
}

func (p *GroqProvider) Call(systemPrompt string, history []models.GeminiContent, tools ...models.Parameters) (models.GeminiContent, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	var groqTools []models.GroqTool
	for _, t := range tools {
		cleanParams := models.Parameters{
			Type:       t.Type,
			Properties: t.Properties,
			Required:   t.Required,
		}
		groqTools = append(groqTools, models.GroqTool{
			Type: "function",
			Function: models.GroqFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  cleanParams,
			},
		})
	}

	// ... (history conversion logic stays the same)
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

		for _, p := range h.Parts {
			if p.Text != "" {
				msg.Content = p.Text
			}
			if p.FunctionCall != nil {
				argsJSON, _ := json.Marshal(p.FunctionCall.Args)
				msg.ToolCalls = append(msg.ToolCalls, models.GroqToolCall{
					ID:   p.FunctionCall.ID,
					Type: "function",
					Function: models.GroqFunction{
						Name:      p.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		messages = append(messages, msg)
	}

	reqBody := models.GroqRequest{
		Model:    p.Model,
		Messages: messages,
		Tools:    groqTools,
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
	var qResp models.GroqResponse
	json.Unmarshal(body, &qResp)

	if qResp.Error.Message != "" {
		return models.GeminiContent{}, fmt.Errorf(qResp.Error.Message)
	}
	if len(qResp.Choices) == 0 {
		return models.GeminiContent{}, fmt.Errorf("no choices returned")
	}

	// Convert Groq response back to Gemini format
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
