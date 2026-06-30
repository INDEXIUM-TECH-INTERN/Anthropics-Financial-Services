package worldnews

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizeAISummaryLengthNoEllipsis(t *testing.T) {
	long := strings.Repeat("Thị trường tài chính ghi nhận biến động. ", 25) + "Nhà đầu tư nên theo dõi chính sách pháp lý và"
	got := normalizeAISummaryLength(long, 700)

	if strings.Contains(got, "...") || strings.Contains(got, "…") {
		t.Fatalf("summary must not contain ellipsis: %q", got)
	}
	if !strings.HasSuffix(got, ".") {
		t.Fatalf("summary should end with a period: %q", got)
	}
	if utf8.RuneCountInString(got) > 700 {
		t.Fatalf("summary too long: %d runes", utf8.RuneCountInString(got))
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

func TestNormalizeAISummaryLengthKeepsShortText(t *testing.T) {
	short := "Bản tin sáng ngắn gọn kết thúc đầy đủ."
	got := normalizeAISummaryLength(short, 700)
	if got != short {
		t.Fatalf("expected unchanged short text, got %q", got)
	}
}