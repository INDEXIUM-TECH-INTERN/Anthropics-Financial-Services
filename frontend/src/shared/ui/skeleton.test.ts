import { describe, it, expect } from 'vitest';
import { createMessageSkeleton } from './skeleton';

describe('createMessageSkeleton', () => {
  it('should create a skeleton element', () => {
    const el = createMessageSkeleton();
    expect(el.className).toBe('skeleton-loading');
  });

  it('should contain skeleton message structure', () => {
    const el = createMessageSkeleton();
    expect(el.querySelector('.skeleton-message')).toBeTruthy();
    expect(el.querySelector('.skeleton-avatar')).toBeTruthy();
    expect(el.querySelector('.skeleton-body')).toBeTruthy();
  });

  it('should have skeleton lines', () => {
    const el = createMessageSkeleton();
    const lines = el.querySelectorAll('.skeleton-line');
    expect(lines.length).toBe(2);
    expect(lines[0]?.classList.contains('short')).toBe(true);
  });
});
