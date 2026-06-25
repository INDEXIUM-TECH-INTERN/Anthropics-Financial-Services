package worldnews

// WorldNewsReport mirrors the frontend dashboard schema.
type WorldNewsReport struct {
	Date                string       `json:"date"`
	QuickHighlights     []string     `json:"quickHighlights"`
	KeyNumbers          []KeyNumber  `json:"keyNumbers"`
	Stocks              StockSection `json:"stocks"`
	Oil                 OilSection   `json:"oil"`
	GoldUsd             GoldUSDSection `json:"goldUsd"`
	VTVIndexNews        []NewsArticle `json:"vtvIndexNews"`
	VietnamFinanceNews  []NewsArticle `json:"vietnamFinanceNews"`
	BreakingNews        []BreakingNews `json:"breakingNews"`
	Watchlist           []WatchlistItem `json:"watchlist"`
	GeneratedAt         string       `json:"generatedAt,omitempty"`
	DataSource          string       `json:"dataSource,omitempty"`
}

type KeyNumber struct {
	Label      string    `json:"label"`
	Value      string    `json:"value"`
	Change     string    `json:"change"`
	IsPositive bool      `json:"isPositive"`
	Sparkline  []float64 `json:"sparkline"`
	Source     string    `json:"source,omitempty"`
	URL        string    `json:"url,omitempty"`
	Symbol     string    `json:"symbol,omitempty"`
}

type StockSection struct {
	IndexName     string   `json:"indexName"`
	Value         string   `json:"value"`
	Change        string   `json:"change"`
	ChangePercent string   `json:"changePercent"`
	IsPositive    bool     `json:"isPositive"`
	ChartPoints   []float64 `json:"chartPoints"`
	ChartLabels   []string `json:"chartLabels"`
	Thumbnail     string   `json:"thumbnail"`
	Highlights    []string `json:"highlights"`
	CNBCUrl       string   `json:"cnbcUrl"`
	WSJUrl        string   `json:"wsjUrl"`
	ReutersUrl    string   `json:"reutersUrl"`
	MarketSource  string   `json:"marketSource,omitempty"`
	MarketURL     string   `json:"marketUrl,omitempty"`
	MarketSymbol  string   `json:"marketSymbol,omitempty"`
}

type OilSection struct {
	WTIPrice         string    `json:"wtiPrice"`
	WTIChange        string    `json:"wtiChange"`
	WTIPercent       string    `json:"wtiPercent"`
	WTIPositive      bool      `json:"wtiPositive"`
	BrentPrice       string    `json:"brentPrice"`
	BrentChange      string    `json:"brentChange"`
	BrentPercent     string    `json:"brentPercent"`
	BrentPositive    bool      `json:"brentPositive"`
	ChartPointsWTI   []float64 `json:"chartPointsWTI"`
	ChartPointsBrent []float64 `json:"chartPointsBrent"`
	ChartLabels      []string  `json:"chartLabels"`
	Thumbnail        string    `json:"thumbnail"`
	Highlights       []string  `json:"highlights"`
	ReutersUrl       string    `json:"reutersUrl"`
	NYTimesUrl       string    `json:"nytimesUrl"`
	WSJUrl           string    `json:"wsjUrl"`
	MarketSource     string    `json:"marketSource,omitempty"`
	MarketURL        string    `json:"marketUrl,omitempty"`
	MarketSymbol     string    `json:"marketSymbol,omitempty"`
}

type GoldUSDSection struct {
	GoldPrice         string    `json:"goldPrice"`
	GoldChange        string    `json:"goldChange"`
	GoldPercent       string    `json:"goldPercent"`
	GoldPositive      bool      `json:"goldPositive"`
	DXYPrice          string    `json:"dxyPrice"`
	DXYChange         string    `json:"dxyChange"`
	DXYPercent        string    `json:"dxyPercent"`
	DXYPositive       bool      `json:"dxyPositive"`
	ChartPointsGold   []float64 `json:"chartPointsGold"`
	ChartPointsDXY    []float64 `json:"chartPointsDXY"`
	ChartLabels       []string  `json:"chartLabels"`
	Highlights        []string  `json:"highlights"`
	ReutersUrl        string    `json:"reutersUrl"`
	MarketSource      string    `json:"marketSource,omitempty"`
	MarketURL         string    `json:"marketUrl,omitempty"`
	MarketSymbol      string    `json:"marketSymbol,omitempty"`
}

type NewsArticle struct {
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Source    string `json:"source"`
	Time      string `json:"time"`
	URL       string `json:"url,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Logo      string `json:"logo,omitempty"`
}

type BreakingNews struct {
	Time      string `json:"time"`
	Source    string `json:"source"`
	Content   string `json:"content"`
	URL       string `json:"url,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Logo      string `json:"logo,omitempty"`
	IsUrgent  bool   `json:"isUrgent"`
}

type WatchlistItem struct {
	Event      string `json:"event"`
	Time       string `json:"time"`
	Importance string `json:"importance"`
	Source     string `json:"source"`
}

type DatesResponse struct {
	Dates       []DateOption `json:"dates"`
	DefaultDate string       `json:"defaultDate"`
	HistoryDays int          `json:"historyDays"`
}

type DateOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	IsToday bool   `json:"isToday"`
}