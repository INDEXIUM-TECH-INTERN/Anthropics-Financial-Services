declare global {
  interface Window {
    __INDEXIUM_API_BASE__?: string;
  }
}

export function getApiBaseUrl(): string {
  const configured = window.__INDEXIUM_API_BASE__?.trim();
  if (configured) return configured.replace(/\/$/, '');
  return window.location.origin;
}