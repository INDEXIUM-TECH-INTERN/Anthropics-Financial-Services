// ═══ Indexium — Glass Chat (TypeScript + Vite) ═══
// Refactored: lean main.ts, logic extracted to services
import { ApiClient } from './services/api';
import { renderMarkdown, highlightCode } from './services/markdown';
import { SSEManager } from './services/sse-manager';
import { diagnoseError, renderErrorHTML } from './services/error-handler';
import { store } from './stores/app-state';
import { $, escHtml, readFileAsBase64 } from './utils/dom';
import {
  createMessageBubble, createMessageActions,
  renderSourcesInline, createFollowupChips,
} from './components/chat/message-bubble';
import { createThinkingCard } from './components/chat/thinking-card';
import { appendCursor, removeCursor } from './components/chat/streaming-cursor';
import { createTypingIndicator } from './components/chat/typing-indicator';
import { renderConversationList } from './components/sidebar/conversation-list';
import { createMessageSkeleton } from './components/ui/skeleton';
import { showToast } from './components/ui/toast';
import { createConnectionStatus } from './components/ui/connection-status';
import type {
  ChatSession, AttachmentPayload, TokenMetrics, HistoryMessage,
} from './types/api';
import './styles/main.css';

// ═══════════════════════════════════════════════════════
// DOM REFS
// ═══════════════════════════════════════════════════════
const chatInput = $<HTMLTextAreaElement>('chat-input')!;
const sendBtn = $<HTMLButtonElement>('send-btn')!;
const chatContent = $('chat-content')!;
const chatViewport = $('chat-viewport')!;
const welcomeState = $('welcome-state')!;
const conversationsList = $('conversations-list')!;
const conversationsSidebar = $('conversations-sidebar')!;
const pipelineSidebar = $('pipeline-sidebar');
const logStream = $('log-stream');
const sourcesPanel = $('sources-panel');
const sourcesList = $('sources-list');
const currentChatTitle = $('current-chat-title');
const attachmentsPreview = $('attachments-preview');
const fileInput = $<HTMLInputElement>('file-input');
const attachBtn = $('attach-btn');
const backendSelect = $<HTMLSelectElement>('backend-select');
const searchInput = $<HTMLInputElement>('search-input');
const runTestBtn = $('run-test-btn');
const scrollToBottomBtn = $('scroll-to-bottom');
const sidebarBackdrop = $('sidebar-backdrop');
const settingsModal = $('settings-modal')!;
const settingsTrigger = $('settings-trigger');
const closeSettings = $('close-settings');
const closeSettingsBtn = $('close-settings-btn');
const saveSettingsBtn = $('save-settings-btn');
const orKeysInput = $<HTMLTextAreaElement>('or-keys-input');
const toggleConversations = $('toggle-conversations');
const togglePipeline = $('toggle-pipeline');
const closePipeline = $('close-pipeline');
const themeToggle = $('theme-toggle');
const newChatBtn = $('new-chat-btn');
const shortcutsPanel = $('shortcuts-panel');
const closeShortcuts = $('close-shortcuts');
const metricsModal = $('metrics-modal');
const closeMetrics = $('close-metrics');
// Pipeline step elements — reserved for future SSE-driven status UI
// const pipeAgent = $('pipe-agent');
// const pipeSkill = $('pipe-skill');
// const pipeTool = $('pipe-tool');
// const pipeReason = $('pipe-reason');

// ═══════════════════════════════════════════════════════
// STATE
// ═══════════════════════════════════════════════════════
const isLocalhost = window.location.hostname === 'localhost'
  || window.location.hostname === '127.0.0.1';

const backends: Record<string, string> = {
  gemini: isLocalhost ? 'http://localhost:8080' : window.location.origin,
  claude: 'http://localhost:8081',
};

let currentBaseUrl = backends[store.getState().currentBackend] ?? backends.gemini!;
let api = new ApiClient(currentBaseUrl);
let abortController: AbortController | null = null;
let pendingAttachments: File[] = [];
let currentBotContentEl: HTMLElement | null = null;
let currentBotFooterEl: HTMLElement | null = null;

const sseManager = new SSEManager();

