export interface SessionEntityState {
  sessions: import('../../../shared/api/types').ChatSession[];
  activeSessionId: string | null;
  isLoading: boolean;
}
