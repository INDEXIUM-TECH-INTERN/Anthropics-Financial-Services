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

type GeminiProvider struct {
	APIKey string
	Model  string
}

func (p *GeminiProvider) GenerateText(systemPrompt, userPrompt string) (string, error) {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "gemini-1.5-flash"
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent", modelPath)

	reqBody := models.GeminiRequest{
		Contents: []models.GeminiContent{
			{
				Role:  "user",
				Parts: []models.GeminiPart{{Text: userPrompt}},
			},
		},
		SystemInstruction: &models.GeminiContent{
			Parts: []models.GeminiPart{{Text: systemPrompt}},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var gResp models.GeminiResponse
	json.Unmarshal(body, &gResp)

	if resp.StatusCode >= http.StatusBadRequest {
		if gResp.Error.Message != "" {
			return "", fmt.Errorf(gResp.Error.Message)
		}
		return "", fmt.Errorf("gemini API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if gResp.Error.Message != "" {
		return "", fmt.Errorf(gResp.Error.Message)
	}
	if len(gResp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	var textParts []string
	for _, part := range gResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}
	return strings.TrimSpace(strings.Join(textParts, "\n")), nil
}

func (p *GeminiProvider) Call(systemPrompt string, history []models.GeminiContent, tools ...models.Parameters) (models.GeminiContent, error) {
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "gemini-1.5-flash"
	}
	modelPath := model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent", modelPath)

	var functionDeclarations []models.GeminiFunctionDeclaration
	for _, t := range tools {
		functionDeclarations = append(functionDeclarations, models.GeminiFunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters: map[string]interface{}{
				"type":       t.Type,
				"properties": t.Properties,
				"required":   t.Required,
			},
		})
	}

	reqBody := models.GeminiRequest{
		Contents:          history,
		SystemInstruction: &models.GeminiContent{Parts: []models.GeminiPart{{Text: systemPrompt}}},
		Tools: []models.GeminiTool{
			{
				FunctionDeclarations: functionDeclarations,
			},
		},

		SafetySettings: []models.SafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
			{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "BLOCK_ONLY_HIGH"},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.GeminiContent{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var gResp models.GeminiResponse
	json.Unmarshal(body, &gResp)

	if resp.StatusCode >= http.StatusBadRequest {
		if gResp.Error.Message != "" {
			return models.GeminiContent{}, fmt.Errorf(gResp.Error.Message)
		}
		return models.GeminiContent{}, fmt.Errorf("gemini API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if gResp.Error.Message != "" {
		return models.GeminiContent{}, fmt.Errorf(gResp.Error.Message)
	}
	if len(gResp.Candidates) == 0 {
		return models.GeminiContent{}, fmt.Errorf("no candidates returned")
	}

	res := gResp.Candidates[0].Content
	if res.Role == "" {
		res.Role = "model"
	}
	return res, nil
}