// Connection status
const headerEl = document.querySelector('.header-right') as HTMLElement | null;
let connStatus: ReturnType<typeof createConnectionStatus> | null = null;
if (headerEl) {
  connStatus = createConnectionStatus(headerEl, {
    onReconnect: () => {
      connStatus?.setStatus('reconnecting');
      sseManager.disconnect();
      sseManager.connect(currentBaseUrl, {
        onMessage: (d) => handleSSEMessage(d),
        onError: (msg) => { addLogEntry(msg, 'error'); connStatus?.setStatus('disconnected'); },
      });
      // Probe the backend
      fetch(`${currentBaseUrl}/`, { method: 'HEAD', signal: AbortSignal.timeout(5000) })
        .then((res) => { connStatus?.setStatus(res.ok ? 'connected' : 'disconnected'); })
        .catch(() => { connStatus?.setStatus('disconnected'); });
    },
  });
  // Initial probe
  fetch(`${currentBaseUrl}/`, { method: 'HEAD', signal: AbortSignal.timeout(5000) })
    .then((res) => { connStatus?.setStatus(res.ok ? 'connected' : 'disconnected'); })
    .catch(() => { connStatus?.setStatus('disconnected'); });
}

// Typing indicator element (created once, shown/hidden during generation)
let typingEl: HTMLElement | null = null;

// ═══════════════════════════════════════════════════════
// THEME
// ═══════════════════════════════════════════════════════
function applyTheme(theme: 'dark' | 'light') {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('theme', theme);
}
applyTheme(store.getState().theme);

themeToggle?.addEventListener('click', () => {
  const next = store.getState().theme === 'dark' ? 'light' : 'dark';
  store.setState({ theme: next });
  applyTheme(next);
});

// ═══════════════════════════════════════════════════════
// SIDEBAR TOGGLES
// ═══════════════════════════════════════════════════════
toggleConversations?.addEventListener('click', () => {
  conversationsSidebar?.classList.toggle('collapsed');
  if (sidebarBackdrop && window.innerWidth <= 768) {
    const open = !conversationsSidebar?.classList.contains('collapsed');
    sidebarBackdrop.classList.toggle('hidden', !open);
    sidebarBackdrop.classList.toggle('visible', open);
  }
});

togglePipeline?.addEventListener('click', () => {
  pipelineSidebar?.classList.toggle('collapsed');
  if (!pipelineSidebar?.classList.contains('collapsed') && logStream) {
    logStream.scrollTop = logStream.scrollHeight;
  }
});

closePipeline?.addEventListener('click', () => pipelineSidebar?.classList.add('collapsed'));

searchInput?.addEventListener('input', (e) => {
  const q = ((e.target as HTMLInputElement).value || '').toLowerCase();
  renderSidebarConversations(q);
});

sidebarBackdrop?.addEventListener('click', () => {
  conversationsSidebar?.classList.add('collapsed');
  sidebarBackdrop.classList.remove('visible');
  sidebarBackdrop.classList.add('hidden');
});

// ═══════════════════════════════════════════════════════
// SETTINGS MODAL
// ═══════════════════════════════════════════════════════
function trapFocus(this: HTMLElement, e: KeyboardEvent) {
  if (e.key !== 'Tab') return;
  const focusable = this.querySelectorAll<HTMLElement>('input, textarea, button, select');
  if (!focusable.length) return;
  if (e.shiftKey && document.activeElement === focusable[0]) {
    e.preventDefault();
    focusable[focusable.length - 1]!.focus();
  } else if (!e.shiftKey && document.activeElement === focusable[focusable.length - 1]) {
    e.preventDefault();
    focusable[0]!.focus();
  }
}

function openSettings() {
  settingsModal.classList.remove('hidden');
  const focusable = settingsModal.querySelectorAll<HTMLElement>('input, textarea, button, select');
  if (focusable.length) focusable[0]!.focus();
  settingsModal.addEventListener('keydown', trapFocus as EventListener);
}

function hideSettings() {
  settingsModal.classList.add('hidden');
  settingsModal.removeEventListener('keydown', trapFocus as EventListener);
}

settingsTrigger?.addEventListener('click', openSettings);
closeSettings?.addEventListener('click', hideSettings);
closeSettingsBtn?.addEventListener('click', hideSettings);
settingsModal?.addEventListener('click', (e) => { if (e.target === settingsModal) hideSettings(); });

