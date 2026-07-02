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

func TestCNBCDailySparkline(t *testing.T) {
	points := cnbcDailySparkline(71.57, 70.76)
	if len(points) != 8 {
		t.Fatalf("expected 8 points, got %d", len(points))
	}
	if points[0] != 71.57 || points[7] != 70.76 {
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
	body := []byte(`{"FormattedQuoteResult":{"FormattedQuote":[{"symbol":"@LCO.1","last":"70.76","change":"-0.81","change_pct":"-1.13%","last_time":"2026-07-02T04:36:00.000+0100","previous_day_closing":"71.57","open":"71.21","high":"71.22","low":"70.63","settlePrice":"70.76"}]}}`)
	q, err := parseCNBCQuoteBody(body, cnbcBrentSymbol)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if q.Last != "70.76" {
		t.Fatalf("expected last 70.76, got %q", q.Last)
	}
}

func TestFormatCNBCPriceTimeLive(t *testing.T) {
	got := formatCNBCPriceTimeLive("2026-07-02T05:30:00.000+0700")
	if got != "02/07 05:30 GMT+7" {
		t.Fatalf("expected 02/07 05:30 GMT+7, got %q", got)
	}
}

func TestResolveCNBCKeyNumberAfterCutoffUsesSettle(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "70.76",
		Change:             "-0.81",
		ChangePct:          "-1.13%",
		LastTime:           "2026-07-02T04:36:00.000+0100",
		PreviousDayClosing: "71.57",
		Open:               "71.21",
		High:               "71.22",
		Low:                "70.63",
		SettlePrice:        "70.76",
	}
	got, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.Price != 70.76 {
		t.Fatalf("expected settle price 70.76, got %.2f", got.Price)
	}
	if diff := got.Change - (-0.81); diff > 0.001 || diff < -0.001 {
		t.Fatalf("expected change -0.81, got %.4f", got.Change)
	}
	if got.PriceTime != "02/07 03:00 GMT+7" {
		t.Fatalf("expected session close label, got %q", got.PriceTime)
	}
	if len(got.Sparkline) != 8 || got.Sparkline[0] != 71.57 || got.Sparkline[7] != 70.76 {
		t.Fatalf("expected daily sparkline 71.57→70.76, got %v", got.Sparkline)
	}
}

func TestResolveCNBCKeyNumberBeforeCutoffUsesLive(t *testing.T) {
	calendarDay := time.Date(2026, 7, 2, 0, 0, 0, 0, vnTimezone)
	q := &cnbcFormattedQuote{
		Last:               "71.10",
		Change:             "-0.47",
		ChangePct:          "-0.66%",
		LastTime:           "2026-07-02T05:30:00.000+0700",
		PreviousDayClosing: "71.57",
		Open:               "71.21",
		High:               "71.30",
		Low:                "71.00",
		SettlePrice:        "70.76",
	}
	got, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if got.Price != 71.10 {
		t.Fatalf("expected live price 71.10, got %.2f", got.Price)
	}
	if got.PriceTime != "02/07 05:30 GMT+7" {
		t.Fatalf("expected live time label, got %q", got.PriceTime)
	}
	if len(got.Sparkline) != 8 || got.Sparkline[7] != 71.10 {
		t.Fatalf("expected intraday sparkline ending at 71.10, got %v", got.Sparkline)
	}
}

func TestCNBCQuotePageURL(t *testing.T) {
	got := cnbcQuotePageURL(cnbcBrentSymbol)
	want := "https://www.cnbc.com/quotes/@LCO.1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}