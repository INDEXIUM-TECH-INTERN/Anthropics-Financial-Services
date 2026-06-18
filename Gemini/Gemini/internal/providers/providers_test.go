package providers

import (
	"context"
	"errors"
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