import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { showToast } from './toast';

describe('showToast', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('should create a toast container', () => {
    showToast({ message: 'Hello' });
    const container = document.querySelector('.toast-container');
    expect(container).toBeTruthy();
  });

  it('should create a toast element with message', () => {
    showToast({ message: 'Test message' });
    const toast = document.querySelector('.toast');
    expect(toast).toBeTruthy();
    expect(toast?.textContent).toContain('Test message');
  });

  it('should apply correct type class', () => {
    showToast({ message: 'Error!', type: 'error' });
    const toast = document.querySelector('.toast-error');
    expect(toast).toBeTruthy();
  });

  it('should default to info type', () => {
    showToast({ message: 'Info' });
    const toast = document.querySelector('.toast-info');
    expect(toast).toBeTruthy();
  });

  it('should set aria attributes on container', () => {
    showToast({ message: 'Hello' });
    const container = document.querySelector('.toast-container');
    expect(container?.getAttribute('aria-live')).toBe('polite');
    expect(container?.getAttribute('aria-atomic')).toBe('true');
  });

  it('should auto-dismiss after duration', async () => {
    vi.useFakeTimers();
    showToast({ message: 'Gone soon', duration: 100 });
    expect(document.querySelector('.toast')).toBeTruthy();
    vi.advanceTimersByTime(100);
    // After duration, toast gets toast-hiding class
    // After 300ms more it's removed
    vi.advanceTimersByTime(300);
    expect(document.querySelector('.toast')).toBeFalsy();
    vi.useRealTimers();
  });
});
