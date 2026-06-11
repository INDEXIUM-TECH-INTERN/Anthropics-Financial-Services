package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/store"
	"gemini-cli/internal/utils"
)

// ── Health check handler ──
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
			fmt.Printf("⚠️ [Health] Write error: %v\n", err)
		}
}

// ── SSE Events handler ──
func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := pubsub.GlobalHub.Register()
	defer pubsub.GlobalHub.Unregister(ch)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "data: {\"type\":\"system\",\"payload\":\"SSE Connected\"}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-ch:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ── Reset handler ──
func handleReset(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			return
		}
		agent.Reset()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
	}
}

// ── Chat sessions CRUD handler ──
func handleChats(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case "GET":
		sessions, err := store.ListSessions()
		if err != nil {
			http.Error(w, "failed to list chats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"chats": sessions})

	case "POST":
		var payload struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad request: invalid JSON", http.StatusBadRequest)
			return
		}

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
}

// ── Main chat handler ──
func handleChat(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit for attachments
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		fmt.Printf("📩 [Server] Received message: %s (chat_id=%s, attachments=%d)\n", req.Message, req.ChatID, len(req.Attachments))

		chatID := req.ChatID
		if chatID == "" {
			chatID = "default"
		}

		sess, err := store.GetSession(chatID)
		if err != nil {
			sess = &store.ChatSession{ID: chatID, Title: "Cuộc trò chuyện mới", Messages: []messaging.Message{}}
		}

		agent.LoadHistory(sess.Messages)

		startTime := time.Now()
		reply, err := agent.ProcessMessage(req.Message, req.Attachments)
		latency := time.Since(startTime).Milliseconds()

		ram, cpu := getSystemMetrics()

		if err != nil {
			resp := ChatResponse{Error: err.Error()}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		updatedHistory := agent.GetHistory()
		sess.Messages = updatedHistory
		sess.Title = generateTitleIfNeeded(sess.Title, req.Message)
		if err := store.SaveSession(sess); err != nil {
				fmt.Printf("⚠️ [SaveSession] Error: %v\n", err)
			}

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
	}
}

// ── Streaming chat handler ──
func handleChatStream(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit for attachments
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		fmt.Printf("📩 [Server-Stream] Received message: %s (chat_id=%s, attachments=%d)\n", req.Message, req.ChatID, len(req.Attachments))

		chatID := req.ChatID
		if chatID == "" {
			chatID = "default"
		}

		sess, err := store.GetSession(chatID)
		if err != nil {
			sess = &store.ChatSession{ID: chatID, Title: "Cuộc trò chuyện mới", Messages: []messaging.Message{}}
		}

		agent.LoadHistory(sess.Messages)
		startTime := time.Now()

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		var fullReply strings.Builder
		streamErr := agent.ProcessMessageStream(req.Message, req.Attachments, func(text string, done bool) {
			if !done && text != "" {
				chunk := map[string]interface{}{"type": "token", "text": text}
				data, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				fullReply.WriteString(text)
			}
			if done {
				latency := time.Since(startTime).Milliseconds()
				ram, cpu := getSystemMetrics()
				final := map[string]interface{}{
					"type":    "done",
					"text":    fullReply.String(),
					"metrics": Metrics{LatencyMs: latency, TokenIn: utils.EstimateTokens(req.Message), TokenOut: utils.EstimateTokens(fullReply.String()), RamMB: ram, CpuLoad: cpu},
				}
				data, _ := json.Marshal(final)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		})

		if streamErr != nil {
			fmt.Printf("❌ [Stream] Error: %v (chat_id=%s)\n", streamErr, chatID)
			errChunk := map[string]interface{}{"type": "error", "error": streamErr.Error()}
			data, _ := json.Marshal(errChunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return
		}

		// Save conversation history
		updatedHistory := agent.GetHistory()
		sess.Messages = updatedHistory
		sess.Title = generateTitleIfNeeded(sess.Title, req.Message)
		if err := store.SaveSession(sess); err != nil {
				fmt.Printf("⚠️ [SaveSession] Error: %v\n", err)
			}
	}
}

// ── History handler ──
func handleHistory(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}
