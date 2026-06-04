package utils

import (
	"strings"
)

// EstimateTokens ước tính số token cho văn bản.
// Sử dụng heuristic đơn giản nhưng đáng tin cậy hơn len(text)/4.
// Dành cho tiếng Việt + tiếng Anh (tài chính).
// Mục tiêu: over-estimate nhẹ để tránh vượt context limit.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	charCount := len([]rune(text))
	wordCount := len(strings.Fields(text))

	// Heuristic an toàn cho mix VN/EN + markdown + JSON tool output
	byChar := int(float64(charCount) / 3.4)
	byWord := int(float64(wordCount) * 1.4)

	est := byChar
	if byWord > est {
		est = byWord
	}

	// Overhead cho cấu trúc (bullet, JSON, code block, tool response)
	overhead := charCount / 480 * 10
	est += overhead

	if est < 1 {
		est = 1
	}
	return est
}

// EstimateMessagesTokens ước tính tổng token của một danh sách messaging.Message
// (không import messaging để tránh circular dep - gọi từ nơi có messaging)
func EstimateMessagesTokens(getHistory func() []interface{ GetRoleAndContent() (string, string) }) int {
	// Cách đơn giản hơn: gọi trực tiếp từ orchestrator với chuỗi đã build
	return 0 // placeholder, sẽ implement trực tiếp ở nơi sử dụng
}

// EstimateFullPrompt ước tính token của toàn bộ prompt sẽ gửi đi (system + history + tool defs)
func EstimateFullPrompt(systemPrompt string, historyTexts []string, toolDefsText string) int {
	total := EstimateTokens(systemPrompt)
	for _, h := range historyTexts {
		total += EstimateTokens(h)
	}
	total += EstimateTokens(toolDefsText)
	// Thêm overhead cho role tags, formatting của provider
	total += len(historyTexts)*4 + 30
	return total
}

// TruncateTextForSummary cắt văn bản thành độ dài hợp lý trước khi đưa vào prompt tóm tắt
func TruncateTextForSummary(text string, maxChars int) string {
	if len([]rune(text)) <= maxChars {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxChars]) + "\n... [đã cắt bớt để tóm tắt]"
}
