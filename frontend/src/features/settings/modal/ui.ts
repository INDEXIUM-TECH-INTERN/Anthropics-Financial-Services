import { $settingsOpen, $apiKeys, closeSettings } from './model';
import { showToast } from '../../../shared/ui/toast';

export interface SettingsModalConfig {
  onSaveKeys: (keys: string) => Promise<void>;
}

export function createSettingsModal(container: HTMLElement, config: SettingsModalConfig): { destroy: () => void } {
  const overlay = document.createElement('div');
  overlay.className = 'settings-modal-overlay';
  overlay.style.display = 'none';

  const modal = document.createElement('div');
  modal.className = 'settings-modal';
  modal.setAttribute('role', 'dialog');
  modal.setAttribute('aria-modal', 'true');
  modal.setAttribute('aria-labelledby', 'settings-modal-title');

  const title = document.createElement('h2');
  title.id = 'settings-modal-title';
  title.textContent = 'Cài đặt';

  const label = document.createElement('label');
  label.textContent = 'API Keys (mỗi key một dòng):';
  label.htmlFor = 'settings-api-keys';

  const textarea = document.createElement('textarea');
  textarea.id = 'settings-api-keys';
  textarea.className = 'settings-api-keys-input';
  textarea.rows = 4;

  const saveBtn = document.createElement('button');
  saveBtn.textContent = 'Lưu';
  saveBtn.className = 'settings-save-btn';
  saveBtn.addEventListener('click', async () => {
    const keys = textarea.value
      .split('\n')
      .map((k) => k.trim())
      .filter(Boolean);
    try {
      await config.onSaveKeys(keys.join('\n'));
      showToast({ message: 'Đã lưu cài đặt.', type: 'success' });
      closeSettings();
    } catch {
      showToast({ message: 'Lỗi lưu cài đặt.', type: 'error' });
    }
  });

  const cancelBtn = document.createElement('button');
  cancelBtn.textContent = 'Hủy';
  cancelBtn.className = 'settings-cancel-btn';
  cancelBtn.addEventListener('click', () => closeSettings());

  modal.appendChild(title);
  modal.appendChild(label);
  modal.appendChild(textarea);
  modal.appendChild(saveBtn);
  modal.appendChild(cancelBtn);
  overlay.appendChild(modal);
  container.appendChild(overlay);

  const unsub = $settingsOpen.subscribe((open) => {
    overlay.style.display = open ? '' : 'none';
    if (open) textarea.value = $apiKeys.get();
  });

  return {
    destroy() {
      unsub();
      overlay.remove();
    },
  };
}
