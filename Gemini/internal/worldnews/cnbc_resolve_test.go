package worldnews

import (
	"testing"
	"time"
)

func TestResolveCNBCKeyNumberMatchesCNBCLive(t *testing.T) {
	q := &cnbcFormattedQuote{
		Last:               "70.80",
		Change:             "-0.77",
		ChangePct:          "-1.08%",
		LastTime:           "2026-07-02T05:04:00.000+0100",
		PreviousDayClosing: "71.57",
		Open:               "71.21",
		High:               "71.22",
		Low:                "70.63",
	}
	got, err := resolveCNBCKeyNumber(q, time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone))
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.Price != 70.80 {
		t.Fatalf("expected price 70.80, got %.2f", got.Price)
	}
	if diff := got.ChangePct - (-1.08); diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected change pct -1.08, got %.2f", got.ChangePct)
	}
	if got.PriceTime != "02/07 05:04 BST" {
		t.Fatalf("expected CNBC BST time label, got %q", got.PriceTime)
	}
}