saveSettingsBtn?.addEventListener('click', async () => {
  const raw = orKeysInput?.value.trim() || '';
  if (saveSettingsBtn) {
    (saveSettingsBtn as HTMLButtonElement).disabled = true;
    saveSettingsBtn.textContent = 'Đang lưu...';
  }
  try {
    const keys = raw.split('\n').map(k => k.trim()).filter(k => k);
    await api.saveConfigKeys(keys);
    addLogEntry('Đã cập nhật API Keys.', 'success');
    showToast({ message: 'Đã lưu API Keys', type: 'success' });
    hideSettings();
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Không xác định';
    addLogEntry(`Lỗi: ${msg}`, 'error');
    showToast({ message: 'Lỗi lưu API Keys', type: 'error' });
  } finally {
    if (saveSettingsBtn) {
      (saveSettingsBtn as HTMLButtonElement).disabled = false;
      saveSettingsBtn.textContent = 'Lưu cấu hình';
    }
  }
});

// ═══════════════════════════════════════════════════════
// METRICS MODAL
// ═══════════════════════════════════════════════════════
function renderMath(el: HTMLElement) {
  if (typeof (window as unknown as Record<string, unknown>).renderMathInElement === 'function') {
    (window as unknown as { renderMathInElement: (el: HTMLElement, opts: Record<string, unknown>) => void })
      .renderMathInElement(el, {
        delimiters: [
          { left: '$$', right: '$$', display: true },
          { left: '$', right: '$', display: false },
          { left: '\\(', right: '\\)', display: false },
          { left: '\\[', right: '\\]', display: true },
        ],
        throwOnError: false,
      });
  }
}

function showMetricsModal(met: TokenMetrics) {
  if (!metricsModal) return;
  const lat = met.latency_ms ? `${(met.latency_ms / 1000).toFixed(2)}s` : '—';
  $<HTMLElement>('metric-val-latency')!.textContent = lat;
  $<HTMLElement>('metric-val-token-in')!.textContent = String(met.token_in);
  $<HTMLElement>('metric-val-token-out')!.textContent = String(met.token_out);
  $<HTMLElement>('metric-val-ram')!.textContent = met.ram_mb || '—';
  $<HTMLElement>('metric-val-cpu')!.textContent = met.cpu_load || '—';
  metricsModal.classList.remove('hidden');
}

function hideMetricsModal() {
  metricsModal?.classList.add('hidden');
}

closeMetrics?.addEventListener('click', hideMetricsModal);
metricsModal?.addEventListener('click', (e) => { if (e.target === metricsModal) hideMetricsModal(); });

// ═══════════════════════════════════════════════════════
// BACKEND SELECTOR
// ═══════════════════════════════════════════════════════
backendSelect?.addEventListener('change', (e) => {
  const b = (e.target as HTMLSelectElement).value;
  currentBaseUrl = backends[b] ?? backends.gemini!;
  api = new ApiClient(currentBaseUrl);
  sseManager.connect(currentBaseUrl, {
    onMessage: (d) => handleSSEMessage(d),
    onError: (msg) => addLogEntry(msg, 'error'),
  });
  store.setState({ currentBackend: b as 'gemini' | 'claude' });
  addLogEntry(`Đã chuyển sang ${b}`, 'info');
});

// ═══════════════════════════════════════════════════════
// SSE
// ═══════════════════════════════════════════════════════
function handleSSEMessage(d: Record<string, unknown>) {
  const type = d.type as string;
  const payload = String(d.payload || '');
  const metadata = (d.metadata || {}) as Record<string, string>;

  addLogEntry(payload, type || 'info');

  if (type === 'agent_selected') {
    store.setPipeline({
      agent: metadata.agent || payload,
      agentStatus: 'active',
      reason: metadata.reason || '',
      reasonStatus: 'active',
    });
  }
  if (type === 'skill_loaded') {
    store.setPipeline({ skill: metadata.skill || payload, skillStatus: 'active' });
  }
  if (type === 'tool_executed') {
    store.setPipeline({ tool: metadata.tool || payload, toolStatus: 'active' });
  }
}

sseManager.connect(currentBaseUrl, {
  onMessage: (d) => handleSSEMessage(d),
  onError: (msg) => addLogEntry(msg, 'error'),
});

// ═══════════════════════════════════════════════════════
// CONVERSATIONS
// ═══════════════════════════════════════════════════════
async function fetchConversations() {
  try {
    store.setState({ chats: await api.getSessions() });
  } catch {
    store.setState({ chats: [] });
  }
  renderSidebarConversations();
}

function renderSidebarConversations(searchQuery = '') {
  renderConversationList(
    conversationsList,
    store.getState().chats,
    store.getState().currentChatId,
    (id) => switchChat(id),
    async (id) => {
      if (confirm('Xóa?')) await deleteChat(id);
    },
    searchQuery,
  );
}

