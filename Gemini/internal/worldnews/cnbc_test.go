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
	body := []byte(`{"FormattedQuoteResult":{"FormattedQuote":[{"symbol":"@LCO.1","last":"70.76","change":"-0.81","change_pct":"-1.13%","last_time":"2026-07-02T04:36:00.000+0100","previous_day_closing":"71.57","open":"71.21","high":"71.22","low":"70.63"}]}}`)
	q, err := parseCNBCQuoteBody(body, cnbcBrentSymbol)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if q.Last != "70.76" {
		t.Fatalf("expected last 70.76, got %q", q.Last)
	}
}

func TestFormatCNBCPriceTimeBST(t *testing.T) {
	cutoff := time.Date(2026, 7, 2, 7, 0, 0, 0, vnTimezone)
	got := formatCNBCPriceTime("2026-07-02T04:36:00.000+0100", cutoff)
	if got != "02/07 10:36 GMT+7" {
		t.Fatalf("expected 02/07 10:36 GMT+7, got %q", got)
	}
}

func TestFormatCNBCPriceTimeBeforeCutoff(t *testing.T) {
	cutoff := time.Date(2026, 7, 2, 7, 0, 0, 0, vnTimezone)
	got := formatCNBCPriceTime("2026-07-01T23:17:19.000-0400", cutoff)
	if got != "02/07 10:17 GMT+7" {
		t.Fatalf("expected 02/07 10:17 GMT+7, got %q", got)
	}
}

func TestCNBCQuotePageURL(t *testing.T) {
	got := cnbcQuotePageURL(cnbcBrentSymbol)
	want := "https://www.cnbc.com/quotes/@LCO.1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}