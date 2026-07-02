package worldnews

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	cnbcRestQuoteAPI = "https://quote.cnbc.com/quote-html-webservice/restQuote/symbolType/symbol"
	cnbcQuoteHTMLAPI = "https://quote.cnbc.com/quote-html-webservice/quote.htm"
	cnbcDataSource   = "CNBC"
	cnbcBrentSymbol  = "@LCO.1" // ICE Brent Crude — khớp giá trên cnbc.com/quotes/@LCO.1
	cnbcGoldSymbol   = "@GC.1"  // COMEX Gold
)

type cnbcFormattedResponse struct {
	FormattedQuoteResult struct {
		FormattedQuote []cnbcFormattedQuote `json:"FormattedQuote"`
	} `json:"FormattedQuoteResult"`
}

type cnbcQuickQuoteResponse struct {
	QuickQuoteResult struct {
		QuickQuote []cnbcFormattedQuote `json:"QuickQuote"`
	} `json:"QuickQuoteResult"`
}

type cnbcFormattedQuote struct {
	Symbol             string `json:"symbol"`
	Name               string `json:"name"`
	Last               string `json:"last"`
	Change             string `json:"change"`
	ChangePct          string `json:"change_pct"`
	LastTime           string `json:"last_time"`
	PreviousDayClosing string `json:"previous_day_closing"`
	Open               string `json:"open"`
	High               string `json:"high"`
	Low                string `json:"low"`
	SettlePrice        string `json:"settlePrice"`
	SettleDate         string `json:"settleDate"`
}

type cnbcResolvedQuote struct {
	Price     float64
	Change    float64
	ChangePct float64
	Sparkline []float64
	PriceTime string
}

func cnbcQuotePageURL(symbol string) string {
	return "https://www.cnbc.com/quotes/" + url.PathEscape(symbol)
}

func cnbcQuoteAPIURL(base, symbol string) string {
	return fmt.Sprintf(
		"%s?symbols=%s&requestMethod=quick&exthrs=1&noform=1&partnerId=2&fund=1&output=json",
		base,
		url.QueryEscape(symbol),
	)
}

func (s *Service) fetchCNBCFormattedQuote(symbol string) (*cnbcFormattedQuote, error) {
	endpoints := []string{
		cnbcQuoteAPIURL(cnbcRestQuoteAPI, symbol),
		cnbcQuoteAPIURL(cnbcQuoteHTMLAPI, symbol),
	}
	var lastErr error
	for _, apiURL := range endpoints {
		body, err := s.httpGet(apiURL)
		if err != nil {
			lastErr = err
			continue
		}
		q, err := parseCNBCQuoteBody(body, symbol)
		if err != nil {
			lastErr = err
			continue
		}
		return q, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("CNBC quote unavailable for %s", symbol)
	}
	return nil, lastErr
}

func parseCNBCQuoteBody(body []byte, symbol string) (*cnbcFormattedQuote, error) {
	var formatted cnbcFormattedResponse
	if err := json.Unmarshal(body, &formatted); err == nil && len(formatted.FormattedQuoteResult.FormattedQuote) > 0 {
		q := formatted.FormattedQuoteResult.FormattedQuote[0]
		if strings.TrimSpace(q.Last) != "" {
			return &q, nil
		}
	}
	var quick cnbcQuickQuoteResponse
	if err := json.Unmarshal(body, &quick); err == nil && len(quick.QuickQuoteResult.QuickQuote) > 0 {
		q := quick.QuickQuoteResult.QuickQuote[0]
		if strings.TrimSpace(q.Last) != "" {
			return &q, nil
		}
	}
	return nil, fmt.Errorf("empty CNBC quote for %s", symbol)
}

func (s *Service) fetchCNBCKeyNumber(symbol, label, displaySymbol string, calendarDay time.Time) (KeyNumber, error) {
	q, err := s.fetchCNBCFormattedQuote(symbol)
	if err != nil {
		return KeyNumber{}, err
	}

	resolved, err := resolveCNBCKeyNumber(q, calendarDay)
	if err != nil {
		return KeyNumber{}, err
	}

	_, pctStr, positive := formatChange(resolved.Change, resolved.ChangePct)
	return KeyNumber{
		Label:           label,
		Value:           formatCNBCFuturesPrice(resolved.Price),
		Change:          pctStr,
		IsPositive:      positive,
		Sparkline:       resolved.Sparkline,
		SparklineLabels: nil,
		PriceTime:       resolved.PriceTime,
		Source:          cnbcDataSource,
		URL:             cnbcQuotePageURL(symbol),
		Symbol:          displaySymbol,
	}, nil
}

