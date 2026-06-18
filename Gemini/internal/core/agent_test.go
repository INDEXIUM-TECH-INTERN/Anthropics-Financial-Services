package core

import (
	"testing"
	"time"

	"gemini-cli/internal/models/messaging"
	"gemini-cli/internal/providers"
)

func TestNewAgent(t *testing.T) {
	a := NewAgent()
	if a == nil {
		t.Fatal("NewAgent returned nil")
	}
	if a.pm == nil {
		t.Error("NewAgent: pm is nil")
	}
	if a.systemPrompt == "" {
		t.Error("NewAgent: systemPrompt is empty")
	}
	if a.conversation == nil {
		t.Error("NewAgent: conversation is nil")
	}
	if a.orchestrator == nil {
		t.Error("NewAgent: orchestrator is nil")
	}
	if a.dispatcher == nil {
		t.Error("NewAgent: dispatcher is nil")
	}
}

func TestAgentGetHistory_Empty(t *testing.T) {
	a := NewAgent()
	history := a.GetHistory()
	if history == nil {
		t.Error("GetHistory returned nil, expected empty slice")
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d messages", len(history))
	}
}

func TestAgentGetHistory_Copy(t *testing.T) {
	a := NewAgent()
	// Access internal conversation directly to add a message
	a.conversation.ContextWindow.AddMessage(messaging.Message{
		Role:    messaging.RoleUser,
		Content: "test",
	})

	history := a.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}

	// Modifying the returned slice should NOT affect the original
	history[0].Content = "modified"
	original := a.GetHistory()
	if original[0].Content == "modified" {
		t.Error("GetHistory should return a copy, not a reference")
	}
}

func TestAgentLoadHistory(t *testing.T) {
	a := NewAgent()
	msgs := []messaging.Message{
		{Role: messaging.RoleUser, Content: "hello"},
		{Role: messaging.RoleAssistant, Content: "hi there"},
		{Role: messaging.RoleUser, Content: "how are you"},
	}
	a.LoadHistory(msgs)

	history := a.GetHistory()
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected 'hello', got %s", history[0].Content)
	}
}

func TestAgentLoadHistory_CopySemantics(t *testing.T) {
	a := NewAgent()
	msgs := []messaging.Message{
		{Role: messaging.RoleUser, Content: "original"},
	}
	a.LoadHistory(msgs)

	// Modifying original slice should not affect agent's history
	msgs[0].Content = "modified"
	history := a.GetHistory()
	if history[0].Content != "original" {
		t.Errorf("LoadHistory should copy messages, got %s", history[0].Content)
	}
}

func TestAgentLoadHistory_Replacement(t *testing.T) {
	a := NewAgent()
	a.conversation.ContextWindow.AddMessage(messaging.Message{
		Role:    messaging.RoleUser,
		Content: "old message",
	})

	a.LoadHistory([]messaging.Message{
		{Role: messaging.RoleUser, Content: "new message"},
	})

	history := a.GetHistory()
	if len(history) != 1 {
		t.Fatalf("expected 1 message after replacement, got %d", len(history))
	}
	if history[0].Content != "new message" {
		t.Errorf("expected 'new message', got %s", history[0].Content)
	}
}

func TestAgentGetProvider(t *testing.T) {
	a := NewAgent()
	p := a.GetProvider()
	if p == nil {
		t.Error("GetProvider returned nil")
	}
}

func TestAgentReset(t *testing.T) {
	a := NewAgent()
	a.conversation.ContextWindow.AddMessage(messaging.Message{
		Role:    messaging.RoleUser,
		Content: "test message",
	})

	a.Reset()

	history := a.GetHistory()
	if len(history) != 0 {
		t.Errorf("expected empty history after reset, got %d", len(history))
	}
}

func TestAgentWithMockProvider(t *testing.T) {
	mp := providers.NewMockProviderWithText("mock response")
	a := newAgentWithProvider(mp)

	if a == nil {
		t.Fatal("newAgentWithProvider returned nil")
	}

	p := a.GetProvider()
	if p == nil {
		t.Error("provider is nil")
	}
}


func TestBuildGroundedSystemPrompt(t *testing.T) {
	// Use a fixed time: 2026-06-14 12:00:00 UTC (Sunday)
	prompt := buildGroundedSystemPrompt(time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC))
	if prompt == "" {
		t.Error("buildGroundedSystemPrompt returned empty string")
	}
}
