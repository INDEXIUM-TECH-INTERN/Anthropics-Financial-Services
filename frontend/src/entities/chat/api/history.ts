import { ApiClient } from '../../../shared/api/client';
import { normalizeMessageRole, type MessageSender } from '../../../shared/lib/message-role';
import type { HistoryResponse, HistoryMessage, AttachmentPayload } from '../../../shared/api/types';

export async function fetchHistory(api: ApiClient, chatId: string): Promise<HistoryMessage[]> {
  const res: HistoryResponse = await api.getHistory(chatId);
  return res.history || [];
}

export interface GroupedMessage {
  role: MessageSender;
  content: string;
  attachments?: AttachmentPayload[];
  metrics?: {
    token_in?: number;
    token_out?: number;
    ram_mb?: string;
    latency_ms?: number;
    cpu_load?: string;
  };
}

function isInternalHistoryMessage(msg: HistoryMessage): boolean {
  if (msg.internal) return true;
  const content = (msg.content || '').toLowerCase();
  return (
    content.includes('anthropic agent configuration') ||
    content.includes('system prompt (from agents/') ||
    content.includes('skill markdown (')
  );
}

function stripLeakedSkillPrefix(content: string): string {
  const marker = 'Chào bạn';
  const idx = content.indexOf(marker);
  if (idx <= 0) return content;
  const head = content.slice(0, idx);
  if (/### Step \d+:|## Important Notes/i.test(head)) {
    return content.slice(idx).trim();
  }
  return content;
}

export function groupHistoryMessages(messages: HistoryMessage[]): GroupedMessage[] {
  const grouped: GroupedMessage[] = [];

  for (const msg of messages) {
    if (isInternalHistoryMessage(msg)) continue;
    let content = msg.content?.trim() || '';
    if (!content && !msg.attachments?.length) continue;
    if (msg.role === 'assistant') {
      content = stripLeakedSkillPrefix(content);
    }

    const role = normalizeMessageRole(msg.role);
    if (!role) continue;

    const last = grouped[grouped.length - 1];
    if (last && last.role === role) {
      last.content += '\n\n' + content;
      if (msg.attachments?.length) {
        last.attachments = [...(last.attachments || []), ...msg.attachments];
      }
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
        attachments: msg.attachments?.length ? [...msg.attachments] : undefined,
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