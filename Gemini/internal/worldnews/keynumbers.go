package worldnews

func keyNumberFromQuote(q *quoteSnapshot, label, source, url, symbol string) KeyNumber {
	_, pct, pos := formatChange(q.Change, q.ChangePct)
	return KeyNumber{
		Label:           label,
		Value:           formatPrice(q.Symbol, q.Price),
		Change:          pct,
		IsPositive:      pos,
		Sparkline:       q.ChartPoints,
		SparklineLabels: q.ChartLabels,
		PriceTime:       q.PriceTime,
		Source:          source,
		URL:             url,
		Symbol:          symbol,
	}
}