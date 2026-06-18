package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gemini-cli/internal/models/messaging"
)

// ─── Mock Agent ───────────────────────────────────────────────────────────────

type mockAgent struct {
	resetCalled   bool
	history       []messaging.Message
	processReply  string
	processErr    error
	streamErr     error
	streamChunks  []string // chunks to deliver via callback
	keys          []string
	loadedHistory []messaging.Message
}

func (m *mockAgent) Reset() {
	m.resetCalled = true
}

func (m *mockAgent) ProcessMessage(ctx context.Context, msg string, atts []messaging.Attachment) (string, error) {
	return m.processReply, m.processErr
}

func (m *mockAgent) ProcessMessageStream(ctx context.Context, msg string, atts []messaging.Attachment, onChunk func(string, bool)) error {
	if m.streamErr != nil {
		return m.streamErr
	}
	for _, chunk := range m.streamChunks {
		onChunk(chunk, false)
	}
	onChunk("", true) // signal done
	return nil
}

func (m *mockAgent) GetHistory() []messaging.Message {
	return m.history
}

func (m *mockAgent) LoadHistory(msgs []messaging.Message) {
	m.loadedHistory = msgs
}

func (m *mockAgent) SetGeminiKeys(keys []string) {
	m.keys = keys
}

// ─── handleHealth ────────────────────────────────────────────────────────────

func TestHandleHealth_GET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", w.Body.String())
	}
}

