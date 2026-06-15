// ═══ App Entry Point ═══
// Replaces src/main.ts — bootstraps the FSD architecture.

import { createChatPage } from '../pages/chat/page';
import { APP_CONFIG } from './config';
import '../styles/main.css';

async function bootstrap() {
  console.log(`[${APP_CONFIG.name} v${APP_CONFIG.version}] Starting...`);

  const page = createChatPage();
  await page.init();

  console.log(`[${APP_CONFIG.name}] Ready.`);
}

bootstrap().catch((err) => {
  console.error('[App] Failed to start:', err);
});
