//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	totalLatency, totalTokens, failed := runTests(tests)
	printResults(len(tests), failed, totalLatency, totalTokens)
}

func runTests(tests []TestCase) (int64, int, int) {
	var totalLatency int64
	var totalTokens, failed int

	for i, t := range tests {
		fmt.Printf("\n▶️ [Test %d] Query: '%s' (Type: %s)\n", i+1, t.Query, t.Type)

		latency, tokens, err := runSingleTest(t)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			failed++
			if strings.Contains(err.Error(), "Quota") || strings.Contains(err.Error(), "Rate limit") {
				fmt.Println("⚠️ Dừng vòng lặp vì các provider đã hết Quota/Token.")
				break
			}
			continue
		}

		totalLatency += latency
		totalTokens += tokens
	}

	return totalLatency, totalTokens, failed
}

func runSingleTest(t TestCase) (int64, int, error) {
	reqBody, err := json.Marshal(ChatRequest{Message: t.Query})
	if err != nil {
		return 0, 0, fmt.Errorf("lỗi marshal: %v", err)
	}
	start := time.Now()

	resp, err := http.Post("http://localhost:8080/api/chat", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, 0, fmt.Errorf("lỗi kết nối: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, 0, fmt.Errorf("lỗi đọc response: %v", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return 0, 0, fmt.Errorf("lỗi parse JSON: %v", err)
	}

	realLatencyMs := time.Since(start).Milliseconds()

	if chatResp.Error != "" {
		return 0, 0, fmt.Errorf("lỗi Backend: %s", chatResp.Error)
	}

	fmt.Printf("   ✅ Trả lời: %s...\n", strings.ReplaceAll(chatResp.Reply[:min(100, len(chatResp.Reply))], "\n", " "))
	fmt.Printf("   ⏱ Độ trễ API: %dms | ⏱ Độ trễ thực tế: %dms\n", chatResp.Metrics.LatencyMs, realLatencyMs)
	fmt.Printf("   🪙 Tokens: IN=%d, OUT=%d\n", chatResp.Metrics.TokenIn, chatResp.Metrics.TokenOut)

	return realLatencyMs, chatResp.Metrics.TokenIn + chatResp.Metrics.TokenOut, nil
}

func printResults(total, failed int, totalLatency int64, totalTokens int) {
	fmt.Println("\n=========================================")
	fmt.Println("📊 KẾT QUẢ ĐÁNH GIÁ (BASELINE)")
	fmt.Printf("Tổng số Test: %d | Thất bại: %d\n", total, failed)
	if total-failed > 0 {
		fmt.Printf("Độ trễ trung bình: %dms\n", totalLatency/int64(total-failed))
		fmt.Printf("Token sử dụng trung bình: %d\n", totalTokens/(total-failed))
	}
	fmt.Println("=========================================")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
