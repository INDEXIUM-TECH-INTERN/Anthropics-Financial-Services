// ═══════════════════════════════════════════════════
// Shared — SSE Connection Manager (EventSource)
// Replaces src/services/sse-manager.ts
// ═══════════════════════════════════════════════════

export interface SSECallbacks {
  onMessage: (data: Record<string, unknown>) => void;
  onError: (message: string) => void;
}

export class SSEManager {
  private eventSource: EventSource | null = null;
  private currentBaseUrl: string | null = null;

  connect(baseUrl: string, callbacks: SSECallbacks): void {
    if (this.eventSource && this.currentBaseUrl === baseUrl) return;
    this.disconnect();
    this.currentBaseUrl = baseUrl;
    this.eventSource = new EventSource(`${baseUrl}/api/events`);
    this.eventSource.onmessage = (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as Record<string, unknown>;
        callbacks.onMessage(data);
      } catch {
        // Ignore malformed frames
      }
    };
    this.eventSource.onerror = () => {
      callbacks.onError('Mất kết nối SSE.');
    };
  }

  disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
      this.currentBaseUrl = null;
    }
  }
}
