import { test, expect } from '@playwright/test';

test('debug: capture full DOM', async ({ page }) => {
  const logs: string[] = [];
  page.on('console', (msg) => {
    logs.push(`[${msg.type()}] ${msg.text()}`);
  });
  page.on('pageerror', (err) => {
    logs.push(`[PAGE ERROR] ${err.message}`);
  });

  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(500);

  const content = await page.content();
  console.log('=== PAGE SIZE:', content.length, 'chars');
  console.log('=== FIRST 2000:', content.substring(0, 2000));

  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(3000);

  const content2 = await page.content();
  console.log('=== AFTER NETWORKIDLE SIZE:', content2.length, 'chars');
  console.log('=== CONSOLE:', JSON.stringify(logs, null, 2));

  await page.screenshot({ path: 'test-results/debug-screenshot2.png', fullPage: true });
});
