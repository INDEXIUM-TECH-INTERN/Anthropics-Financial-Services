package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gemini-cli/internal/models/messaging"
)

// ─── MockProvider ─────────────────────────────────────────────────────────────

func TestMockProvider_GenerateText(t *testing.T) {
	t.Run("returns configured texts in order", func(t *testing.T) {
		p := NewMockProviderWithText("first", "second")
		r1, err := p.GenerateText("sys", "user")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r1 != "first" {
			t.Errorf("expected 'first', got %q", r1)
		}
		r2, _ := p.GenerateText("sys", "user")
		if r2 != "second" {
			t.Errorf("expected 'second', got %q", r2)
		}
	})

	t.Run("returns default after exhausting texts", func(t *testing.T) {
		p := NewMockProviderWithText("only")
		p.GenerateText("sys", "user") // exhaust
		r, _ := p.GenerateText("sys", "user")
		if r != "mock response" {
			t.Errorf("expected 'mock response', got %q", r)
		}
	})

	t.Run("returns error when configured", func(t *testing.T) {
		p := &MockProvider{GenerateTextErr: errors.New("fail")}
		_, err := p.GenerateText("sys", "user")
		if err == nil || err.Error() != "fail" {
			t.Errorf("expected 'fail' error, got %v", err)
		}
	})
}

