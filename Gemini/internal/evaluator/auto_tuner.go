package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply   string `json:"reply"`
	Metrics struct {
		LatencyMs int64  `json:"latency_ms"`
		TokenIn   int    `json:"token_in"`
		TokenOut  int    `json:"token_out"`
	} `json:"metrics"`
	Error string `json:"error,omitempty"`
}

type TestCase struct {
	Query string
	Type  string // "casual", "complex"
}

func main() {
	fmt.Println("🚀 BẮT ĐẦU VÒNG LẶP ĐÁNH GIÁ VÀ TỐI ƯU TỰ ĐỘNG...")
	
	tests := []TestCase{
		{"Hello bạn", "casual"},
		{"Bạn là ai?", "casual"},
		{"Phân tích chi phí dự phòng của VCB năm 2023", "complex"},
		{"Tổng quan ngành ngân hàng hôm nay", "complex"},
	}

	totalLatency := int64(0)
	totalTokens := 0
	failed := 0

	for i, t := range tests {
		fmt.Printf("\n▶️ [Test %d] Query: '%s' (Type: %s)\n", i+1, t.Query, t.Type)
		
		reqBody, _ := json.Marshal(ChatRequest{Message: t.Query})
		start := time.Now()
		
		// Gửi request đến server đang chạy
		resp, err := http.Post("http://localhost:8080/api/chat", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Printf("❌ Lỗi kết nối: %v\n", err)
			failed++
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		var chatResp ChatResponse
		json.Unmarshal(body, &chatResp)
		
		realLatencyMs := time.Since(start).Milliseconds()
		
		if chatResp.Error != "" {
			fmt.Printf("❌ Lỗi Backend: %s\n", chatResp.Error)
			failed++
			if strings.Contains(chatResp.Error, "Quota") || strings.Contains(chatResp.Error, "Rate limit") || strings.Contains(chatResp.Error, "Tất cả các dịch vụ") {
				fmt.Println("⚠️ Dừng vòng lặp vì các provider đã hết Quota/Token.")
				break
			}
			continue
		}

		fmt.Printf("   ✅ Trả lời: %s...\n", strings.ReplaceAll(chatResp.Reply[:min(100, len(chatResp.Reply))], "\n", " "))
		fmt.Printf("   ⏱ Độ trễ API: %dms | ⏱ Độ trễ thực tế: %dms\n", chatResp.Metrics.LatencyMs, realLatencyMs)
		fmt.Printf("   🪙 Tokens: IN=%d, OUT=%d\n", chatResp.Metrics.TokenIn, chatResp.Metrics.TokenOut)

		totalLatency += realLatencyMs
		totalTokens += chatResp.Metrics.TokenIn + chatResp.Metrics.TokenOut
	}

	fmt.Println("\n=========================================")
	fmt.Println("📊 KẾT QUẢ ĐÁNH GIÁ (BASELINE)")
	fmt.Printf("Tổng số Test: %d | Thất bại: %d\n", len(tests), failed)
	if len(tests)-failed > 0 {
		fmt.Printf("Độ trễ trung bình: %dms\n", totalLatency/int64(len(tests)-failed))
		fmt.Printf("Token sử dụng trung bình: %d\n", totalTokens/(len(tests)-failed))
	}
	fmt.Println("=========================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
