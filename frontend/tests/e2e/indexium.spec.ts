import { test, expect } from '@playwright/test';

test.describe('Indexium Glass Chat — E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate and wait for the app to be ready (not networkidle — SSE keeps connections open)
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    // Wait for the welcome state to confirm the app has rendered
    await page.waitForSelector('.welcome-state', { state: 'visible', timeout: 15000 });
  });

  test('1. Page loads without critical console errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') errors.push(msg.text());
    });

    // Wait for the app to be fully rendered (welcome state visible = app ready)
    await expect(page.locator('.welcome-state')).toBeVisible({ timeout: 15000 });

    // Filter: allow expected backend connection errors when backend is offline
    // and known CDN/CSP warnings that are not app bugs
    const critical = errors.filter(e =>
      !e.includes('localhost:8080') &&       // backend offline — expected
      !e.includes('ERR_CONNECTION_REFUSED') && // backend offline — expected
      !e.includes('favicon.ico') &&            // favicon — minor, not critical
      !e.includes('integrity')                 // CDN SRI — transient CDN issue
    );

    if (critical.length > 0) {
      console.log('Critical console errors:', critical);
    }
    expect(critical).toHaveLength(0);
  });

  test('2. Welcome screen shows with title and quick-reply chips', async ({ page }) => {
    const title = page.locator('.welcome-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText(/Tôi có thể giúp gì cho bạn?/i);

    const desc = page.locator('.welcome-desc');
    await expect(desc).toBeVisible();

    const chips = page.locator('.welcome-chip');
    await expect(chips).toHaveCount(3);
    await expect(chips.nth(0)).toBeVisible();
    await expect(chips.nth(1)).toBeVisible();
    await expect(chips.nth(2)).toBeVisible();
  });

  test('3. Chat input is visible and accepts text', async ({ page }) => {
    const input = page.locator('#chat-input');
    await expect(input).toBeVisible();
    await expect(input).toBeEditable();

    await input.click();
    await input.fill('Xin chào');
    await expect(input).toHaveValue('Xin chào');

    const sendBtn = page.locator('#send-btn');
    await expect(sendBtn).toBeVisible();
  });

  test('4. Theme toggle switches dark ↔ light', async ({ page }) => {
    const html = page.locator('html');
    await expect(html).toHaveAttribute('data-theme', 'dark');

    const toggle = page.locator('#theme-toggle');
    await expect(toggle).toBeVisible();
    await toggle.click({ timeout: 10000 });

    await page.waitForFunction(
      () => document.documentElement.getAttribute('data-theme') === 'light',
      { timeout: 5000 }
    ).catch(() => { /* theme may already be light */ });
    expect(await page.evaluate(() => document.documentElement.getAttribute('data-theme'))).toBe('light');

    await toggle.click({ timeout: 10000 });
    await page.waitForFunction(
      () => document.documentElement.getAttribute('data-theme') === 'dark',
      { timeout: 5000 }
    ).catch(() => { /* theme may already be dark */ });
    await expect(html).toHaveAttribute('data-theme', 'dark');
  });

  test('5. Settings modal opens and shows API key inputs', async ({ page }) => {
    const trigger = page.locator('#settings-trigger');
    await expect(trigger).toBeVisible();
    await trigger.click();

    const modal = page.locator('#settings-modal');
    await expect(modal).toBeVisible();

    await expect(page.locator('#gemini-key-input')).toBeVisible();
    await expect(page.locator('#or-keys-input')).toBeVisible();
    await expect(page.locator('#save-settings-btn')).toBeVisible();

    await page.locator('#close-settings').click();
    await expect(modal).toHaveClass(/hidden/);
  });

  test('6. Sidebar toggle shows/hides conversation list', async ({ page }) => {
    const sidebar = page.locator('#conversations-sidebar');
    const toggle = page.locator('#toggle-conversations');

    await expect(toggle).toBeVisible();
    await expect(sidebar).not.toHaveClass(/collapsed/);

    await toggle.click();
    await expect(sidebar).toHaveClass(/collapsed/);
  });

  test('7. Shortcut panel opens with keyboard shortcut', async ({ page }) => {
    // Initially hidden
    const panel = page.locator('#shortcuts-panel');
    await expect(panel).toHaveClass(/hidden/);

    // Press Ctrl+?
    await page.keyboard.press('Control+?');
    await expect(panel).not.toHaveClass(/hidden/);

    // Close with Escape
    await page.keyboard.press('Escape');
    await expect(panel).toHaveClass(/hidden/);
  });
});
