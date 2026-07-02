package worldnews

import (
	"strings"
	"testing"
)

func TestNormalizeAISummaryWordsNoEllipsis(t *testing.T) {
	long := strings.Repeat("Thị trường tài chính ghi nhận biến động đáng chú ý. ", 80) + "Nhà đầu tư nên theo dõi chính sách pháp lý và"
	got := normalizeAISummaryWords(long, HighlightSummaryMinWords, HighlightSummaryMaxWords)

	if strings.Contains(got, "...") || strings.Contains(got, "…") {
		t.Fatalf("summary must not contain ellipsis: %q", got)
	}
	if !strings.HasSuffix(got, ".") {
		t.Fatalf("summary should end with a period: %q", got)
	}
	if wordCount(got) > HighlightSummaryMaxWords {
		t.Fatalf("summary too long: %d words", wordCount(got))
	}
}

func TestStripTrailingEllipsis(t *testing.T) {
	got := stripTrailingEllipsis("Nhà đầu tư nên thận trọng theo dõi các tác động từ chính sách pháp lý và...............")
	if strings.Contains(got, "...") {
		t.Fatalf("expected ellipsis removed, got %q", got)
	}
	if !strings.HasSuffix(got, "pháp lý") && !strings.HasSuffix(got, "pháp lý và") {
		t.Fatalf("unexpected ending: %q", got)
	}
}

func TestNormalizeAISummaryWordsKeepsShortText(t *testing.T) {
	short := "Bản tin sáng ngắn gọn kết thúc đầy đủ."
	got := normalizeAISummaryWords(short, HighlightSummaryMinWords, HighlightSummaryMaxWords)
	if got != short {
		t.Fatalf("expected unchanged short text, got %q", got)
	}
}