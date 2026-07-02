package worldnews

import (
	"testing"
	"time"
)

func TestCNBCKeyNumberPriceTimeClampsBSTAfterCutoff(t *testing.T) {
	day := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	_, cutoff := morningDigestWindow(day)
	got := cnbcKeyNumberPriceTime("2026-07-02T05:04:00.000+0100", day, cutoff)
	if got != "02/07 06:59 GMT+7" {
		t.Fatalf("expected 02/07 06:59 GMT+7, got %q", got)
	}
}