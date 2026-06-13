export interface ChatSession {
  id: string;
  title: string;
  updated_at?: string;
  created_at?: string;
}

export interface ChatSessionsResponse {
  chats: ChatSession[];
}

export interface AttachmentPayload {
  name: string;
  type: string;
  data: string;
}

export interface TokenMetrics {
  token_in: number;
  token_out: number;
  ram_mb: string;
  latency_ms?: number;
  cpu_load?: string;
}

export interface HistoryMessage {
  role: string;
  content: string;
  latency_ms?: number;
  token_in?: number;
  token_out?: number;
  ram_mb?: string;
  cpu_load?: string;
}

export interface HistoryResponse {
  history: HistoryMessage[];
}

export interface ConfigKeysRequest {
  keys: string[];
}

export interface ConfigKeysResponse {
  status: string;
  message?: string;
}

// ═══ Data Card Types ═══

export type DataCardType = 'chart' | 'table' | 'metric' | 'comparison' | 'entity';

export interface ChartData {
  chartType: 'line' | 'bar' | 'bar-horizontal' | 'pie';
  labels: string[];
  datasets: { label: string; values: number[]; color?: string }[];
}

export interface TableData {
  headers: string[];
  rows: string[][];
  align?: ('left' | 'center' | 'right')[];
}

export interface MetricData {
  label: string;
  value: string;
  change?: string;
  changeType?: 'positive' | 'negative' | 'neutral';
  unit?: string;
}

export interface ComparisonItem {
  label: string;
  value: number;
  displayValue?: string;
  color?: string;
}

export interface ComparisonData {
  items: ComparisonItem[];
  maxValue?: number;
}

export interface EntityTag {
  name: string;
  type: 'company' | 'index' | 'metric' | 'sector';
}

export interface DataCard {
  type: DataCardType;
  title?: string;
  subtitle?: string;
  data: ChartData | TableData | MetricData | ComparisonData | EntityTag[];
}

export interface SSEEventData {
  type: string;
  text?: string;
  payload?: string;
  error?: string;
  metrics?: TokenMetrics;
  metadata?: Record<string, string>;
}
