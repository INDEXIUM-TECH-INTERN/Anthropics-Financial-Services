import { atom } from 'nanostores';

export const $inputMessage = atom<string>('');
export const $isSending = atom<boolean>(false);
export const $attachments = atom<{ name: string; type: string; data: string }[]>([]);

export function setInputMessage(msg: string) {
  $inputMessage.set(msg);
}
export function setSending(s: boolean) {
  $isSending.set(s);
}
export function setAttachments(a: { name: string; type: string; data: string }[]) {
  $attachments.set(a);
}
export function clearInput() {
  $inputMessage.set('');
  $attachments.set([]);
}
