package worldnews

import "testing"

func TestKeyNumberFromQuoteIncludesTimeLabels(t *testing.T) {
	q := &quoteSnapshot{
		Symbol:      "%5EGSPC",
		Price:       5400,
		Change:      12,
		ChangePct:   0.22,
		ChartPoints: []float64{5380, 5390, 5400},
		ChartLabels: []string{"23/06 16:00", "24/06 16:00", "25/06 16:00"},
		PriceTime:   "25/06 16:00 ET",
	}
	kn := keyNumberFromQuote(q, "S&P 500", "CNBC", "https://www.cnbc.com/world/", "S&P 500")
	if kn.PriceTime != "25/06 16:00 ET" {
		t.Fatalf("expected price time, got %q", kn.PriceTime)
	}
	if len(kn.SparklineLabels) != 3 {
		t.Fatalf("expected sparkline labels, got %d", len(kn.SparklineLabels))
	}
}