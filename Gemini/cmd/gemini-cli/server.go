package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gemini-cli/internal/models"
)

// Giao diện đã được nhúng vào Go binary (nếu thư mục dist tồn tại)
// Để đơn giản khi phát triển, tôi sẽ tắt phần embed nếu bạn đã xóa thư mục frontend.
// Nếu muốn bật lại, hãy chạy 'npm run build' trong thư mục frontend và phục hồi cấu trúc dist.

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply   string                 `json:"reply"`
	History []models.GeminiContent `json:"history"`
	Error   string                 `json:"error,omitempty"`
}

func StartServer(agent *Agent) {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			return
		}
		agent.Reset()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
	})

	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		fmt.Printf("📩 [Server] Received message: %s\n", req.Message)
		reply, err := agent.ProcessMessage(req.Message)
		if err != nil {
			resp := ChatResponse{Error: err.Error()}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		resp := ChatResponse{
			Reply:   reply,
			History: agent.GetHistory(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Static files (tạm thời tắt embed để tránh lỗi build khi không có thư mục dist)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Anthropic Financial Agent Backend is running. API is at /api/chat")
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("🚀 [Server] Backend Go is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ [Error] Server failed: %v\n", err)
	}
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
