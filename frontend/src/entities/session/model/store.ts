import { atom } from 'nanostores';
import type { ChatSession } from '../../../shared/api/types';

export const $sessions = atom<ChatSession[]>([]);
export function setSessions(s: ChatSession[]) {
  $sessions.set(s);
}
export function addSession(s: ChatSession) {
  $sessions.set([s, ...$sessions.get()]);
}
export function removeSession(id: string) {
  $sessions.set($sessions.get().filter((s) => s.id !== id));
}
export function updateSession(id: string, u: Partial<ChatSession>) {
  $sessions.set($sessions.get().map((s) => (s.id === id ? { ...s, ...u } : s)));
}
export function getSessionById(id: string) {
  return $sessions.get().find((s) => s.id === id);
}
