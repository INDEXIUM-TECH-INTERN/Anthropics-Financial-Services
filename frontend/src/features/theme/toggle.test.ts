import { describe, it, expect } from 'vitest';
import { $theme, toggleTheme, setTheme } from './toggle';

describe('theme toggle', () => {
  it('should default to light', () => {
    expect($theme.get()).toBe('light');
  });

  it('should toggle theme', () => {
    toggleTheme();
    expect($theme.get()).toBe('dark');
    toggleTheme();
    expect($theme.get()).toBe('light');
  });

  it('should set theme explicitly', () => {
    setTheme('dark');
    expect($theme.get()).toBe('dark');
    setTheme('light');
    expect($theme.get()).toBe('light');
  });

  it('should update document element data attribute', () => {
    setTheme('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    setTheme('light');
    expect(document.documentElement.dataset.theme).toBe('light');
  });
});
