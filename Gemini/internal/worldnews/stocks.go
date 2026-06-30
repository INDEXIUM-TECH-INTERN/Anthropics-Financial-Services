package worldnews

import (
	"sync"
	"time"
)

type stockSymbolDef struct {
	Encoded string
	Label   string
}

type stockTabDef struct {
	ID      string
	Label   string
	Symbols []stockSymbolDef
}

// StockTabCount — số tab biểu đồ trong mục Chứng khoán Thế giới.
const StockTabCount = 4

// WorldStockTabDefs — 4 nhóm mã; muốn xem thêm → Yahoo Finance.
var WorldStockTabDefs = []stockTabDef{
	{
		ID:    "indices",
		Label: "Chỉ số Mỹ",
		Symbols: []stockSymbolDef{
			{Encoded: "%5EGSPC", Label: "S&P 500"},
			{Encoded: "%5EIXIC", Label: "Nasdaq Composite"},
			{Encoded: "%5EDJI", Label: "Dow Jones"},
		},
	},
	{
		ID:    "tech-1",
		Label: "Apple · Microsoft",
		Symbols: []stockSymbolDef{
			{Encoded: "AAPL", Label: "Apple"},
			{Encoded: "MSFT", Label: "Microsoft"},
		},
	},
	{
		ID:    "tech-2",
		Label: "NVIDIA · Alphabet",
		Symbols: []stockSymbolDef{
			{Encoded: "NVDA", Label: "NVIDIA"},
			{Encoded: "GOOGL", Label: "Alphabet"},
		},
	},
	{
		ID:    "tech-3",
		Label: "Amazon · Meta",
		Symbols: []stockSymbolDef{
			{Encoded: "AMZN", Label: "Amazon"},
			{Encoded: "META", Label: "Meta"},
		},
	},
}

func quoteToStockInstrument(q *quoteSnapshot) StockInstrument {
	chg, pct, pos := formatChange(q.Change, q.ChangePct)
	displaySymbol := yahooFinanceDisplaySymbol(q.Symbol)
	return StockInstrument{
		Symbol:        displaySymbol,
		Label:         q.Label,
		Value:         formatPrice(q.Symbol, q.Price),
		Change:        chg,
		ChangePercent: pct,
		IsPositive:    pos,
		ChartPoints:   q.ChartPoints,
		ChartLabels:   q.ChartLabels,
		QuoteURL:      yahooFinanceQuoteURL(q.Symbol),
	}
}

func (s *Service) fetchWorldStockQuotes(calendarDay time.Time) (map[string]*quoteSnapshot, []StockInstrument, []StockTab) {
	type job struct {
		tabIdx  int
		symIdx  int
		encoded string
		label   string
	}
	var jobs []job
	for ti, tab := range WorldStockTabDefs {
		for si, sym := range tab.Symbols {
			jobs = append(jobs, job{tabIdx: ti, symIdx: si, encoded: sym.Encoded, label: sym.Label})
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
			q, err := s.fetchQuote(j.encoded, j.label, calendarDay)
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
		tabBuckets[r.job.tabIdx][r.job.symIdx] = quoteToStockInstrument(r.quote)
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