// resolveCNBCKeyNumber uses CNBC session OHLC/change and snaps the displayed time to
// before 07:00 GMT+7 when the live quote timestamp is after the morning digest cutoff.
func resolveCNBCKeyNumber(q *cnbcFormattedQuote, calendarDay time.Time) (cnbcResolvedQuote, error) {
	last, err := parseCNBCPrice(q.Last)
	if err != nil {
		return cnbcResolvedQuote{}, err
	}
	last = math.Round(last*100) / 100

	day := calendarDay.In(vnTimezone)
	_, cutoff := morningDigestWindow(day)

	prev, _ := parseCNBCPriceOptional(q.PreviousDayClosing)
	open, _ := parseCNBCPriceOptional(q.Open)
	low, _ := parseCNBCPriceOptional(q.Low)
	high, _ := parseCNBCPriceOptional(q.High)

	change, changePct := cnbcResolvedChange(q, last, prev)
	return cnbcResolvedQuote{
		Price:     last,
		Change:    change,
		ChangePct: changePct,
		Sparkline: cnbcSessionSparkline(prev, open, low, high, last),
		PriceTime: cnbcKeyNumberPriceTime(q.LastTime, day, cutoff),
	}, nil
}

func cnbcResolvedChange(q *cnbcFormattedQuote, last, prev float64) (float64, float64) {
	if raw := strings.TrimSpace(q.Change); raw != "" && raw != "-" {
		if ch, err := parseCNBCPrice(raw); err == nil {
			if rawPct := strings.TrimSpace(q.ChangePct); rawPct != "" && rawPct != "-" {
				if pct, err := parseCNBCPercentOptional(rawPct); err == nil {
					return ch, pct
				}
			}
			if prev != 0 {
				return ch, (ch / prev) * 100
			}
			return ch, 0
		}
	}
	if prev != 0 {
		ch := last - prev
		return ch, (ch / prev) * 100
	}
	return 0, 0
}

func cnbcKeyNumberPriceTime(lastTime string, reportDay, cutoff time.Time) string {
	lastTS, err := parseCNBCLastTime(lastTime)
	if err != nil {
		return cnbcDigestCutoffTimeLabel(reportDay)
	}
	vnLast := lastTS.In(vnTimezone)
	sameDay := vnLast.Year() == reportDay.Year() &&
		vnLast.Month() == reportDay.Month() &&
		vnLast.Day() == reportDay.Day()
	if sameDay && !vnLast.After(cutoff) {
		return formatDigestPriceTimeLabel(vnLast)
	}
	return cnbcDigestCutoffTimeLabel(reportDay)
}

func cnbcDigestCutoffTimeLabel(reportDay time.Time) string {
	t := time.Date(
		reportDay.Year(), reportDay.Month(), reportDay.Day(),
		MorningDigestHour-1, 59, 0, 0, vnTimezone,
	)
	return formatDigestPriceTimeLabel(t)
}

// cnbcSessionSparkline builds an 8-point sparkline from CNBC OHLC + previous close.
func cnbcSessionSparkline(prev, open, low, high, last float64) []float64 {
	if prev == 0 && open == 0 && low == 0 && high == 0 && last == 0 {
		return nil
	}
	if prev == 0 {
		prev = open
	}
	if open == 0 {
		open = prev
	}
	if low == 0 {
		low = open
	}
	if high == 0 {
		high = last
	}
	if last == 0 {
		last = high
	}
	mid := (low + high) / 2
	return []float64{
		prev,
		prev + (open-prev)*0.5,
		open,
		open + (low-open)*0.5,
		low,
		mid,
		high,
		last,
	}
}

func parseCNBCPrice(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if raw == "" || raw == "-" {
		return 0, fmt.Errorf("empty CNBC price")
	}
	return strconv.ParseFloat(raw, 64)
}

func parseCNBCPriceOptional(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if raw == "" || raw == "-" {
		return 0, fmt.Errorf("empty CNBC price")
	}
	return strconv.ParseFloat(raw, 64)
}

func parseCNBCPercentOptional(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if raw == "" || raw == "-" {
		return 0, fmt.Errorf("empty CNBC percent")
	}
	return strconv.ParseFloat(raw, 64)
}

func formatCNBCFuturesPrice(price float64) string {
	return fmt.Sprintf("$%.2f", price)
}

func parseCNBCLastTime(lastTime string) (time.Time, error) {
	lastTime = strings.TrimSpace(lastTime)
	if lastTime == "" {
		return time.Time{}, fmt.Errorf("empty CNBC last_time")
	}
	layouts := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
		time.RFC3339,
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, lastTime)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

