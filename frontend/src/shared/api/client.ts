// ═══════════════════════════════════════════════════
// Shared — API Client
// Replaces src/services/api.ts
// ═══════════════════════════════════════════════════

import type {
  ChatSession,
  ChatSessionsResponse,
  AttachmentPayload,
  TokenMetrics,
  HistoryResponse,
  ConfigKeysRequest,
  ConfigKeysResponse,
  SSEEventData,
} from './types';
import type { WorldNewsReport, WorldNewsDatesResponse } from './mock-news';
import { retryWithBackoff } from '../lib/markdown';

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly statusCode?: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export class ApiClient {
  constructor(public baseUrl: string) {}

  async getSessions(): Promise<ChatSession[]> {
    const res = await retryWithBackoff(() => fetch(`${this.baseUrl}/api/chats`), { maxRetries: 3, initialDelay: 1000 });
    if (!res.ok) throw new ApiError('Failed to fetch sessions', res.status);
    const data: ChatSessionsResponse = await res.json();
    return data.chats || [];
  }

  async createSession(title: string): Promise<ChatSession> {
    const res = await retryWithBackoff(
      () =>
        fetch(`${this.baseUrl}/api/chats`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ title }),
        }),
      { maxRetries: 3, initialDelay: 1000 },
    );
    if (!res.ok) throw new ApiError('Failed to create session', res.status);
    return res.json() as Promise<ChatSession>;
  }

  async deleteSession(chatId: string): Promise<void> {
    const res = await retryWithBackoff(
      () => fetch(`${this.baseUrl}/api/chats?chat_id=${encodeURIComponent(chatId)}`, { method: 'DELETE' }),
      { maxRetries: 3, initialDelay: 1000 },
    );
    if (!res.ok) throw new ApiError('Failed to delete session', res.status);
  }

  async getHistory(chatId: string): Promise<HistoryResponse> {
    const res = await retryWithBackoff(
      () => fetch(`${this.baseUrl}/api/history?chat_id=${encodeURIComponent(chatId)}`),
      { maxRetries: 3, initialDelay: 1000 },
    );
    if (!res.ok) throw new ApiError('Failed to fetch history', res.status);
    return res.json() as Promise<HistoryResponse>;
  }

  async saveConfigKeys(keys: string[]): Promise<ConfigKeysResponse> {
    const res = await retryWithBackoff(
      () =>
        fetch(`${this.baseUrl}/api/config/keys`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ keys } satisfies ConfigKeysRequest),
        }),
      { maxRetries: 3, initialDelay: 1000 },
    );
    if (!res.ok) throw new ApiError('Failed to save config keys', res.status);
    return res.json() as Promise<ConfigKeysResponse>;
  }

  async getWorldNews(date?: string): Promise<WorldNewsReport> {
    const qs = date ? `?date=${encodeURIComponent(date)}` : '';
    const res = await retryWithBackoff(() => fetch(`${this.baseUrl}/api/world-news${qs}`), {
      maxRetries: 2,
      initialDelay: 800,
    });
    if (!res.ok) {
      const errBody = await res.json().catch(() => ({}));
      const msg = (errBody as { error?: string }).error || 'Failed to fetch world news';
      throw new ApiError(msg, res.status);
    }
    return res.json() as Promise<WorldNewsReport>;
  }

  async getWorldNewsDates(): Promise<WorldNewsDatesResponse> {
    const res = await retryWithBackoff(() => fetch(`${this.baseUrl}/api/world-news/dates`), {
      maxRetries: 2,
      initialDelay: 800,
    });
    if (!res.ok) throw new ApiError('Failed to fetch world news dates', res.status);
    return res.json() as Promise<WorldNewsDatesResponse>;
  }

  async streamChat(
    message: string,
    chatId: string | undefined,
    attachments: AttachmentPayload[],
    signal: AbortSignal,
    onToken: (t: string) => void,
    onDone: (m: TokenMetrics) => void,
    onError: (e: string) => void,
  ): Promise<void> {
    const body: Record<string, unknown> = { message };
    if (chatId) body.chat_id = chatId;
    if (attachments.length > 0) body.attachments = attachments;
    const response = await fetch(`${this.baseUrl}/api/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    });
    if (!response.ok) throw new ApiError(`HTTP ${response.status}`, response.status);
    const reader = response.body?.getReader();
    if (!reader) throw new Error('No response body');
    const decoder = new TextDecoder();
    let buffer = '';
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() ?? '';
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed.startsWith('data: ')) continue;
        let data: SSEEventData;
        try {
          data = JSON.parse(trimmed.substring(6));
        } catch {
          continue;
        }
        if (data.type === 'token' && data.text) onToken(data.text);
        if (data.type === 'done' && data.metrics) onDone(data.metrics);
        if (data.type === 'error' && data.error) onError(data.error);
      }
    }
  }
}
