// ═══ Features Layer ═══

// Chat — Send
export {
  $inputMessage,
  $isSending,
  $attachments,
  setInputMessage,
  setSending,
  setAttachments,
  clearInput,
} from './chat/send/model';
export { createSendForm } from './chat/send/ui';
export type { SendFormConfig } from './chat/send/ui';

// Sidebar — Toggle
export { $sidebarOpen, toggleSidebar, setSidebarOpen } from './sidebar/toggle/model';
export { createSidebarToggle } from './sidebar/toggle/ui';

// Settings — Modal
export { $settingsOpen, $apiKeys, openSettings, closeSettings, setApiKeys } from './settings/modal/model';
export { createSettingsModal } from './settings/modal/ui';
export type { SettingsModalConfig } from './settings/modal/ui';

// Theme
export { $theme, toggleTheme, setTheme } from './theme/toggle';
export type { ThemeMode } from './theme/toggle';
