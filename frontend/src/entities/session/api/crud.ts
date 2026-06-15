import { ApiClient } from '../../../shared/api/client';
import type { ChatSession } from '../../../shared/api/types';

export async function fetchSessions(api: ApiClient): Promise<ChatSession[]> {
  return api.getSessions();
}

export async function createNewSession(api: ApiClient, title: string): Promise<ChatSession> {
  return api.createSession(title);
}

export async function removeSession(api: ApiClient, chatId: string): Promise<void> {
  return api.deleteSession(chatId);
}
