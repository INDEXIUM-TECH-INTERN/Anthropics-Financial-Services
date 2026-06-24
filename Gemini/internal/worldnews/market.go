package worldnews

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const yahooChartURL = "https://query1.finance.yahoo.com/v8/finance/chart"

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
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
}

func (s *Service) fetchQuote(symbol, label string, tradingDay time.Time) (*quoteSnapshot, error) {
	isToday := sameTradingDay(tradingDay, time.Now().In(vnTimezone))
	var apiURL string
	if isToday {
		apiURL = fmt.Sprintf("%s/%s?interval=30m&range=1d", yahooChartURL, symbol)
	} else {
		loc := usEastern
		start := time.Date(tradingDay.Year(), tradingDay.Month(), tradingDay.Day(), 9, 0, 0, 0, loc)
		end := time.Date(tradingDay.Year(), tradingDay.Month(), tradingDay.Day(), 17, 0, 0, 0, loc)
		apiURL = fmt.Sprintf("%s/%s?period1=%d&period2=%d&interval=30m", yahooChartURL, symbol, start.Unix(), end.Unix())
	}

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
	meta := result.Meta
	prev := meta.ChartPreviousClose
	if prev == 0 {
		prev = meta.PreviousClose
	}

	price := meta.RegularMarketPrice
	points, labels := extractSeries(result.Timestamp, result.Indicators.Quote[0].Close)
	if len(points) > 0 {
		price = points[len(points)-1]
	}
	if prev == 0 && len(points) > 1 {
		prev = points[0]
	}

	change := price - prev
	var pct float64
	if prev != 0 {
		pct = (change / prev) * 100
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
	}, nil
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
		return fmt.Sprintf("%.2f", price)
	}
}

func formatChange(change, pct float64) (string, string, bool) {
	positive := change >= 0
	sign := ""
	if positive && change > 0 {
		sign = "+"
	}
	absChange := change
	if absChange < 0 {
		absChange = -absChange
	}
	changeStr := fmt.Sprintf("%s%.2f", sign, change)
	if strings.Contains(changeStr, "$") == false && absChange > 100 {
		changeStr = fmt.Sprintf("%s%.2f", sign, change)
	}
	pctStr := fmt.Sprintf("%s%.2f%%", sign, pct)
	return changeStr, pctStr, positive
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