func TestMockProvider_Generate(t *testing.T) {
	t.Run("returns configured responses", func(t *testing.T) {
		p := &MockProvider{
			Responses: []messaging.Message{
				{Role: messaging.RoleAssistant, Content: "hello"},
			},
		}
		msg, err := p.Generate(context.Background(), messaging.Request{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.Content != "hello" {
			t.Errorf("expected 'hello', got %q", msg.Content)
		}
	})

	t.Run("returns error when configured", func(t *testing.T) {
		p := &MockProvider{GenerateErr: errors.New("gen fail")}
		_, err := p.Generate(context.Background(), messaging.Request{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("tracks call count", func(t *testing.T) {
		p := &MockProvider{}
		p.Generate(context.Background(), messaging.Request{})
		p.Generate(context.Background(), messaging.Request{})
		if p.CallCount != 2 {
			t.Errorf("expected CallCount 2, got %d", p.CallCount)
		}
	})
}

func TestMockProvider_GenerateStream(t *testing.T) {
	t.Run("delivers chunks and done", func(t *testing.T) {
		p := &MockProvider{
			StreamChunks: []StreamChunk{
				{Text: "chunk1"},
				{Text: "chunk2"},
			},
		}
		var received []string
		err := p.GenerateStream(context.Background(), messaging.Request{}, func(sc StreamChunk) {
			if !sc.Done {
				received = append(received, sc.Text)
			}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(received) != 2 || received[0] != "chunk1" || received[1] != "chunk2" {
			t.Errorf("unexpected chunks: %v", received)
		}
	})

	t.Run("returns error when configured", func(t *testing.T) {
		p := &MockProvider{GenerateStreamErr: errors.New("stream fail")}
		err := p.GenerateStream(context.Background(), messaging.Request{}, func(sc StreamChunk) {})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNewMockProviderWithTools(t *testing.T) {
	p := NewMockProviderWithTools("search", map[string]any{"query": "test"}, "final answer")
	if len(p.Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(p.Responses))
	}
	if len(p.Responses[0].ToolCalls) != 1 {
		t.Fatal("expected 1 tool call in first response")
	}
	if p.Responses[0].ToolCalls[0].Name != "search" {
		t.Errorf("expected tool name 'search', got %q", p.Responses[0].ToolCalls[0].Name)
	}
}

func TestErrIsProviderFailure(t *testing.T) {
	if ErrIsProviderFailure(nil) {
		t.Error("nil should not be a provider failure")
	}
	if !ErrIsProviderFailure(errors.New("Tat ca cac dich vu AI deu fail")) {
		t.Error("expected provider failure detection")
	}
	if ErrIsProviderFailure(errors.New("some other error")) {
		t.Error("non-provider error should not match")
	}
}

// ─── isQuotaOrRateLimitError ──────────────────────────────────────────────────

func TestIsQuotaOrRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"quota error", errors.New("quota exceeded"), true},
		{"rate limit error", errors.New("rate limit reached"), true},
		{"429 error", errors.New("HTTP 429 too many requests"), true},
		{"exceeded error", errors.New("limit exceeded"), true},
		{"auth error", errors.New("401 unauthorized"), false},
		{"random error", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isQuotaOrRateLimitError(tt.err)
			if got != tt.want {
				t.Errorf("isQuotaOrRateLimitError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// ─── isGeminiSupportedMime ────────────────────────────────────────────────────

func TestIsGeminiSupportedMime(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		// Images
		{"image/png", true},
		{"image/jpeg", true},
		{"image/webp", true},
		// Video
		{"video/mp4", true},
		// Audio
		{"audio/mpeg", true},
		// Documents
		{"application/pdf", true},
		// Text formats
		{"text/plain", true},
		{"text/html", true},
		{"text/css", true},
		{"text/javascript", true},
		{"text/x-typescript", true},
		{"text/csv", true},
		{"text/markdown", true},
		{"text/x-python", true},
		{"text/x-go", true},
		{"application/json", true},
		{"application/xml", true},
		// Unsupported
		{"application/zip", false},
		{"application/x-executable", false},
		{"", false},
		{"text/unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := isGeminiSupportedMime(tt.mime)
			if got != tt.want {
				t.Errorf("isGeminiSupportedMime(%q) = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}

// ─── MultiProvider ────────────────────────────────────────────────────────────

func TestNewMultiProvider(t *testing.T) {
	primary := NewMockProviderWithText("primary")
	fb := NewMockProviderWithText("fallback")
	mp := NewMultiProvider(primary, []Provider{fb})
	if mp == nil {
		t.Fatal("expected non-nil MultiProvider")
	}
	if mp.primary != primary {
		t.Error("primary not set correctly")
	}
	if len(mp.fallbacks) != 1 {
		t.Errorf("expected 1 fallback, got %d", len(mp.fallbacks))
	}
}

func TestMultiProvider_GenerateText_PrimarySuccess(t *testing.T) {
	primary := NewMockProviderWithText("from primary")
	fb := NewMockProviderWithText("from fallback")
	mp := NewMultiProvider(primary, []Provider{fb})

	result, err := mp.GenerateText("sys", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "from primary" {
		t.Errorf("expected 'from primary', got %q", result)
	}
}

func TestMultiProvider_GenerateText_FallbackOnPrimaryError(t *testing.T) {
	primary := &MockProvider{GenerateTextErr: errors.New("primary failed")}
	fb := NewMockProviderWithText("from fallback")
	mp := NewMultiProvider(primary, []Provider{fb})

	result, err := mp.GenerateText("sys", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "from fallback" {
		t.Errorf("expected 'from fallback', got %q", result)
	}
}

func TestMultiProvider_GenerateText_AllFail(t *testing.T) {
	primary := &MockProvider{GenerateTextErr: errors.New("primary failed")}
	fb := &MockProvider{GenerateTextErr: errors.New("fallback failed")}
	mp := NewMultiProvider(primary, []Provider{fb})

	_, err := mp.GenerateText("sys", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Tất cả các dịch vụ AI") {
		t.Errorf("expected aggregated error, got %v", err)
	}
}

func TestMultiProvider_GenerateText_NoFallbacks(t *testing.T) {
	primary := &MockProvider{GenerateTextErr: errors.New("primary failed")}
	mp := NewMultiProvider(primary, nil)

	_, err := mp.GenerateText("sys", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no fallbacks configured") {
		t.Errorf("expected 'no fallbacks' error, got %v", err)
	}
}

func TestMultiProvider_Generate_PrimarySuccess(t *testing.T) {
	primary := &MockProvider{
		Responses: []messaging.Message{
			{Role: messaging.RoleAssistant, Content: "primary response"},
		},
	}
	fb := NewMockProviderWithText("fallback response")
	mp := NewMultiProvider(primary, []Provider{fb})

	msg, err := mp.Generate(context.Background(), messaging.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Content != "primary response" {
		t.Errorf("expected 'primary response', got %q", msg.Content)
	}
}

func TestMultiProvider_Generate_FallbackOnPrimaryError(t *testing.T) {
	primary := &MockProvider{GenerateErr: errors.New("primary failed")}
	fb := &MockProvider{
		Responses: []messaging.Message{
			{Role: messaging.RoleAssistant, Content: "fallback response"},
		},
	}
	mp := NewMultiProvider(primary, []Provider{fb})

	msg, err := mp.Generate(context.Background(), messaging.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Content != "fallback response" {
		t.Errorf("expected 'fallback response', got %q", msg.Content)
	}
}

func TestMultiProvider_Generate_QuotaSkipsPrimary(t *testing.T) {
	callCount := 0
	primary := &MockProvider{
		GenerateErr: fmt.Errorf("quota exceeded"),
	}
	fb := &MockProvider{
		Responses: []messaging.Message{
			{Role: messaging.RoleAssistant, Content: "fallback"},
		},
	}
	mp := NewMultiProvider(primary, []Provider{fb})

	// First call: primary fails with quota, fallback succeeds
	_, err := mp.Generate(context.Background(), messaging.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After quota error, skipPrimaryUntil should be set
	mp.mu.Lock()
	skip := mp.skipPrimaryUntil
	mp.mu.Unlock()
	if skip <= 0 {
		t.Error("expected skipPrimaryUntil > 0 after quota error")
	}
	_ = callCount
}

func TestMultiProvider_GenerateStream_PrimarySuccess(t *testing.T) {
	primary := &MockProvider{
		StreamChunks: []StreamChunk{{Text: "streamed"}},
	}
	fb := &MockProvider{}
	mp := NewMultiProvider(primary, []Provider{fb})

	var received []string
	err := mp.GenerateStream(context.Background(), messaging.Request{}, func(sc StreamChunk) {
		if !sc.Done {
			received = append(received, sc.Text)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) != 1 || received[0] != "streamed" {
		t.Errorf("unexpected stream chunks: %v", received)
	}
}

func TestMultiProvider_GenerateStream_FallbackOnError(t *testing.T) {
	primary := &MockProvider{GenerateStreamErr: errors.New("stream failed")}
	fb := &MockProvider{
		StreamChunks: []StreamChunk{{Text: "fallback stream"}},
	}
	mp := NewMultiProvider(primary, []Provider{fb})

	var received []string
	err := mp.GenerateStream(context.Background(), messaging.Request{}, func(sc StreamChunk) {
		if !sc.Done {
			received = append(received, sc.Text)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) != 1 || received[0] != "fallback stream" {
		t.Errorf("unexpected stream chunks: %v", received)
	}
}

func TestMultiProvider_GenerateStream_AllFail(t *testing.T) {
	primary := &MockProvider{GenerateStreamErr: errors.New("primary stream fail")}
	fb := &MockProvider{GenerateStreamErr: errors.New("fallback stream fail")}
	mp := NewMultiProvider(primary, []Provider{fb})

	err := mp.GenerateStream(context.Background(), messaging.Request{}, func(sc StreamChunk) {})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Tất cả các dịch vụ AI streaming") {
		t.Errorf("expected aggregated stream error, got %v", err)
	}
}

func TestMultiProvider_RoundRobinFallbacks(t *testing.T) {
	fb1 := &MockProvider{GenerateErr: errors.New("fb1 fail")}
	fb2 := &MockProvider{
		Responses: []messaging.Message{
			{Role: messaging.RoleAssistant, Content: "fb2 response"},
		},
	}
	primary := &MockProvider{GenerateErr: errors.New("primary fail")}
	mp := NewMultiProvider(primary, []Provider{fb1, fb2})

	msg, err := mp.Generate(context.Background(), messaging.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Content != "fb2 response" {
		t.Errorf("expected 'fb2 response', got %q", msg.Content)
	}

	// Next call should start from fb2 (round-robin)
	mp.currentIdx = 0 // reset for test
	fb1.GenerateErr = nil
	fb1.Responses = []messaging.Message{
		{Role: messaging.RoleAssistant, Content: "fb1 response"},
	}
}

func TestMultiProvider_ResetFailuresOnSuccess(t *testing.T) {
	primary := &MockProvider{
		Responses: []messaging.Message{
			{Role: messaging.RoleAssistant, Content: "ok"},
		},
	}
	mp := NewMultiProvider(primary, nil)

	// Simulate prior failures
	mp.primaryFailures = 5
	mp.skipPrimaryUntil = 10

	_, err := mp.Generate(context.Background(), messaging.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mp.primaryFailures != 0 {
		t.Errorf("expected primaryFailures reset to 0, got %d", mp.primaryFailures)
	}
	if mp.skipPrimaryUntil != 0 {
		t.Errorf("expected skipPrimaryUntil reset to 0, got %d", mp.skipPrimaryUntil)
	}
}

// ─── StreamChunk ──────────────────────────────────────────────────────────────

func TestStreamChunk_Done(t *testing.T) {
	sc := StreamChunk{Done: true, Text: "complete", Metrics: &StreamMetrics{TokenIn: 10, TokenOut: 20}}
	if !sc.Done {
		t.Error("expected Done=true")
	}
	if sc.Text != "complete" {
		t.Errorf("expected Text='complete', got %q", sc.Text)
	}
	if sc.Metrics == nil {
		t.Fatal("expected non-nil Metrics")
	}
	if sc.Metrics.TokenIn != 10 {
		t.Errorf("expected TokenIn=10, got %d", sc.Metrics.TokenIn)
	}
	if sc.Metrics.TokenOut != 20 {
		t.Errorf("expected TokenOut=20, got %d", sc.Metrics.TokenOut)
	}
}

// ─── OpenRouter free models ──────────────────────────────────────────────────

func TestOpenRouterFreeModelsNotEmpty(t *testing.T) {
	if len(openRouterFreeModels) == 0 {
		t.Error("openRouterFreeModels should not be empty")
	}
	for i, m := range openRouterFreeModels {
		if m == "" {
			t.Errorf("model at index %d is empty", i)
		}
	}
}
