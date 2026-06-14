// ═══ Typing Indicator — Animated dots shown while AI is thinking ═══

export function createTypingIndicator(): HTMLElement {
  const el = document.createElement('div');
  el.className = 'typing-indicator';
  el.setAttribute('aria-label', 'AI đang suy nghĩ');
  el.setAttribute('role', 'status');

  for (let i = 0; i < 3; i++) {
    const dot = document.createElement('span');
    dot.className = 'typing-dot';
    el.appendChild(dot);
  }

  return el;
}
