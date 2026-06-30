package worldnews

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	yahooChartURL  = "https://query1.finance.yahoo.com/v8/finance/chart"
	yahooFinanceURL = "https://finance.yahoo.com"
	marketDataSource = "Yahoo Finance"
)

func yahooFinanceQuoteURL(encodedSymbol string) string {
	return yahooFinanceURL + "/quote/" + encodedSymbol
}

func yahooFinanceDisplaySymbol(encodedSymbol string) string {
	decoded, err := url.PathUnescape(encodedSymbol)
	if err != nil || decoded == "" {
		return encodedSymbol
	}
	return decoded
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				PreviousClose      float64 `json:"previousClose"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

type quoteBar struct {
	ts    int64
	close float64
	day   string
}

type quoteSnapshot struct {
	Symbol       string
	Label        string
	Price        float64
	PrevClose    float64
	Change       float64
	ChangePct    float64
	IsPositive   bool
	ChartPoints  []float64
	ChartLabels  []string
	PriceTime    string
}

func (s *Service) fetchQuote(symbol, label string, calendarDay time.Time) (*quoteSnapshot, error) {
	day := calendarDay.In(vnTimezone)
	_, cutoff := morningDigestWindow(day)
	quoteDay := digestMarketQuoteDay(day)

	priceSnap, err := s.fetchPriceAtCutoff(symbol, label, quoteDay, cutoff)
	if err != nil {
		return s.fetchDailyQuote(symbol, label, quoteDay, cutoff)
	}

	dailySnap, err := s.fetchDailyQuote(symbol, label, quoteDay, cutoff)
	if err != nil {
		return priceSnap, nil
	}

	priceSnap.ChartPoints = dailySnap.ChartPoints
	priceSnap.ChartLabels = dailySnap.ChartLabels
	return priceSnap, nil
}

func (s *Service) fetchPriceAtCutoff(symbol, label string, quoteDay time.Time, cutoff time.Time) (*quoteSnapshot, error) {
	loc := usEastern
	periodStart := cutoff.Add(-12 * 24 * time.Hour).In(loc)
	periodEnd := cutoff.In(loc)
	apiURL := fmt.Sprintf(
		"%s/%s?period1=%d&period2=%d&interval=30m",
		yahooChartURL, symbol, periodStart.Unix(), periodEnd.Unix(),
	)
	body, err := s.httpGet(apiURL)
	if err != nil {
		return nil, err
	}

	var data yahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("empty yahoo intraday result for %s", symbol)
	}

	result := data.Chart.Result[0]
	timestamps := result.Timestamp
	closes := result.Indicators.Quote[0].Close
	if len(timestamps) == 0 || len(closes) == 0 {
		return nil, fmt.Errorf("no intraday price data for %s", symbol)
	}

	cutoffUnix := cutoff.Unix()
	type bar struct {
		ts    int64
		close float64
	}
	var bars []bar
	for i, c := range closes {
		if c == 0 || i >= len(timestamps) || timestamps[i] > cutoffUnix {
			continue
		}
		bars = append(bars, bar{ts: timestamps[i], close: c})
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("no intraday bars before cutoff for %s", symbol)
	}

	last := bars[len(bars)-1]
	price := last.close
	prev := last.close
	if len(bars) > 1 {
		prev = bars[len(bars)-2].close
	}

	meta := result.Meta
	quoteKey := quoteDay.In(loc).Format("2006-01-02")
	if meta.RegularMarketTime > 0 && meta.RegularMarketTime <= cutoffUnix && meta.RegularMarketPrice > 0 {
		rm := time.Unix(meta.RegularMarketTime, 0).In(loc)
		if rm.Format("2006-01-02") == quoteKey && isEquityLikeSymbol(symbol) {
			price = meta.RegularMarketPrice
			last.ts = meta.RegularMarketTime
		}
	}

	change := price - prev
	var pct float64
	if prev != 0 {
		pct = (change / prev) * 100
	}

	return &quoteSnapshot{
		Symbol:     symbol,
		Label:      label,
		Price:      price,
		PrevClose:  prev,
		Change:     change,
		ChangePct:  pct,
		IsPositive: change >= 0,
		PriceTime:  resolvePriceTime(meta.RegularMarketTime, quoteKey, last.ts, true, cutoff, loc),
	}, nil
}

func isEquityLikeSymbol(symbol string) bool {
	decoded, err := url.PathUnescape(symbol)
	if err != nil || decoded == "" {
		decoded = symbol
	}
	return strings.HasPrefix(decoded, "^") || (!strings.Contains(decoded, "=F") && !strings.Contains(decoded, "-"))
}

func (s *Service) fetchDailyQuote(symbol, label string, tradingDay time.Time, cutoff time.Time) (*quoteSnapshot, error) {
	loc := usEastern
	periodStart := time.Date(tradingDay.Year(), tradingDay.Month(), tradingDay.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -45)
	periodEnd := cutoff.In(loc)
	apiURL := fmt.Sprintf("%s/%s?period1=%d&period2=%d&interval=1d", yahooChartURL, symbol, periodStart.Unix(), periodEnd.Unix())
	return s.parseChartResponse(symbol, label, apiURL, tradingDay, cutoff, false)
}

func (s *Service) parseChartResponse(symbol, label, apiURL string, tradingDay time.Time, cutoff time.Time, intraday bool) (*quoteSnapshot, error) {
	body, err := s.httpGet(apiURL)
	if err != nil {
		return nil, err
	}

	var data yahooChartResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("empty yahoo result for %s", symbol)
	}

	result := data.Chart.Result[0]
	timestamps := result.Timestamp
	closes := result.Indicators.Quote[0].Close
	if len(timestamps) == 0 || len(closes) == 0 {
		return nil, fmt.Errorf("no price data for %s on %s", symbol, tradingDay.Format("2006-01-02"))
	}

	loc := usEastern
	targetKey := tradingDay.In(loc).Format("2006-01-02")

	cutoffUnix := cutoff.Unix()
	var bars []quoteBar
	for i, c := range closes {
		if c == 0 || i >= len(timestamps) || timestamps[i] > cutoffUnix {
			continue
		}
		dayKey := time.Unix(timestamps[i], 0).In(loc).Format("2006-01-02")
		bars = append(bars, quoteBar{ts: timestamps[i], close: c, day: dayKey})
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("no valid bars for %s", symbol)
	}

	idx := selectBarIndex(bars, targetKey, cutoff)

	price := bars[idx].close
	prev := bars[idx].close
	meta := result.Meta
	if meta.RegularMarketTime > 0 &&
		meta.RegularMarketTime <= cutoff.Unix() &&
		meta.RegularMarketPrice > 0 &&
		isEquityLikeSymbol(symbol) {
		rm := time.Unix(meta.RegularMarketTime, 0).In(loc)
		if rm.Format("2006-01-02") == targetKey {
			price = meta.RegularMarketPrice
		}
	}
	if idx > 0 {
		prev = bars[idx-1].close
	} else {
		if meta.ChartPreviousClose > 0 {
			prev = meta.ChartPreviousClose
		} else if meta.PreviousClose > 0 {
			prev = meta.PreviousClose
		}
	}

	change := price - prev
	var pct float64
	if prev != 0 {
		pct = (change / prev) * 100
	}

	var points []float64
	var labels []string
	if intraday {
		for i, c := range closes {
			if c == 0 || i >= len(timestamps) {
				continue
			}
			points = append(points, c)
			labels = append(labels, formatChartTimeLabel(timestamps[i], loc, true))
		}
	} else {
		start := idx - 7
		if start < 0 {
			start = 0
		}
		for i := start; i <= idx; i++ {
			points = append(points, bars[i].close)
			labels = append(labels, formatChartTimeLabel(bars[i].ts, loc, false))
		}
	}

	return &quoteSnapshot{
		Symbol:      symbol,
		Label:       label,
		Price:       price,
		PrevClose:   prev,
		Change:      change,
		ChangePct:   pct,
		IsPositive:  change >= 0,
		ChartPoints: points,
		ChartLabels: labels,
		PriceTime:   resolvePriceTime(result.Meta.RegularMarketTime, targetKey, bars[idx].ts, intraday, cutoff, loc),
	}, nil
}

func selectBarIndex(bars []quoteBar, targetKey string, cutoff time.Time) int {
	cutoffUnix := cutoff.Unix()
	idx := -1
	for i, b := range bars {
		if b.ts > cutoffUnix || b.day != targetKey {
			continue
		}
		idx = i
	}
	if idx >= 0 {
		return idx
	}
	for i := len(bars) - 1; i >= 0; i-- {
		if bars[i].ts <= cutoffUnix {
			return i
		}
	}
	return len(bars) - 1
}

func resolvePriceTime(regularMarketTime int64, targetKey string, barTS int64, intraday bool, cutoff time.Time, loc *time.Location) string {
	if regularMarketTime > 0 {
		rm := time.Unix(regularMarketTime, 0)
		if !rm.After(cutoff) {
			rmET := rm.In(loc)
			if rmET.Format("2006-01-02") == targetKey {
				return formatDigestPriceTimeLabel(rm)
			}
		}
	}
	if intraday {
		barTime := time.Unix(barTS, 0)
		if !barTime.After(cutoff) {
			return formatDigestPriceTimeLabel(barTime)
		}
	}
	day, err := time.ParseInLocation("2006-01-02", targetKey, loc)
	if err != nil {
		return formatDigestPriceTimeLabel(time.Unix(barTS, 0))
	}
	closeTime := time.Date(day.Year(), day.Month(), day.Day(), 16, 0, 0, 0, loc)
	return formatDigestPriceTimeLabel(closeTime)
}

func formatDigestPriceTimeLabel(t time.Time) string {
	return t.In(vnTimezone).Format("02/01 15:04") + " GMT+7"
}

func formatPriceTimeLabel(t time.Time) string {
	return t.Format("02/01 15:04")
}

func formatChartTimeLabel(ts int64, loc *time.Location, intraday bool) string {
	t := time.Unix(ts, 0).In(loc)
	if intraday {
		return t.Format("15:04")
	}
	return formatPriceTimeLabel(t)
}

func extractSeries(timestamps []int64, closes []float64) ([]float64, []string) {
	var points []float64
	var labels []string
	loc := usEastern
	for i, c := range closes {
		if c == 0 || i >= len(timestamps) {
			continue
		}
		points = append(points, c)
		t := time.Unix(timestamps[i], 0).In(loc)
		labels = append(labels, t.Format("15:04"))
	}
	return points, labels
}

func formatPrice(symbol string, price float64) string {
	switch {
	case strings.HasPrefix(symbol, "^"):
		return fmt.Sprintf("%.2f", price)
	case strings.Contains(symbol, "=F"):
		return fmt.Sprintf("$%.2f", price)
	default:
		return fmt.Sprintf("$%.2f", price)
	}
}

func formatChange(change, pct float64) (string, string, bool) {
	positive := change >= 0
	sign := ""
	if positive && change > 0 {
		sign = "+"
	}
	changeStr := fmt.Sprintf("%s%.2f", sign, change)
	pctStr := fmt.Sprintf("%s%.2f%%", sign, pct)
	return changeStr, pctStr, positive
}

func formatVNDateLabel(iso string) string {
	parts := strings.Split(iso, "-")
	if len(parts) != 3 {
		return iso
	}
	return fmt.Sprintf("%s/%s", parts[2], parts[1])
}

func (s *Service) httpGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}