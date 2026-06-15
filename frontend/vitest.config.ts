// ═══════════════════════════════════════════════════
// Vitest Config
// ═══════════════════════════════════════════════════
import { defineConfig } from 'vitest/config';
import { resolve } from 'path';

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    exclude: ['src/**/*.spec.ts', 'tests/**'],
    setupFiles: ['src/shared/testing/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.ts'],
      exclude: ['src/**/*.test.ts', 'src/**/*.d.ts', 'src/main.ts'],
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
});
