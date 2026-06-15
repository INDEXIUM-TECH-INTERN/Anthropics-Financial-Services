import { atom } from 'nanostores';

export type ThemeMode = 'light' | 'dark';

export const $theme = atom<ThemeMode>('light');

export function toggleTheme() {
  const next = $theme.get() === 'light' ? 'dark' : 'light';
  $theme.set(next);
  document.documentElement.dataset.theme = next;
}

export function setTheme(mode: ThemeMode) {
  $theme.set(mode);
  document.documentElement.dataset.theme = mode;
}
