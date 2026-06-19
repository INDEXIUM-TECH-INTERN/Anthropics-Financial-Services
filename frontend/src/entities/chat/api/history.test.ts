import { describe, it, expect } from 'vitest';
import { groupHistoryMessages } from './history';
import type { HistoryMessage } from '../../../shared/api/types';

describe('groupHistoryMessages', () => {
  it('should return empty array for empty input', () => {
    expect(groupHistoryMessages([])).toEqual([]);
  });

  it('should group consecutive user messages', () => {
    const messages: HistoryMessage[] = [
      { role: 'user', content: 'Hello' },
      { role: 'user', content: 'World' },
    ];
    const result = groupHistoryMessages(messages);
    expect(result.length).toBe(1);
    expect(result[0]?.role).toBe('user');
    expect(result[0]?.content).toBe('Hello\n\nWorld');
  });

  it('should group consecutive assistant messages and normalize role to bot', () => {
    const messages: HistoryMessage[] = [
      { role: 'assistant', content: 'Hello' },
      { role: 'assistant', content: 'World' },
    ];
    const result = groupHistoryMessages(messages);
    expect(result.length).toBe(1);
    expect(result[0]?.role).toBe('bot');
    expect(result[0]?.content).toBe('Hello\n\nWorld');
  });

  it('should not group different roles together', () => {
    const messages: HistoryMessage[] = [
      { role: 'user', content: 'Hi' },
      { role: 'assistant', content: 'Hello' },
    ];
    const result = groupHistoryMessages(messages);
    expect(result.length).toBe(2);
    expect(result[0]?.role).toBe('user');
    expect(result[1]?.role).toBe('bot');
  });

  it('should skip empty and system messages', () => {
    const messages: HistoryMessage[] = [
      { role: 'assistant', content: '   ' },
      { role: 'system', content: 'hidden bootstrap' },
      { role: 'user', content: 'Hi' },
    ];
    const result = groupHistoryMessages(messages);
    expect(result.length).toBe(1);
    expect(result[0]?.role).toBe('user');
  });

  it('should keep latest metrics when grouping', () => {
    const messages: HistoryMessage[] = [
      { role: 'assistant', content: 'First', latency_ms: 100, token_in: 10, token_out: 5, ram_mb: '100' },
      { role: 'assistant', content: 'Second', latency_ms: 200, token_in: 20, token_out: 10, ram_mb: '200' },
    ];
    const result = groupHistoryMessages(messages);
    expect(result.length).toBe(1);
    expect(result[0]?.metrics?.latency_ms).toBe(200);
    expect(result[0]?.metrics?.token_in).toBe(20);
  });
});
