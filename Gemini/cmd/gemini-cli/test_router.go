//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gemini-cli/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/gemini-cli/test_router.go <query>")
		os.Exit(1)
	}
	query := os.Args[len(os.Args)-1]

	routePlan := core.SelectRoutePlan(query)

	b, err := json.Marshal(routePlan)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("RESULT_JSON: %s\n", string(b))
}
