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

func TestSelectBarIndexUsesLastDailyBarBeforeCutoff(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 30, 7, 0, 0, 0, vnTimezone)
	bars := []quoteBar{
		{ts: time.Date(2026, 6, 29, 0, 0, 0, 0, loc).Unix(), close: 73.15, day: "2026-06-29"},
		{ts: time.Date(2026, 6, 29, 23, 17, 0, 0, loc).Unix(), close: 73.84, day: "2026-06-29"},
	}
	idx := selectBarIndex(bars, "2026-06-29", cutoff)
	if bars[idx].close != 73.15 {
		t.Fatalf("expected first daily bar before cutoff, got %.2f", bars[idx].close)
	}
}

func TestFormatChartTimeLabelUsesGMT7(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	// US session close 29/06 16:00 ET -> 30/06 03:00 GMT+7
	sessionClose := time.Date(2026, 6, 29, 16, 0, 0, 0, loc).Unix()
	gotDaily := formatChartTimeLabel(sessionClose, false)
	if gotDaily != "30/06" {
		t.Fatalf("expected daily label 30/06, got %q", gotDaily)
	}

	intradayTS := time.Date(2026, 6, 30, 6, 30, 0, 0, vnTimezone).Unix()
	gotIntra := formatChartTimeLabel(intradayTS, true)
	if gotIntra != "30/06 06:30" {
		t.Fatalf("expected intraday label 30/06 06:30, got %q", gotIntra)
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