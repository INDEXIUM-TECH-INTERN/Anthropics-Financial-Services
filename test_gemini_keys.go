package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gemini-cli/internal/providers"
	"gemini-cli/internal/models/messaging"
)

func main() {
	fmt.Println("🧪 [TEST] Bắt đầu test 5 Gemini API keys...")
	fmt.Println(strings.Repeat("=", 50))

	// Đọc API keys từ environment
	keys := []string{
		os.Getenv("GEMINI_API_KEY"),
		os.Getenv("GEMINI_API_KEY_2"),
		os.Getenv("GEMINI_API_KEY_3"),
		os.Getenv("GEMINI_API_KEY_4"),
		os.Getenv("GEMINI_API_KEY_5"),
	}

	var validKeys []string
	var invalidKeys []string

	// Test từng key
	for i, key := range keys {
		if key == "" {
			fmt.Printf("❌ [Key %d] Trống hoặc không được set\n", i+1)
			continue
		}

		fmt.Printf("🔍 [Key %d] Đang test...\n", i+1)

		// Tạo provider với key này
		provider := &providers.GeminiProvider{
			APIKey: key,
			Model:  "models/gemini-2.0-flash", // Dùng model nhẹ để test
		}

		// Test message đơn giản
		req := messaging.Request{
			History: []messaging.Message{
				{Role: messaging.RoleUser, Content: "Xin chào, hãy chỉ trả lời 'OK'"},
			},
		}

		start := time.Now()
		resp, err := provider.Generate(context.Background(), req)
		duration := time.Since(start)

		if err != nil {
			errorMsg := err.Error()
			fmt.Printf("❌ [Key %d] Lỗi: %s (Thời gian: %v)\n", i+1, errorMsg, duration)

			// Phân loại lỗi
			if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "invalid API key") {
				fmt.Printf("   → Lỗi API key không hợp lệ\n")
			} else if strings.Contains(errorMsg, "quota") || strings.Contains(errorMsg, "429") {
				fmt.Printf("   → Hết hạn mức (quota exceeded)\n")
			} else if strings.Contains(errorMsg, "503") || strings.Contains(errorMsg, "overloaded") {
				fmt.Printf("   → Dịch vụ quá tải\n")
			} else {
				fmt.Printf("   → Lỗi không xác định\n")
			}

			invalidKeys = append(invalidKeys, fmt.Sprintf("Key %d: %s", i+1, errorMsg))
			continue
		}

		// Kiểm tra response hợp lệ
		if resp.Content == "" {
			fmt.Printf("❌ [Key %d] Response trống\n", i+1)
			invalidKeys = append(invalidKeys, fmt.Sprintf("Key %d: Response trống", i+1))
			continue
		}

		// Check if response contains OK as requested
		if strings.Contains(resp.Content, "OK") {
			fmt.Printf("✅ [Key %d] HOẠT ĐỘNG tốt - Response: '%s' (Thời gian: %v)\n", i+1, resp.Content, duration)
			validKeys = append(validKeys, fmt.Sprintf("Key %d", i+1))
		} else {
			fmt.Printf("⚠️ [Key %2d] Hoạt động nhưng response không mong đợi - Response: '%s' (Thời gian: %v)\n", i+1, resp.Content, duration)
			validKeys = append(validKeys, fmt.Sprintf("Key %d (response khác kỳ)", i+1))
		}
	}

	// Tổng kết
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 [KẾT QUẢ TỔNG KẾT]")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("✅ Số key hợp lệ: %d/%d\n", len(validKeys), len(keys))
	fmt.Printf("❌ Số key không hợp lệ: %d/%d\n", len(invalidKeys), len(keys))

	if len(validKeys) > 0 {
		fmt.Println("\n🎯 Key hoạt động:")
		for _, key := range validKeys {
			fmt.Printf("   ✓ %s\n", key)
		}
	}

	if len(invalidKeys) > 0 {
		fmt.Println("\n🚨 Key lỗi:")
		for _, key := range invalidKeys {
			fmt.Printf("   ✗ %s\n", key)
		}
	}

	// Đánh giá tổng thể
	fmt.Println("\n📋 [ĐÁNH GIÁ]")
	if len(validKeys) == len(keys) {
		fmt.Println("🎉 TẤT CẢ CÁC API KEY ĐỀU HOẠT ĐỘNG TỐT!")
	} else if len(validKeys) >= 3 {
		fmt.Println("👍 ĐẠT: Hầu hết API keys đều hoạt động, đủ dùng cho production")
	} else if len(validKeys) >= 1 {
		fmt.Println("⚠️ CẦN CẢI TIẾN: Chỉ có 1-2 keys hoạt động, nên bổ sung thêm")
	} else {
		fmt.Println("❌ KHÔNG ĐƯỢC: Không có API key nào hoạt động, cần kiểm tra ngay!")
	}
}