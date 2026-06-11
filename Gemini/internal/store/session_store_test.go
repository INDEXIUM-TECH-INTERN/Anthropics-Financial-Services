package store

import (
	"testing"
	"time"

	"gemini-cli/internal/models/messaging"
)

func TestSaveSession(t *testing.T) {
	sess := &ChatSession{
		ID:       "test_001",
		Title:    "Test Chat",
		Messages: []messaging.Message{},
	}
	if err := SaveSession(sess); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSaveSession_EmptyID(t *testing.T) {
	sess := &ChatSession{ID: ""}
	err := SaveSession(sess)
	if err == nil {
		t.Error("expected error for empty session ID")
	}
}

func TestGetSession_Existing(t *testing.T) {
	sess := &ChatSession{
		ID:    "test_get_001",
		Title: "Existing Chat",
	}
	SaveSession(sess)

	got, err := GetSession("test_get_001")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got.ID != "test_get_001" {
		t.Errorf("expected ID test_get_001, got %s", got.ID)
	}
	if got.Title != "Existing Chat" {
		t.Errorf("expected title 'Existing Chat', got %s", got.Title)
	}
}

func TestGetSession_NonExistent(t *testing.T) {
	got, err := GetSession("nonexistent_id_xyz")
	if err != nil {
		t.Fatalf("GetSession should not error for nonexistent: %v", err)
	}
	if got.ID != "nonexistent_id_xyz" {
		t.Errorf("expected ID nonexistent_id_xyz, got %s", got.ID)
	}
	if got.Title != "Cuộc trò chuyện mới" {
		t.Errorf("expected default title, got %s", got.Title)
	}
}

func TestListSessions(t *testing.T) {
	// Clear existing
	memMu.Lock()
	memSessions = make(map[string]*ChatSession)
	memMu.Unlock()

	SaveSession(&ChatSession{ID: "list_001", Title: "Chat 1"})
	SaveSession(&ChatSession{ID: "list_002", Title: "Chat 2"})

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
	// Sessions should not include messages (lightweight)
	for _, s := range sessions {
		if s.Messages != nil {
			t.Error("expected nil messages in lightweight listing")
		}
	}
}

func TestDeleteSession(t *testing.T) {
	SaveSession(&ChatSession{ID: "del_001", Title: "To Delete"})

	if err := DeleteSession("del_001"); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// After delete, should return default session
	got, err := GetSession("del_001")
	if err != nil {
		t.Fatalf("GetSession after delete failed: %v", err)
	}
	if got.Title != "Cuộc trò chuyện mới" {
		t.Error("expected default session after delete")
	}
}

func TestSessionKey(t *testing.T) {
	key := sessionKey("test123")
	if key != "chat:session:test123" {
		t.Errorf("expected chat:session:test123, got %s", key)
	}
}

func TestListKey(t *testing.T) {
	key := listKey()
	if key != "chat:sessions:list" {
		t.Errorf("expected chat:sessions:list, got %s", key)
	}
}

func TestChatSession_Serialize(t *testing.T) {
	now := time.Now()
	sess := &ChatSession{
		ID:        "serial_001",
		Title:     "Serialization Test",
		Messages:  []messaging.Message{{Role: messaging.RoleUser, Content: "hello"}},
		UpdatedAt: now,
	}

	if sess.ID != "serial_001" {
		t.Errorf("expected ID serial_001, got %s", sess.ID)
	}
	if len(sess.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(sess.Messages))
	}
}
