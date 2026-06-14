// ═══ Streaming Cursor — Claude-style blinking cursor ═══
let cursorEl = null;
export function getStreamingCursor() {
    if (!cursorEl) {
        cursorEl = document.createElement('span');
        cursorEl.className = 'streaming-cursor';
    }
    return cursorEl;
}
export function appendCursor(target) {
    removeCursor();
    target.appendChild(getStreamingCursor());
}
export function removeCursor() {
    if (cursorEl && cursorEl.parentNode) {
        cursorEl.parentNode.removeChild(cursorEl);
    }
}
