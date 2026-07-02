package worldnews

import (
	"fmt"
	"sync"
	"time"
)

type stockSymbolDef struct {
	Encoded    string
	Label      string
	CNBCSymbol string
}

type stockTabDef struct {
	ID      string
	Label   string
	Symbols []stockSymbolDef
}

// StockTabCount — số tab biểu đồ trong mục Chứng khoán Thế giới.
const StockTabCount = 4

// StockChartsPerTab — mỗi tab hiển thị đúng 4 biểu đồ (lưới 2×2).
const StockChartsPerTab = 4

// WorldStockTabDefs — 4 tab × 4 mã; muốn xem thêm → Yahoo Finance.
var WorldStockTabDefs = []stockTabDef{
	{
		ID:    "tab-1",
		Label: "Chỉ số Mỹ",
		Symbols: []stockSymbolDef{
			{Encoded: "%5EGSPC", Label: "S&P 500", CNBCSymbol: ".SPX"},
			{Encoded: "%5EIXIC", Label: "Nasdaq Composite", CNBCSymbol: ".IXIC"},
			{Encoded: "%5EDJI", Label: "Dow Jones", CNBCSymbol: ".DJI"},
			{Encoded: "%5ERUT", Label: "Russell 2000", CNBCSymbol: ".RUT"},
		},
	},
	{
		ID:    "tab-2",
		Label: "Big Tech I",
		Symbols: []stockSymbolDef{
			{Encoded: "AAPL", Label: "Apple", CNBCSymbol: "AAPL"},
			{Encoded: "MSFT", Label: "Microsoft", CNBCSymbol: "MSFT"},
			{Encoded: "NVDA", Label: "NVIDIA", CNBCSymbol: "NVDA"},
			{Encoded: "GOOGL", Label: "Alphabet", CNBCSymbol: "GOOGL"},
		},
	},
	{
		ID:    "tab-3",
		Label: "Big Tech II",
		Symbols: []stockSymbolDef{
			{Encoded: "AMZN", Label: "Amazon", CNBCSymbol: "AMZN"},
			{Encoded: "META", Label: "Meta", CNBCSymbol: "META"},
			{Encoded: "TSLA", Label: "Tesla", CNBCSymbol: "TSLA"},
			{Encoded: "NFLX", Label: "Netflix", CNBCSymbol: "NFLX"},
		},
	},
	{
		ID:    "tab-4",
		Label: "Blue chip",
		Symbols: []stockSymbolDef{
			{Encoded: "JPM", Label: "JPMorgan", CNBCSymbol: "JPM"},
			{Encoded: "V", Label: "Visa", CNBCSymbol: "V"},
			{Encoded: "BRK-B", Label: "Berkshire Hathaway", CNBCSymbol: "BRK.B"},
			{Encoded: "ORCL", Label: "Oracle", CNBCSymbol: "ORCL"},
		},
	},
}

func quoteToStockInstrument(q *quoteSnapshot, cnbcSymbol string) StockInstrument {
	chg, pct, pos := formatChange(q.Change, q.ChangePct)
	displaySymbol := yahooFinanceDisplaySymbol(q.Symbol)
	quoteURL := yahooFinanceQuoteURL(q.Symbol)
	if cnbcSymbol != "" {
		quoteURL = cnbcQuotePageURL(cnbcSymbol)
	}
	return StockInstrument{
		Symbol:        displaySymbol,
		Label:         q.Label,
		Value:         formatPrice(q.Symbol, q.Price),
		Change:        chg,
		ChangePercent: pct,
		IsPositive:    pos,
		ChartPoints:   q.ChartPoints,
		ChartLabels:   q.ChartLabels,
		PriceTime:     q.PriceTime,
		QuoteURL:      quoteURL,
	}
}

func (s *Service) fetchWorldStockQuotes(calendarDay time.Time) (map[string]*quoteSnapshot, []StockInstrument, []StockTab) {
	type job struct {
		tabIdx     int
		symIdx     int
		encoded    string
		label      string
		cnbcSymbol string
	}
	var jobs []job
	for ti, tab := range WorldStockTabDefs {
		for si, sym := range tab.Symbols {
			jobs = append(jobs, job{
				tabIdx:     ti,
				symIdx:     si,
				encoded:    sym.Encoded,
				label:      sym.Label,
				cnbcSymbol: sym.CNBCSymbol,
			})
		}
	}

	type result struct {
		job   job
		quote *quoteSnapshot
		ok    bool
	}

	ch := make(chan result, len(jobs))
	var wg sync.WaitGroup

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			q, err := s.fetchCNBCStockQuote(j.cnbcSymbol, j.label, j.encoded, calendarDay)
			if err != nil {
				fmt.Printf("⚠️ [WorldNews] CNBC stock quote failed (%s), fallback Yahoo %s: %v\n", j.cnbcSymbol, j.encoded, err)
				q, err = s.fetchQuote(j.encoded, j.label, calendarDay)
			}
			if err != nil {
				ch <- result{job: j, ok: false}
				return
			}
			ch <- result{job: j, quote: q, ok: true}
		}(j)
	}

	wg.Wait()
	close(ch)

	quotes := make(map[string]*quoteSnapshot)
	tabBuckets := make([]map[int]StockInstrument, len(WorldStockTabDefs))
	for i := range tabBuckets {
		tabBuckets[i] = make(map[int]StockInstrument)
	}

	for r := range ch {
		if !r.ok {
			continue
		}
		quotes[r.job.encoded] = r.quote
		tabBuckets[r.job.tabIdx][r.job.symIdx] = quoteToStockInstrument(r.quote, r.job.cnbcSymbol)
	}

	tabs := make([]StockTab, 0, len(WorldStockTabDefs))
	var flat []StockInstrument
	for ti, tabDef := range WorldStockTabDefs {
		tab := StockTab{ID: tabDef.ID, Label: tabDef.Label}
		for si := range tabDef.Symbols {
			if inst, ok := tabBuckets[ti][si]; ok {
				tab.Instruments = append(tab.Instruments, inst)
				flat = append(flat, inst)
			}
		}
		if len(tab.Instruments) > 0 {
			tabs = append(tabs, tab)
		}
	}
	return quotes, flat, tabs
}

func flattenStockTabInstruments(tabs []StockTab) []StockInstrument {
	var out []StockInstrument
	for _, tab := range tabs {
		out = append(out, tab.Instruments...)
	}
	return out
}