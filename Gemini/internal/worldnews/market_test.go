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
	intradayTS := time.Date(2026, 6, 30, 6, 30, 0, 0, vnTimezone).Unix()
	gotIntra := formatChartTimeLabel(intradayTS, true)
	if gotIntra != "30/06 06:30" {
		t.Fatalf("expected intraday label 30/06 06:30, got %q", gotIntra)
	}
}

func TestChartLabelForDailyBarBefore7AM(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 30, 7, 0, 0, 0, vnTimezone)
	got := chartLabelForDailyBar("2026-06-29", time.Date(2026, 6, 29, 16, 0, 0, 0, loc).Unix(), cutoff, loc)
	if got != "30/06 03:00" {
		t.Fatalf("expected 30/06 03:00, got %q", got)
	}
}

func TestChartLabelForDailyBarMapsYahoo930ToSessionClose(t *testing.T) {
	loc := time.FixedZone("ET", -4*3600)
	cutoff := time.Date(2026, 6, 30, 7, 0, 0, 0, vnTimezone)
	// Yahoo daily bar timestamp often displays as 09:30 GMT+7.
	yahooBarTS := time.Date(2026, 6, 17, 9, 30, 0, 0, vnTimezone).Unix()
	dayKey := time.Unix(yahooBarTS, 0).In(loc).Format("2006-01-02")
	got := chartLabelForDailyBar(dayKey, yahooBarTS, cutoff, loc)
	if got == "17/06 09:30" {
		t.Fatalf("must not expose raw Yahoo bar time 09:30, got %q", got)
	}
	if got != "17/06 03:00" {
		t.Fatalf("expected session close 17/06 03:00, got %q", got)
	}
}

func TestFormatChartVNLabelClampsAfter7AM(t *testing.T) {
	got := formatChartVNLabel(time.Date(2026, 6, 17, 9, 30, 0, 0, vnTimezone))
	if got != "17/06 06:59" {
		t.Fatalf("expected 17/06 06:59, got %q", got)
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