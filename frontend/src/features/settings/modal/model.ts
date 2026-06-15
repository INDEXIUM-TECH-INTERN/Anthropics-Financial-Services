import { atom } from 'nanostores';

export const $settingsOpen = atom<boolean>(false);
export const $apiKeys = atom<string>('');

export function openSettings() {
  $settingsOpen.set(true);
}
export function closeSettings() {
  $settingsOpen.set(false);
}
export function setApiKeys(keys: string) {
  $apiKeys.set(keys);
}
