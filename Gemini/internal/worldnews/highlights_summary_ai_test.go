package worldnews

import (
	"strings"
	"testing"
	"time"
)

type mockTextGenerator struct {
	response string
	err      error
}

func (m *mockTextGenerator) GenerateText(_, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestBuildHighlightSummaryMarketData(t *testing.T) {
	sp := &quoteSnapshot{Symbol: "%5EGSPC", Price: 5400, Change: 12, ChangePct: 0.22, IsPositive: true}
	nd := &quoteSnapshot{Symbol: "%5EIXIC", Price: 17800, Change: -40, ChangePct: -0.22, IsPositive: false}
	wti := &quoteSnapshot{Symbol: "CL%3DF", Price: 70.5, Change: 0.8, ChangePct: 1.1, IsPositive: true}

	data := buildHighlightSummaryMarketData(sp, nd, wti, nil, nil, nil, "29/06/2026 (Mỹ)")
	for _, part := range []string{"S&P 500", "Nasdaq", "WTI"} {
		if !strings.Contains(data, part) {
			t.Fatalf("market data missing %q: %s", part, data)
		}
	}
}

func TestSanitizeAISummary(t *testing.T) {
	raw := "## Tiêu đề\n\n**S&P 500** tăng mạnh.\n\nKết luận."
	got := sanitizeAISummary(raw)
	if strings.Contains(got, "#") || strings.Contains(got, "**") {
		t.Fatalf("expected markdown removed, got %q", got)
	}
	if !strings.Contains(got, "\n\n") {
		t.Fatalf("expected paragraph breaks preserved, got %q", got)
	}
	if len(splitSummaryParagraphs(got)) < 2 {
		t.Fatalf("expected multiple paragraphs, got %q", got)
	}
}

func TestGenerateHighlightSummaryAIUsesMock(t *testing.T) {
	sp := &quoteSnapshot{Symbol: "%5EGSPC", Price: 5400, Change: 12, ChangePct: 0.22, IsPositive: true}
	news := []rssItem{{
		Title:   "Fed holds rates steady",
		Source:  "Reuters",
		PubDate: time.Date(2026, 6, 30, 6, 0, 0, 0, vnTimezone),
	}}

	long := strings.Repeat("Thị trường tài chính toàn cầu ghi nhận biến động đáng chú ý. ", 20)
	gen := &mockTextGenerator{response: long}

	summary, err := generateHighlightSummaryAI(gen, sp, nil, nil, nil, nil, nil, news, "29/06/2026 (Mỹ)", "29/06/2026 07:00 – 30/06/2026 07:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := wordCount(summary)
	if w > HighlightSummaryMaxWords {
		t.Fatalf("summary too long: %d words", w)
	}
	if strings.Contains(summary, "...") {
		t.Fatalf("AI summary must not end with ellipsis: %q", summary)
	}
}

func TestResolveHighlightSummaryFallsBackWithoutGenerator(t *testing.T) {
	svc := NewService()
	sp := &quoteSnapshot{Symbol: "%5EGSPC", Price: 5400, Change: 12, ChangePct: 0.22, IsPositive: true}
	summary := svc.resolveHighlightSummary(sp, nil, nil, nil, nil, nil, nil, "29/06/2026 (Mỹ)", "29/06/2026 07:00 – 30/06/2026 07:00")
	if !strings.Contains(summary, "S&P 500") {
		t.Fatalf("expected template fallback, got %q", summary)
	}
}