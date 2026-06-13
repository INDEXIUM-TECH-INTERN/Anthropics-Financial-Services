import { test, expect } from '@playwright/test';

test.describe('Indexium Glass Chat — E2E', () => {
  test('1. Page loads without console errors', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    await page.waitForTimeout(1000);
    page.removeAllListeners('console');
    expect(errors, `Console errors:\n${errors.join('\n')}`).toHaveLength(0);
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

    try {
      await page.waitForFunction(
        () => document.documentElement.getAttribute('data-theme') === 'light',
        { timeout: 3000 }
      );
    } catch {
      // fallback
    }
    const afterFirst = await html.getAttribute('data-theme');
    expect(afterFirst).toBe('light');

    await toggle.click({ timeout: 10000 });
    try {
      await page.waitForFunction(
        () => document.documentElement.getAttribute('data-theme') === 'dark',
        { timeout: 3000 }
      );
    } catch {
      // fallback
    }
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
    await page.waitForTimeout(300);
  });
});
