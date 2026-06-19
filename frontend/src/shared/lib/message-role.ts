export type MessageSender = 'user' | 'bot';

/** Map API/history roles to UI sender classes. */
export function normalizeMessageRole(role: string): MessageSender | null {
  if (role === 'user') return 'user';
  if (role === 'assistant' || role === 'bot' || role === 'model') return 'bot';
  return null;
}