async function deleteChat(id: string) {
  try {
    await api.deleteSession(id);
    store.setState({ chats: store.getState().chats.filter(c => c.id !== id) });
    addLogEntry('Đã xóa.', 'success');
    showToast({ message: 'Đã xóa cuộc trò chuyện', type: 'success' });
    if (store.getState().currentChatId === id) {
      const chats = store.getState().chats;
      if (chats.length > 0) {
        await switchChat(chats[0]!.id);
      } else {
        await createNewChat();
      }
    } else {
      renderSidebarConversations();
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'Lỗi';
    addLogEntry(`Lỗi: ${msg}`, 'error');
    showToast({ message: 'Lỗi xóa cuộc trò chuyện', type: 'error' });
  }
}

async function createNewChat(title = 'Cuộc trò chuyện mới') {
  try {
    const c = await api.createSession(title);
    store.setState({
      chats: [c, ...store.getState().chats],
      currentChatId: c.id,
      currentChatTitle: title,
    });
    clearChatUI();
    if (welcomeState) welcomeState.style.display = 'flex';
    if (currentChatTitle) currentChatTitle.textContent = title;
    renderSidebarConversations();
    addLogEntry('Đã tạo mới.', 'success');
    return c;
  } catch {
    const fb: ChatSession = { id: `local_${Date.now()}`, title };
    store.setState({
      chats: [fb, ...store.getState().chats],
      currentChatId: fb.id,
      currentChatTitle: title,
    });
    clearChatUI();
    if (welcomeState) welcomeState.style.display = 'flex';
    if (currentChatTitle) currentChatTitle.textContent = title;
    renderSidebarConversations();
    return fb;
  }
}

function groupHistoryMessages(
  history: HistoryMessage[],
): Array<{ role: 'user' | 'bot'; content: string; metrics?: TokenMetrics }> {
  const grouped: Array<{ role: 'user' | 'bot'; content: string; metrics?: TokenMetrics }> = [];

  for (let i = 0; i < history.length; i++) {
    const msg = history[i];
    if (!msg) continue;
    const role = String(msg.role || '').toLowerCase();
    if (role === 'tool') continue;

    if (role === 'user') {
      grouped.push({ role: 'user', content: msg.content || '' });
      continue;
    }

    const text = msg.content || '';
    if (!text) continue;

    // Check if this is a "thinking" message (followed by another assistant msg)
    let isThinking = false;
    for (let j = i + 1; j < history.length; j++) {
      const next = history[j];
      if (!next) continue;
      const nextRole = String(next.role || '').toLowerCase();
      if (nextRole === 'user') break;
      if (nextRole === 'assistant') { isThinking = true; break; }
    }

    let formattedContent = text;
    if (isThinking) {
      formattedContent =
        `<details class="thinking-details">` +
        `<summary class="thinking-summary">Suy nghĩ của AI</summary>` +
        `<div class="thinking-content">${text}</div></details>\n\n`;
    }

    const last = grouped[grouped.length - 1];
    if (last && last.role === 'bot') {
      last.content += formattedContent;
      if (msg.token_in !== undefined && msg.token_in > 0) {
        last.metrics = {
          token_in: msg.token_in,
          token_out: msg.token_out || 0,
          ram_mb: msg.ram_mb || '',
          latency_ms: msg.latency_ms,
          cpu_load: msg.cpu_load,
        };
      }
    } else {
      const metrics: TokenMetrics | undefined =
        msg.token_in !== undefined && msg.token_in > 0
          ? {
              token_in: msg.token_in,
              token_out: msg.token_out || 0,
              ram_mb: msg.ram_mb || '',
              latency_ms: msg.latency_ms,
              cpu_load: msg.cpu_load,
            }
          : undefined;
      grouped.push({ role: 'bot', content: formattedContent, metrics });
    }
  }

  return grouped;
}

async function switchChat(id: string) {
  store.setState({ currentChatId: id });
  const chat = store.getState().chats.find(c => c.id === id);
  if (currentChatTitle) currentChatTitle.textContent = chat?.title || id;
  clearChatUI();
  renderSidebarConversations();

  const skeleton = createMessageSkeleton();
  chatContent?.appendChild(skeleton);

  try {
    const data = await api.getHistory(id);
    skeleton.remove();
    let hasMessages = false;
    let idx = 0;
    const grouped = groupHistoryMessages(data.history);
    for (const g of grouped) {
      appendMessageBubble(g.content, g.role, false, g.metrics);
      const msgs = chatContent.querySelectorAll('.message');
      (msgs[msgs.length - 1] as HTMLElement).style.animationDelay = `${idx * 0.06}s`;
      idx++;
      hasMessages = true;
    }
    if (hasMessages && welcomeState) welcomeState.style.display = 'none';
    else if (welcomeState) welcomeState.style.display = 'flex';
    scrollToBottom();
  } catch {
    skeleton.remove();
    if (welcomeState) welcomeState.style.display = 'flex';
  }

  addLogEntry(`Chuyển: ${chat?.title || id}`, 'info');
}

function clearChatUI() {
  if (!chatContent) return;
  chatContent.querySelectorAll('.message, .skeleton-loading, .thinking-card')
    .forEach(m => m.remove());
  if (logStream) logStream.innerHTML = '<div class="log-stream-empty">Sẵn sàng.</div>';
  if (sourcesPanel) sourcesPanel.style.display = 'none';
  if (sourcesList) sourcesList.innerHTML = '';
  store.resetPipeline();
}

// ═══════════════════════════════════════════════════════
// ATTACHMENTS
// ═══════════════════════════════════════════════════════
attachBtn?.addEventListener('click', () => fileInput?.click());

fileInput?.addEventListener('change', () => {
  if (fileInput.files?.length) {
    pendingAttachments = [...pendingAttachments, ...Array.from(fileInput.files)];
    renderAttachments();
    fileInput.value = '';
  }
});

function renderAttachments() {
  if (!attachmentsPreview) return;
  attachmentsPreview.innerHTML = '';
  pendingAttachments.forEach((file, i) => {
    const chip = document.createElement('div');
    chip.className = 'attachment-chip';
    const icon = file.type.startsWith('image/') ? '🖼️' : '📎';
    chip.innerHTML =
      `<span>${icon}</span>` +
      `<span class="attachment-chip-name">${escHtml(file.name)}</span>` +
      `<button class="attachment-remove-btn">✕</button>`;
    chip.querySelector('.attachment-remove-btn')?.addEventListener('click', (e) => {
      e.stopPropagation();
      pendingAttachments.splice(i, 1);
      renderAttachments();
    });
    attachmentsPreview.appendChild(chip);
  });
}

// ═══════════════════════════════════════════════════════
// MESSAGES
// ═══════════════════════════════════════════════════════
function appendMessageBubble(
  text: string,
  sender: 'user' | 'bot',
  isStreaming = false,
  metrics?: TokenMetrics,
) {
  if (!chatContent) return { el: null, content: null, footer: null };
  if (welcomeState) welcomeState.style.display = 'none';

  const { el, content, footer } = createMessageBubble({
    text, sender, isStreaming, metrics,
  });

  // Bot message post-processing
  if (sender === 'bot' && !isStreaming && text) {
    renderSourcesInline(text, content);
    highlightCode(content);
    renderMath(content);

    // Action buttons
    const actions = createMessageActions(content, text, undefined);
    const body = el.querySelector('.msg-body') as HTMLElement | null;
    if (body) body.appendChild(actions);

    // Follow-up chips
    const target = body || el;
    createFollowupChips(target, (chipText) => {
      if (chatInput) chatInput.value = chipText;
      sendMessage();
    });
  }

  // User message: click-to-edit
  if (sender === 'user' && !isStreaming) {
    el.addEventListener('click', function handler(this: HTMLElement, evt: Event) {
      if ((evt.target as HTMLElement).closest('.edit-message-container')) return;
      this.removeEventListener('click', handler);
      const orig = content.textContent || '';
      const container = document.createElement('div');
      container.className = 'edit-message-container';
      const ta = document.createElement('textarea');
      ta.className = 'edit-message-textarea';
      ta.value = orig;
      ta.rows = 3;
      const actionBar = document.createElement('div');
      actionBar.className = 'edit-message-actions';
      const submitBtn = document.createElement('button');
      submitBtn.className = 'btn btn-primary btn-sm';
      submitBtn.textContent = 'Gửi lại';
      const cancelBtn = document.createElement('button');
      cancelBtn.className = 'btn btn-ghost btn-sm';
      cancelBtn.textContent = 'Hủy';
      actionBar.appendChild(cancelBtn);
      actionBar.appendChild(submitBtn);
      container.appendChild(ta);
      container.appendChild(actionBar);
      content.style.display = 'none';
      const msgBody = this.querySelector('.msg-body');
      if (msgBody) msgBody.insertBefore(container, content.nextSibling);
      ta.focus();

      cancelBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        container.remove();
        content.style.display = '';
        this.addEventListener('click', handler);
      });

      submitBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const newText = ta.value.trim();
        if (!newText) return;
        // Remove subsequent bot messages
        let sibling = this.nextElementSibling;
        while (sibling) {
          const next = sibling.nextElementSibling;
          if (sibling.classList.contains('message') || sibling.classList.contains('followup-chips')) {
            sibling.remove();
          }
          sibling = next;
        }
        this.remove();
        sendMessage(newText);
      });
    });
  }

  // Streaming cursor for bot
  if (sender === 'bot' && isStreaming) {
    appendCursor(content);
  }

  chatContent.appendChild(el);
  scrollToBottom();
  return { el, content, footer };
}

