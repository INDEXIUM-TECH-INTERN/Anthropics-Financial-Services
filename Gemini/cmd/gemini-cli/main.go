package main

import (
	"flag"
)

func main() {
	serverMode := flag.Bool("server", false, "Chạy Agent ở chế độ Web Server")
	flag.Parse()

	agent := NewAgent()

	if *serverMode {
		StartServer(agent)
	} else {
		agent.Start()
	}
}
