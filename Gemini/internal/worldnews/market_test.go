package worldnews

import (
	"testing"
	"time"
)

func TestResolvePriceTimeUsesRegularMarketTimeBeforeCutoff(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 30, 7, 0, 0, 0, vnTimezone)
	targetKey := "2026-06-29"
	regularMarketTime := time.Date(2026, 6, 29, 17, 42, 3, 0, loc).Unix()
	barTS := time.Date(2026, 6, 29, 9, 30, 0, 0, loc).Unix()

	got := resolvePriceTime(regularMarketTime, targetKey, barTS, false, cutoff, loc)
	want := "30/06 04:42 GMT+7"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePriceTimeIgnoresQuoteAfterCutoff(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 30, 7, 0, 0, 0, vnTimezone)
	targetKey := "2026-06-29"
	regularMarketTime := time.Date(2026, 6, 30, 17, 42, 3, 0, loc).Unix()
	barTS := time.Date(2026, 6, 29, 9, 30, 0, 0, loc).Unix()

	got := resolvePriceTime(regularMarketTime, targetKey, barTS, false, cutoff, loc)
	want := "30/06 03:00 GMT+7"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolvePriceTimeFallsBackToSessionCloseForHistoricalDay(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 26, 7, 0, 0, 0, vnTimezone)
	targetKey := "2026-06-25"
	regularMarketTime := time.Date(2026, 6, 29, 17, 42, 3, 0, loc).Unix()
	barTS := time.Date(2026, 6, 25, 9, 30, 0, 0, loc).Unix()

	got := resolvePriceTime(regularMarketTime, targetKey, barTS, false, cutoff, loc)
	want := "26/06 03:00 GMT+7"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}