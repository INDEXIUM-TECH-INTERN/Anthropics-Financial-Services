package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/redis"
)

type AgentInterface interface {
	Reset()
	ProcessMessage(string, []messaging.Attachment) (string, error)
	ProcessMessageStream(string, []messaging.Attachment, func(string, bool)) error
	GetHistory() []messaging.Message
	LoadHistory([]messaging.Message)
	SetOpenRouterKeys([]string)
}

type ChatRequest struct {
	Message     string                 `json:"message"`
	ChatID      string                 `json:"chat_id,omitempty"`
	Attachments []messaging.Attachment `json:"attachments,omitempty"`
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
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/api/events", handleSSE)
	mux.HandleFunc("/api/reset", handleReset(agent))
	mux.HandleFunc("/api/chats", handleChats)
	mux.HandleFunc("/api/chat", handleChat(agent))
	mux.HandleFunc("/api/chat/stream", handleChatStream(agent))
	mux.HandleFunc("/api/history", handleHistory(agent))

	// Static files
	frontendDir := resolveFrontendDir()
	fmt.Printf("📂 [Server] Serving static files from: %s\n", frontendDir)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       120 * time.Second,
		IdleTimeout:       600 * time.Second,
		// Gỡ bỏ WriteTimeout vì nó làm ngắt kết nối SSE dài hạn
	}

	fmt.Printf("🚀 [Server] Backend Go is running on http://0.0.0.0:%s\n", port)
	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		fmt.Printf("\n🛑 [Server] Received signal %v, shutting down gracefully...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("❌ [Server] Shutdown error: %v\n", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("❌ [Error] Server failed: %v\n", err)
	}
	fmt.Println("✅ [Server] Stopped.")
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
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "*" // fallback for dev; set ALLOWED_ORIGIN in production
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// generateTitleIfNeeded creates a nice title from the first user message if still default.
func generateTitleIfNeeded(currentTitle, userMsg string) string {
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
