import { atom } from 'nanostores';

export const $sidebarOpen = atom<boolean>(false);

export function toggleSidebar() {
  $sidebarOpen.set(!$sidebarOpen.get());
}
export function setSidebarOpen(open: boolean) {
  $sidebarOpen.set(open);
}
