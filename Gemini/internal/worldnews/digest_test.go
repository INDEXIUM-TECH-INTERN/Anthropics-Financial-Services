package worldnews

import (
	"testing"
	"time"
)

func TestMorningDigestWindow(t *testing.T) {
	day := time.Date(2026, 6, 24, 12, 0, 0, 0, vnTimezone)
	since, until := morningDigestWindow(day)
	if until.Hour() != MorningDigestHour || until.Day() != 24 {
		t.Fatalf("unexpected until: %v", until)
	}
	if since.Day() != 23 || since.Hour() != MorningDigestHour {
		t.Fatalf("unexpected since: %v", since)
	}
	if formatDigestWindow(since, until) == "" {
		t.Fatal("expected digest window label")
	}
}

func TestDigestMarketQuoteDayBeforeMorningCutoff(t *testing.T) {
	// 30/06/2026 07:00 GMT+7 = 29/06 20:00 ET → phiên Mỹ gần nhất là 29/06 đóng cửa.
	day := time.Date(2026, 6, 30, 0, 0, 0, 0, vnTimezone)
	quoteDay := digestMarketQuoteDay(day)
	if quoteDay.Format("2006-01-02") != "2026-06-29" {
		t.Fatalf("expected 2026-06-29, got %s", quoteDay.Format("2006-01-02"))
	}
}

func TestDigestMarketQuoteDaySkipsWeekend(t *testing.T) {
	// Thứ Hai 29/06/2026 07:00 GMT+7 → phiên cuối là thứ Sáu 26/06.
	monday := time.Date(2026, 6, 29, 0, 0, 0, 0, vnTimezone)
	if monday.Weekday() != time.Monday {
		t.Fatalf("test fixture should be Monday, got %v", monday.Weekday())
	}
	quoteDay := digestMarketQuoteDay(monday)
	if quoteDay.Format("2006-01-02") != "2026-06-26" {
		t.Fatalf("expected 2026-06-26, got %s", quoteDay.Format("2006-01-02"))
	}
}

func TestFilterNewsBeforeDigestCutoff(t *testing.T) {
	day := time.Date(2026, 6, 25, 0, 0, 0, 0, vnTimezone)
	since, until := morningDigestWindow(day)

	items := []rssItem{
		{Title: "Before cutoff", PubDate: time.Date(2026, 6, 25, 6, 30, 0, 0, vnTimezone)},
		{Title: "After cutoff", PubDate: time.Date(2026, 6, 25, 8, 0, 0, 0, vnTimezone)},
		{Title: "Too early", PubDate: time.Date(2026, 6, 24, 6, 0, 0, 0, vnTimezone)},
	}
	filtered := filterNewsBetween(items, since, until)
	if len(filtered) != 1 || filtered[0].Title != "Before cutoff" {
		t.Fatalf("expected only pre-7am item, got %#v", filtered)
	}
}