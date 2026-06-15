// ═══ App Configuration ═══

export const APP_CONFIG = {
  name: 'Indexium Glass Chat',
  version: '2.0.0',
  defaultTheme: 'dark' as const,
  defaultBackend: 'gemini' as const,
  maxAttachmentSize: 10 * 1024 * 1024, // 10MB
  supportedFileTypes: ['image/*', 'application/pdf', '.txt', '.csv', '.json'],
  toastDuration: 3000,
  scrollThreshold: 200,
  maxTextareaRows: 8,
} as const;

export type AppConfig = typeof APP_CONFIG;
