package main

import (
	"flag"
	"fmt"
)

func main() {
	serverMode := flag.Bool("server", false, "Chạy Agent ở chế độ Web Server")
	flag.Parse()

	agent := NewAgent()

	if *serverMode {
		StartServer(agent)
	} else {
		// Nếu có tham số sau các flag, chạy một lần rồi thoát
		args := flag.Args()
		if len(args) > 0 {
			userInput := args[0]
			fmt.Printf("👤 User: %s\n", userInput)
			reply, err := agent.ProcessMessage(userInput)
			if err != nil {
				fmt.Printf("❌ [Lỗi] %v\n", err)
			} else {
				fmt.Printf("\n🤖 Agent: %s\n", reply)
			}
			return
		}
		agent.Start()
	}
}