// ═══════════════════════════════════════════════════════
// SEND MESSAGE
// ═══════════════════════════════════════════════════════
async function sendMessage(textOverride: string | null = null, isRegenerate = false) {
  const text = textOverride !== null ? textOverride : (chatInput?.value.trim() || '');
  if (!text && pendingAttachments.length === 0) return;

  if (!isRegenerate) {
    if (!store.getState().currentChatId) await createNewChat();
    appendMessageBubble(text, 'user');
  }

  const files = [...pendingAttachments];
  pendingAttachments = [];
  renderAttachments();

  if (chatInput) {
    chatInput.value = '';
    chatInput.style.height = '24px';
  }

  store.setState({ isGenerating: true });
  updateSendButtonState();

  // Thinking card + typing indicator
  const thinkingCard = createThinkingCard();
  chatContent?.appendChild(thinkingCard);

  typingEl = createTypingIndicator();
  typingEl.style.padding = '4px 0';
  chatContent?.appendChild(typingEl);

  const startTime = Date.now();
  const result = appendMessageBubble('', 'bot', true) as {
    content: HTMLElement; footer: HTMLElement;
  };
  currentBotContentEl = result.content;
  currentBotFooterEl = result.footer;

  abortController = new AbortController();

  try {
    const att: AttachmentPayload[] = [];
    for (const f of files) {
      att.push({ name: f.name, type: f.type, data: await readFileAsBase64(f) });
    }

    let fullText = '';
    await api.streamChat(
      text,
      store.getState().currentChatId || undefined,
      att,
      abortController.signal,
      (tok) => {
        // Hide typing indicator when first token arrives
        if (typingEl) { typingEl.remove(); typingEl = null; }
        fullText += tok;
        if (currentBotContentEl) {
          currentBotContentEl.innerHTML = renderMarkdown(fullText);
          appendCursor(currentBotContentEl);
          scrollToBottom();
        }
      },
      (met) => {
        const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
        removeCursor();
        if (currentBotContentEl) {
          currentBotContentEl.innerHTML = renderMarkdown(fullText);
          highlightCode(currentBotContentEl);
          renderMath(currentBotContentEl);
        }
        if (currentBotFooterEl) {
          currentBotFooterEl.innerHTML = '';
          const badge = document.createElement('button');
          badge.className = 'metrics-badge';
          badge.innerHTML = `📊 ${elapsed}s · ↑${met.token_in} · ↓${met.token_out} · ${met.ram_mb}`;
          badge.title = 'Xem chi tiết hiệu năng';
          if (!met.latency_ms) met.latency_ms = Date.now() - startTime;
          badge.addEventListener('click', (e) => {
            e.stopPropagation();
            showMetricsModal(met);
          });
          currentBotFooterEl.appendChild(badge);
        }
        addCopyButtons(currentBotContentEl!);
        thinkingCard.remove();
        fetchConversations();
      },
      (err) => { throw new Error(err); },
    );

    store.setState({ isGenerating: false });
    updateSendButtonState();
    abortController = null;
    currentBotContentEl = null;
    currentBotFooterEl = null;
  } catch (error) {
    store.setState({ isGenerating: false });
    updateSendButtonState();
    abortController = null;

    if ((error as Error).name === 'AbortError') return;

    removeCursor();
    thinkingCard.remove();
    if (typingEl) { typingEl.remove(); typingEl = null; }
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
    const diag = diagnoseError((error as Error).name, (error as Error).message);

    if (currentBotContentEl) {
      currentBotContentEl.innerHTML = renderErrorHTML(diag);
      // Add retry button for network/quota errors
      if (diag.name === 'TypeError' || diag.msg.toLowerCase().includes('network') || diag.msg.toLowerCase().includes('quota') || diag.msg.toLowerCase().includes('429')) {
        const retryBtn = document.createElement('button');
        retryBtn.className = 'btn btn-ghost btn-sm';
        retryBtn.style.marginTop = '8px';
        retryBtn.innerHTML = '🔄 Thử lại';
        retryBtn.addEventListener('click', () => {
          // Remove the error message bubble
          const msgEl = currentBotContentEl?.closest('.message');
          if (msgEl) msgEl.remove();
          currentBotContentEl = null;
          currentBotFooterEl = null;
          // Retry with exponential backoff
          sendMessage(text, false);
        });
        currentBotContentEl.appendChild(retryBtn);
      }
    }
    if (currentBotFooterEl) {
      currentBotFooterEl.innerHTML =
        `<span class="msg-timer" style="color:var(--danger)">Thất bại · ${elapsed}s</span>`;
    }
    currentBotContentEl = null;
    currentBotFooterEl = null;
  } finally {
    if (sendBtn) (sendBtn as HTMLButtonElement).disabled = false;
    chatInput?.focus();
    scrollToBottom();
  }
}

