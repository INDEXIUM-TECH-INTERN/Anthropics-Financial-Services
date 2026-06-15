import { createIconButton } from '../../../shared/ui/icon-button';

export interface SendFormConfig {
  onSend: (message: string) => void;
}

export function createSendForm(container: HTMLElement, config: SendFormConfig): { destroy: () => void } {
  const form = document.createElement('form');
  form.className = 'send-form';

  const textarea = document.createElement('textarea');
  textarea.className = 'send-input';
  textarea.placeholder = 'Nhập tin nhắn…';
  textarea.rows = 1;

  const sendBtn = createIconButton({
    icon: '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>',
    title: 'Gửi tin nhắn',
    className: 'send-btn',
  });

  form.appendChild(textarea);
  form.appendChild(sendBtn);
  container.appendChild(form);

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    const msg = textarea.value.trim();
    if (!msg) return;
    config.onSend(msg);
    textarea.value = '';
  };

  form.addEventListener('submit', handleSubmit);

  return {
    destroy() {
      form.removeEventListener('submit', handleSubmit);
      form.remove();
    },
  };
}
