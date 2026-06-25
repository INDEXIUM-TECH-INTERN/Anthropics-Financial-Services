import { escHtml } from './dom';
import type { AttachmentPayload } from '../api/types';

export interface MessageDisplayAttachment {
  name: string;
  type: string;
  data?: string;
  previewUrl?: string;
}

const IMAGE_EXT_RE = /\.(png|jpe?g|gif|webp|bmp|svg|heic|heif)$/i;

const EXT_MIME: Record<string, string> = {
  png: 'image/png',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  gif: 'image/gif',
  webp: 'image/webp',
  bmp: 'image/bmp',
  svg: 'image/svg+xml',
  heic: 'image/heic',
  heif: 'image/heif',
};

export function inferMimeType(name: string, mimeType: string): string {
  const normalized = (mimeType || '').trim().toLowerCase();
  if (normalized && normalized !== 'application/octet-stream') return normalized;
  const ext = name.split('.').pop()?.toLowerCase() || '';
  return EXT_MIME[ext] || normalized || 'application/octet-stream';
}

export function isImageAttachment(att: MessageDisplayAttachment): boolean {
  const type = inferMimeType(att.name, att.type);
  if (type.startsWith('image/')) return true;
  return IMAGE_EXT_RE.test(att.name);
}

export function fileToDisplayAttachment(file: File): MessageDisplayAttachment {
  return {
    name: file.name,
    type: inferMimeType(file.name, file.type),
    previewUrl: URL.createObjectURL(file),
  };
}

export function payloadToDisplayAttachment(att: AttachmentPayload): MessageDisplayAttachment {
  return {
    name: att.name,
    type: inferMimeType(att.name, att.type),
    data: att.data,
  };
}

export function attachmentImageSrc(att: MessageDisplayAttachment): string {
  if (att.data) return `data:${inferMimeType(att.name, att.type)};base64,${att.data}`;
  if (att.previewUrl) return att.previewUrl;
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
    if (isImageAttachment(att)) {
      const src = attachmentImageSrc(att);
      if (!src) continue;
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'msg-attachment-image-btn';
      btn.title = `Xem ảnh: ${att.name}`;
      btn.setAttribute('aria-label', `Xem ảnh ${att.name}`);

      const img = document.createElement('img');
      img.className = 'msg-attachment-image';
      img.alt = att.name;
      img.loading = 'lazy';
      img.draggable = false;
      img.src = src;

      btn.appendChild(img);
      wrap.appendChild(btn);
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