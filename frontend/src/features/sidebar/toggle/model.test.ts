import { describe, it, expect } from 'vitest';
import { $sidebarOpen, toggleSidebar, setSidebarOpen } from './model';

describe('sidebar toggle model', () => {
  it('should default to open', () => {
    expect($sidebarOpen.get()).toBe(true);
  });

  it('should toggle sidebar state', () => {
    toggleSidebar();
    expect($sidebarOpen.get()).toBe(false);
    toggleSidebar();
    expect($sidebarOpen.get()).toBe(true);
  });

  it('should set sidebar open explicitly', () => {
    setSidebarOpen(false);
    expect($sidebarOpen.get()).toBe(false);
    setSidebarOpen(true);
    expect($sidebarOpen.get()).toBe(true);
  });
});
