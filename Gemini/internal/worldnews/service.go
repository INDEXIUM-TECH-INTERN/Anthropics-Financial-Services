package worldnews

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	vnTimezone = time.FixedZone("ICT", 7*3600)
	usEastern  *time.Location
)

func init() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		usEastern = time.FixedZone("ET", -5*3600)
	} else {
		usEastern = loc
	}
}

// Service aggregates live market data and RSS headlines.
type Service struct {
	client *http.Client
	mu     sync.RWMutex
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	report    WorldNewsReport
	expiresAt time.Time
}

// DefaultService is the shared instance used by HTTP handlers.
var DefaultService = NewService()

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 25 * time.Second},
		cache:  make(map[string]cacheEntry),
	}
}

func (s *Service) GetAvailableDates() DatesResponse {
	now := time.Now().In(vnTimezone)
	var dates []DateOption
	today := now.Format("2006-01-02")

	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -i)
		trading := normalizeTradingDay(d)
		value := trading.Format("2006-01-02")
		label := trading.Format("02/01/2006")
		if value == today {
			label += " (Hôm nay)"
		}
		dates = append(dates, DateOption{
			Value:   value,
			Label:   label,
			IsToday: value == today,
		})
	}

	return DatesResponse{Dates: dates, DefaultDate: dates[0].Value}
}

func (s *Service) GetReport(dateStr string) (*WorldNewsReport, error) {
	tradingDay, err := parseDate(dateStr)
	if err != nil {
		return nil, err
	}
	tradingDay = normalizeTradingDay(tradingDay)
	key := tradingDay.Format("2006-01-02")

	s.mu.RLock()
	if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		r := entry.report
		s.mu.RUnlock()
		return &r, nil
	}
	s.mu.RUnlock()

	report, err := s.buildReport(tradingDay)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{report: *report, expiresAt: time.Now().Add(30 * time.Minute)}
	s.mu.Unlock()

	return report, nil
}

func (s *Service) buildReport(tradingDay time.Time) (*WorldNewsReport, error) {
	sp500, err := s.fetchQuote("%5EGSPC", "S&P 500", tradingDay)
	if err != nil {
		return nil, fmt.Errorf("S&P 500: %w", err)
	}
	nasdaq, err := s.fetchQuote("%5EIXIC", "Nasdaq", tradingDay)
	if err != nil {
		nasdaq = nil
	}
	wti, err := s.fetchQuote("CL%3DF", "WTI", tradingDay)
	if err != nil {
		return nil, fmt.Errorf("WTI: %w", err)
	}
	brent, err := s.fetchQuote("BZ%3DF", "Brent", tradingDay)
	if err != nil {
		brent = wti
	}
	gold, err := s.fetchQuote("GC%3DF", "Vàng", tradingDay)
	if err != nil {
		gold = nil
	}
	dxy, err := s.fetchQuote("DX-Y.NYB", "DXY", tradingDay)
	if err != nil {
		dxy = nil
	}

	newsItems, _ := s.fetchAllNews()
	since := time.Now().Add(-24 * time.Hour)

	var vtvItems, vnItems, breakingItems []rssItem
	for _, it := range newsItems {
		if it.PubDate.Before(since) {
			continue
		}
		switch it.Kind {
		case "vtv":
			if len(vtvItems) < 5 {
				vtvItems = append(vtvItems, it)
			}
		case "vietnam":
			if len(vnItems) < 6 {
				vnItems = append(vnItems, it)
			}
		case "breaking", "global":
			if len(breakingItems) < 8 {
				breakingItems = append(breakingItems, it)
			}
		}
	}

	stockChange, stockPct, stockPos := formatChange(sp500.Change, sp500.ChangePct)
	nasdaqVal := ""
	nasdaqChg := ""
	nasdaqPct := ""
	if nasdaq != nil {
		nasdaqVal = formatPrice(nasdaq.Symbol, nasdaq.Price)
		_, nasdaqPct, _ = formatChange(nasdaq.Change, nasdaq.ChangePct)
		nasdaqChg = fmt.Sprintf("%+.2f", nasdaq.Change)
	}

	wtiChg, wtiPct, wtiPos := formatChange(wti.Change, wti.ChangePct)
	brentChg, brentPct, brentPos := formatChange(brent.Change, brent.ChangePct)

	report := &WorldNewsReport{
		Date: tradingDay.Format("2006-01-02"),
		QuickHighlights: buildQuickHighlights(sp500, nasdaq, wti, brent, gold, dxy, breakingItems),
		KeyNumbers: []KeyNumber{
			{
				Label:      "S&P 500",
				Value:      formatPrice(sp500.Symbol, sp500.Price),
				Change:     stockPct,
				IsPositive: stockPos,
				Sparkline:  sp500.ChartPoints,
			},
			{
				Label:      "Dầu Brent",
				Value:      formatPrice(brent.Symbol, brent.Price),
				Change:     brentPct,
				IsPositive: brentPos,
				Sparkline:  brent.ChartPoints,
			},
		},
		Stocks: StockSection{
			IndexName:     "S&P 500 & Nasdaq Composite",
			Value:         joinValues(formatPrice(sp500.Symbol, sp500.Price), nasdaqVal),
			Change:        joinValues(stockChange, nasdaqChg),
			ChangePercent: joinValues(stockPct, nasdaqPct),
			IsPositive:    stockPos,
			ChartPoints:   sp500.ChartPoints,
			ChartLabels:   sp500.ChartLabels,
			Thumbnail:     "/images/cnbc_trader.png",
			Highlights:    buildStockHighlights(sp500, nasdaq, breakingItems),
			CNBCUrl:       "https://www.cnbc.com/world/",
			WSJUrl:        "https://www.wsj.com/finance/stocks?mod=nav_top_subsection",
			ReutersUrl:    "https://www.reuters.com/markets/stocks/",
		},
		Oil: OilSection{
			WTIPrice:         formatPrice(wti.Symbol, wti.Price),
			WTIChange:        wtiChg,
			WTIPercent:       wtiPct,
			WTIPositive:      wtiPos,
			BrentPrice:       formatPrice(brent.Symbol, brent.Price),
			BrentChange:      brentChg,
			BrentPercent:     brentPct,
			BrentPositive:    brentPos,
			ChartPointsWTI:   wti.ChartPoints,
			ChartPointsBrent: brent.ChartPoints,
			ChartLabels:      coalesceLabels(wti.ChartLabels, brent.ChartLabels),
			Thumbnail:        "/images/reuters_oil.png",
			Highlights:       buildOilHighlights(wti, brent, breakingItems),
			ReutersUrl:       "https://www.reuters.com/markets/",
			NYTimesUrl:       "https://www.nytimes.com/section/business/energy-environment?page=2",
			WSJUrl:           "https://www.wsj.com/business/energy-oil?mod=nav_top_subsection",
		},
		GoldUsd: buildGoldSection(gold, dxy, breakingItems),
		VTVIndexNews:       toNewsArticles(vtvItems),
		VietnamFinanceNews: toNewsArticles(vnItems),
		BreakingNews:       toBreakingNews(takeItems(breakingItems, 5)),
		Watchlist:          buildWatchlist(breakingItems),
		GeneratedAt:        time.Now().In(vnTimezone).Format(time.RFC3339),
		DataSource:         "Yahoo Finance (thị trường) + RSS (CNBC, Bloomberg, FT, CNN, VNeconomy, VNExpress, Vietstock...)",
	}

	if gold != nil {
		_, goldPct, goldPos := formatChange(gold.Change, gold.ChangePct)
		report.KeyNumbers = append(report.KeyNumbers, KeyNumber{
			Label:      "Vàng thế giới",
			Value:      formatPrice(gold.Symbol, gold.Price),
			Change:     goldPct,
			IsPositive: goldPos,
			Sparkline:  gold.ChartPoints,
		})
	}

	return report, nil
}