function addCopyButtons(el: HTMLElement) {
  el.querySelectorAll('pre').forEach(pre => {
    if (pre.querySelector('.copy-code-btn')) return;
    const btn = document.createElement('button');
    btn.className = 'copy-code-btn';
    btn.textContent = '📋 Copy';
    btn.addEventListener('click', () => {
      const code = pre.querySelector('code')?.innerText || pre.innerText;
      navigator.clipboard.writeText(code).then(() => {
        btn.textContent = '✅ Copied!';
        setTimeout(() => { btn.textContent = '📋 Copy'; }, 2000);
      });
    });
    pre.style.position = 'relative';
    pre.appendChild(btn);
  });
}

function updateSendButtonState() {
  if (!sendBtn) return;
  if (store.getState().isGenerating) {
    sendBtn.innerHTML = '■';
    sendBtn.title = 'Dừng';
    sendBtn.classList.add('stop-mode');
  } else {
    sendBtn.innerHTML = '↑';
    sendBtn.title = 'Gửi';
    sendBtn.classList.remove('stop-mode');
  }
}

function stopGeneration() {
  if (abortController) {
    abortController.abort();
    abortController = null;
  }
  store.setState({ isGenerating: false });
  updateSendButtonState();
  removeCursor();
  if (currentBotContentEl && !currentBotContentEl.innerHTML) {
    currentBotContentEl.innerHTML = '<em>Đã dừng.</em>';
  }
  if (currentBotFooterEl) {
    currentBotFooterEl.innerHTML = '<span class="msg-timer">Đã dừng</span>';
  }
  currentBotContentEl = null;
  currentBotFooterEl = null;
}

