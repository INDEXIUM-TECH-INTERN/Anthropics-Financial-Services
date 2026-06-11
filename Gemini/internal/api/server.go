package api

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/redis"
)

type AgentInterface interface {
	Reset()
	ProcessMessage(string) (string, error)
	ProcessMessageStream(string, func(string, bool)) error
	GetHistory() []messaging.Message
	LoadHistory([]messaging.Message)
	SetOpenRouterKeys([]string)
}

type ChatRequest struct {
	Message string `json:"message"`
	ChatID  string `json:"chat_id,omitempty"`
}

type Metrics struct {
	LatencyMs int64  `json:"latency_ms"`
	TokenIn   int    `json:"token_in"`
	TokenOut  int    `json:"token_out"`
	RamMB     string `json:"ram_mb"`
	CpuLoad   string `json:"cpu_load"`
}

type ChatResponse struct {
	Reply   string              `json:"reply"`
	History []messaging.Message `json:"history"`
	Metrics Metrics             `json:"metrics"`
	Error   string              `json:"error,omitempty"`
}

func getSystemMetrics() (string, string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ram := fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
	cpu := fmt.Sprintf("%d Goroutines (Active)", runtime.NumGoroutine())
	return ram, cpu
}

func StartServer(agent AgentInterface) {
	if err := redis.Init(); err != nil {
		fmt.Printf("⚠️ [Redis] %v — falling back to in-memory only\n", err)
	} else {
		defer redis.Close()
	}

	mux := http.NewServeMux()

	// API routes — each handler is in handlers.go
	mux.HandleFunc("/api/events", handleSSE)
	mux.HandleFunc("/api/reset", handleReset(agent))
	mux.HandleFunc("/api/chats", handleChats)
	mux.HandleFunc("/api/chat", handleChat(agent))
		mux.HandleFunc("/api/chat/stream", handleChatStream(agent))
	mux.HandleFunc("/api/config/keys", handleConfigKeys(agent))
	mux.HandleFunc("/api/history", handleHistory(agent))

	// Static files
	frontendDir := resolveFrontendDir()
	fmt.Printf("📂 [Server] Serving static files from: %s\n", frontendDir)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	server := &http.Server{
		Addr:        ":8080",
		Handler:     mux,
		ReadTimeout: 300 * time.Second,
		IdleTimeout: 600 * time.Second,
		// Gỡ bỏ WriteTimeout vì nó làm ngắt kết nối SSE dài hạn
	}

	fmt.Println("🚀 [Server] Backend Go is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ [Error] Server failed: %v\n", err)
	}
}

func resolveFrontendDir() string {
	for _, dir := range []string{"frontend", "../../frontend", "../frontend"} {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return "frontend"
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// generateTitleIfNeeded creates a nice title from the first user message if still default.
func generateTitleIfNeeded(currentTitle, userMsg, reply string) string {
	if currentTitle != "" && currentTitle != "Cuộc trò chuyện mới" {
		return currentTitle
	}
	runes := []rune(userMsg)
	if len(runes) > 60 {
		return string(runes[:57]) + "..."
	}
	if userMsg != "" {
		return userMsg
	}
	return "Cuộc trò chuyện mới"
}
