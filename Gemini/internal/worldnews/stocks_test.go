package worldnews

import (
	"fmt"
	"testing"
)

func TestIsAllowedStockNewsSource(t *testing.T) {
	cases := []struct {
		item rssItem
		want bool
	}{
		{rssItem{Source: "CNBC", PublisherHost: "cnbc.com"}, true},
		{rssItem{Source: "Reuters", PublisherHost: "reuters.com"}, true},
		{rssItem{Source: "WSJ", PublisherHost: "wsj.com"}, true},
		{rssItem{Source: "Wall Street Journal", PublisherHost: "wsj.com"}, true},
		{rssItem{Source: "Bloomberg", PublisherHost: "bloomberg.com"}, false},
		{rssItem{Source: "Google News (Thế giới)", PublisherHost: "news.google.com"}, false},
	}

	for _, tc := range cases {
		got := isAllowedStockNewsSource(tc.item)
		if got != tc.want {
			t.Fatalf("source %q host %q: got %v want %v", tc.item.Source, tc.item.PublisherHost, got, tc.want)
		}
	}
}

func TestQuoteToStockInstrument(t *testing.T) {
	q := &quoteSnapshot{
		Symbol:      "%5EGSPC",
		Label:       "S&P 500",
		Price:       5400.5,
		Change:      12.3,
		ChangePct:   0.23,
		IsPositive:  true,
		ChartPoints: []float64{5380, 5390, 5400.5},
		ChartLabels: []string{"01/06", "02/06", "03/06"},
	}
	inst := quoteToStockInstrument(q)
	if inst.Symbol != "^GSPC" {
		t.Fatalf("expected ^GSPC, got %s", inst.Symbol)
	}
	if len(inst.ChartPoints) != 3 {
		t.Fatalf("expected chart points, got %d", len(inst.ChartPoints))
	}
	if inst.QuoteURL == "" {
		t.Fatal("expected quote url")
	}
}

func TestSplitStockInstruments(t *testing.T) {
	all := make([]StockInstrument, 9)
	for i := range all {
		all[i] = StockInstrument{Symbol: fmt.Sprintf("SYM%d", i)}
	}
	primary, more := splitStockInstruments(all)
	if len(primary) != PrimaryStockDisplayCount {
		t.Fatalf("expected %d primary, got %d", PrimaryStockDisplayCount, len(primary))
	}
	if len(more) != 5 {
		t.Fatalf("expected 5 more, got %d", len(more))
	}
}

func TestFilterStockNewsSources(t *testing.T) {
	items := []rssItem{
		{Title: "A", Source: "CNBC", PublisherHost: "cnbc.com"},
		{Title: "B", Source: "Bloomberg", PublisherHost: "bloomberg.com"},
		{Title: "C", Source: "Reuters", PublisherHost: "reuters.com"},
	}
	filtered := filterStockNewsSources(items)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 stock items, got %d", len(filtered))
	}
}