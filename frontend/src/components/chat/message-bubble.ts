// ═══ Message Bubble — Renders a single chat message ═══

import { renderMarkdown, extractUrls } from '../../services/markdown';
import type { TokenMetrics } from '../../types/api';

interface MessageBubbleOptions {
  text: string;
  sender: 'user' | 'bot';
  isStreaming?: boolean;
  metrics?: TokenMetrics;
  animationDelay?: string;
}

export function createMessageBubble(options: MessageBubbleOptions): {
  el: HTMLElement;
  content: HTMLElement;
  footer: HTMLElement;
} {
  const { text, sender, isStreaming = false, metrics, animationDelay } = options;

  const el = document.createElement('div');
  el.className = `message ${sender}`;
  if (animationDelay) {
    el.style.animationDelay = animationDelay;
  }

  // Avatar
  const avatar = document.createElement('div');
  avatar.className = 'msg-avatar';
  avatar.textContent = sender === 'user' ? 'U' : 'IX';

  // Body
  const body = document.createElement('div');
  body.className = 'msg-body';

  // Sender label
  const senderLabel = document.createElement('div');
  senderLabel.className = 'msg-sender';
  senderLabel.textContent = sender === 'user' ? 'Bạn' : 'Indexium AI';

  // Content
  const content = document.createElement('div');
  content.className = 'msg-content glass-panel';

  if (sender === 'bot') {
    if (isStreaming) {
      content.innerHTML = '';
    } else {
      content.innerHTML = renderMarkdown(text);
    }
  } else {
    content.textContent = text;
  }

  body.appendChild(senderLabel);
  body.appendChild(content);

  // Footer
  const footer = document.createElement('div');
  footer.className = 'msg-footer';

  if (sender === 'bot') {
    if (isStreaming) {
      footer.innerHTML = '<span class="msg-timer streaming">Đang trả lời…</span>';
    } else if (metrics) {
      const btn = document.createElement('button');
      btn.className = 'metrics-badge';
      const latSec = metrics.latency_ms ? (metrics.latency_ms / 1000).toFixed(1) + 's' : '—';
      btn.innerHTML = `📊 ${latSec} · ↑${metrics.token_in} · ↓${metrics.token_out} · ${metrics.ram_mb}`;
      btn.title = 'Xem chi tiết hiệu năng';
      footer.appendChild(btn);
    } else {
      footer.innerHTML = '<span class="msg-timer">—</span>';
    }
    body.appendChild(footer);
  }

  el.appendChild(avatar);
  el.appendChild(body);

  return { el, content, footer };
}

// ═══ Message Actions (Copy, Regenerate, Thumbs) ═══

export function createMessageActions(
  content: HTMLElement,
  _text: string,
  onRegenerate?: () => void,
): HTMLElement {
  const actions = document.createElement('div');
  actions.className = 'msg-actions';

  // Copy button
  const copyBtn = document.createElement('button');
  copyBtn.className = 'msg-action-btn';
  copyBtn.title = 'Copy';
  copyBtn.setAttribute('aria-label', 'Sao chép nội dung');
  copyBtn.textContent = '📋';
  copyBtn.addEventListener('click', () => {
    navigator.clipboard.writeText(content.innerText).then(() => {
      copyBtn.textContent = '✅';
      setTimeout(() => { copyBtn.textContent = '📋'; }, 2000);
    });
  });
  actions.appendChild(copyBtn);

  // Regenerate button
  if (onRegenerate) {
    const regenBtn = document.createElement('button');
    regenBtn.className = 'msg-action-btn';
    regenBtn.title = 'Tạo lại';
    regenBtn.setAttribute('aria-label', 'Tạo lại phản hồi');
    regenBtn.textContent = '🔄';
    regenBtn.addEventListener('click', onRegenerate);
    actions.appendChild(regenBtn);
  }

  // Thumbs up
  const thumbsUp = document.createElement('button');
  thumbsUp.className = 'msg-action-btn';
  thumbsUp.title = 'Hữu ích';
  thumbsUp.setAttribute('aria-label', 'Hữu ích');
  thumbsUp.textContent = '👍';
  thumbsUp.addEventListener('click', () => {
    thumbsUp.classList.toggle('active');
    thumbsDown.classList.remove('active');
  });
  actions.appendChild(thumbsUp);

  // Thumbs down
  const thumbsDown = document.createElement('button');
  thumbsDown.className = 'msg-action-btn';
  thumbsDown.title = 'Chưa hữu ích';
  thumbsDown.setAttribute('aria-label', 'Không hữu ích');
  thumbsDown.textContent = '👎';
  thumbsDown.addEventListener('click', () => {
    thumbsDown.classList.toggle('active');
    thumbsUp.classList.remove('active');
  });
  actions.appendChild(thumbsDown);

  return actions;
}

// ═══ Sources Inline ═══

export function renderSourcesInline(md: string, target: HTMLElement): void {
  if (target.querySelector('.message-sources')) return;
  const urls = extractUrls(md);
  if (!urls.length) return;

  const container = document.createElement('div');
  container.className = 'message-sources';
  container.innerHTML = '<div class="sources-label">Nguồn tham khảo</div>';

  const grid = document.createElement('div');
  grid.className = 'sources-grid';

  urls.forEach((url, i) => {
    let display = url;
    try { display = new URL(url).hostname.replace('www.', ''); } catch { /* keep raw */ }
    const a = document.createElement('a');
    a.href = url;
    a.target = '_blank';
    a.rel = 'noopener noreferrer';
    a.className = 'source-chip';
    a.innerHTML = `<span class="source-chip-index">${i + 1}</span>${display}`;
    grid.appendChild(a);
  });

  container.appendChild(grid);
  target.appendChild(container);
}

// ═══ Follow-up Chips ═══

export const FOLLOWUP_SUGGESTIONS: string[] = [
  'Giải thích thêm chi tiết',
  'So sánh với chỉ số khác',
  'Tóm tắt ngắn gọn',
];

export function createFollowupChips(
  container: HTMLElement,
  onSelect: (text: string) => void,
  suggestions: string[] = FOLLOWUP_SUGGESTIONS,
): void {
  const chips = document.createElement('div');
  chips.className = 'followup-chips';
  suggestions.forEach(s => {
    const c = document.createElement('button');
    c.className = 'followup-chip';
    c.textContent = s;
    c.addEventListener('click', () => onSelect(s));
    chips.appendChild(c);
  });
  container.appendChild(chips);
}
