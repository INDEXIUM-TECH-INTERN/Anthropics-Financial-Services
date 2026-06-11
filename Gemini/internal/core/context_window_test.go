package core

import (
	"testing"

	"gemini-cli/internal/models/messaging"
)

func TestNewContextWindow(t *testing.T) {
	cw := NewContextWindow()
	if cw == nil {
		t.Fatal("NewContextWindow returned nil")
	}
	if len(cw.History) != 0 {
		t.Errorf("expected empty history, got %d", len(cw.History))
	}
	if cw.MemorySummary != "" {
		t.Errorf("expected empty summary, got %s", cw.MemorySummary)
	}
}

func TestAddMessage(t *testing.T) {
	cw := NewContextWindow()
	msg := messaging.Message{Role: messaging.RoleUser, Content: "hello"}
	cw.AddMessage(msg)

	if len(cw.History) != 1 {
		t.Errorf("expected 1 message, got %d", len(cw.History))
	}
	if cw.History[0].Content != "hello" {
		t.Errorf("expected 'hello', got %s", cw.History[0].Content)
	}
}

func TestReset(t *testing.T) {
	cw := NewContextWindow()
	cw.AddMessage(messaging.Message{Role: messaging.RoleUser, Content: "test"})
	cw.MemorySummary = "some summary"
	cw.Reset()

	if len(cw.History) != 0 {
		t.Errorf("expected 0 history after reset, got %d", len(cw.History))
	}
	if cw.MemorySummary != "" {
		t.Error("expected empty summary after reset")
	}
}

func TestBuildLLMHistory_Empty(t *testing.T) {
	cw := NewContextWindow()
	result := cw.BuildLLMHistory(7)
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestBuildLLMHistory_WithSummary(t *testing.T) {
	cw := NewContextWindow()
	cw.MemorySummary = "previous context summary"
	cw.History = []messaging.Message{
		{Role: messaging.RoleUser, Content: "old msg 1"},
		{Role: messaging.RoleAssistant, Content: "old msg 2"},
		{Role: messaging.RoleUser, Content: "recent msg"},
	}

	result := cw.BuildLLMHistory(1)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 messages (summary + recent), got %d", len(result))
	}
	// First message should be the summary
	if result[0].Role != messaging.RoleUser {
		t.Errorf("expected summary as user role, got %s", result[0].Role)
	}
}

func TestBuildLLMHistory_ProtectedBootstrap(t *testing.T) {
	cw := NewContextWindow()
	cw.History = []messaging.Message{
		{Role: messaging.RoleUser, Content: "user query"},
		{Role: messaging.RoleUser, Content: "bootstrap context"},
		{Role: messaging.RoleAssistant, Content: "response 1"},
		{Role: messaging.RoleUser, Content: "follow up"},
		{Role: messaging.RoleAssistant, Content: "response 2"},
	}

	result := cw.BuildLLMHistory(2)
	// Should have: bootstrap (2 first) + last 2 recent
	if len(result) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(result))
	}
	// First message should be the original user query
	if result[0].Content != "user query" {
		t.Errorf("expected first msg to be 'user query', got %s", result[0].Content)
	}
}

func TestShouldSummarize(t *testing.T) {
	cw := NewContextWindow()
	// Not enough history
	cw.History = []messaging.Message{
		{Role: messaging.RoleUser, Content: "hi"},
	}
	if cw.ShouldSummarize(100, 7) {
		t.Error("should not summarize with only 1 message")
	}

	// Many messages but small content
	for i := 0; i < 20; i++ {
		cw.History = append(cw.History, messaging.Message{
			Role:    messaging.RoleUser,
			Content: "small",
		})
	}
	// 20 small messages = ~100 tokens, threshold 100
	if cw.ShouldSummarize(50, 7) {
		t.Error("should not summarize when under threshold")
	}
}

func TestEstimateCurrentTokens(t *testing.T) {
	cw := NewContextWindow()
	cw.History = []messaging.Message{
		{Role: messaging.RoleUser, Content: "hello world"},
	}
	tokens := cw.EstimateCurrentTokens()
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
	// "hello world" = 11 chars, /4 = ~2 tokens + overhead
	if tokens < 2 {
		t.Errorf("expected at least 2 tokens, got %d", tokens)
	}
}

func TestGetFullHistory(t *testing.T) {
	cw := NewContextWindow()
	cw.History = []messaging.Message{
		{Role: messaging.RoleUser, Content: "test"},
	}
	full := cw.GetFullHistory()
	if len(full) != 1 {
		t.Fatalf("expected 1 message, got %d", len(full))
	}
	// Verify it's a copy
	full[0].Content = "modified"
	if cw.History[0].Content == "modified" {
		t.Error("GetFullHistory should return a copy, not a reference")
	}
}

func TestUpdateSummary(t *testing.T) {
	cw := NewContextWindow()
	cw.UpdateSummary("new summary", 5)
	if cw.MemorySummary != "new summary" {
		t.Errorf("expected 'new summary', got %s", cw.MemorySummary)
	}
	if cw.lastSummarizedIdx != 5 {
		t.Errorf("expected lastSummarizedIdx=5, got %d", cw.lastSummarizedIdx)
	}
}
