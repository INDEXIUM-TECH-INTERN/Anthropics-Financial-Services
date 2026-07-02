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

func TestCNBCResolvedChangeMatchesReferenceBar(t *testing.T) {
	q := &cnbcFormattedQuote{
		Change:             "-0.73",
		ChangePct:          "-1.02%",
		PreviousDayClosing: "71.57",
	}
	_, pct := cnbcResolvedChange(q, 70.84, 71.57)
	if pct != -1.02 {
		t.Fatalf("expected -1.02%%, got %.2f", pct)
	}
}