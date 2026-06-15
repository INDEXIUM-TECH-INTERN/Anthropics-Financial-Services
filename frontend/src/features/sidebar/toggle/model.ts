import { atom } from 'nanostores';

export const $sidebarOpen = atom<boolean>(true);

export function toggleSidebar() {
  $sidebarOpen.set(!$sidebarOpen.get());
}
export function setSidebarOpen(open: boolean) {
  $sidebarOpen.set(open);
}
