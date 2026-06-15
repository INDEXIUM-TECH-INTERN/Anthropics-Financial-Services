package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestErrProviderFailure(t *testing.T) {
	inner := errors.New("connection refused")
	err := &ErrProviderFailure{Err: inner}

	want := "all AI providers failed: connection refused"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Unwrap should return inner error
	if got := errors.Unwrap(err); got != inner {
		t.Errorf("Unwrap() = %v, want %v", got, inner)
	}
}

func TestErrProviderFailure_NilInner(t *testing.T) {
	err := &ErrProviderFailure{Err: nil}
	got := err.Error()
	if !strings.Contains(got, "all AI providers failed") {
		t.Errorf("Error() should contain 'all AI providers failed', got %q", got)
	}
}

func TestErrRoutingFailure(t *testing.T) {
	err := &ErrRoutingFailure{Query: "test query for routing", Reason: "no match"}

	got := err.Error()
	if !strings.Contains(got, "Routing failed") {
		t.Errorf("Error() should contain 'Routing failed', got %q", got)
	}
	if !strings.Contains(got, "no match") {
		t.Errorf("Error() should contain reason, got %q", got)
	}
}

func TestErrRoutingFailure_LongQuery(t *testing.T) {
	longQuery := strings.Repeat("a", 100)
	err := &ErrRoutingFailure{Query: longQuery, Reason: "test"}
	got := err.Error()

	// Should truncate query to 40 chars + "..."
	if strings.Contains(got, longQuery) {
		t.Error("Error() should truncate long query")
	}
	if !strings.Contains(got, "...") {
		t.Error("Error() should contain '...' for truncated query")
	}
}

func TestErrContextOverflow(t *testing.T) {
	err := &ErrContextOverflow{Tokens: 5000, MaxTokens: 4096}

	want := "Context overflow: 5000 tokens exceeds limit of 4096"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrSessionNotFound(t *testing.T) {
	err := &ErrSessionNotFound{SessionID: "sess-123"}

	want := "Session not found: sess-123"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrToolExecution(t *testing.T) {
	inner := errors.New("timeout")
	err := &ErrToolExecution{ToolName: "search", Err: inner}

	want := "Tool 'search' execution failed: timeout"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Unwrap should return inner error
	if got := errors.Unwrap(err); got != inner {
		t.Errorf("Unwrap() = %v, want %v", got, inner)
	}
}

func TestErrToolExecution_NilInner(t *testing.T) {
	err := &ErrToolExecution{ToolName: "fetch", Err: nil}
	got := err.Error()
	if !strings.Contains(got, "Tool 'fetch' execution failed") {
		t.Errorf("Error() should contain tool name, got %q", got)
	}
}

func TestNewProviderFailure(t *testing.T) {
	inner := errors.New("rate limited")
	err := NewProviderFailure(inner)
	if err == nil {
		t.Fatal("NewProviderFailure returned nil")
	}
	var pf *ErrProviderFailure
	if !errors.As(err, &pf) {
		t.Error("NewProviderFailure should return *ErrProviderFailure")
	}
}

func TestNewRoutingFailure(t *testing.T) {
	err := NewRoutingFailure("query", "reason")
	if err == nil {
		t.Fatal("NewRoutingFailure returned nil")
	}
	var rf *ErrRoutingFailure
	if !errors.As(err, &rf) {
		t.Error("NewRoutingFailure should return *ErrRoutingFailure")
	}
	if rf.Query != "query" || rf.Reason != "reason" {
		t.Errorf("unexpected fields: query=%s reason=%s", rf.Query, rf.Reason)
	}
}

func TestNewContextOverflow(t *testing.T) {
	err := NewContextOverflow(100, 50)
	if err == nil {
		t.Fatal("NewContextOverflow returned nil")
	}
	var co *ErrContextOverflow
	if !errors.As(err, &co) {
		t.Error("NewContextOverflow should return *ErrContextOverflow")
	}
	if co.Tokens != 100 || co.MaxTokens != 50 {
		t.Errorf("unexpected fields: tokens=%d max=%d", co.Tokens, co.MaxTokens)
	}
}

func TestNewSessionNotFound(t *testing.T) {
	err := NewSessionNotFound("abc")
	if err == nil {
		t.Fatal("NewSessionNotFound returned nil")
	}
	var snf *ErrSessionNotFound
	if !errors.As(err, &snf) {
		t.Error("NewSessionNotFound should return *ErrSessionNotFound")
	}
	if snf.SessionID != "abc" {
		t.Errorf("unexpected session ID: %s", snf.SessionID)
	}
}

func TestNewToolExecution(t *testing.T) {
	inner := errors.New("fail")
	err := NewToolExecution("tool", inner)
	if err == nil {
		t.Fatal("NewToolExecution returned nil")
	}
	var te *ErrToolExecution
	if !errors.As(err, &te) {
		t.Error("NewToolExecution should return *ErrToolExecution")
	}
	if te.ToolName != "tool" {
		t.Errorf("unexpected tool name: %s", te.ToolName)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"unicode", "xin chào", 4, "xin ..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
