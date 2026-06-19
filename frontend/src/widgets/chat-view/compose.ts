// ═══ Chat View Widget ═══
// Composes chat message display, streaming, and related UI.

import { renderMarkdown, highlightCode } from '../../shared/lib/markdown';
import { renderErrorHTML, diagnoseError } from '../../shared/lib/errors';
import { createMessageSkeleton } from '../../shared/ui/skeleton';
import { escHtml } from '../../shared/lib/dom';
import type { TokenMetrics } from '../../shared/api/types';

export interface ChatViewWidget {
  appendMessage: (
    text: string,
    sender: 'user' | 'bot',
    isStreaming?: boolean,
    metrics?: TokenMetrics,
  ) => { el: HTMLElement | null; content: HTMLElement | null; footer: HTMLElement | null };
  appendStreamingBotMessage: () => { el: HTMLElement; content: HTMLElement; footer: HTMLElement };
  finalizeStreaming: (fullText: string, metrics: TokenMetrics, elapsed: string) => void;
  showError: (error: Error, elapsed: string) => void;
  showSkeleton: () => HTMLElement;
  removeSkeleton: (el: HTMLElement) => void;
  clearMessages: () => void;
  scrollToBottom: () => void;
  destroy: () => void;
}

export function createChatViewWidget(container: HTMLElement, _viewport: HTMLElement): ChatViewWidget {
  let abortController: AbortController | null = null;

  function scrollToBottom() {
    requestAnimationFrame(() => {
      const lastMessage = container.lastElementChild;
      if (lastMessage) {
        lastMessage.scrollIntoView({ behavior: 'smooth', block: 'end' });
        return;
      }
      window.scrollTo({ top: document.documentElement.scrollHeight, behavior: 'smooth' });
    });
  }

  function appendMessage(
    text: string,
    sender: 'user' | 'bot',
    _isStreaming = false,
    metrics?: TokenMetrics,
  ): { el: HTMLElement | null; content: HTMLElement | null; footer: HTMLElement | null } {
    const el = document.createElement('div');
    el.className = `message message-${sender}`;

    const avatar = document.createElement('div');
    avatar.className = 'msg-avatar';
    avatar.textContent = sender === 'user' ? '👤' : '🤖';

    const body = document.createElement('div');
    body.className = 'msg-body';

    const content = document.createElement('div');
    content.className = 'msg-content';
    if (text) content.innerHTML = sender === 'bot' ? renderMarkdown(text) : escHtml(text);

    body.appendChild(content);

    const footer = document.createElement('div');
    footer.className = 'msg-footer';
    if (metrics) {
      const badge = document.createElement('span');
      badge.className = 'metrics-badge';
      badge.textContent = `↑${metrics.token_in} · ↓${metrics.token_out} · ${metrics.ram_mb}`;
      footer.appendChild(badge);
    }
    body.appendChild(footer);

    el.appendChild(avatar);
    el.appendChild(body);
    container.appendChild(el);
    scrollToBottom();

    return { el, content, footer };
  }

  function appendStreamingBotMessage(): { el: HTMLElement; content: HTMLElement; footer: HTMLElement } {
    const result = appendMessage('', 'bot', true);
    return { el: result.el!, content: result.content!, footer: result.footer! };
  }

  function finalizeStreaming(fullText: string, metrics: TokenMetrics, elapsed: string) {
    const msgs = container.querySelectorAll('.message');
    const lastMsg = msgs[msgs.length - 1];
    if (!lastMsg) return;

    const content = lastMsg.querySelector('.msg-content') as HTMLElement | null;
    const footer = lastMsg.querySelector('.msg-footer') as HTMLElement | null;

    if (content) {
      content.innerHTML = renderMarkdown(fullText);
      highlightCode(content);
    }
    if (footer) {
      footer.innerHTML = '';
      const badge = document.createElement('button');
      badge.className = 'metrics-badge';
      badge.textContent = `📊 ${elapsed}s · ↑${metrics.token_in} · ↓${metrics.token_out} · ${metrics.ram_mb}`;
      footer.appendChild(badge);
    }
  }

  function showError(error: Error, elapsed: string) {
    const msgs = container.querySelectorAll('.message');
    const lastMsg = msgs[msgs.length - 1];
    if (!lastMsg) return;

    const content = lastMsg.querySelector('.msg-content') as HTMLElement | null;
    const footer = lastMsg.querySelector('.msg-footer') as HTMLElement | null;

    const diag = diagnoseError(error.name, error.message);
    if (content) content.innerHTML = renderErrorHTML(diag);
    if (footer) footer.innerHTML = `<span class="msg-timer" style="color:var(--danger)">Thất bại · ${elapsed}s</span>`;
  }

  function showSkeleton(): HTMLElement {
    const skeleton = createMessageSkeleton();
    container.appendChild(skeleton);
    return skeleton;
  }

  function removeSkeleton(el: HTMLElement) {
    el.remove();
  }

  function clearMessages() {
    container.querySelectorAll('.message, .skeleton-loading, .thinking-card').forEach((m) => m.remove());
  }

  function destroy() {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
  }

  return {
    appendMessage,
    appendStreamingBotMessage,
    finalizeStreaming,
    showError,
    showSkeleton,
    removeSkeleton,
    clearMessages,
    scrollToBottom,
    destroy,
  };
}
