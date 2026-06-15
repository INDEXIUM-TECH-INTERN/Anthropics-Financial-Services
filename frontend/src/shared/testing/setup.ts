// ═══════════════════════════════════════════════════
// Vitest Setup — Global test configuration
// ═══════════════════════════════════════════════════
import { afterEach } from 'vitest';
import { cleanup } from './helpers';

// Clean DOM after each test
afterEach(() => {
  cleanup();
});
