package worldnews

import (
	"testing"
	"time"
)

func TestResolvePriceTimeUsesRegularMarketTime(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	targetKey := "2026-06-29"
	regularMarketTime := time.Date(2026, 6, 29, 17, 42, 3, 0, loc).Unix()
	barTS := time.Date(2026, 6, 29, 9, 30, 0, 0, loc).Unix()

	got := resolvePriceTime(regularMarketTime, targetKey, barTS, false, loc)
	want := "29/06 17:42 ET"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePriceTimeFallsBackToSessionCloseForHistoricalDay(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	targetKey := "2026-06-25"
	regularMarketTime := time.Date(2026, 6, 29, 17, 42, 3, 0, loc).Unix()
	barTS := time.Date(2026, 6, 25, 9, 30, 0, 0, loc).Unix()

	got := resolvePriceTime(regularMarketTime, targetKey, barTS, false, loc)
	want := "25/06 16:00 ET"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}