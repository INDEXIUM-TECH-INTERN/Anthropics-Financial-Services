package utils

import (
	"testing"
	"time"
)

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		keywords []string
		want     bool
	}{
		{"empty string", "", []string{"a"}, false},
		{"no keywords", "hello", nil, false},
		{"single keyword found", "hello world", []string{"world"}, true},
		{"single keyword not found", "hello world", []string{"foo"}, false},
		{"multiple keywords first match", "hello world", []string{"hello", "foo"}, true},
		{"multiple keywords second match", "hello world", []string{"foo", "world"}, true},
		{"multiple keywords no match", "hello world", []string{"foo", "bar"}, false},
		{"unicode text", "xin chào Việt Nam", []string{"Việt"}, true},
		{"case sensitive", "Hello World", []string{"hello"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsAny(tt.s, tt.keywords...)
			if got != tt.want {
				t.Errorf("ContainsAny(%q, %v) = %v, want %v", tt.s, tt.keywords, got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		min   int
		max   int
	}{
		{"empty", "", 0, 0},
		{"single char", "a", 1, 5},
		{"short english", "hello world", 2, 20},
		{"vietnamese text", "xin chào Việt Nam", 2, 30},
		{"long text", string(make([]byte, 3400)), 50, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.min || got > tt.max {
				t.Errorf("EstimateTokens() = %d, want between %d and %d", got, tt.min, tt.max)
			}
		})
	}
}

func TestEstimateTokens_NeverNegative(t *testing.T) {
	got := EstimateTokens("a")
	if got < 1 {
		t.Errorf("EstimateTokens('a') = %d, want >= 1", got)
	}
}

func TestEstimateFullPrompt(t *testing.T) {
	got := EstimateFullPrompt("system prompt", []string{"user msg", "assistant msg"}, "tool defs")
	if got <= 0 {
		t.Errorf("EstimateFullPrompt() = %d, want > 0", got)
	}
}

func TestEstimateFullPrompt_Empty(t *testing.T) {
	got := EstimateFullPrompt("", nil, "")
	// Should still have overhead: len(nil)*4 + 30 = 30
	if got < 30 {
		t.Errorf("EstimateFullPrompt(empty) = %d, want >= 30", got)
	}
}

func TestTruncateTextForSummary(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxChars int
		wantLen  int // expected rune count
	}{
		{"short text no truncate", "hello", 100, 5},
		{"exact length", "hello", 5, 5},
		{"needs truncate", "hello world", 5, 5 + len([]rune("\n... [đã cắt bớt để tóm tắt]"))},
		{"unicode truncate", "xin chào Việt Nam", 4, 4 + len([]rune("\n... [đã cắt bớt để tóm tắt]"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateTextForSummary(tt.text, tt.maxChars)
			gotRunes := len([]rune(got))
			if gotRunes != tt.wantLen {
				t.Errorf("TruncateTextForSummary() rune count = %d, want %d", gotRunes, tt.wantLen)
			}
		})
	}
}

func TestTruncateTextForSummary_NoTruncation(t *testing.T) {
	text := "short"
	got := TruncateTextForSummary(text, 100)
	if got != text {
		t.Errorf("TruncateTextForSummary() = %q, want %q", got, text)
	}
}

func TestTranslateWeekday(t *testing.T) {
	tests := []struct {
		name string
		day  time.Weekday
		want string
	}{
		{"Sunday", time.Sunday, "Chủ Nhật"},
		{"Monday", time.Monday, "Thứ Hai"},
		{"Tuesday", time.Tuesday, "Thứ Ba"},
		{"Wednesday", time.Wednesday, "Thứ Tư"},
		{"Thursday", time.Thursday, "Thứ Năm"},
		{"Friday", time.Friday, "Thứ Sáu"},
		{"Saturday", time.Saturday, "Thứ Bảy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateWeekday(tt.day)
			if got != tt.want {
				t.Errorf("TranslateWeekday(%d) = %q, want %q", tt.day, got, tt.want)
			}
		})
	}
}

func TestGetFileContentWrapper(t *testing.T) {
	got := GetFileContentWrapper("test.txt", "content here")
	if got == "" {
		t.Error("GetFileContentWrapper returned empty string")
	}
	// Should contain the filename
	if !containsStr(got, "test.txt") {
		t.Error("GetFileContentWrapper should contain filename")
	}
	// Should contain the content
	if !containsStr(got, "content here") {
		t.Error("GetFileContentWrapper should contain file content")
	}
}

func containsStr(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && containsSubstring(s, sub)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