// ═══════════════════════════════════════════════════════
// LOG
// ═══════════════════════════════════════════════════════
function addLogEntry(text: string, type = 'info') {
  if (!logStream) return;
  const empty = logStream.querySelector('.log-stream-empty');
  if (empty) empty.remove();

  const el = document.createElement('div');
  el.className = `log-entry ${type}`;

  const now = new Date();
  const time = now.toLocaleTimeString('en-US', {
    hour12: false,
    minute: '2-digit',
    second: '2-digit',
  });

  const prefixMap: Record<string, string> = {
    routing: '[R]', process: '[P]', tool: '[T]', success: '[✓]', error: '[✗]',
  };
  const prefix = prefixMap[type] || '';

  el.innerHTML =
    `<span class="log-time">${time}</span>` +
    (prefix ? `<span class="log-prefix">${prefix}</span>` : '') +
    `<span class="log-text">${escHtml(text)}</span>`;

  logStream.appendChild(el);
  logStream.scrollTop = logStream.scrollHeight;
}

// ═══════════════════════════════════════════════════════
// SCROLL
// ═══════════════════════════════════════════════════════
function scrollToBottom() {
  requestAnimationFrame(() => {
    if (chatViewport) chatViewport.scrollTop = chatViewport.scrollHeight;
  });
}

