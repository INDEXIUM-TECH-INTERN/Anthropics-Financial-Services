// ═══ World News Types ═══
// Type definitions for the World Finance News Dashboard (data from /api/world-news).

export interface StockInstrument {
  symbol: string;
  label: string;
  value: string;
  change: string;
  changePercent: string;
  isPositive: boolean;
  chartPoints: number[];
  chartLabels: string[];
  quoteUrl?: string;
}

export interface StockMarketData {
  indexName: string;
  value: string;
  change: string;
  changePercent: string;
  isPositive: boolean;
  chartPoints: number[];
  chartLabels: string[];
  instruments: StockInstrument[];
  moreInstruments?: StockInstrument[];
  thumbnail: string;
  highlights: string[];
  cnbcUrl: string;
  wsjUrl: string;
  reutersUrl: string;
  marketSource?: string;
  marketUrl?: string;
  marketSymbol?: string;
}

export interface OilMarketData {
  wtiPrice: string;
  wtiChange: string;
  wtiPercent: string;
  wtiPositive: boolean;
  brentPrice: string;
  brentChange: string;
  brentPercent: string;
  brentPositive: boolean;
  chartPointsWTI: number[];
  chartPointsBrent: number[];
  chartLabels: string[];
  thumbnail: string;
  highlights: string[];
  reutersUrl: string;
  nytimesUrl: string;
  wsjUrl: string;
  marketSource?: string;
  marketUrl?: string;
  marketSymbol?: string;
}

export interface GoldUSDData {
  goldPrice: string;
  goldChange: string;
  goldPercent: string;
  goldPositive: boolean;
  dxyPrice: string;
  dxyChange: string;
  dxyPercent: string;
  dxyPositive: boolean;
  chartPointsGold: number[];
  chartPointsDXY: number[];
  chartLabels: string[];
  highlights: string[];
  reutersUrl: string;
  marketSource?: string;
  marketUrl?: string;
  marketSymbol?: string;
}

export interface KeyNumberMetric {
  label: string;
  value: string;
  change: string;
  isPositive: boolean;
  sparkline: number[];
  sparklineLabels?: string[];
  priceTime?: string;
  source?: string;
  url?: string;
  symbol?: string;
}

export interface NewsArticle {
  title: string;
  summary: string;
  source: string;
  time: string;
  url?: string;
  thumbnail?: string;
  logo?: string;
}

export interface BreakingNews {
  time: string;
  source: string;
  content: string;
  url?: string;
  thumbnail?: string;
  logo?: string;
  isUrgent: boolean;
}

export interface WatchlistItem {
  event: string;
  time: string;
  importance: 'high' | 'medium' | 'low';
  source: string;
}

export interface QuickHighlight {
  text: string;
  url?: string;
  source?: string;
  logo?: string;
  time?: string;
}

export interface WorldNewsReport {
  date: string;
  quickHighlights: QuickHighlight[];
  keyNumbers: KeyNumberMetric[];
  stocks: StockMarketData;
  oil: OilMarketData;
  goldUsd: GoldUSDData;
  vtvIndexNews: NewsArticle[];
  vietnamFinanceNews: NewsArticle[];
  breakingNews: BreakingNews[];
  watchlist: WatchlistItem[];
  generatedAt?: string;
  dataSource?: string;
  digestWindow?: string;
  digestUntil?: string;
}

export interface WorldNewsDateOption {
  value: string;
  label: string;
  isToday: boolean;
}

export interface WorldNewsDatesResponse {
  dates: WorldNewsDateOption[];
  defaultDate: string;
  historyDays?: number;
}