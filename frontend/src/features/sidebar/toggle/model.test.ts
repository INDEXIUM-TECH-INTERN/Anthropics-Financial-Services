import { describe, it, expect, beforeEach } from 'vitest';
import { $sidebarOpen, toggleSidebar, setSidebarOpen } from './model';

describe('sidebar toggle model', () => {
  beforeEach(() => {
    $sidebarOpen.set(false);
  });

  it('should default to closed', () => {
    expect($sidebarOpen.get()).toBe(false);
  });

  it('should toggle sidebar state', () => {
    toggleSidebar();
    expect($sidebarOpen.get()).toBe(true);
    toggleSidebar();
    expect($sidebarOpen.get()).toBe(false);
  });

  it('should set sidebar open explicitly', () => {
    setSidebarOpen(true);
    expect($sidebarOpen.get()).toBe(true);
    setSidebarOpen(false);
    expect($sidebarOpen.get()).toBe(false);
  });
});