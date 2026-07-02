package worldnews

import (
	"testing"
	"time"
)

func TestCNBCSessionSparkline(t *testing.T) {
	points := cnbcSessionSparkline(68.58, 68.03, 67.56, 68.17, 67.60)
	if len(points) != 8 {
		t.Fatalf("expected 8 points, got %d", len(points))
	}
	if points[0] != 68.58 || points[7] != 67.60 {
		t.Fatalf("unexpected endpoints: %v", points)
	}
}

func TestParseCNBCPercent(t *testing.T) {
	got, err := parseCNBCPercent("-1.43%")
	if err != nil || got != -1.43 {
		t.Fatalf("expected -1.43, got %v err=%v", got, err)
	}
}

func TestFormatCNBCPriceTimeBeforeCutoff(t *testing.T) {
	cutoff := time.Date(2026, 7, 2, 7, 0, 0, 0, vnTimezone)
	got := formatCNBCPriceTime("2026-07-01T23:17:19.000-0400", cutoff)
	if got != "02/07 10:17 GMT+7" {
		t.Fatalf("expected 02/07 10:17 GMT+7, got %q", got)
	}
}

func TestCNBCQuotePageURL(t *testing.T) {
	got := cnbcQuotePageURL(cnbcBrentSymbol)
	want := "https://www.cnbc.com/quotes/@LCO.1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}