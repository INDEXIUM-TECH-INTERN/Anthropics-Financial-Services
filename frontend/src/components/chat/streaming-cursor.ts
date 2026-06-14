// ═══ Streaming Cursor — Claude-style blinking cursor ═══

let cursorEl: HTMLElement | null = null;

export function getStreamingCursor(): HTMLElement {
  if (!cursorEl) {
    cursorEl = document.createElement('span');
    cursorEl.className = 'streaming-cursor';
  }
  return cursorEl;
}

export function appendCursor(target: HTMLElement): void {
  removeCursor();
  target.appendChild(getStreamingCursor());
}

export function removeCursor(): void {
  if (cursorEl && cursorEl.parentNode) {
    cursorEl.parentNode.removeChild(cursorEl);
  }
}
