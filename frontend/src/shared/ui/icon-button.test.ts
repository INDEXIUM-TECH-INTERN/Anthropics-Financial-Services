import { describe, it, expect, vi } from 'vitest';
import { createIconButton } from './icon-button';

describe('createIconButton', () => {
  it('should create a button element', () => {
    const btn = createIconButton({ icon: '🔍', title: 'Search' });
    expect(btn.tagName).toBe('BUTTON');
  });

  it('should set title', () => {
    const btn = createIconButton({ icon: '🔍', title: 'Search' });
    expect(btn.title).toBe('Search');
  });

  it('should set icon HTML', () => {
    const btn = createIconButton({ icon: '<svg>...</svg>', title: 'Icon' });
    expect(btn.innerHTML).toBe('<svg>...</svg>');
  });

  it('should add base class', () => {
    const btn = createIconButton({ icon: 'X', title: 'Close' });
    expect(btn.classList.contains('icon-btn')).toBe(true);
  });

  it('should add custom class', () => {
    const btn = createIconButton({ icon: 'X', title: 'Close', className: 'custom-btn' });
    expect(btn.classList.contains('custom-btn')).toBe(true);
    expect(btn.classList.contains('icon-btn')).toBe(true);
  });

  it('should attach click handler', () => {
    const handler = vi.fn();
    const btn = createIconButton({ icon: 'X', title: 'Close', onClick: handler });
    btn.click();
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it('should not throw without onClick', () => {
    const btn = createIconButton({ icon: 'X', title: 'Close' });
    expect(() => btn.click()).not.toThrow();
  });
});