func buildGoldSection(gold, dxy *quoteSnapshot, news []rssItem) GoldUSDSection {
	sec := GoldUSDSection{
		ChartLabels: []string{},
		Highlights:  []string{},
		ReutersUrl:  "https://www.reuters.com/markets/",
	}
	if gold != nil {
		gChg, gPct, gPos := formatChange(gold.Change, gold.ChangePct)
		sec.GoldPrice = formatPrice(gold.Symbol, gold.Price)
		sec.GoldChange = gChg
		sec.GoldPercent = gPct
		sec.GoldPositive = gPos
		sec.ChartPointsGold = gold.ChartPoints
		sec.ChartLabels = gold.ChartLabels
		sec.Highlights = append(sec.Highlights,
			fmt.Sprintf("Vàng giao dịch quanh %s (%s) — dữ liệu Yahoo Finance.", sec.GoldPrice, gPct))
	}
	if dxy != nil {
		dChg, dPct, dPos := formatChange(dxy.Change, dxy.ChangePct)
		sec.DXYPrice = formatPrice(dxy.Symbol, dxy.Price)
		sec.DXYChange = dChg
		sec.DXYPercent = dPct
		sec.DXYPositive = dPos
		sec.ChartPointsDXY = dxy.ChartPoints
		if len(sec.ChartLabels) == 0 {
			sec.ChartLabels = dxy.ChartLabels
		}
		sec.Highlights = append(sec.Highlights,
			fmt.Sprintf("Chỉ số USD (DXY) ở mức %s (%s).", sec.DXYPrice, dPct))
	}
	for _, it := range news {
		lower := strings.ToLower(it.Title)
		if strings.Contains(lower, "gold") || strings.Contains(lower, "dollar") || strings.Contains(lower, "usd") {
			sec.Highlights = append(sec.Highlights, fmt.Sprintf("%s: %s", it.Source, it.Title))
			if len(sec.Highlights) >= 4 {
				break
			}
		}
	}
	return sec
}

