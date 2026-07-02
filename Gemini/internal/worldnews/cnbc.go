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
	cnbcQuoteAPI   = "https://quote.cnbc.com/quote-html-webservice/restQuote/symbolType/symbol"
	cnbcDataSource = "CNBC"
	cnbcBrentSymbol = "@LCO.1" // ICE Brent Crude
	cnbcGoldSymbol  = "@GC.1"  // COMEX Gold
)

type cnbcFormattedResponse struct {
	FormattedQuoteResult struct {
		FormattedQuote []cnbcFormattedQuote `json:"FormattedQuote"`
	} `json:"FormattedQuoteResult"`
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

func (s *Service) fetchCNBCFormattedQuote(symbol string) (*cnbcFormattedQuote, error) {
	apiURL := fmt.Sprintf(
		"%s?symbols=%s&requestMethod=quick&exthrs=1&noform=1&partnerId=2&fund=1&output=json",
		cnbcQuoteAPI,
		url.QueryEscape(symbol),
	)
	body, err := s.httpGet(apiURL)
	if err != nil {
		return nil, err
	}
	var data cnbcFormattedResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.FormattedQuoteResult.FormattedQuote) == 0 {
		return nil, fmt.Errorf("empty CNBC quote for %s", symbol)
	}
	q := data.FormattedQuoteResult.FormattedQuote[0]
	if strings.TrimSpace(q.Last) == "" {
		return nil, fmt.Errorf("CNBC quote missing last price for %s", symbol)
	}
	return &q, nil
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
	change, err := parseCNBCPrice(q.Change)
	if err != nil {
		return KeyNumber{}, err
	}
	changePct, err := parseCNBCPercent(q.ChangePct)
	if err != nil {
		return KeyNumber{}, err
	}

	_, cutoff := morningDigestWindow(calendarDay.In(vnTimezone))
	prev, _ := parseCNBCPrice(q.PreviousDayClosing)
	open, _ := parseCNBCPrice(q.Open)
	low, _ := parseCNBCPrice(q.Low)
	high, _ := parseCNBCPrice(q.High)

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

func parseCNBCPercent(raw string) (float64, error) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
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