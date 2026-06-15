// ═══════════════════════════════════════════════════
// Test Helpers — DOM cleanup & utilities
// ═══════════════════════════════════════════════════

/**
 * Remove all children from document.body.
 * Call in afterEach to prevent test pollution.
 */
export function cleanup(): void {
  document.body.innerHTML = '';
}

/**
 * Create a detached container element for component testing.
 * Returns { container, cleanup } — call cleanup() when done.
 */
export function createContainer(): { container: HTMLElement; cleanup: () => void } {
  const container = document.createElement('div');
  document.body.appendChild(container);
  return {
    container,
    cleanup: () => container.remove(),
  };
}

/**
 * Create a mock Response object for fetch mocking.
 */
export function mockResponse(body: string, init: ResponseInit = {}): Response {
  return new Response(body, {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
}

/**
 * Create a mock SSE stream response for testing streaming.
 */
export function mockSSEStream(events: Array<Record<string, unknown>>): Response {
  const encoder = new TextEncoder();
  const stream = new ReadableStream({
    start(controller) {
      for (const event of events) {
        controller.enqueue(encoder.encode(`data: ${JSON.stringify(event)}\n\n`));
      }
      controller.close();
    },
  });
  return new Response(stream, {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  });
}
