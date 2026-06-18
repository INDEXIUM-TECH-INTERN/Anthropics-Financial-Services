package utils

import (
	"fmt"
	"time"
)

// ExampleRoundRobin demostrate round-robin key selection
func ExampleRoundRobin() {
	// Giả sử có 5 Gemini API keys
	keys := "key1,key2,key3,key4,key5"

	keyPool, _ := NewSimpleKeyPool(keys)

	fmt.Println("=== Round-Robin Key Selection Demo ===")
	fmt.Println("Keys in pool:", keyPool.GetAllKeys())
	fmt.Println()

	// Demo 10 lần gọi để thấy round-robin behavior
	for i := 0; i < 10; i++ {
		key := keyPool.GetRoundRobinKey()
		fmt.Printf("Request %d: Using key %s\n", i+1, key)
		time.Sleep(100 * time.Millisecond) // Giả sử có delay
	}

	fmt.Println()
	stats := keyPool.Stats()
	fmt.Println("Usage Statistics:")
	for key, lastUsed := range stats {
		fmt.Printf("- %s: last used at %v\n", key, lastUsed)
	}
}

// Example usage in main:
/*
func main() {
    ExampleRoundRobin()

    // Trong actual usage:
    provider := NewSimpleGeminiProvider(
        os.Getenv("GEMINI_API_KEYS"), // "key1,key2,key3,key4,key5"
        "gemini-3.1-flash-lite",
        3,
    )

    // Mỗi call SendMessage() sẽ tự động dùng round-robin
    resp, err := provider.SendMessage(ctx, "Hello")
}
*/