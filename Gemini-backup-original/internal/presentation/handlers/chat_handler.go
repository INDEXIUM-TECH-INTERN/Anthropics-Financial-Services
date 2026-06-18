package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gemini-cli/internal/domain/entities"
	"gemini-cli/internal/domain/interfaces"
	"gemini-cli/internal/presentation/middleware"
)

// ChatHandler handles chat-related HTTP requests
type ChatHandler struct {
	agentService interfaces.AgentService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(agentService interfaces.AgentService) *ChatHandler {
	return &ChatHandler{
		agentService: agentService,
	}
}

// HandleChatStream handles SSE chat streaming
func (h *ChatHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Get chat ID from request
	var req ChatStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Set response writer flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create context
	ctx := r.Context()

	// Process message with streaming
	err := h.agentService.ProcessMessageStream(ctx, req.Message, req.Attachments, func(chunk string, done bool) {
		if done {
			event := StreamEvent{
				Type:    "done",
				Data:    "",
				Done:    true,
				Message: "",
			}
			data, err := json.Marshal(event)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(data))
			}
			flusher.Flush()
			return
		}

		// Send chunk
		event := StreamEvent{
			Type:    "chunk",
			Data:    chunk,
			Done:    false,
			Message: chunk,
		}

		data, err := json.Marshal(event)
		if err == nil {
			fmt.Fprintf(w, "data: %s\n\n", string(data))
		}
		flusher.Flush()
	})

	if err != nil {
		event := StreamEvent{
			Type:    "error",
			Data:    err.Error(),
			Done:    true,
			Message: err.Error(),
		}

		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	}
}

// HandleChat handles regular chat requests
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Process request
	resp, err := h.agentService.Process(ctx, &interfaces.ProcessRequest{
		Message:     req.Message,
		ChatID:      req.ChatID,
		Attachments: req.Attachments,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Reply:      resp.Reply,
		History:    resp.History,
		UsedAgent:  resp.UsedAgent,
		TokenUsage: resp.TokenUsage,
		ExecTime:   resp.ExecTime,
		ToolCalls:  resp.ToolCalls,
	})
}

// HandleReset handles conversation reset
func (h *ChatHandler) HandleReset(w http.ResponseWriter, r *http.Request) {
	h.agentService.Reset()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Conversation reset successfully",
	})
}

// HandleHistory handles conversation history requests
func (h *ChatHandler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	history := h.agentService.GetHistory()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HistoryResponse{
		History: history,
	})
}

// HandleHealth handles health check requests
func (h *ChatHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	health := HealthResponse{
		Status:   "healthy",
		Message:  "Service is running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Message     string                  `json:"message"`
	ChatID      string                  `json:"chat_id,omitempty"`
	Attachments []entities.Attachment `json:"attachments,omitempty"`
}

// ChatStreamRequest represents a chat streaming request
type ChatStreamRequest struct {
	Message     string                  `json:"message"`
	ChatID      string                  `json:"chat_id,omitempty"`
	Attachments []entities.Attachment `json:"attachments,omitempty"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Reply      string                  `json:"reply"`
	History    []entities.Message      `json:"history"`
	UsedAgent  string                  `json:"used_agent"`
	TokenUsage interfaces.TokenUsage  `json:"token_usage"`
	ExecTime   int64                   `json:"exec_time_ms"`
	ToolCalls  []entities.ToolCall    `json:"tool_calls,omitempty"`
}

// HistoryResponse represents a history response
type HistoryResponse struct {
	History []entities.Message `json:"history"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// StreamEvent represents a SSE event
type StreamEvent struct {
	Type    string `json:"type"`
	Data    string `json:"data"`
	Done    bool   `json:"done"`
	Message string `json:"message,omitempty"`
}

// ChatRoutes configures chat routes
func ChatRoutes(mux *http.ServeMux, chatHandler *ChatHandler) {
	mux.HandleFunc("/api/chat/stream", middleware.WithCors(middleware.WithLogging(chatHandler.HandleChatStream)))
	mux.HandleFunc("/api/chat", middleware.WithCors(middleware.WithLogging(chatHandler.HandleChat)))
	mux.HandleFunc("/api/reset", middleware.WithCors(middleware.WithLogging(chatHandler.HandleReset)))
	mux.HandleFunc("/api/history", middleware.WithCors(middleware.WithLogging(chatHandler.HandleHistory)))
	mux.HandleFunc("/health", middleware.WithCors(middleware.WithLogging(chatHandler.HandleHealth)))
}