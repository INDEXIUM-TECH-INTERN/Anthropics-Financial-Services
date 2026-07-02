package worldnews

import (
	"testing"
	"time"
)

func TestCNBCSessionSparkline(t *testing.T) {
	points := cnbcSessionSparkline(68.58, 68.03, 67.56, 68.17, 67.60)
	if len(points) != 8 {
		t.Fatalf("expected 8 points, got %d", len(points))
	}
	if points[0] != 68.58 || points[7] != 67.60 {
		t.Fatalf("unexpected endpoints: %v", points)
	}
}

func TestParseCNBCPercent(t *testing.T) {
	got, err := parseCNBCPercentOptional("-1.43%")
	if err != nil || got != -1.43 {
		t.Fatalf("expected -1.43, got %v err=%v", got, err)
	}
}

func TestParseCNBCQuoteBodyBrentSample(t *testing.T) {
	body := []byte(`{"FormattedQuoteResult":{"FormattedQuote":[{"symbol":"@LCO.1","last":"70.80","change":"-0.77","change_pct":"-1.08%","last_time":"2026-07-02T05:04:00.000+0100","previous_day_closing":"71.57","open":"71.21","high":"71.22","low":"70.63"}]}}`)
	q, err := parseCNBCQuoteBody(body, cnbcBrentSymbol)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if q.Last != "70.80" {
		t.Fatalf("expected last 70.80, got %q", q.Last)
	}
}

func TestCNBCResolvedChangeUsesAPIFields(t *testing.T) {
	q := &cnbcFormattedQuote{
		Last:               "70.84",
		Change:             "-0.73",
		ChangePct:          "-1.02%",
		PreviousDayClosing: "71.57",
	}
	ch, pct := cnbcResolvedChange(q, 70.84, 71.57)
	if ch != -0.73 {
		t.Fatalf("expected change -0.73, got %.2f", ch)
	}
	if pct != -1.02 {
		t.Fatalf("expected pct -1.02, got %.2f", pct)
	}
}

func TestResolveCNBCKeyNumberAfterCutoffKeepsCNBCPrice(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "70.84",
		Change:             "-0.73",
		ChangePct:          "-1.02%",
		LastTime:           "2026-07-02T05:04:00.000+0100",
		PreviousDayClosing: "71.57",
		Open:               "71.21",
		High:               "71.22",
		Low:                "70.63",
	}
	got, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.Price != 70.84 {
		t.Fatalf("expected CNBC price 70.84, got %.2f", got.Price)
	}
	if diff := got.ChangePct - (-1.02); diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected change pct -1.02, got %.2f", got.ChangePct)
	}
	if got.PriceTime != "02/07 06:59 GMT+7" {
		t.Fatalf("expected pre-7h GMT+7 label, got %q", got.PriceTime)
	}
}

func TestResolveCNBCKeyNumberBeforeCutoffUsesLiveGMT7(t *testing.T) {
	calendarDay := time.Date(2099, 1, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "71.10",
		Change:             "-0.47",
		ChangePct:          "-0.66%",
		LastTime:           "2099-01-02T05:30:00.000+0700",
		PreviousDayClosing: "71.57",
		Open:               "71.21",
		High:               "71.30",
		Low:                "71.00",
	}
	got, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.PriceTime != "02/01 05:30 GMT+7" {
		t.Fatalf("expected live GMT+7 label, got %q", got.PriceTime)
	}
}

func TestCNBCIsAfterDigestCutoffGoldEDT(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	// 12:46 AM EDT on 07/02 = 11:46 GMT+7 — after 07:00 digest cutoff.
	if !cnbcIsAfterDigestCutoff("2026-07-02T00:46:00.000-0400", calendarDay) {
		t.Fatal("expected gold EDT quote after cutoff")
	}
}

func TestApplyCNBCCutoffPriceGold(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "4074.70",
		Change:             "-7.70",
		ChangePct:          "-0.19%",
		LastTime:           "2026-07-02T00:46:00.000-0400",
		PreviousDayClosing: "4082.40",
		Open:               "4049.20",
		High:               "4086.30",
		Low:                "4042.80",
	}
	base, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	got := applyCNBCCutoffPrice(base, 4047.40, 4082.40, calendarDay, "02/07 06:59 GMT+7")
	if got.Price != 4047.40 {
		t.Fatalf("expected cutoff price 4047.40, got %.2f", got.Price)
	}
	if diff := got.ChangePct - (-0.8578); diff > 0.05 || diff < -0.05 {
		t.Fatalf("expected ~-0.86%% change vs prev close, got %.2f", got.ChangePct)
	}
	if got.PriceTime != "02/07 06:59 GMT+7" {
		t.Fatalf("expected 02/07 06:59 GMT+7, got %q", got.PriceTime)
	}
	if len(got.Sparkline) == 0 || got.Sparkline[len(got.Sparkline)-1] != 4047.40 {
		t.Fatalf("expected sparkline to end at cutoff price, got %v", got.Sparkline)
	}
}

func TestLiveGoldCutoffSnap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live gold cutoff test in short mode")
	}
	s := NewService()
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	kn, err := s.fetchCNBCKeyNumber(cnbcGoldSymbol, "Vàng thế giới", "Gold COMEX", calendarDay)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if kn.PriceTime != "02/07 06:59 GMT+7" {
		t.Fatalf("expected 02/07 06:59 GMT+7, got %q", kn.PriceTime)
	}
	// Yahoo 1m at 06:59 GMT+7 should be ~4047, not live CNBC ~4074.
	if kn.Value != "$4047.40" && kn.Value != "$4047.39" && kn.Value != "$4047.41" {
		t.Fatalf("expected ~$4047.40 cutoff price, got %q", kn.Value)
	}
}

func TestCNBCQuotePageURL(t *testing.T) {
	got := cnbcQuotePageURL(cnbcBrentSymbol)
	want := "https://www.cnbc.com/quotes/@LCO.1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}