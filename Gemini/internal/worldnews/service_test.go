package worldnews

import (
	"testing"
	"time"
)

func TestGetAvailableDates(t *testing.T) {
	resp := DefaultService.GetAvailableDates()
	if len(resp.Dates) != 7 {
		t.Fatalf("expected 7 dates, got %d", len(resp.Dates))
	}
	if resp.DefaultDate == "" {
		t.Fatal("expected default date")
	}
	seen := make(map[string]int)
	for _, d := range resp.Dates {
		seen[d.Value]++
	}
	for value, count := range seen {
		if count > 1 {
			t.Fatalf("duplicate calendar date in picker: %s", value)
		}
	}
}

func TestNormalizeTradingDay(t *testing.T) {
	sat := time.Date(2026, 6, 20, 12, 0, 0, 0, vnTimezone)
	fri := normalizeTradingDay(sat)
	if fri.Weekday() != time.Friday {
		t.Fatalf("expected Friday, got %v", fri.Weekday())
	}
}

func TestMorningDigestWindow(t *testing.T) {
	day := time.Date(2026, 6, 24, 12, 0, 0, 0, vnTimezone)
	since, until := morningDigestWindow(day)
	if until.Hour() != 7 || until.Day() != 24 {
		t.Fatalf("unexpected until: %v", until)
	}
	if since.Day() != 23 || since.Hour() != 7 {
		t.Fatalf("unexpected since: %v", since)
	}
}

func TestBuildReportDifferentDates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network integration test")
	}

	r17, err := DefaultService.GetReport("2026-06-17")
	if err != nil {
		t.Fatalf("GetReport 2026-06-17 failed: %v", err)
	}
	r23, err := DefaultService.GetReport("2026-06-23")
	if err != nil {
		t.Fatalf("GetReport 2026-06-23 failed: %v", err)
	}

	if r17.Date != "2026-06-17" || r23.Date != "2026-06-23" {
		t.Fatalf("unexpected report dates: %s vs %s", r17.Date, r23.Date)
	}
	if r17.KeyNumbers[0].Value == r23.KeyNumbers[0].Value {
		t.Fatalf("expected different S&P values, both %s", r17.KeyNumbers[0].Value)
	}
}

func TestBuildReportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network integration test")
	}
	report, err := DefaultService.GetReport("")
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if len(report.KeyNumbers) < 2 {
		t.Fatalf("expected key numbers, got %d", len(report.KeyNumbers))
	}
	if report.Stocks.Value == "" {
		t.Fatal("expected stock value")
	}
}