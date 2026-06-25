package worldnews

import "testing"

func TestBuildQuickHighlightsWithLinks(t *testing.T) {
	sp := &quoteSnapshot{
		Symbol:     "%5EGSPC",
		Price:      5400,
		Change:     10,
		ChangePct:  0.2,
		IsPositive: true,
	}
	news := []rssItem{{
		Title:  "Fed signals rate pause",
		Source: "Reuters",
		Link:   "https://www.reuters.com/markets/example",
	}}
	out := buildQuickHighlights(sp, nil, nil, nil, nil, nil, news, "24/06/2026")
	if len(out) < 2 {
		t.Fatalf("expected at least 2 highlights, got %d", len(out))
	}
	if out[0].URL != "https://www.cnbc.com/world/" {
		t.Fatalf("expected CNBC link for S&P, got %q", out[0].URL)
	}
	if out[len(out)-1].URL != news[0].Link {
		t.Fatalf("expected article link, got %q", out[len(out)-1].URL)
	}
}