// ═══════════════════════════════════════════════════════
// EVENT LISTENERS
// ═══════════════════════════════════════════════════════
sendBtn?.addEventListener('click', () => {
  if (store.getState().isGenerating) stopGeneration();
  else sendMessage();
});

chatInput?.addEventListener('keypress', (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    sendMessage();
  }
});

chatInput?.addEventListener('input', function (this: HTMLTextAreaElement) {
  this.style.height = '24px';
  this.style.height = `${Math.min(this.scrollHeight, 160)}px`;
});

// Welcome chips
document.querySelectorAll<HTMLButtonElement>('.welcome-chip[data-quick]').forEach(chip => {
  chip.addEventListener('click', () => {
    if (chatInput) chipInput(chip.dataset.quick || '');
    sendMessage();
  });
});

function chipInput(value: string) {
  if (chatInput) chatInput.value = value;
}

// Scroll-to-bottom FAB
if (scrollToBottomBtn && chatViewport) {
  chatViewport.addEventListener('scroll', () => {
    const distance = chatViewport.scrollHeight - chatViewport.scrollTop - chatViewport.clientHeight;
    scrollToBottomBtn.classList.toggle('hidden', distance <= 200);
  });
  scrollToBottomBtn.addEventListener('click', () => {
    scrollToBottom();
    scrollToBottomBtn.classList.add('hidden');
  });
}

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
  if (e.ctrlKey && !e.shiftKey && e.key === 'n') {
    e.preventDefault();
    createNewChat();
    clearChatUI();
    if (welcomeState) welcomeState.style.display = 'flex';
  }
  if (e.ctrlKey && e.shiftKey && e.key === 'S') {
    e.preventDefault();
    conversationsSidebar?.classList.toggle('collapsed');
  }
  if (e.ctrlKey && e.key === 'p') {
    e.preventDefault();
    pipelineSidebar?.classList.toggle('collapsed');
  }
  if (e.ctrlKey && (e.key === '/' || e.key === '?')) {
    e.preventDefault();
    shortcutsPanel?.classList.toggle('hidden');
  }
  if (e.key === 'Escape') {
    if (settingsModal && !settingsModal.classList.contains('hidden')) hideSettings();
    else if (metricsModal && !metricsModal.classList.contains('hidden')) hideMetricsModal();
    else if (shortcutsPanel && !shortcutsPanel.classList.contains('hidden')) {
      shortcutsPanel.classList.add('hidden');
    } else {
      pipelineSidebar?.classList.add('collapsed');
      conversationsSidebar?.classList.add('collapsed');
    }
  }
});

closeShortcuts?.addEventListener('click', () => shortcutsPanel?.classList.add('hidden'));
shortcutsPanel?.addEventListener('click', (e) => {
  if (e.target === shortcutsPanel) shortcutsPanel.classList.add('hidden');
});

newChatBtn?.addEventListener('click', async () => {
  await createNewChat();
  clearChatUI();
  if (welcomeState) welcomeState.style.display = 'flex';
});

// Run test button
const testQueries = [
  'Tổng tài sản HDB 10 năm?',
  'Chi phí dự phòng HDB 2024',
  'Tổng quát ngân hàng 2025',
  'So sánh HDB và ACB 3 năm',
];

runTestBtn?.addEventListener('click', async () => {
  if (!runTestBtn) return;
  (runTestBtn as HTMLButtonElement).disabled = true;
  addLogEntry('--- BẮT ĐẦU TEST ---', 'process');
  for (let i = 0; i < testQueries.length; i++) {
    addLogEntry(`[Test ${i + 1}/${testQueries.length}]`, 'process');
    await sendMessage(testQueries[i]);
    await new Promise(r => setTimeout(r, 3000));
  }
  addLogEntry('--- HOÀN THÀNH ---', 'success');
  (runTestBtn as HTMLButtonElement).disabled = false;
});

// ═══════════════════════════════════════════════════════
// INIT
// ═══════════════════════════════════════════════════════
await fetchConversations();
if (!store.getState().chats.length) {
  await createNewChat('Cuộc trò chuyện đầu tiên');
} else {
  store.setState({ currentChatId: store.getState().chats[0]!.id });
  if (currentChatTitle) {
    currentChatTitle.textContent = store.getState().chats[0]!.title || 'Cuộc trò chuyện mới';
  }
  renderSidebarConversations();
}
setTimeout(() => chatInput?.focus(), 100);
