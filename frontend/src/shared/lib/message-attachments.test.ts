import { describe, it, expect } from 'vitest';
import {
  inferMimeType,
  isImageAttachment,
  attachmentImageSrc,
  payloadToDisplayAttachment,
} from './message-attachments';

describe('message-attachments', () => {
  it('infers image mime from extension when browser omits type', () => {
    expect(inferMimeType('photo.png', '')).toBe('image/png');
    expect(inferMimeType('scan.JPG', 'application/octet-stream')).toBe('image/jpeg');
  });

  it('detects images without mime type', () => {
    expect(
      isImageAttachment({ name: 'chart.png', type: 'application/octet-stream' }),
    ).toBe(true);
  });

  it('builds data url for payload attachments', () => {
    const att = payloadToDisplayAttachment({
      name: 'a.png',
      type: 'image/png',
      data: 'abc',
    });
    expect(attachmentImageSrc(att)).toBe('data:image/png;base64,abc');
  });
});