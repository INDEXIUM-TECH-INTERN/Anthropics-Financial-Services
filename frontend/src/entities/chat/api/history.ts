import { ApiClient } from '../../../shared/api/client';
import { normalizeMessageRole, type MessageSender } from '../../../shared/lib/message-role';
import type { HistoryResponse, HistoryMessage } from '../../../shared/api/types';

export async function fetchHistory(api: ApiClient, chatId: string): Promise<HistoryMessage[]> {
  const res: HistoryResponse = await api.getHistory(chatId);
  return res.history || [];
}

export interface GroupedMessage {
  role: MessageSender;
  content: string;
  metrics?: {
    token_in?: number;
    token_out?: number;
    ram_mb?: string;
    latency_ms?: number;
    cpu_load?: string;
  };
}

export function groupHistoryMessages(messages: HistoryMessage[]): GroupedMessage[] {
  const grouped: GroupedMessage[] = [];

  for (const msg of messages) {
    const content = msg.content?.trim();
    if (!content) continue;

    const role = normalizeMessageRole(msg.role);
    if (!role) continue;

    const last = grouped[grouped.length - 1];
    if (last && last.role === role) {
      last.content += '\n\n' + content;
      if (msg.latency_ms) {
        last.metrics = {
          token_in: msg.token_in,
          token_out: msg.token_out,
          ram_mb: msg.ram_mb,
          latency_ms: msg.latency_ms,
          cpu_load: msg.cpu_load,
        };
      }
    } else {
      grouped.push({
        role,
        content,
        metrics: msg.latency_ms
          ? {
              token_in: msg.token_in,
              token_out: msg.token_out,
              ram_mb: msg.ram_mb,
              latency_ms: msg.latency_ms,
              cpu_load: msg.cpu_load,
            }
          : undefined,
      });
    }
  }

  return grouped;
}