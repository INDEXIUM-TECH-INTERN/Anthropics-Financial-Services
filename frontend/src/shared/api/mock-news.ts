// ═══ World News Types ═══
// Type definitions for the World Finance News Dashboard (data from /api/world-news).

export interface StockMarketData {
  indexName: string;
  value: string;
  change: string;
  changePercent: string;
  isPositive: boolean;
  chartPoints: number[];
  chartLabels: string[];
  thumbnail: string;
  highlights: string[];
  cnbcUrl: string;
  wsjUrl: string;
  reutersUrl: string;
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
}

export interface NewsArticle {
  title: string;
  summary: string;
  source: string;
  time: string;
  url?: string;
  logo?: string;
}

export interface BreakingNews {
  time: string;
  source: string;
  content: string;
  url?: string;
  isUrgent: boolean;
}

export interface WatchlistItem {
  event: string;
  time: string;
  importance: 'high' | 'medium' | 'low';
  source: string;
}

export interface WorldNewsReport {
  date: string;
  quickHighlights: string[];
  keyNumbers: {
    label: string;
    value: string;
    change: string;
    isPositive: boolean;
    sparkline: number[];
  }[];
  stocks: StockMarketData;
  oil: OilMarketData;
  goldUsd: GoldUSDData;
  vtvIndexNews: NewsArticle[];
  vietnamFinanceNews: NewsArticle[];
  breakingNews: BreakingNews[];
  watchlist: WatchlistItem[];
  generatedAt?: string;
  dataSource?: string;
}

export interface WorldNewsDateOption {
  value: string;
  label: string;
  isToday: boolean;
}

export interface WorldNewsDatesResponse {
  dates: WorldNewsDateOption[];
  defaultDate: string;
}