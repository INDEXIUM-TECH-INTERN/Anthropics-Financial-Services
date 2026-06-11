package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/redis"
	"gemini-cli/internal/store"
	"gemini-cli/internal/utils"
)

type AgentInterface interface {
	Reset()
	ProcessMessage(string) (string, error)
	GetHistory() []messaging.Message
	LoadHistory([]messaging.Message)
	SetOpenRouterKeys([]string)
}

type ChatRequest struct {
	Message string `json:"message"`
	ChatID  string `json:"chat_id,omitempty"` // for multi-chat / Redis sessions
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

	// Ước lượng độ tải thông qua số Goroutine đang chạy
	cpu := fmt.Sprintf("%d Goroutines (Active)", runtime.NumGoroutine())
	return ram, cpu
}

func StartServer(agent AgentInterface) {
	// Init Redis for persistent chat sessions (multi-chat support)
	if err := redis.Init(); err != nil {
		fmt.Printf("⚠️ [Redis] %v — falling back to in-memory only\n", err)
	} else {
		defer redis.Close()
	}

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

	// === Redis-backed multi-chat session management ===
	mux.HandleFunc("/api/chats", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			return
		}

		switch r.Method {
		case "GET":
			// List all sessions (lightweight)
			sessions, err := store.ListSessions()
			if err != nil {
				http.Error(w, "failed to list chats", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"chats": sessions})

		case "POST":
			// Create new session
			var payload struct {
				Title string `json:"title"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)

			title := payload.Title
			if title == "" {
				title = "Cuộc trò chuyện mới"
			}

			sess := &store.ChatSession{
				ID:       fmt.Sprintf("chat_%d", time.Now().UnixNano()),
				Title:    title,
				Messages: []messaging.Message{},
			}
			if err := store.SaveSession(sess); err != nil {
				http.Error(w, "failed to create chat", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sess)

		case "DELETE":
			chatID := r.URL.Query().Get("chat_id")
			if chatID == "" {
				http.Error(w, "missing chat_id query parameter", http.StatusBadRequest)
				return
			}
			if err := store.DeleteSession(chatID); err != nil {
				http.Error(w, "failed to delete chat session", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "deleted",
				"chat_id": chatID,
			})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
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
		// Limit request body to 1MB to prevent DoS
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		fmt.Printf("📩 [Server] Received message: %s (chat_id=%s)\n", req.Message, req.ChatID)

		chatID := req.ChatID
		if chatID == "" {
			chatID = "default" // fallback for legacy single-chat clients
		}

		// Load session history from Redis (or empty)
		sess, err := store.GetSession(chatID)
		if err != nil {
			sess = &store.ChatSession{ID: chatID, Title: "Cuộc trò chuyện mới", Messages: []messaging.Message{}}
		}

		// Load into agent for this request (multi-session support)
		agent.LoadHistory(sess.Messages)

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

		// Save updated history back to Redis
		updatedHistory := agent.GetHistory()
		sess.Messages = updatedHistory
		sess.Title = generateTitleIfNeeded(sess.Title, req.Message, reply)
		_ = store.SaveSession(sess)

		resp := ChatResponse{
			Reply:   reply,
			History: updatedHistory,
			Metrics: Metrics{
				LatencyMs: latency,
				TokenIn:   utils.EstimateTokens(req.Message),
				TokenOut:  utils.EstimateTokens(reply),
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
		if r.Method == "OPTIONS" {
			return
		}
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

	// GET current full history (for UI to restore chat segments on refresh/F5)
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			return
		}
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		chatID := r.URL.Query().Get("chat_id")
		var history []messaging.Message
		if chatID != "" {
			sess, err := store.GetSession(chatID)
			if err == nil && sess != nil {
				history = sess.Messages
			}
		} else {
			history = agent.GetHistory()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"history": history,
		})
	})

	server := &http.Server{
		Addr:        ":8080",
		Handler:     mux,
		ReadTimeout: 15 * time.Second,
		IdleTimeout: 120 * time.Second,
		// Gỡ bỏ WriteTimeout vì nó làm ngắt kết nối SSE dài hạn
	}

	fmt.Println("🚀 [Server] Backend Go is running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("❌ [Error] Server failed: %v\n", err)
	}
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
	// simple title from first user message (use rune count for Unicode safety)
	runes := []rune(userMsg)
	if len(runes) > 60 {
		return string(runes[:57]) + "..."
	}
	if userMsg != "" {
		return userMsg
	}
	return "Cuộc trò chuyện mới"
}
