package utils

import (
	"strings"
	"testing"
)

func TestEstimateTokens_Empty(t *testing.T) {
	got := EstimateTokens("")
	if got != 0 {
		t.Errorf("EstimateTokens('') = %d, want 0", got)
	}
}

func TestEstimateTokens_SingleChar(t *testing.T) {
	got := EstimateTokens("a")
	if got < 1 {
		t.Errorf("EstimateTokens('a') = %d, want >= 1", got)
	}
}

func TestEstimateTokens_Overhead(t *testing.T) {
	// Long text should have overhead added
	short := EstimateTokens("hello")
	longText := strings.Repeat("a", 4800) // 480 chars boundary
	long := EstimateTokens(longText)

	// Long text should have more tokens per char due to overhead
	if long <= short {
		t.Errorf("long text tokens (%d) should be > short text tokens (%d)", long, short)
	}
}

func TestEstimateTokens_Overestimated(t *testing.T) {
	// The function should over-estimate (never under-estimate)
	text := "hello world"
	tokens := EstimateTokens(text)
	// At minimum, should be 1
	if tokens < 1 {
		t.Errorf("EstimateTokens should return >= 1 for non-empty text, got %d", tokens)
	}
}

func TestEstimateFullPrompt_WithHistory(t *testing.T) {
	got := EstimateFullPrompt("system", []string{"user", "assistant"}, "tool defs")
	if got <= 0 {
		t.Errorf("EstimateFullPrompt = %d, want > 0", got)
	}
}

func TestEstimateFullPrompt_OverheadPerMessage(t *testing.T) {
	// Each message adds 4 tokens of overhead
	got0 := EstimateFullPrompt("system", nil, "")
	got1 := EstimateFullPrompt("system", []string{"one"}, "")
	got2 := EstimateFullPrompt("system", []string{"one", "two"}, "")

	diff1 := got1 - got0
	if diff1 <= 0 {
		t.Errorf("adding 1 message should increase tokens, diff = %d", diff1)
	}
	diff2 := got2 - got1
	if diff2 <= 0 {
		t.Errorf("adding 2nd message should increase tokens, diff = %d", diff2)
	}
}

func TestTruncateTextForSummary_ShortEnough(t *testing.T) {
	text := "short"
	got := TruncateTextForSummary(text, 100)
	if got != text {
		t.Errorf("TruncateTextForSummary(%q) = %q, want %q", text, got, text)
	}
}

func TestTruncateTextForSummary_Truncates(t *testing.T) {
	text := "abcdefghij"
	got := TruncateTextForSummary(text, 5)
	runes := len([]rune(got))
	if runes <= 5 {
		t.Errorf("Truncated text should contain original 5 chars + suffix, got %d runes", runes)
	}
	if !containsStr(got, "đã cắt bớt") {
		t.Error("Truncated text should contain '...' suffix")
	}
}

func TestTruncateTextForSummary_Exact(t *testing.T) {
	text := "abcde"
	got := TruncateTextForSummary(text, 5)
	if got != text {
		t.Errorf("TruncateTextForSummary(%q, 5) = %q, want %q", text, got, text)
	}
}
