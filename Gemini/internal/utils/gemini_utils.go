package utils

import (
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// IsAPIKeyError kiểm tra nếu lỗi là do API key
func IsAPIKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "API_KEY_INVALID") ||
		strings.Contains(errStr, "API_KEY_EXPIRED") ||
		strings.Contains(errStr, "QUOTA_EXCEEDED") ||
		strings.Contains(errStr, "PERMISSION_DENIED")
}

// FormatGeminiResponse format response từ Gemini
func FormatGeminiResponse(resp *genai.GenerateContentResponse) string {
	var builder strings.Builder

	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					builder.WriteString(string(textPart))
				}
			}
		}
	}

	return builder.String()
}

// MinValue trả về giá trị nhỏ hơn của hai số
func MinValue(a, b int) int {
	if a < b {
		return a
	}
	return b
}