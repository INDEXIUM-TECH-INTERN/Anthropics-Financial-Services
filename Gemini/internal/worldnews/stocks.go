package worldnews

import (
	"sync"
	"time"
)

type stockSymbolDef struct {
	Encoded string
	Label   string
}

// WorldStockSymbols — các mã hiển thị biểu đồ trong mục Chứng khoán Thế giới.
var WorldStockSymbols = []stockSymbolDef{
	{Encoded: "%5EGSPC", Label: "S&P 500"},
	{Encoded: "%5EIXIC", Label: "Nasdaq Composite"},
	{Encoded: "%5EDJI", Label: "Dow Jones"},
	{Encoded: "AAPL", Label: "Apple"},
	{Encoded: "MSFT", Label: "Microsoft"},
	{Encoded: "NVDA", Label: "NVIDIA"},
	{Encoded: "GOOGL", Label: "Alphabet"},
	{Encoded: "AMZN", Label: "Amazon"},
	{Encoded: "META", Label: "Meta"},
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

func (s *Service) fetchWorldStockQuotes(calendarDay time.Time, live bool) (map[string]*quoteSnapshot, []StockInstrument) {
	type result struct {
		key   string
		quote *quoteSnapshot
		order int
		ok    bool
	}

	ch := make(chan result, len(WorldStockSymbols))
	var wg sync.WaitGroup

	for i, def := range WorldStockSymbols {
		wg.Add(1)
		go func(order int, def stockSymbolDef) {
			defer wg.Done()
			q, err := s.fetchQuote(def.Encoded, def.Label, calendarDay, live)
			if err != nil {
				ch <- result{key: def.Encoded, order: order, ok: false}
				return
			}
			ch <- result{key: def.Encoded, quote: q, order: order, ok: true}
		}(i, def)
	}

	wg.Wait()
	close(ch)

	quotes := make(map[string]*quoteSnapshot, len(WorldStockSymbols))
	bucket := make(map[int]StockInstrument, len(WorldStockSymbols))
	for r := range ch {
		if !r.ok {
			continue
		}
		quotes[r.key] = r.quote
		bucket[r.order] = quoteToStockInstrument(r.quote)
	}

	instruments := make([]StockInstrument, 0, len(WorldStockSymbols))
	for i := 0; i < len(WorldStockSymbols); i++ {
		if inst, ok := bucket[i]; ok {
			instruments = append(instruments, inst)
		}
	}
	return quotes, instruments
}