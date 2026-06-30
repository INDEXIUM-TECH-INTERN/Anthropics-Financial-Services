package worldnews

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ReportHistoryDays — số ngày lịch có thể chọn trong dropdown bản tin.
const ReportHistoryDays = 90

// reportCacheVersion — tăng khi đổi schema báo cáo để tránh trả cache cũ.
const reportCacheVersion = 5

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
	client  *http.Client
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	textGen TextGenerator
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

// SetTextGenerator wires Gemini (or any LLM) for AI-written highlight summaries.
func (s *Service) SetTextGenerator(gen TextGenerator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.textGen = gen
}

func (s *Service) GetAvailableDates() DatesResponse {
	now := time.Now().In(vnTimezone)
	var dates []DateOption
	today := now.Format("2006-01-02")

	for i := 0; i < ReportHistoryDays; i++ {
		d := now.AddDate(0, 0, -i)
		value := d.Format("2006-01-02")
		label := d.Format("02/01/2006")
		if value == today {
			label += " (Hôm nay)"
		}
		dates = append(dates, DateOption{
			Value:   value,
			Label:   label,
			IsToday: value == today,
		})
	}

	return DatesResponse{Dates: dates, DefaultDate: dates[0].Value, HistoryDays: ReportHistoryDays}
}

func (s *Service) GetReport(dateStr string) (*WorldNewsReport, error) {
	calendarDay, err := parseDate(dateStr)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s:v%d", calendarDay.Format("2006-01-02"), reportCacheVersion)

	s.mu.RLock()
	if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		r := entry.report
		s.mu.RUnlock()
		return &r, nil
	}
	s.mu.RUnlock()

	report, err := s.buildReport(calendarDay)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{report: *report, expiresAt: time.Now().Add(30 * time.Minute)}
	s.mu.Unlock()

	return report, nil
}

