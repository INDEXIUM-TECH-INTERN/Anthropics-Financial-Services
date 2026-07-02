package worldnews

import (
	"strings"
	"testing"
	"time"
)

func TestBuildHighlightSummaryWordRange(t *testing.T) {
	sp := &quoteSnapshot{Symbol: "%5EGSPC", Price: 5400, Change: 12, ChangePct: 0.22, IsPositive: true}
	nd := &quoteSnapshot{Symbol: "%5EIXIC", Price: 17800, Change: -40, ChangePct: -0.22, IsPositive: false}
	wti := &quoteSnapshot{Symbol: "CL%3DF", Price: 70.5, Change: 0.8, ChangePct: 1.1, IsPositive: true}
	brent := &quoteSnapshot{Symbol: "BZ%3DF", Price: 73.8, Change: 0.6, ChangePct: 0.9, IsPositive: true}
	gold := &quoteSnapshot{Symbol: "GC%3DF", Price: 3980, Change: -20, ChangePct: -0.5, IsPositive: false}
	dxy := &quoteSnapshot{Symbol: "DX-Y.NYB", Price: 104.2, Change: 0.1, ChangePct: 0.1, IsPositive: true}
	news := []rssItem{
		{
			Title:         "Fed signals cautious stance on rates",
			Source:        "Reuters",
			PubDate:       time.Date(2026, 6, 30, 6, 15, 0, 0, vnTimezone),
			PublisherHost: "reuters.com",
		},
		{
			Title:         "Oil prices steady as OPEC+ meets",
			Source:        "WSJ",
			PubDate:       time.Date(2026, 6, 30, 5, 40, 0, 0, vnTimezone),
			PublisherHost: "wsj.com",
		},
	}

	summary := buildHighlightSummary(sp, nd, wti, brent, gold, dxy, news, "29/06/2026 (Mỹ)", "29/06/2026 07:00 – 30/06/2026 07:00")
	w := wordCount(summary)
	if w < HighlightSummaryMinWords || w > HighlightSummaryMaxWords {
		t.Fatalf("expected %d–%d words, got %d", HighlightSummaryMinWords, HighlightSummaryMaxWords, w)
	}
	for _, part := range []string{"S&P 500", "Nasdaq", "WTI", "Brent", "Reuters"} {
		if !strings.Contains(summary, part) {
			t.Fatalf("summary missing %q: %s", part, summary)
		}
	}
}