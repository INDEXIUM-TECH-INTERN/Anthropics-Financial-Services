package main

import (
	"context"
	"flag"
	"fmt"

	"gemini-cli/internal/api"
	"gemini-cli/internal/core"
)

func main() {
	serverMode := flag.Bool("server", false, "Chạy Agent ở chế độ Web Server")
	flag.Parse()

	agent := core.NewAgent()

	if *serverMode {
		api.StartServer(agent)
		return
	}

	// Nếu có tham số sau các flag, chạy một lần rồi thoát
	args := flag.Args()
	if len(args) > 0 {
		userInput := args[0]
		fmt.Printf("👤 User: %s\n", userInput)
		reply, err := agent.ProcessMessage(context.Background(), userInput, nil)
		if err != nil {
			fmt.Printf("❌ [Lỗi] %v\n", err)
		} else {
			fmt.Printf("\n🤖 Agent: %s\n", reply)
		}
		return
	}

	// Không có mode nào được chỉ định — hướng dẫn sử dụng
	fmt.Println("Usage:")
	fmt.Println("  --server          Chạy ở chế độ Web Server")
	fmt.Println("  <message>         Chạy một lần với message")
	fmt.Println("")
	fmt.Println("Ví dụ:")
	fmt.Println("  gemini-cli --server")
	fmt.Println("  gemini-cli \"Phân tích cổ phiếu VNM\"")
}