func (s *Service) buildReport(calendarDay time.Time) (*WorldNewsReport, error) {
	calendarDay = calendarDay.In(vnTimezone)
	quoteDay := digestMarketQuoteDay(calendarDay)
	quoteLabel := formatQuoteSessionLabel(quoteDay)
	since, until := morningDigestWindow(calendarDay)
	digestWindow := formatDigestWindow(since, until)

	stockQuotes, stockInstruments, stockTabs := s.fetchWorldStockQuotes(calendarDay)
	sp500 := stockQuotes["%5EGSPC"]
	if sp500 == nil {
		return nil, fmt.Errorf("S&P 500: no quote data")
	}
	nasdaq := stockQuotes["%5EIXIC"]
	wti, err := s.fetchQuote("CL%3DF", "WTI", calendarDay)
	if err != nil {
		return nil, fmt.Errorf("WTI: %w", err)
	}
	brent, err := s.fetchQuote("BZ%3DF", "Brent", calendarDay)
	if err != nil {
		brent = wti
	}
	gold, err := s.fetchQuote("GC%3DF", "Vàng", calendarDay)
	if err != nil {
		gold = nil
	}
	dxy, err := s.fetchQuote("DX-Y.NYB", "DXY", calendarDay)
	if err != nil {
		dxy = nil
	}

	newsItems := s.fetchNewsForReport(calendarDay)
	stockNewsItems := s.fetchStockNewsForReport(calendarDay)

	var vtvItems, vnItems, breakingItems []rssItem
	for _, it := range newsItems {
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

	dataSource := fmt.Sprintf(
		"Yahoo Finance (giá chốt trước 07:00 GMT+7, phiên %s) + tin tức trước 07:00 sáng, khung %s (GMT+7)",
		quoteLabel,
		digestWindow,
	)

	report := &WorldNewsReport{
		Date:            calendarDay.Format("2006-01-02"),
		HighlightSummary: s.resolveHighlightSummary(sp500, nasdaq, wti, brent, gold, dxy, breakingItems, quoteLabel, digestWindow),
		KeyNumbers: []KeyNumber{
			keyNumberFromQuote(sp500, "S&P 500", "CNBC", "https://www.cnbc.com/world/", "S&P 500"),
			keyNumberFromQuote(brent, "Dầu Brent", marketDataSource, yahooFinanceQuoteURL("BZ%3DF"), yahooFinanceDisplaySymbol("BZ%3DF")),
		},
		Stocks: StockSection{
			IndexName:     "Chỉ số & cổ phiếu theo dõi",
			Value:         joinValues(formatPrice(sp500.Symbol, sp500.Price), nasdaqVal),
			Change:        joinValues(stockChange, nasdaqChg),
			ChangePercent: joinValues(stockPct, nasdaqPct),
			IsPositive:    stockPos,
			ChartPoints:   sp500.ChartPoints,
			ChartLabels:   sp500.ChartLabels,
			Tabs:          stockTabs,
			Instruments:   stockInstruments,
			Thumbnail:     FaviconProxyPath("cnbc.com"),
			Highlights:    buildStockHighlights(sp500, nasdaq, stockNewsItems, quoteLabel),
			CNBCUrl:       "https://www.cnbc.com/world/",
			WSJUrl:        "https://www.wsj.com/finance/stocks?mod=nav_top_subsection",
			ReutersUrl:    "https://www.reuters.com/markets/stocks/",
			MarketSource:  marketDataSource,
			MarketURL:     yahooFinanceURL + "/world/",
			MarketSymbol:  "S&P 500, Nasdaq, Dow Jones & cổ phiếu lớn",
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
			Thumbnail:        FaviconProxyPath("reuters.com"),
			Highlights:       buildOilHighlights(wti, brent, breakingItems, quoteLabel),
			ReutersUrl:       "https://www.reuters.com/markets/",
			NYTimesUrl:       "https://www.nytimes.com/section/business/energy-environment?page=2",
			WSJUrl:           "https://www.wsj.com/business/energy-oil?mod=nav_top_subsection",
			MarketSource:     marketDataSource,
			MarketURL:        yahooFinanceQuoteURL("CL%3DF"),
			MarketSymbol:     yahooFinanceDisplaySymbol("CL%3DF") + ", " + yahooFinanceDisplaySymbol("BZ%3DF"),
		},
		GoldUsd:            buildGoldSection(gold, dxy, breakingItems, quoteLabel),
		VTVIndexNews:       toNewsArticles(vtvItems),
		VietnamFinanceNews: toNewsArticles(vnItems),
		BreakingNews:       toBreakingNews(takeItems(breakingItems, 5)),
		Watchlist:          buildWatchlist(breakingItems, calendarDay),
		GeneratedAt:        time.Now().In(vnTimezone).Format(time.RFC3339),
		DataSource:         dataSource,
		DigestWindow:       digestWindow,
		DigestUntil:        formatDigestUntil(until),
	}

	if gold != nil {
		report.KeyNumbers = append(report.KeyNumbers, keyNumberFromQuote(
			gold,
			"Vàng thế giới",
			marketDataSource,
			yahooFinanceQuoteURL("GC%3DF"),
			yahooFinanceDisplaySymbol("GC%3DF"),
		))
	}

	return report, nil
}

func formatVNDate(t time.Time) string {
	return t.In(vnTimezone).Format("02/01/2006")
}

func buildGoldSection(gold, dxy *quoteSnapshot, news []rssItem, tradingLabel string) GoldUSDSection {
	sec := GoldUSDSection{
		ChartLabels:  []string{},
		Highlights:   []string{},
		ReutersUrl:   "https://www.reuters.com/markets/",
		MarketSource: marketDataSource,
		MarketURL:    yahooFinanceQuoteURL("GC%3DF"),
		MarketSymbol: yahooFinanceDisplaySymbol("GC%3DF") + ", " + yahooFinanceDisplaySymbol("DX-Y.NYB"),
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
			fmt.Sprintf("Vàng phiên %s: %s (%s) — Yahoo Finance.", tradingLabel, sec.GoldPrice, gPct))
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
			fmt.Sprintf("DXY phiên %s: %s (%s).", tradingLabel, sec.DXYPrice, dPct))
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

func quickHighlightLogo(source, publisherHost string) string {
	if publisherHost != "" {
		return mediaField(publisherHost, false)
	}
	switch strings.ToLower(source) {
	case "cnbc":
		return FaviconProxyPath("cnbc.com")
	case "reuters":
		return FaviconProxyPath("reuters.com")
	case "wsj", "wall street journal":
		return FaviconProxyPath("wsj.com")
	case "yahoo finance":
		return FaviconProxyPath("finance.yahoo.com")
	default:
		return ""
	}
}

func buildQuickHighlights(sp, nd, wti, brent, gold, dxy *quoteSnapshot, news []rssItem, tradingLabel string) []QuickHighlight {
	var out []QuickHighlight
	if sp != nil {
		dir := "giảm"
		if sp.IsPositive {
			dir = "tăng"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		out = append(out, QuickHighlight{
			Text:   fmt.Sprintf("S&P 500 %s %s, đóng cửa phiên %s ở %s.", dir, pct, tradingLabel, formatPrice(sp.Symbol, sp.Price)),
			Source: "CNBC",
			URL:    "https://www.cnbc.com/world/",
			Logo:   quickHighlightLogo("CNBC", ""),
		})
	}
	if wti != nil && brent != nil {
		_, wPct, _ := formatChange(wti.Change, wti.ChangePct)
		_, bPct, _ := formatChange(brent.Change, brent.ChangePct)
		out = append(out, QuickHighlight{
			Text:   fmt.Sprintf("Dầu WTI %s, Brent %s — WTI %s, Brent %s.", wPct, bPct, formatPrice(wti.Symbol, wti.Price), formatPrice(brent.Symbol, brent.Price)),
			Source: marketDataSource,
			URL:    yahooFinanceQuoteURL("CL%3DF"),
			Logo:   quickHighlightLogo(marketDataSource, ""),
		})
	}
	if gold != nil {
		_, gPct, _ := formatChange(gold.Change, gold.ChangePct)
		out = append(out, QuickHighlight{
			Text:   fmt.Sprintf("Vàng thế giới %s (%s).", formatPrice(gold.Symbol, gold.Price), gPct),
			Source: marketDataSource,
			URL:    yahooFinanceQuoteURL("GC%3DF"),
			Logo:   quickHighlightLogo(marketDataSource, ""),
		})
	}
	if dxy != nil {
		_, dPct, _ := formatChange(dxy.Change, dxy.ChangePct)
		out = append(out, QuickHighlight{
			Text:   fmt.Sprintf("USD Index (DXY) %s.", dPct),
			Source: marketDataSource,
			URL:    yahooFinanceQuoteURL("DX-Y.NYB"),
			Logo:   quickHighlightLogo(marketDataSource, ""),
		})
	}
	for _, it := range news {
		if len(out) >= 6 {
			break
		}
		out = append(out, QuickHighlight{
			Text:   it.Title,
			Source: it.Source,
			URL:    it.Link,
			Logo:   quickHighlightLogo(it.Source, it.PublisherHost),
			Time:   it.PubDate.In(vnTimezone).Format("02/01/2006 15:04"),
		})
	}
	return out
}

func buildStockHighlights(sp, nd *quoteSnapshot, news []rssItem, tradingLabel string) []string {
	var out []string
	if sp != nil {
		dir := "điều chỉnh giảm"
		if sp.IsPositive {
			dir = "tăng điểm"
		}
		_, pct, _ := formatChange(sp.Change, sp.ChangePct)
		out = append(out, fmt.Sprintf("S&P 500 %s %s trong phiên %s.", dir, pct, tradingLabel))
	}
	if nd != nil {
		_, pct, _ := formatChange(nd.Change, nd.ChangePct)
		out = append(out, fmt.Sprintf("Nasdaq Composite %s, chốt %s.", pct, formatPrice(nd.Symbol, nd.Price)))
	}
	for _, it := range news {
		if !isAllowedStockNewsSource(it) {
			continue
		}
		out = append(out, fmt.Sprintf("%s: %s", it.Source, it.Title))
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func buildOilHighlights(wti, brent *quoteSnapshot, news []rssItem, tradingLabel string) []string {
	var out []string
	if wti != nil {
		_, pct, pos := formatChange(wti.Change, wti.ChangePct)
		dir := "giảm"
		if pos {
			dir = "tăng"
		}
		out = append(out, fmt.Sprintf("Dầu WTI (NYMEX) %s %s, chốt phiên %s: %s.", dir, pct, tradingLabel, formatPrice(wti.Symbol, wti.Price)))
	}
	if brent != nil {
		_, pct, pos := formatChange(brent.Change, brent.ChangePct)
		dir := "giảm"
		if pos {
			dir = "tăng"
		}
		out = append(out, fmt.Sprintf("Dầu Brent (ICE) %s %s, chốt phiên %s: %s.", dir, pct, tradingLabel, formatPrice(brent.Symbol, brent.Price)))
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

func buildWatchlist(news []rssItem, calendarDay time.Time) []WatchlistItem {
	when := formatVNDate(calendarDay)
	items := []WatchlistItem{
		{Event: "Theo dõi lịch kinh tế Mỹ & châu Âu", Time: when, Importance: "high", Source: "Bloomberg"},
		{Event: "Diễn biến giá dầu & vàng sau phiên chốt", Time: when, Importance: "medium", Source: "Reuters"},
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

func sameCalendarDay(a, b time.Time) bool {
	a = a.In(vnTimezone)
	b = b.In(vnTimezone)
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
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