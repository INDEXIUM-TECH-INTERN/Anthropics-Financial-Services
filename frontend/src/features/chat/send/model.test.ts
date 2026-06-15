import { describe, it, expect } from 'vitest';
import {
  $inputMessage,
  $isSending,
  $attachments,
  setInputMessage,
  setSending,
  setAttachments,
  clearInput,
} from './model';

describe('chat/send model', () => {
  it('should set input message', () => {
    setInputMessage('Hello');
    expect($inputMessage.get()).toBe('Hello');
  });

  it('should set sending state', () => {
    setSending(true);
    expect($isSending.get()).toBe(true);
    setSending(false);
    expect($isSending.get()).toBe(false);
  });

  it('should set attachments', () => {
    const files = [{ name: 'test.png', type: 'image/png', data: 'base64' }];
    setAttachments(files);
    expect($attachments.get()).toEqual(files);
  });

  it('should clear input and attachments', () => {
    setInputMessage('Hello');
    setAttachments([{ name: 'f.png', type: 'image/png', data: 'x' }]);
    clearInput();
    expect($inputMessage.get()).toBe('');
    expect($attachments.get()).toEqual([]);
  });
});
