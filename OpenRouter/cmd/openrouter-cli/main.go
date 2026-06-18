package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"openrouter-cli/internal/providers"
	"openrouter-cli/internal/models/messaging"
)

func main() {
	mode := flag.String("mode", "cli", "Chạy ở chế độ gì (cli/server)")
	model := flag.String("model", "", "Model OpenRouter muốn dùng")
	flag.Parse()

	// Đọc API keys từ environment
	keys := make([]string, 0)
	for i := 1; i <= 5; i++ {
		key := os.Getenv(fmt.Sprintf("OPENROUTER_API_KEY_%d", i))
		if key != "" {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		fmt.Println("❌ Không tìm thấy OPENROUTER_API_KEY nào trong environment")
		os.Exit(1)
	}

	// Tạo provider với model được chỉ định hoặc model mặc định
	selectedModel := *model
	if selectedModel == "" {
		selectedModel = "meta-llama/llama-3.3-70b-instruct:free"
	}

	// Tạo multi-provider cho OpenRouter
	var primary providers.Provider
	var fallbacks []providers.Provider

	// Dùng key đầu tiên làm primary
	primary = &providers.OpenRouterProvider{
		APIKey: keys[0],
		Model:  selectedModel,
	}

	// Các key còn lại làm fallback
	for i := 1; i < len(keys); i++ {
		fallbacks = append(fallbacks, &providers.OpenRouterProvider{
			APIKey: keys[i],
			Model:  selectedModel,
		})
	}

	// Xử lý các mode
	switch *mode {
	case "cli":
		args := flag.Args()
		if len(args) > 0 {
			userInput := args[0]
			fmt.Printf("👤 User: %s\n", userInput)

			// Test với provider
			req := messaging.Request{
				History: []messaging.Message{
					{Role: messaging.RoleUser, Content: userInput},
				},
			}

			provider := providers.NewMultiProvider(primary, fallbacks)
			resp, err := provider.Generate(context.Background(), req)
			if err != nil {
				fmt.Printf("❌ Lỗi: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🤖 AI: %s\n", resp.Content)
		} else {
			fmt.Println("Usage:")
			fmt.Println("  openrouter-cli --mode cli --model <model> \"your message\"")
			fmt.Println("  openrouter-cli --mode server --model <model>")
		}
	case "server":
		fmt.Printf("🚀 OpenRouter Server sẽ khởi động với model: %s\n", selectedModel)
		fmt.Printf("🔑 Sử dụng %d API keys\n", len(keys))
		fmt.Println("🔄 Server mode sẽ được implement sau...")
	default:
		fmt.Println("❌ Unknown mode:", *mode)
		os.Exit(1)
	}
}