// ═══ Connection Status Indicator — Shows backend connectivity ═══

export type ConnStatus = 'connected' | 'disconnected' | 'reconnecting';

export interface ConnStatusCallbacks {
  onReconnect?: () => void;
}

export function createConnectionStatus(
  container: HTMLElement,
  callbacks: ConnStatusCallbacks = {},
): { setStatus: (s: ConnStatus) => void; destroy: () => void } {
  const el = document.createElement('div');
  el.className = 'conn-status-indicator';
  el.setAttribute('role', 'status');
  el.setAttribute('aria-live', 'polite');

  const dot = document.createElement('span');
  dot.className = 'conn-status-dot';

  const label = document.createElement('span');
  label.className = 'conn-status-label';

  const retryBtn = document.createElement('button');
  retryBtn.className = 'conn-status-retry-btn';
  retryBtn.textContent = 'Thử lại';
  retryBtn.title = 'Kết nối lại';
  retryBtn.addEventListener('click', () => {
    if (callbacks.onReconnect) callbacks.onReconnect();
  });

  el.appendChild(dot);
  el.appendChild(label);
  el.appendChild(retryBtn);
  container.appendChild(el);

  let currentStatus: ConnStatus = 'disconnected';

  const statusConfig: Record<ConnStatus, { label: string; className: string }> = {
    connected: { label: 'Đã kết nối', className: 'conn-connected' },
    disconnected: { label: 'Mất kết nối', className: 'conn-disconnected' },
    reconnecting: { label: 'Đang kết nối lại…', className: 'conn-reconnecting' },
  };

  function setStatus(s: ConnStatus) {
    if (s === currentStatus && s !== 'reconnecting') return;
    currentStatus = s;
    const cfg = statusConfig[s];
    label.textContent = cfg.label;
    el.className = `conn-status-indicator ${cfg.className}`;
    retryBtn.style.display = s === 'disconnected' ? '' : 'none';
  }

  function destroy() {
    el.remove();
  }

  return { setStatus, destroy };
}
