import { describe, it, expect, vi } from 'vitest';
import { fetchSessions, createNewSession, removeSession } from './crud';
import { ApiClient } from '../../../shared/api/client';

describe('session CRUD', () => {
  it('should fetch sessions via API', async () => {
    const mockApi = {
      getSessions: vi.fn().mockResolvedValue([{ id: '1', title: 'Test' }]),
    } as unknown as ApiClient;

    const result = await fetchSessions(mockApi);
    expect(mockApi.getSessions).toHaveBeenCalled();
    expect(result).toHaveLength(1);
    expect(result[0]?.id).toBe('1');
  });

  it('should create session via API', async () => {
    const mockApi = {
      createSession: vi.fn().mockResolvedValue({ id: '2', title: 'New Chat' }),
    } as unknown as ApiClient;

    const result = await createNewSession(mockApi, 'New Chat');
    expect(mockApi.createSession).toHaveBeenCalledWith('New Chat');
    expect(result.id).toBe('2');
  });

  it('should delete session via API', async () => {
    const mockApi = {
      deleteSession: vi.fn().mockResolvedValue(undefined),
    } as unknown as ApiClient;

    await removeSession(mockApi, 'chat-123');
    expect(mockApi.deleteSession).toHaveBeenCalledWith('chat-123');
  });
});
