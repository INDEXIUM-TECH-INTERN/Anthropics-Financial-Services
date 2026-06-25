package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/pubsub"
	"gemini-cli/internal/store"
	"gemini-cli/internal/utils"
	"gemini-cli/internal/worldnews"
)

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("chat_%d", time.Now().UnixNano())
	}
	return "chat_" + hex.EncodeToString(b)
}

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
		enableCORS(w, r)
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
	enableCORS(w, r)
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
			ID:       generateSessionID(),
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
// validateChatRequest validates a ChatRequest and writes HTTP errors on failure.
func validateChatRequest(w http.ResponseWriter, req *ChatRequest) bool {
	if len(req.Message) > 50000 {
		http.Error(w, `{"error":"Message too long (max 50000 characters)"}`, http.StatusBadRequest)
		return false
	}
	if len(req.Attachments) > 10 {
		http.Error(w, `{"error":"Too many attachments (max 10)"}`, http.StatusBadRequest)
		return false
	}
	return true
}

// resolveChatSession returns an existing session or creates a default one.
func resolveChatSession(chatID string) *store.ChatSession {
	if chatID == "" {
		chatID = "default"
	}
	sess, err := store.GetSession(chatID)
	if err != nil {
		return &store.ChatSession{ID: chatID, Title: "Cuộc trò chuyện mới", Messages: []messaging.Message{}}
	}
	return sess
}

func handleChat(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
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
		if !validateChatRequest(w, &req) {
			return
		}

		fmt.Printf("📩 [Server] Received message: %s (chat_id=%s, attachments=%d)\n", req.Message, req.ChatID, len(req.Attachments))

		sess := resolveChatSession(req.ChatID)
		agent.LoadHistory(sess.Messages)

		startTime := time.Now()
		reply, err := agent.ProcessMessage(r.Context(), req.Message, req.Attachments)
		latency := time.Since(startTime).Milliseconds()

		ram, cpu := getSystemMetrics()

		if err != nil {
			resp := ChatResponse{Error: err.Error()}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		updatedHistory := agent.GetHistory()
		if len(updatedHistory) > 0 {
			for idx := len(updatedHistory) - 1; idx >= 0; idx-- {
				if updatedHistory[idx].Role == messaging.RoleAssistant {
					updatedHistory[idx].LatencyMs = latency
					updatedHistory[idx].TokenIn = utils.EstimateTokens(req.Message)
					updatedHistory[idx].TokenOut = utils.EstimateTokens(reply)
					updatedHistory[idx].RamMB = ram
					updatedHistory[idx].CpuLoad = cpu
					break
				}
			}
		}
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
		enableCORS(w, r)
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

		if !validateChatRequest(w, &req) {
			return
		}

		fmt.Printf("📩 [Server-Stream] Received message: %s (chat_id=%s, attachments=%d)\n", req.Message, req.ChatID, len(req.Attachments))

		sess := resolveChatSession(req.ChatID)

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
		streamErr := agent.ProcessMessageStream(r.Context(), req.Message, req.Attachments, func(text string, done bool) {
			if !done && text != "" {
				chunk := map[string]interface{}{"type": "token", "text": text}
				data, err := json.Marshal(chunk)
				if err != nil {
					fmt.Printf("⚠️ [Stream] Marshal token chunk error: %v\n", err)
					return
				}
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
				data, err := json.Marshal(final)
				if err != nil {
					fmt.Printf("⚠️ [Stream] Marshal done chunk error: %v\n", err)
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		})

		if streamErr != nil {
			fmt.Printf("❌ [Stream] Error: %v (chat_id=%s)\n", streamErr, req.ChatID)
			errChunk := map[string]interface{}{"type": "error", "error": streamErr.Error()}
			data, err := json.Marshal(errChunk)
			if err != nil {
				fmt.Printf("⚠️ [Stream] Marshal error chunk error: %v\n", err)
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return
		}

		// Save conversation history
		updatedHistory := agent.GetHistory()
		latency := time.Since(startTime).Milliseconds()
		ram, cpu := getSystemMetrics()
		if len(updatedHistory) > 0 {
			for idx := len(updatedHistory) - 1; idx >= 0; idx-- {
				if updatedHistory[idx].Role == messaging.RoleAssistant {
					updatedHistory[idx].LatencyMs = latency
					updatedHistory[idx].TokenIn = utils.EstimateTokens(req.Message)
					updatedHistory[idx].TokenOut = utils.EstimateTokens(updatedHistory[idx].Content)
					updatedHistory[idx].RamMB = ram
					updatedHistory[idx].CpuLoad = cpu
					break
				}
			}
		}
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
		enableCORS(w, r)
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
			"history": messaging.FilterPublicHistory(history),
		})
	}
}

// ── Config Keys handler ──
// ── World News handlers ──
func handleWorldNews(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	date := strings.TrimSpace(r.URL.Query().Get("date"))
	report, err := worldnews.DefaultService.GetReport(date)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func handleWorldNewsDates(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worldnews.DefaultService.GetAvailableDates())
}

func handleWorldNewsFavicon(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	host := strings.TrimSpace(r.URL.Query().Get("host"))
	data, mime, err := worldnews.DefaultService.FetchFavicon(host)
	if err != nil {
		http.Error(w, "favicon not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleWorldNewsImage(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	imageURL := strings.TrimSpace(r.URL.Query().Get("url"))
	data, mime, err := worldnews.DefaultService.FetchProxiedImage(imageURL)
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleConfigKeys(agent AgentInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w, r)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Bad request: invalid JSON", http.StatusBadRequest)
			return
		}

		// Filter empty lines
		var validKeys []string
		for _, k := range payload.Keys {
			k = strings.TrimSpace(k)
			if k != "" {
				validKeys = append(validKeys, k)
			}
		}

		agent.SetGeminiKeys(validKeys)
		fmt.Printf("🔑 [Config] Updated %d Gemini keys\n", len(validKeys))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"count":  len(validKeys),
		})
	}
}
