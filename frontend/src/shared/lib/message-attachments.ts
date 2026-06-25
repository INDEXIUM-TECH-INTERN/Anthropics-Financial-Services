import { escHtml } from './dom';
import type { AttachmentPayload } from '../api/types';

export interface MessageDisplayAttachment {
  name: string;
  type: string;
  data?: string;
  previewUrl?: string;
}

export function fileToDisplayAttachment(file: File): MessageDisplayAttachment {
  return {
    name: file.name,
    type: file.type || 'application/octet-stream',
    previewUrl: URL.createObjectURL(file),
  };
}

export function payloadToDisplayAttachment(att: AttachmentPayload): MessageDisplayAttachment {
  return {
    name: att.name,
    type: att.type,
    data: att.data,
  };
}

export function attachmentImageSrc(att: MessageDisplayAttachment): string {
  if (att.previewUrl) return att.previewUrl;
  if (att.data) return `data:${att.type};base64,${att.data}`;
  return '';
}

export function appendAttachmentsToElement(
  container: HTMLElement,
  attachments: MessageDisplayAttachment[],
): void {
  if (!attachments.length) return;

  const wrap = document.createElement('div');
  wrap.className = 'msg-attachments';

  for (const att of attachments) {
    if (att.type.startsWith('image/')) {
      const src = attachmentImageSrc(att);
      if (!src) continue;
      const img = document.createElement('img');
      img.className = 'msg-attachment-image';
      img.alt = att.name;
      img.loading = 'lazy';
      img.src = src;
      wrap.appendChild(img);
      continue;
    }

    const chip = document.createElement('div');
    chip.className = 'msg-attachment-file';
    chip.innerHTML = `<span class="msg-attachment-file-icon">📎</span><span class="msg-attachment-file-name">${escHtml(att.name)}</span>`;
    wrap.appendChild(chip);
  }

  if (wrap.childElementCount > 0) {
    container.appendChild(wrap);
  }
}