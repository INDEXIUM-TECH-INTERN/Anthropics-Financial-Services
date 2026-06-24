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
}

func TestNormalizeTradingDay(t *testing.T) {
	sat := time.Date(2026, 6, 20, 12, 0, 0, 0, vnTimezone)
	fri := normalizeTradingDay(sat)
	if fri.Weekday() != time.Friday {
		t.Fatalf("expected Friday, got %v", fri.Weekday())
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