package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"gemini-cli/internal/models/messaging"
)

type AgentInterface interface {
	Reset()
	ProcessMessage(string) (string, error)
	GetHistory() []messaging.Message
	SetOpenRouterKeys([]string)
}

type ChatRequest struct {
	Message string `json:"message"`
}

type Metrics struct {
	LatencyMs int64  `json:"latency_ms"`
	TokenIn   int    `json:"token_in"`
	TokenOut  int    `json:"token_out"`
	RamMB     string `json:"ram_mb"`
	CpuLoad   string `json:"cpu_load"`
}

type ChatResponse struct {
	Reply   string                 `json:"reply"`
	History []messaging.Message `json:"history"`
	Metrics Metrics                `json:"metrics"`
	Error   string                 `json:"error,omitempty"`
}

func getSystemMetrics() (string, string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ram := fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)

	// Ước lượng độ tải thông qua số Goroutine đang chạy
	cpu := fmt.Sprintf("%d Goroutines (Active)", runtime.NumGoroutine())
	return ram, cpu
}

func StartServer(agent AgentInterface) {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := GlobalHub.Register()
		defer GlobalHub.Unregister(ch)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Gửi sự kiện chào mừng
		fmt.Fprintf(w, "data: {\"type\":\"system\",\"payload\":\"SSE Connected\"}\n\n")
		flusher.Flush()

		// Heartbeat (Ping) mỗi 15s để giữ kết nối
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case event := <-ch:
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-ticker.C:
				// Gửi ping comment
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

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

		startTime := time.Now()
		reply, err := agent.ProcessMessage(req.Message)
		latency := time.Since(startTime).Milliseconds()

		ram, cpu := getSystemMetrics()

		if err != nil {
			resp := ChatResponse{Error: err.Error()}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := ChatResponse{
			Reply:   reply,
			History: agent.GetHistory(),
			Metrics: Metrics{
				LatencyMs: latency,
				TokenIn:   len(req.Message) / 4, // Ước tính tạm
				TokenOut:  len(reply) / 4,       // Ước tính tạm
				RamMB:     ram,
				CpuLoad:   cpu,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Static files - Try finding the frontend dir from multiple possible locations
	var frontendDir string
	if _, err := os.Stat("frontend"); err == nil {
		frontendDir = "frontend"
	} else if _, err := os.Stat("../../frontend"); err == nil {
		frontendDir = "../../frontend"
	} else if _, err := os.Stat("../frontend"); err == nil {
		frontendDir = "../frontend"
	} else {
		// Fallback to absolute path search if possible, or current dir
		frontendDir = "frontend"
	}
	fmt.Printf("📂 [Server] Serving static files from: %s\n", frontendDir)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	mux.HandleFunc("/api/config/keys", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" { return }
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		agent.SetOpenRouterKeys(payload.Keys)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "OpenRouter keys updated"})
	})


	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  120 * time.Second,
		// Gỡ bỏ WriteTimeout vì nó làm ngắt kết nối SSE dài hạn
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