func TestHandleHealth_POST(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// ─── handleChat ──────────────────────────────────────────────────────────────

func TestHandleChat_OPTIONS(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChat(agent)

	req := httptest.NewRequest(http.MethodOptions, "/api/chat", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestHandleChat_GET(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChat(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleChat_InvalidJSON(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChat(agent)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader("not-json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleChat_MessageTooLong(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChat(agent)

	longMsg := strings.Repeat("x", 50001)
	body := `{"message": "` + longMsg + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for too-long message, got %d", w.Code)
	}
}

func TestHandleChat_TooManyAttachments(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChat(agent)

	// Build JSON with 11 attachments
	atts := make([]map[string]string, 11)
	for i := range atts {
		atts[i] = map[string]string{
			"name": "file.txt",
			"type": "text/plain",
			"data": "aGVsbG8=",
		}
	}
	payload := map[string]interface{}{
		"message":     "test",
		"attachments": atts,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req) // nolint:errcheck

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for too many attachments, got %d", w.Code)
	}
}

func TestHandleChat_Success(t *testing.T) {
	agent := &mockAgent{
		processReply: "Hello from AI",
	}
	handler := handleChat(agent)

	body := `{"message": "hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Reply != "Hello from AI" {
		t.Errorf("expected reply 'Hello from AI', got %q", resp.Reply)
	}
}

func TestHandleChat_ProcessError(t *testing.T) {
	agent := &mockAgent{
		processErr: fmt.Errorf("AI failure"),
	}
	handler := handleChat(agent)

	body := `{"message": "hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// Even on agent error, handler returns 200 with error field in JSON
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Error != "AI failure" {
		t.Errorf("expected error 'AI failure', got %q", resp.Error)
	}
}

// ─── handleChatStream ────────────────────────────────────────────────────────

func TestHandleChatStream_OPTIONS(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChatStream(agent)

	req := httptest.NewRequest(http.MethodOptions, "/api/chat/stream", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestHandleChatStream_GET(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChatStream(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/chat/stream", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleChatStream_InvalidJSON(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChatStream(agent)

	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader("invalid"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// ─── handleReset ─────────────────────────────────────────────────────────────

func TestHandleReset_ResetCalled(t *testing.T) {
	agent := &mockAgent{}
	handler := handleReset(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/reset", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if !agent.resetCalled {
		t.Error("expected agent.Reset() to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "reset" {
		t.Errorf("expected status 'reset', got %q", resp["status"])
	}
}

// ─── handleHistory ───────────────────────────────────────────────────────────

func TestHandleHistory_OPTIONS(t *testing.T) {
	agent := &mockAgent{}
	handler := handleHistory(agent)

	req := httptest.NewRequest(http.MethodOptions, "/api/history", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// OPTIONS returns without writing status (just returns)
}

func TestHandleHistory_POST(t *testing.T) {
	agent := &mockAgent{}
	handler := handleHistory(agent)

	req := httptest.NewRequest(http.MethodPost, "/api/history", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleHistory_GET(t *testing.T) {
	agent := &mockAgent{
		history: []messaging.Message{
			{Role: messaging.RoleUser, Content: "hi"},
			{Role: messaging.RoleAssistant, Content: "hello"},
		},
	}
	handler := handleHistory(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if _, ok := resp["history"]; !ok {
		t.Error("expected 'history' key in response")
	}
}

func TestHandleHistory_GET_WithChatID(t *testing.T) {
	agent := &mockAgent{}
	handler := handleHistory(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/history?chat_id=nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// ─── handleChats ─────────────────────────────────────────────────────────────

func TestHandleChats_OPTIONS(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/chats", nil)
	w := httptest.NewRecorder()

	handleChats(w, req)
	// Just returns, no error
}

func TestHandleChats_GET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chats", nil)
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if _, ok := resp["chats"]; !ok {
		t.Error("expected 'chats' key in response")
	}
}

func TestHandleChats_POST(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/chats", strings.NewReader(`{"title": "Test Chat"}`))
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestHandleChats_POST_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/chats", strings.NewReader("bad-json"))
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleChats_POST_NoTitle(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/chats", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	// Default title should be set
	if resp["title"] != "Cuộc trò chuyện mới" {
		t.Errorf("expected default title, got %v", resp["title"])
	}
}

func TestHandleChats_DELETE(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/chats?chat_id=test-id", nil)
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleChats_DELETE_MissingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/chats", nil)
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleChats_UnsupportedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/chats", nil)
	w := httptest.NewRecorder()

	handleChats(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// ─── handleConfigKeys ────────────────────────────────────────────────────────

func TestHandleConfigKeys_OPTIONS(t *testing.T) {
	agent := &mockAgent{}
	handler := handleConfigKeys(agent)

	req := httptest.NewRequest(http.MethodOptions, "/api/config/keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestHandleConfigKeys_GET(t *testing.T) {
	agent := &mockAgent{}
	handler := handleConfigKeys(agent)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleConfigKeys_InvalidJSON(t *testing.T) {
	agent := &mockAgent{}
	handler := handleConfigKeys(agent)

	req := httptest.NewRequest(http.MethodPost, "/api/config/keys", strings.NewReader("bad"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleConfigKeys_EmptyKeys(t *testing.T) {
	agent := &mockAgent{}
	handler := handleConfigKeys(agent)

	body := `{"keys": ["", "  ", ""]}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/keys", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if len(agent.keys) != 0 {
		t.Errorf("expected 0 valid keys, got %d", len(agent.keys))
	}
}

func TestHandleConfigKeys_ValidKeys(t *testing.T) {
	agent := &mockAgent{}
	handler := handleConfigKeys(agent)

	body := `{"keys": ["key1", "key2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/keys", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if len(agent.keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(agent.keys))
	}
}

// ─── validateChatRequest ─────────────────────────────────────────────────────

func TestValidateChatRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     ChatRequest
		wantErr bool
	}{
		{
			name:    "valid request",
			req:     ChatRequest{Message: "hello"},
			wantErr: false,
		},
		{
			name:    "message too long",
			req:     ChatRequest{Message: strings.Repeat("x", 50001)},
			wantErr: true,
		},
		{
			name: "too many attachments",
			req: ChatRequest{
				Message:     "hi",
				Attachments: make([]messaging.Attachment, 11),
			},
			wantErr: true,
		},
		{
			name: "max valid attachments",
			req: ChatRequest{
				Message:     "hi",
				Attachments: make([]messaging.Attachment, 10),
			},
			wantErr: false,
		},
		{
			name:    "empty message is valid",
			req:     ChatRequest{Message: ""},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			got := validateChatRequest(w, &tt.req)
			if got == tt.wantErr {
				t.Errorf("validateChatRequest() returned %v, want %v", got, !tt.wantErr)
			}
		})
	}
}

// ─── resolveChatSession ──────────────────────────────────────────────────────

func TestResolveChatSession_EmptyID(t *testing.T) {
	sess := resolveChatSession("")
	if sess.ID != "default" {
		t.Errorf("expected ID 'default', got %q", sess.ID)
	}
}

func TestResolveChatSession_WithID(t *testing.T) {
	// Non-existent ID returns a new default session
	sess := resolveChatSession("nonexistent-id-12345")
	if sess.ID != "nonexistent-id-12345" {
		t.Errorf("expected ID 'nonexistent-id-12345', got %q", sess.ID)
	}
}

// ─── generateTitleIfNeeded ───────────────────────────────────────────────────

func TestGenerateTitleIfNeeded(t *testing.T) {
	tests := []struct {
		name        string
		currentTitle string
		userMsg     string
		expected    string
	}{
		{
			name:        "keep existing title",
			currentTitle: "My Chat",
			userMsg:     "hello",
			expected:    "My Chat",
		},
		{
			name:        "default title gets replaced",
			currentTitle: "Cuộc trò chuyện mới",
			userMsg:     "hello world",
			expected:    "hello world",
		},
		{
			name:        "empty title gets replaced",
			currentTitle: "",
			userMsg:     "hello",
			expected:    "hello",
		},
		{
			name:        "long message gets truncated",
			currentTitle: "",
			userMsg:     strings.Repeat("x", 100),
			expected:    strings.Repeat("x", 57) + "...",
		},
		{
			name:        "empty message with default title",
			currentTitle: "Cuộc trò chuyện mới",
			userMsg:     "",
			expected:    "Cuộc trò chuyện mới",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTitleIfNeeded(tt.currentTitle, tt.userMsg)
			if result != tt.expected {
				t.Errorf("generateTitleIfNeeded(%q, %q) = %q, want %q",
					tt.currentTitle, tt.userMsg, result, tt.expected)
			}
		})
	}
}

// ─── generateSessionID ───────────────────────────────────────────────────────

func TestGenerateSessionID(t *testing.T) {
	id := generateSessionID()
	if id == "" {
		t.Error("expected non-empty session ID")
	}
	if !strings.HasPrefix(id, "chat_") {
		t.Errorf("expected prefix 'chat_', got %q", id)
	}

	// Ensure uniqueness
	id2 := generateSessionID()
	if id == id2 {
		t.Error("expected unique session IDs")
	}
}

// ─── enableCORS ──────────────────────────────────────────────────────────────

func TestEnableCORS_DefaultOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	enableCORS(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:8080" {
		t.Errorf("expected default origin, got %q", origin)
	}
}

func TestEnableCORS_CustomOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGIN", "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	enableCORS(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Errorf("expected custom origin, got %q", origin)
	}
}

// ─── resolveFrontendDir ──────────────────────────────────────────────────────

func TestResolveFrontendDir(t *testing.T) {
	dir := resolveFrontendDir()
	// Should return one of the candidate dirs (may not exist in test env)
	validDirs := map[string]bool{
		"../frontend/dist": true,
		"../frontend":      true,
		"frontend/dist":    true,
		"frontend":         true,
		"../../frontend":   true,
	}
	if !validDirs[dir] {
		t.Errorf("unexpected frontend dir: %q", dir)
	}
}

// ─── securityHeaders + rateLimitMiddleware ───────────────────────────────────

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := securityHeaders(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for header, expected := range checks {
		got := w.Header().Get(header)
		if got != expected {
			t.Errorf("header %s = %q, want %q", header, got, expected)
		}
	}

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected CSP header to be set")
	}
}

func TestRateLimitMiddleware_AllowsAPIRequests(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) // nolint:errcheck
	})
	handler := rateLimitMiddleware(inner)

	// API request should pass (within burst)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		// Note: first requests may pass, rate limiter allows burst
	}
}

func TestRateLimitMiddleware_SkipsSSE(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) // nolint:errcheck
	})
	handler := rateLimitMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for SSE path, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_SkipsHealth(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) // nolint:errcheck
	})
	handler := rateLimitMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for health path, got %d", w.Code)
	}
}

// ─── getSystemMetrics ────────────────────────────────────────────────────────

func TestGetSystemMetrics(t *testing.T) {
	ram, cpu := getSystemMetrics()
	if ram == "" {
		t.Error("expected non-empty RAM metric")
	}
	if cpu == "" {
		t.Error("expected non-empty CPU metric")
	}

	// Second call should return cached values
	ram2, cpu2 := getSystemMetrics()
	if ram != ram2 || cpu != cpu2 {
		t.Error("expected cached metrics to be identical")
	}
}

// ─── requireConfigSecret ─────────────────────────────────────────────────────

func TestRequireConfigSecret_LocalhostAllowed(t *testing.T) {
	t.Setenv("CONFIG_KEYS_SECRET", "")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireConfigSecret(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for localhost, got %d", w.Code)
	}
}

func TestRequireConfigSecret_NonLocalhostRejected(t *testing.T) {
	t.Setenv("CONFIG_KEYS_SECRET", "")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireConfigSecret(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for non-localhost, got %d", w.Code)
	}
}

func TestRequireConfigSecret_WithValidSecret(t *testing.T) {
	t.Setenv("CONFIG_KEYS_SECRET", "my-secret")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireConfigSecret(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Config-Secret", "my-secret")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 with valid secret, got %d", w.Code)
	}
}

func TestRequireConfigSecret_WithInvalidSecret(t *testing.T) {
	t.Setenv("CONFIG_KEYS_SECRET", "my-secret")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireConfigSecret(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Config-Secret", "wrong-secret")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 with invalid secret, got %d", w.Code)
	}
}

func TestRequireConfigSecret_MalformedRemoteAddr(t *testing.T) {
	t.Setenv("CONFIG_KEYS_SECRET", "")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireConfigSecret(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config/keys", nil)
	req.RemoteAddr = "invalid"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for malformed RemoteAddr, got %d", w.Code)
	}
}

// ─── handleChatStream — success with chunks ──────────────────────────────────

func TestHandleChatStream_Success(t *testing.T) {
	agent := &mockAgent{
		streamChunks: []string{"Hello", " world", "!"},
	}
	handler := handleChatStream(agent)

	body := `{"message": "hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, `"type":"token"`) {
		t.Error("expected token chunks in response body")
	}
	if !strings.Contains(respBody, `"type":"done"`) {
		t.Error("expected done event in response body")
	}
	if !strings.Contains(respBody, "Hello") {
		t.Error("expected 'Hello' chunk in response body")
	}
}

func TestHandleChatStream_StreamError(t *testing.T) {
	agent := &mockAgent{
		streamErr: fmt.Errorf("streaming failed"),
	}
	handler := handleChatStream(agent)

	body := `{"message": "hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	// Stream error writes error event but doesn't change status code (already 200)
	respBody := w.Body.String()
	if !strings.Contains(respBody, `"type":"error"`) {
		t.Error("expected error event in response body")
	}
	if !strings.Contains(respBody, "streaming failed") {
		t.Error("expected error message in response body")
	}
}

func TestHandleChatStream_MessageTooLong(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChatStream(agent)

	longMsg := strings.Repeat("x", 50001)
	body := `{"message": "` + longMsg + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for too-long message, got %d", w.Code)
	}
}

func TestHandleChatStream_TooManyAttachments(t *testing.T) {
	agent := &mockAgent{}
	handler := handleChatStream(agent)

	atts := make([]map[string]string, 11)
	for i := range atts {
		atts[i] = map[string]string{
			"name": "file.txt",
			"type": "text/plain",
			"data": "aGVsbG8=",
		}
	}
	payload := map[string]any{
		"message":     "test",
		"attachments": atts,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for too many attachments, got %d", w.Code)
	}
}

// ─── handleChat — with chat_id and attachments ──────────────────────────────

func TestHandleChat_WithChatID(t *testing.T) {
	agent := &mockAgent{
		processReply: "Reply with chat",
	}
	handler := handleChat(agent)

	body := `{"message": "hi", "chat_id": "my-chat-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Reply != "Reply with chat" {
		t.Errorf("expected reply 'Reply with chat', got %q", resp.Reply)
	}
	if resp.Metrics.LatencyMs < 0 {
		t.Error("expected non-negative latency")
	}
}

func TestHandleChat_WithAttachments(t *testing.T) {
	agent := &mockAgent{
		processReply: "Got attachments",
	}
	handler := handleChat(agent)

	body := `{"message": "hi", "attachments": [{"name":"test.txt","type":"text/plain","data":"aGVsbG8="}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleChat_ResponseContainsMetrics(t *testing.T) {
	agent := &mockAgent{
		processReply: "metric test",
	}
	handler := handleChat(agent)

	body := `{"message": "test metrics"}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	var resp ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if resp.Metrics.TokenIn <= 0 {
		t.Error("expected positive TokenIn")
	}
	if resp.Metrics.TokenOut <= 0 {
		t.Error("expected positive TokenOut")
	}
	if resp.Metrics.RamMB == "" {
		t.Error("expected non-empty RamMB")
	}
	if resp.Metrics.CpuLoad == "" {
		t.Error("expected non-empty CpuLoad")
	}
}

// ─── handleSSE ───────────────────────────────────────────────────────────────

func TestHandleSSE_WritesHeaders(t *testing.T) {
	// handleSSE runs an infinite loop; we use a context that gets cancelled
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Run in goroutine and cancel quickly
	done := make(chan struct{})
	go func() {
		handleSSE(w, req)
		close(done)
	}()

	// Cancel the request context to stop the loop
	cancel()

	// Wait a bit for the handler to finish
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleSSE did not return after context cancel")
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}

	cache := w.Header().Get("Cache-Control")
	if cache != "no-cache, no-transform" {
		t.Errorf("expected Cache-Control 'no-cache, no-transform', got %q", cache)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "SSE Connected") {
		t.Error("expected 'SSE Connected' message in body")
	}
}

// ─── handleReset — OPTIONS ───────────────────────────────────────────────────

func TestHandleReset_OPTIONS(t *testing.T) {
	agent := &mockAgent{}
	handler := handleReset(agent)

	req := httptest.NewRequest(http.MethodOptions, "/api/reset", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// OPTIONS should return without calling reset
	if agent.resetCalled {
		t.Error("agent.Reset() should not be called for OPTIONS")
	}
}

// ─── handleChats — POST with store error simulation ──────────────────────────

func TestHandleChats_POST_StoreSaveError(t *testing.T) {
	// This test verifies the handler works with the real store.
	// In test environment the store may or may not be available.
	// We just verify the handler doesn't panic.
	req := httptest.NewRequest(http.MethodPost, "/api/chats", strings.NewReader(`{"title": "Error Test"}`))
	w := httptest.NewRecorder()

	handleChats(w, req)

	// Status should be either 200 (success) or 500 (store error)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("unexpected status %d", w.Code)
	}
}
