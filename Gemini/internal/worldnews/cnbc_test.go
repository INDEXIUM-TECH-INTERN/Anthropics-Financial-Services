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

func TestResolveCNBCKeyNumberAfterCutoffKeepsCNBCPrice(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
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
	got, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.Price != 70.80 {
		t.Fatalf("expected CNBC price 70.80, got %.2f", got.Price)
	}
	if diff := got.ChangePct - (-1.08); diff > 0.01 || diff < -0.01 {
		t.Fatalf("expected change pct -1.08, got %.2f", got.ChangePct)
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

func TestCNBCQuotePageURL(t *testing.T) {
	got := cnbcQuotePageURL(cnbcBrentSymbol)
	want := "https://www.cnbc.com/quotes/@LCO.1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}