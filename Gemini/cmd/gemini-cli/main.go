package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"gemini-cli/internal/models"
	"gemini-cli/internal/tools"
	"gemini-cli/internal/utils"
)

func main() {
	utils.LoadEnv()
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-flash-latest"
	}

	// Cấu hình các công cụ AI có thể gọi
	aiTools := []models.Tool{
		{
			FunctionDeclarations: []models.FunctionDeclaration{
				{
					Name:        "get_current_time",
					Description: "Lấy thời gian hiện tại của hệ thống (Runtime Datetime). BẮT BUỘC gọi trước khi trả lời các câu hỏi về 'hôm nay', 'hiện tại', hoặc các mốc thời gian khác.",
					Parameters: models.Parameters{
						Type:       "object",
						Properties: map[string]models.Property{},
						Required:   []string{},
					},
				},
				{
					Name:        "google_search",
					Description: "Lấy dữ liệu TÀI CHÍNH THỜI GIAN THỰC (giá cổ phiếu, tỷ giá, vốn hóa) và tin tức mới nhất từ internet. Hãy sử dụng công cụ này sau khi gọi get_current_time để lấy dữ liệu thực tế.",
					Parameters: models.Parameters{
						Type: "object",
						Properties: map[string]models.Property{
							"query": {Type: "string", Description: "Từ khóa tìm kiếm (VD: 'giá cổ phiếu FPT hôm nay')"},
						},
						Required: []string{"query"},
					},
				},
				{
					Name:        "load_financial_context",
					Description: "Nạp hướng dẫn chuyên môn cho Agent hoặc Skill từ hệ thống tài chính.",
					Parameters: models.Parameters{
						Type: "object",
						Properties: map[string]models.Property{
							"type": {Type: "string", Description: "Loại tài liệu: 'agent' hoặc 'skill'"},
							"name": {Type: "string", Description: "Tên tài liệu (ví dụ: 'pitch-agent', 'comps-analysis')"},
						},
						Required: []string{"type", "name"},
					},
				},
			},
		},
	}

	var history []models.Content
	// Thiết lập System Instruction (Đọc từ file bên ngoài)
	promptBytes, err := os.ReadFile("internal/prompt/system_prompt.txt")
	var systemPrompt string
	if err != nil {
		fmt.Printf("⚠️ Cảnh báo: Không thể đọc file system_prompt.txt, sử dụng prompt mặc định. Lỗi: %v\n", err)
		systemPrompt = "Bạn là một trợ lý tài chính thông minh."
	} else {
		systemPrompt = string(promptBytes)
	}

	systemInstruction := &models.Content{
		Parts: []models.Part{{Text: systemPrompt}},
	}

	// Nhận yêu cầu từ người dùng
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\n👤 Nhập yêu cầu tài chính của bạn: ")
	if !scanner.Scan() {
		return
	}
	userInput := scanner.Text()
	history = append(history, models.Content{Role: "user", Parts: []models.Part{{Text: userInput}}})

	// Vòng lặp Agentic (Suy nghĩ -> Gọi công cụ -> Trả lời)
	for {
		fmt.Println("🔄 [Hệ thống] Đang gửi yêu cầu đến Gemini...")
		reqBody := models.GeminiRequest{
			Contents:          history,
			Tools:             aiTools,
			SystemInstruction: systemInstruction,
		}
		jsonData, _ := json.Marshal(reqBody)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Lỗi kết nối: %v\n", err)
			break
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var geminiResp models.GeminiResponse
		err = json.Unmarshal(body, &geminiResp)
		if err != nil {
			fmt.Printf("❌ Lỗi giải mã JSON: %v\n", err)
			fmt.Printf("Raw Body: %s\n", string(body))
			break
		}

		if geminiResp.Error.Message != "" {
			fmt.Printf("❌ Lỗi API: %s\n", geminiResp.Error.Message)
			break
		}

		if geminiResp.UsageMetadata.TotalTokenCount > 0 {
			fmt.Printf("📊 [Token Usage] Prompt: %d | Response: %d | Total: %d\n", 
				geminiResp.UsageMetadata.PromptTokenCount, 
				geminiResp.UsageMetadata.CandidatesTokenCount, 
				geminiResp.UsageMetadata.TotalTokenCount)
		}

		if len(geminiResp.Candidates) == 0 {
			fmt.Println("⚠️ [Hệ thống] AI không trả về candidate nào.")
			fmt.Printf("Raw Body: %s\n", string(body))
			break
		}

		candidate := geminiResp.Candidates[0]
		if len(candidate.Content.Parts) == 0 {
			fmt.Println("⚠️ [Hệ thống] AI trả về candidate trống (có thể bị chặn bởi bộ lọc an toàn).")
			fmt.Printf("Raw Body: %s\n", string(body))
			break
		}

		aiMessage := candidate.Content
		if aiMessage.Role == "" {
			aiMessage.Role = "model" // Gemini yêu cầu role model cho phản hồi của nó
		}
		history = append(history, aiMessage)

		hasToolCall := false
		for _, part := range aiMessage.Parts {
			if part.FunctionCall != nil {
				hasToolCall = true
				fmt.Printf("🤖 [AI] Yêu cầu gọi công cụ: %s với args: %v\n", part.FunctionCall.Name, part.FunctionCall.Args)
				var result string
				if part.FunctionCall.Name == "google_search" {
					result = tools.SearchGoogle(part.FunctionCall.Args["query"].(string))
				} else if part.FunctionCall.Name == "load_financial_context" {
					result = tools.LoadDocument(part.FunctionCall.Args["type"].(string), part.FunctionCall.Args["name"].(string))
				} else if part.FunctionCall.Name == "get_current_time" {
					result = tools.GetCurrentTime()
				}

				// Gửi kết quả công cụ lại cho AI
				history = append(history, models.Content{
					Role: "function",
					Parts: []models.Part{{
						FunctionResponse: &models.FunctionResponse{
							Name:     part.FunctionCall.Name,
							Response: map[string]interface{}{"content": result},
						},
					}},
				})
			}
		}

		if !hasToolCall {
			foundText := false
			// Hiển thị phản hồi cuối cùng
			for _, part := range aiMessage.Parts {
				if part.Text != "" {
					foundText = true
					fmt.Println("\n✨ --- PHẢN HỒI TỪ AGENT ---")
					fmt.Println(part.Text)
				}
			}
			if !foundText {
				fmt.Println("⚠️ [Hệ thống] AI không trả lời bằng văn bản và cũng không gọi công cụ.")
				fmt.Printf("Raw Body: %s\n", string(body))
			}
			break
		}
	}
}
