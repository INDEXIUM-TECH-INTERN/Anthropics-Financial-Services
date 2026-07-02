package worldnews

import (
	"encoding/json"
	"fmt"
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

	last, err := parseCNBCPrice(q.Last)
	if err != nil {
		return KeyNumber{}, err
	}
	prev, _ := parseCNBCPriceOptional(q.PreviousDayClosing)
	open, _ := parseCNBCPriceOptional(q.Open)
	low, _ := parseCNBCPriceOptional(q.Low)
	high, _ := parseCNBCPriceOptional(q.High)

	change, _ := parseCNBCPriceOptional(q.Change)
	if change == 0 && prev != 0 {
		change = last - prev
	}
	changePct, _ := parseCNBCPercentOptional(q.ChangePct)
	if changePct == 0 && prev != 0 {
		changePct = (change / prev) * 100
	}

	_, cutoff := morningDigestWindow(calendarDay.In(vnTimezone))
	_, pctStr, positive := formatChange(change, changePct)
	return KeyNumber{
		Label:           label,
		Value:           formatCNBCFuturesPrice(last),
		Change:          pctStr,
		IsPositive:      positive,
		Sparkline:       cnbcSessionSparkline(prev, open, low, high, last),
		SparklineLabels: nil,
		PriceTime:       formatCNBCPriceTime(q.LastTime, cutoff),
		Source:          cnbcDataSource,
		URL:             cnbcQuotePageURL(symbol),
		Symbol:          displaySymbol,
	}, nil
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

func formatCNBCPriceTime(lastTime string, cutoff time.Time) string {
	if strings.TrimSpace(lastTime) == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
		time.RFC3339,
	}
	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, lastTime)
		if err == nil {
			break
		}
	}
	if err != nil {
		return ""
	}
	vn := t.In(vnTimezone)
	if !vn.After(cutoff) {
		return formatChartVNLabel(vn) + " GMT+7"
	}
	return formatDigestPriceTimeLabel(t)
}