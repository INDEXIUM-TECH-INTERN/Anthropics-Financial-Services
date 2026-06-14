export type ConnectionStatus = 'connected' | 'disconnected' | 'reconnecting';

export interface ConnectionStatusOptions {
  onStatusChange: (status: ConnectionStatus) => void;
  reconnectDelay?: number;
  maxRetries?: number;
}

export function initConnectionStatus(options: ConnectionStatusOptions): {
  setStatus: (s: ConnectionStatus) => void;
  destroy: () => void;
} {
  let currentStatus: ConnectionStatus = 'disconnected';

  const setStatus = (s: ConnectionStatus) => {
    if (s !== currentStatus) {
      currentStatus = s;
      options.onStatusChange(s);
    }
  };

  const destroy = () => {
    // No timers to clean up — kept for API compatibility
  };

  return { setStatus, destroy };
}