func buildQuickHighlights(sp, nd, wti, brent, gold, dxy *quoteSnapshot, news []rssItem) []string {
	var out []string
	if sp != nil {
		dir := "giảm"
		if sp.IsPositive {
			dir = "tăng"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		out = append(out, fmt.Sprintf("S&P 500 %s %s, đóng cửa ở %s (Yahoo Finance).", dir, pct, formatPrice(sp.Symbol, sp.Price)))
	}
	if wti != nil && brent != nil {
		_, wPct, _ := formatChange(wti.Change, wti.ChangePct)
		_, bPct, _ := formatChange(brent.Change, brent.ChangePct)
		out = append(out, fmt.Sprintf("Dầu WTI %s, Brent %s — WTI %s, Brent %s.", wPct, bPct, formatPrice(wti.Symbol, wti.Price), formatPrice(brent.Symbol, brent.Price)))
	}
	if gold != nil {
		_, gPct, _ := formatChange(gold.Change, gold.ChangePct)
		out = append(out, fmt.Sprintf("Vàng thế giới %s (%s).", formatPrice(gold.Symbol, gold.Price), gPct))
	}
	if dxy != nil {
		_, dPct, _ := formatChange(dxy.Change, dxy.ChangePct)
		out = append(out, fmt.Sprintf("USD Index (DXY) %s.", dPct))
	}
	for _, it := range news {
		if len(out) >= 6 {
			break
		}
		out = append(out, fmt.Sprintf("[%s] %s", it.Source, it.Title))
	}
	return out
}

func buildStockHighlights(sp, nd *quoteSnapshot, news []rssItem) []string {
	var out []string
	if sp != nil {
		dir := "điều chỉnh giảm"
		if sp.IsPositive {
			dir = "tăng điểm"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		out = append(out, fmt.Sprintf("S&P 500 %s %s trong phiên gần nhất (dữ liệu Yahoo Finance).", dir, pct))
	}
	if nd != nil {
		_, pct, _ := formatChange(nd.Change, nd.ChangePct)
		out = append(out, fmt.Sprintf("Nasdaq Composite %s, chốt %s.", pct, formatPrice(nd.Symbol, nd.Price)))
	}
	for _, it := range news {
		lower := strings.ToLower(it.Title)
		if strings.Contains(lower, "stock") || strings.Contains(lower, "market") || strings.Contains(lower, "s&p") || strings.Contains(lower, "nasdaq") || it.Source == "CNBC" {
			out = append(out, fmt.Sprintf("%s: %s", it.Source, it.Title))
			if len(out) >= 4 {
				break
			}
		}
	}
	return out
}

func buildOilHighlights(wti, brent *quoteSnapshot, news []rssItem) []string {
	var out []string
	if wti != nil {
		_, pct, pos := formatChange(wti.Change, wti.ChangePct)
		dir := "giảm"
		if pos {
			dir = "tăng"
		}
		out = append(out, fmt.Sprintf("Dầu WTI (NYMEX) %s %s, giá %s.", dir, pct, formatPrice(wti.Symbol, wti.Price)))
	}
	if brent != nil {
		_, pct, pos := formatChange(brent.Change, brent.ChangePct)
		dir := "giảm"
		if pos {
			dir = "tăng"
		}
		out = append(out, fmt.Sprintf("Dầu Brent (ICE) %s %s, giá %s.", dir, pct, formatPrice(brent.Symbol, brent.Price)))
	}
	for _, it := range news {
		lower := strings.ToLower(it.Title)
		if strings.Contains(lower, "oil") || strings.Contains(lower, "crude") || strings.Contains(lower, "opec") || strings.Contains(lower, "energy") {
			out = append(out, fmt.Sprintf("%s: %s", it.Source, it.Title))
			if len(out) >= 4 {
				break
			}
		}
	}
	return out
}

func buildWatchlist(news []rssItem) []WatchlistItem {
	items := []WatchlistItem{
		{Event: "Theo dõi lịch kinh tế Mỹ & châu Âu", Time: "Hôm nay", Importance: "high", Source: "Bloomberg"},
		{Event: "Diễn biến giá dầu & vàng sau phiên chốt", Time: "Sáng nay (GMT+7)", Importance: "medium", Source: "Reuters"},
	}
	for _, it := range news {
		lower := strings.ToLower(it.Title)
		if strings.Contains(lower, "fed") || strings.Contains(lower, "cpi") || strings.Contains(lower, "ecb") || strings.Contains(lower, "jobs") {
			items = append(items, WatchlistItem{
				Event:      it.Title,
				Time:       relativeTime(it.PubDate),
				Importance: "high",
				Source:     it.Source,
			})
			if len(items) >= 5 {
				break
			}
		}
	}
	return items
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now().In(vnTimezone), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, vnTimezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %s", s)
	}
	return t, nil
}

func normalizeTradingDay(d time.Time) time.Time {
	d = d.In(vnTimezone)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func sameTradingDay(a, b time.Time) bool {
	a = a.In(vnTimezone)
	b = b.In(vnTimezone)
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func joinValues(a, b string) string {
	if b == "" {
		return a
	}
	if a == "" {
		return b
	}
	return a + " / " + b
}

func coalesceLabels(a, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
}

func takeItems(items []rssItem, n int) []rssItem {
	if len(items) <= n {
		return items
	}
	return items[:n]
}