package worldnews

import (
	"testing"
	"time"
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
		PriceTime:   "29/06 03:00 GMT+7",
	}
	inst := quoteToStockInstrument(q, ".SPX")
	if inst.Symbol != "^GSPC" {
		t.Fatalf("expected ^GSPC, got %s", inst.Symbol)
	}
	if len(inst.ChartPoints) != 3 {
		t.Fatalf("expected chart points, got %d", len(inst.ChartPoints))
	}
	if inst.QuoteURL != "https://www.cnbc.com/quotes/.SPX" {
		t.Fatalf("expected CNBC quote url, got %s", inst.QuoteURL)
	}
	if inst.PriceTime != "29/06 03:00 GMT+7" {
		t.Fatalf("expected price time, got %q", inst.PriceTime)
	}
}

func TestWorldStockTabDefsHaveCNBCSymbols(t *testing.T) {
	for _, tab := range WorldStockTabDefs {
		for _, sym := range tab.Symbols {
			if sym.CNBCSymbol == "" {
				t.Fatalf("tab %q symbol %q missing CNBCSymbol", tab.ID, sym.Label)
			}
		}
	}
}

func TestCNBCQuoteToSnapshotSPX(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "7,483.23",
		Change:             "-16.13",
		ChangePct:          "-0.22%",
		LastTime:           "2026-07-01",
		PreviousDayClosing: "7,499.36",
		Open:               "7,478.84",
		High:               "7,521.81",
		Low:                "7,449.63",
	}
	got, err := cnbcQuoteToSnapshot(q, "S&P 500", "%5EGSPC", calendarDay)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if got.Price != 7483.23 {
		t.Fatalf("expected 7483.23, got %.2f", got.Price)
	}
	if len(got.ChartPoints) != 8 {
		t.Fatalf("expected 8-point CNBC sparkline, got %d", len(got.ChartPoints))
	}
}

func TestWorldStockTabDefs(t *testing.T) {
	if len(WorldStockTabDefs) != StockTabCount {
		t.Fatalf("expected %d tabs, got %d", StockTabCount, len(WorldStockTabDefs))
	}
	total := 0
	for _, tab := range WorldStockTabDefs {
		if tab.ID == "" || tab.Label == "" {
			t.Fatal("tab id/label required")
		}
		total += len(tab.Symbols)
	}
	if total != StockTabCount*StockChartsPerTab {
		t.Fatalf("expected %d symbols across tabs, got %d", StockTabCount*StockChartsPerTab, total)
	}
	for _, tab := range WorldStockTabDefs {
		if len(tab.Symbols) != StockChartsPerTab {
			t.Fatalf("tab %q: expected %d symbols, got %d", tab.ID, StockChartsPerTab, len(tab.Symbols))
		}
	}
}

func TestFlattenStockTabInstruments(t *testing.T) {
	tabs := []StockTab{
		{ID: "a", Label: "A", Instruments: []StockInstrument{{Symbol: "A1"}, {Symbol: "A2"}}},
		{ID: "b", Label: "B", Instruments: []StockInstrument{{Symbol: "B1"}}},
	}
	flat := flattenStockTabInstruments(tabs)
	if len(flat) != 3 {
		t.Fatalf("expected 3 instruments, got %d", len(flat))
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