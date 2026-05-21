package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/gemini-cli/test_router.go <query>")
		return
	}

	query := os.Args[1]
	agent := NewAgent()
	agent.userInput = query

	fmt.Printf("Testing query: %s\n", query)
	route := agent.selectRoutePlan()

	fmt.Printf("RESULT_JSON: {\"agent\":\"%s\",\"skills\":%v,\"temporal\":{\"intent\":\"%s\",\"resolved_date\":\"%s\",\"is_future\":%v},\"reason\":\"%s\"}\n",
		route.Agent, route.Skills, route.Temporal.Intent, route.Temporal.ResolvedDate, route.Temporal.IsFuture, route.Reason)
}
