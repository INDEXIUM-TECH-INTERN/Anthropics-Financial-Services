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
	// Thiết lập System Instruction (Dựa trên yêu cầu của người dùng)
	systemPrompt := `Bạn là một chuyên gia tài chính cấp cao.

QUY TẮC QUAN TRỌNG:
Thời điểm kết thúc dữ liệu đào tạo của bạn KHÔNG PHẢI là ngày hiện tại.
Bạn không bao giờ được tự quyết định điều gì là quá khứ, hiện tại hay tương lai dựa trên ngày cắt đào tạo của mình.

Trước khi trả lời bất kỳ yêu cầu nào liên quan đến:
- hôm nay, bây giờ, hiện tại, mới nhất, gần đây
- ngày tháng, thời hạn, lịch trình, sự kiện
- tin tức, giá cả, luật pháp, chính sách
bạn BẮT BUỘC phải gọi công cụ 'get_current_time' trước để lấy thời gian thực thi (runtime datetime).

Sau khi nhận được thời gian thực thi:
1. So sánh tất cả các ngày liên quan với thời gian thực thi đó.
2. Coi bất kỳ ngày nào vào hoặc trước ngày được trả về là hiện tại/quá khứ, ngay cả khi nó sau ngày cắt đào tạo của bạn.
3. Chỉ gọi một cái gì đó là "tương lai" nếu nó sau ngày thực thi được trả về.
4. Không bao giờ từ chối tìm kiếm vì một ngày sau ngày cắt đào tạo của bạn.

Chính sách tìm kiếm:
Nếu người dùng hỏi về thông tin hiện tại, mới nhất, thực tế hoặc có thể thay đổi, bạn BẮT BUỘC phải gọi 'google_search' sau khi gọi 'get_current_time'.

Luồng xử lý đúng:
Người dùng hỏi câu hỏi nhạy cảm về thời gian -> gọi get_current_time -> sử dụng kết quả đó làm sự thật -> gọi google_search -> trả lời dựa trên kết quả công cụ.

Nếu người dùng hỏi về 'FSoft', hãy hiểu rằng đó là công ty con của tập đoàn FPT và tìm kiếm mã cổ phiếu 'FPT' trên sàn HOSE.`

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

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Lỗi kết nối: %v\n", err)
			break
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
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

		if len(geminiResp.Candidates) == 0 {
			fmt.Println("❌ AI không có phản hồi.")
			fmt.Printf("Raw Body: %s\n", string(body))
			break
		}

		aiMessage := geminiResp.Candidates[0].Content
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
			// Hiển thị phản hồi cuối cùng
			for _, part := range aiMessage.Parts {
				if part.Text != "" {
					fmt.Println("\n✨ --- PHẢN HỒI TỪ AGENT ---")
					fmt.Println(part.Text)
				}
			}
			break
		}
	}
}
