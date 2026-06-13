// ═══ Indexium — Glass Chat (TypeScript + Vite) ═══
import { ApiClient } from './services/api';
import { renderMarkdown, highlightCode, extractUrls } from './services/markdown';
import { store } from './stores/app-state';
import { $, escHtml, formatTime, readFileAsBase64 } from './utils/dom';
import type { ChatSession, AttachmentPayload, TokenMetrics, HistoryMessage } from './types/api';
import './styles/main.css';

// ═══ DOM REFS ═══
const chatInput = $<HTMLTextAreaElement>('chat-input')!;
const sendBtn = $<HTMLButtonElement>('send-btn')!;
const chatContent = $('chat-content')!;
const chatViewport = $('chat-viewport')!;
const welcomeState = $('welcome-state')!;
const conversationsList = $('conversations-list')!;
const conversationsSidebar = $('conversations-sidebar')!;
const pipelineSidebar = $('pipeline-sidebar')!;
const logStream = $('log-stream')!;
const sourcesPanel = $('sources-panel')!;
const sourcesList = $('sources-list')!;
const currentChatTitle = $('current-chat-title')!;
const attachmentsPreview = $('attachments-preview')!;
const fileInput = $<HTMLInputElement>('file-input')!;
const attachBtn = $('attach-btn')!;
const backendSelect = $<HTMLSelectElement>('backend-select')!;
const searchInput = $<HTMLInputElement>('search-input')!;
const runTestBtn = $('run-test-btn');
const scrollToBottomBtn = $('scroll-to-bottom');
const sidebarBackdrop = $('sidebar-backdrop');
const settingsModal = $('settings-modal')!;
const settingsTrigger = $('settings-trigger')!;
const closeSettings = $('close-settings')!;
const closeSettingsBtn = $('close-settings-btn')!;
const saveSettingsBtn = $('save-settings-btn')!;
const geminiKeyInput = $<HTMLInputElement>('gemini-key-input')!;
const orKeysInput = $<HTMLTextAreaElement>('or-keys-input')!;
const toggleConversations = $('toggle-conversations')!;
const togglePipeline = $('toggle-pipeline')!;
const closePipeline = $('close-pipeline')!;
const themeToggle = $('theme-toggle')!;
const newChatBtn = $('new-chat-btn')!;
const shortcutsPanel = $('shortcuts-panel')!;
const closeShortcuts = $('close-shortcuts')!;
const metricsModal = $('metrics-modal')!;
const closeMetrics = $('close-metrics')!;
void $('pipe-agent');
void $('pipe-skill');
void $('pipe-tool');
void $('pipe-reason');

// ═══ STATE ═══
const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
const backends: Record<string, string> = {
  gemini: isLocalhost ? 'http://localhost:8080' : window.location.origin,
  claude: 'http://localhost:8081',
};
let currentBaseUrl = backends[store.getState().currentBackend] ?? backends['gemini']!;
let api = new ApiClient(currentBaseUrl);
let eventSource: EventSource | null = null;
let abortController: AbortController | null = null;
let pendingAttachments: File[] = [];
let currentBotContentEl: HTMLElement | null = null;
let currentBotFooterEl: HTMLElement | null = null;

// ═══ THEME ═══
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

// ═══ SIDEBAR TOGGLES ═══
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
  if (!pipelineSidebar?.classList.contains('collapsed')) logStream.scrollTop = logStream.scrollHeight;
});
closePipeline?.addEventListener('click', () => pipelineSidebar?.classList.add('collapsed'));
searchInput?.addEventListener('input', (e) => {
  const q = ((e.target as HTMLInputElement).value || '').toLowerCase();
  conversationsList.querySelectorAll('.conv-item').forEach(el => {
    const t = el.querySelector('.conv-item-title')?.textContent?.toLowerCase() || '';
    (el as HTMLElement).style.display = t.includes(q) ? '' : 'none';
  });
});

// ═══ SETTINGS MODAL ═══
function trapFocus(e: KeyboardEvent) {
  if (e.key !== 'Tab') return;
  const f = settingsModal.querySelectorAll<HTMLElement>('input, textarea, button, select');
  if (!f.length) return;
  if (e.shiftKey && document.activeElement === f[0]) { e.preventDefault(); f[f.length - 1]!.focus(); }
  else if (!e.shiftKey && document.activeElement === f[f.length - 1]) { e.preventDefault(); f[0]!.focus(); }
}
const openSettings = () => {
  settingsModal.classList.remove('hidden');
  const f = settingsModal.querySelectorAll<HTMLElement>('input, textarea, button, select');
  if (f.length) f[0]!.focus();
  settingsModal.addEventListener('keydown', trapFocus);
};
const hideSettings = () => {
  settingsModal.classList.add('hidden');
  settingsModal.removeEventListener('keydown', trapFocus);
};
settingsTrigger?.addEventListener('click', openSettings);
closeSettings?.addEventListener('click', hideSettings);
closeSettingsBtn?.addEventListener('click', hideSettings);
settingsModal?.addEventListener('click', (e) => { if (e.target === settingsModal) hideSettings(); });
// API keys are sent to server via /api/config/keys and stored server-side only
saveSettingsBtn?.addEventListener('click', async () => {
  const gk = geminiKeyInput?.value.trim() || '';
  const ok = orKeysInput?.value.trim() || '';
  if (saveSettingsBtn) { (saveSettingsBtn as HTMLButtonElement).disabled = true; saveSettingsBtn.textContent = 'Đang lưu...'; }
  try {
    await api.saveConfigKeys(ok.split('\n').map(k => k.trim()).filter(k => k));
    addLogEntry('Đã cập nhật API Keys.', 'success');
    hideSettings();
  } catch (err) {
    addLogEntry('Lỗi: ' + (err instanceof Error ? err.message : 'Không xác định'), 'error');
  } finally {
    if (saveSettingsBtn) { (saveSettingsBtn as HTMLButtonElement).disabled = false; saveSettingsBtn.textContent = 'Lưu cấu hình'; }
  }
});

// ═══ METRICS MODAL & MATH ═══
function renderMath(el: HTMLElement) {
  if (typeof (window as any).renderMathInElement === 'function') {
    (window as any).renderMathInElement(el, {
      delimiters: [
        {left: '$$', right: '$$', display: true},
        {left: '$', right: '$', display: false},
        {left: '\\(', right: '\\)', display: false},
        {left: '\\[', right: '\\]', display: true}
      ],
      throwOnError: false
    });
  }
}
function showMetricsModal(met: TokenMetrics) {
  if (!metricsModal) return;
  const lat = met.latency_ms ? (met.latency_ms / 1000).toFixed(2) + 's' : '—';
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

// ═══ BACKEND SELECTOR ═══
backendSelect?.addEventListener('change', (e) => {
  const b = (e.target as HTMLSelectElement).value;
  currentBaseUrl = backends[b] ?? backends['gemini']!;
  api = new ApiClient(currentBaseUrl);
  setupEventSource();
  store.setState({ currentBackend: b as 'gemini' | 'claude' });
  addLogEntry(`Đã chuyển sang ${b}`, 'info');
});

// ═══ SSE ═══
function setupEventSource() {
  if (eventSource) eventSource.close();
  eventSource = new EventSource(`${currentBaseUrl}/api/events`);
  eventSource.onmessage = (e) => {
    try { const d = JSON.parse(e.data); addLogEntry(d.payload || '', d.type || 'info'); updatePipeline(d); } catch { /* */ }
  };
  eventSource.onerror = () => addLogEntry('Mất kết nối SSE.', 'error');
}
function updatePipeline(d: { type: string; payload?: string; metadata?: Record<string, string> }) {
  const msg = String(d.payload || '');
  const m = d.metadata || {};
  if (d.type === 'agent_selected') store.setPipeline({ agent: m.agent || msg, agentStatus: 'active', reason: m.reason || '', reasonStatus: 'active' });
  if (d.type === 'skill_loaded') store.setPipeline({ skill: m.skill || msg, skillStatus: 'active' });
  if (d.type === 'tool_executed') store.setPipeline({ tool: m.tool || msg, toolStatus: 'active' });
}

// ═══ CONVERSATIONS ═══
async function fetchConversations() {
  try { store.setState({ chats: await api.getSessions() }); } catch { store.setState({ chats: [] }); }
  renderConversations();
}
function renderConversations() {
  if (!conversationsList) return;
  const chats = store.getState().chats;
  conversationsList.innerHTML = '';
  if (!chats.length) { conversationsList.innerHTML = '<div style="padding:16px;font-size:13px;opacity:0.5;text-align:center;">Chưa có cuộc trò chuyện.</div>'; return; }
  const now = new Date(), today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const y = new Date(today); y.setDate(y.getDate() - 1);
  const w7 = new Date(today); w7.setDate(w7.getDate() - 7);
  const m30 = new Date(today); m30.setDate(m30.getDate() - 30);
  const groups: Record<string, ChatSession[]> = { 'Hôm nay': [], 'Hôm qua': [], '7 ngày trước': [], '30 ngày trước': [], 'Cũ hơn': [] };
  for (const c of chats) { const d = c.updated_at ? new Date(c.updated_at) : new Date(0); if (d >= today) groups['Hôm nay']!.push(c); else if (d >= y) groups['Hôm qua']!.push(c); else if (d >= w7) groups['7 ngày trước']!.push(c); else if (d >= m30) groups['30 ngày trước']!.push(c); else groups['Cũ hơn']!.push(c); }
  for (const [label, gc] of Object.entries(groups)) {
    if (!gc.length) continue;
    const sl = document.createElement('div'); sl.className = 'sidebar-section-label'; sl.textContent = label; conversationsList.appendChild(sl);
    for (const chat of gc) {
      const isActive = chat.id === store.getState().currentChatId;
      const el = document.createElement('div'); el.className = `conv-item ${isActive ? 'active' : ''}`;
      el.innerHTML = `<span class="conv-item-title">${escHtml(chat.title || 'Cuộc trò chuyện mới')}</span><span class="conv-item-meta">${formatTime(chat.updated_at)}</span><button class="conv-item-delete" title="Xóa">✕</button>`;
      el.querySelector('.conv-item-title')?.addEventListener('click', () => switchChat(chat.id));
      el.querySelector('.conv-item-delete')?.addEventListener('click', async (e) => { e.stopPropagation(); if (confirm('Xóa?')) await deleteChat(chat.id); });
      const ts = el.querySelector('.conv-item-title') as HTMLElement;
      ts.addEventListener('dblclick', (e) => {
        e.stopPropagation();
        const inp = document.createElement('input'); inp.type = 'text'; inp.className = 'conv-rename-input'; inp.value = chat.title || '';
        ts.style.display = 'none'; ts.parentNode?.insertBefore(inp, ts); inp.focus(); inp.select();
        const fin = async (save: boolean) => { const t = inp.value.trim(); inp.remove(); ts.style.display = ''; if (save && t && t !== chat.title) { chat.title = t; ts.textContent = escHtml(t); if (chat.id === store.getState().currentChatId && currentChatTitle) currentChatTitle.textContent = t; } };
        inp.addEventListener('blur', () => fin(true));
        inp.addEventListener('keydown', (ke) => { if (ke.key === 'Enter') { ke.preventDefault(); inp.blur(); } else if (ke.key === 'Escape') { ke.preventDefault(); fin(false); } });
      });
      conversationsList.appendChild(el);
    }
  }
}
async function deleteChat(id: string) {
  try { await api.deleteSession(id); store.setState({ chats: store.getState().chats.filter(c => c.id !== id) }); addLogEntry('Đã xóa.', 'success'); if (store.getState().currentChatId === id) { const chats = store.getState().chats; if (chats.length > 0) await switchChat(chats[0]!.id); else await createNewChat(); } else renderConversations(); } catch (err) { addLogEntry('Lỗi: ' + (err instanceof Error ? err.message : 'Lỗi'), 'error'); }
}
async function createNewChat(title = 'Cuộc trò chuyện mới') {
  try { const c = await api.createSession(title); store.setState({ chats: [c, ...store.getState().chats], currentChatId: c.id, currentChatTitle: title }); clearChatUI(); if (welcomeState) welcomeState.style.display = 'flex'; if (currentChatTitle) currentChatTitle.textContent = title; renderConversations(); addLogEntry('Đã tạo mới.', 'success'); return c; } catch { const fb: ChatSession = { id: 'local_' + Date.now(), title }; store.setState({ chats: [fb, ...store.getState().chats], currentChatId: fb.id, currentChatTitle: title }); clearChatUI(); if (welcomeState) welcomeState.style.display = 'flex'; if (currentChatTitle) currentChatTitle.textContent = title; renderConversations(); return fb; }
}
function groupHistoryMessages(history: HistoryMessage[]): { role: 'user' | 'bot'; content: string; metrics?: TokenMetrics }[] {
  const grouped: { role: 'user' | 'bot'; content: string; metrics?: TokenMetrics }[] = [];
  for (let i = 0; i < history.length; i++) {
    const msg = history[i];
    if (!msg) continue;
    const role = String(msg.role || '').toLowerCase();
    if (role === 'tool') continue;
    if (role === 'user') {
      grouped.push({ role: 'user', content: msg.content || '' });
      continue;
    }
    let isThinking = false;
    for (let j = i + 1; j < history.length; j++) {
      const nextMsg = history[j];
      if (!nextMsg) continue;
      const nextRole = String(nextMsg.role || '').toLowerCase();
      if (nextRole === 'user') break;
      if (nextRole === 'assistant') {
        isThinking = true;
        break;
      }
    }
    const text = msg.content || '';
    if (!text) continue;
    let formattedContent = text;
    if (isThinking) {
      formattedContent = `<details class="thinking-details"><summary class="thinking-summary">Suy nghĩ của AI</summary><div class="thinking-content">${text}</div></details>\n\n`;
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
          cpu_load: msg.cpu_load
        };
      }
    } else {
      const metrics: TokenMetrics | undefined = msg.token_in !== undefined && msg.token_in > 0 ? {
        token_in: msg.token_in,
        token_out: msg.token_out || 0,
        ram_mb: msg.ram_mb || '',
        latency_ms: msg.latency_ms,
        cpu_load: msg.cpu_load
      } : undefined;
      grouped.push({ role: 'bot', content: formattedContent, metrics });
    }
  }
  return grouped;
}

async function switchChat(id: string) {
  store.setState({ currentChatId: id }); const chat = store.getState().chats.find(c => c.id === id);
  if (currentChatTitle) currentChatTitle.textContent = chat?.title || id; clearChatUI(); renderConversations();
  const sk = document.createElement('div'); sk.className = 'skeleton-loading'; sk.innerHTML = '<div class="skeleton-message"><div class="skeleton-avatar"></div><div class="skeleton-body"><div class="skeleton-line short"></div><div class="skeleton-line"></div></div></div>'; chatContent?.appendChild(sk);
  try {
    const data = await api.getHistory(id);
    sk.remove();
    let has = false, idx = 0;
    const grouped = groupHistoryMessages(data.history);
    for (const g of grouped) {
      appendMessageBubble(g.content, g.role, false, g.metrics);
      const msgs = chatContent.querySelectorAll('.message');
      (msgs[msgs.length - 1] as HTMLElement).style.animationDelay = `${idx * 0.06}s`;
      idx++;
      has = true;
    }
    if (has && welcomeState) welcomeState.style.display = 'none';
    else if (welcomeState) welcomeState.style.display = 'flex';
    scrollToBottom();
  } catch {
    sk.remove();
    if (welcomeState) welcomeState.style.display = 'flex';
  }
  addLogEntry(`Chuyển: ${chat?.title || id}`, 'info');
}
function clearChatUI() {
  if (!chatContent) return;
  chatContent.querySelectorAll('.message, .skeleton-loading, .thinking-card').forEach(m => m.remove());
  if (logStream) logStream.innerHTML = '<div class="log-stream-empty">Sẵn sàng.</div>';
  if (sourcesPanel) sourcesPanel.style.display = 'none';
  if (sourcesList) sourcesList.innerHTML = '';
  store.resetPipeline();
}

// ═══ ATTACHMENTS ═══
attachBtn?.addEventListener('click', () => fileInput?.click());
fileInput?.addEventListener('change', () => { if (fileInput.files?.length) { pendingAttachments = [...pendingAttachments, ...Array.from(fileInput.files)]; renderAttachments(); fileInput.value = ''; } });
function renderAttachments() {
  if (!attachmentsPreview) return; attachmentsPreview.innerHTML = '';
  pendingAttachments.forEach((file, i) => { const ch = document.createElement('div'); ch.className = 'attachment-chip'; ch.innerHTML = `<span>${file.type.startsWith('image/') ? '🖼️' : '📎'}</span><span class="attachment-chip-name">${escHtml(file.name)}</span><button class="attachment-remove-btn">✕</button>`; ch.querySelector('.attachment-remove-btn')?.addEventListener('click', (e) => { e.stopPropagation(); pendingAttachments.splice(i, 1); renderAttachments(); }); attachmentsPreview.appendChild(ch); });
}

// ═══ MESSAGES ═══
function appendMessageBubble(text: string, sender: 'user' | 'bot', isStreaming = false, metrics?: TokenMetrics) {
  if (!chatContent) return;
  if (welcomeState) welcomeState.style.display = 'none';
  const el = document.createElement('div'); el.className = `message ${sender}`;
  const avatar = document.createElement('div'); avatar.className = 'msg-avatar'; avatar.textContent = sender === 'user' ? 'U' : 'IX';
  const body = document.createElement('div'); body.className = 'msg-body';
  const sl = document.createElement('div'); sl.className = 'msg-sender'; sl.textContent = sender === 'user' ? 'Bạn' : 'Indexium AI';
  const content = document.createElement('div'); content.className = 'msg-content glass-panel';
  if (sender === 'bot') { if (isStreaming) content.innerHTML = ''; else content.innerHTML = renderMarkdown(text); } else content.textContent = text;
  body.appendChild(sl); body.appendChild(content);
  const footer = document.createElement('div'); footer.className = 'msg-footer';
  if (sender === 'bot') {
    if (isStreaming) {
      footer.innerHTML = '<span class="msg-timer streaming">Đang trả lời…</span>';
    } else if (metrics) {
      const btn = document.createElement('button');
      btn.className = 'metrics-badge';
      const latSec = metrics.latency_ms ? (metrics.latency_ms / 1000).toFixed(1) + 's' : '—';
      btn.innerHTML = `📊 ${latSec} · ↑${metrics.token_in} · ↓${metrics.token_out} · ${metrics.ram_mb}`;
      btn.title = 'Xem chi tiết hiệu năng';
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        showMetricsModal(metrics);
      });
      footer.appendChild(btn);
    } else {
      footer.innerHTML = '<span class="msg-timer">—</span>';
    }
    body.appendChild(footer);
  }
  if (sender === 'bot' && !isStreaming) {
    const actions = document.createElement('div'); actions.className = 'msg-actions';
    const cb = document.createElement('button'); cb.className = 'msg-action-btn'; cb.title = 'Copy'; cb.textContent = '📋'; cb.addEventListener('click', () => { navigator.clipboard.writeText(content.innerText).then(() => { cb.textContent = '✅'; setTimeout(() => { cb.textContent = '📋'; }, 2000); }); }); actions.appendChild(cb);
    const rb = document.createElement('button'); rb.className = 'msg-action-btn'; rb.title = 'Tạo lại'; rb.textContent = '🔄'; rb.addEventListener('click', () => { const msgs = chatContent!.querySelectorAll('.message'); let prev: string | null = null; for (let i = msgs.length - 1; i >= 0; i--) { const msg = msgs[i]!; if (msg === el) continue; if (msg.classList.contains('user')) { prev = msg.querySelector('.msg-content')?.textContent || null; break; } } if (prev) { el.remove(); sendMessage(prev, true); } }); actions.appendChild(rb);
    const tu = document.createElement('button'); tu.className = 'msg-action-btn'; tu.title = 'Hữu ích'; tu.textContent = '👍'; tu.addEventListener('click', () => { tu.classList.toggle('active'); td.classList.remove('active'); }); actions.appendChild(tu);
    const td = document.createElement('button'); td.className = 'msg-action-btn'; td.title = 'Chưa hữu ích'; td.textContent = '👎'; td.addEventListener('click', () => { td.classList.toggle('active'); tu.classList.remove('active'); }); actions.appendChild(td);
    body.appendChild(actions);
  }
  if (sender === 'user' && !isStreaming) {
    el.addEventListener('click', function handler(evt) { if ((evt.target as HTMLElement).closest('.edit-message-container')) return; el.removeEventListener('click', handler); const orig = content.textContent || ''; const cont = document.createElement('div'); cont.className = 'edit-message-container'; const ta = document.createElement('textarea'); ta.className = 'edit-message-textarea'; ta.value = orig; ta.rows = 3; const aa = document.createElement('div'); aa.className = 'edit-message-actions'; const sb = document.createElement('button'); sb.className = 'btn btn-primary btn-sm'; sb.textContent = 'Gửi lại'; const ca = document.createElement('button'); ca.className = 'btn btn-ghost btn-sm'; ca.textContent = 'Hủy'; aa.appendChild(ca); aa.appendChild(sb); cont.appendChild(ta); cont.appendChild(aa); content.style.display = 'none'; body.insertBefore(cont, content.nextSibling); ta.focus(); ca.addEventListener('click', (e) => { e.stopPropagation(); cont.remove(); content.style.display = ''; el.addEventListener('click', handler); }); sb.addEventListener('click', (e) => { e.stopPropagation(); const nt = ta.value.trim(); if (!nt) return; let sib = el.nextElementSibling; while (sib) { const n = sib.nextElementSibling; if (sib.classList.contains('message') || sib.classList.contains('followup-chips')) sib.remove(); sib = n; } el.remove(); sendMessage(nt); }); });
  }
  el.appendChild(avatar); el.appendChild(body); chatContent.appendChild(el); scrollToBottom();
  if (sender === 'bot' && !isStreaming && text) { renderSourcesInline(text, content); highlightCode(content); renderMath(content); }
  if (sender === 'bot' && !isStreaming) { const chips = document.createElement('div'); chips.className = 'followup-chips'; ['Giải thích thêm chi tiết', 'So sánh với chỉ số khác', 'Tóm tắt ngắn gọn'].forEach(s => { const c = document.createElement('button'); c.className = 'followup-chip'; c.textContent = s; c.addEventListener('click', () => { if (chatInput) chatInput.value = s; sendMessage(); }); chips.appendChild(c); }); body.appendChild(chips); }
  return { el, content, footer };
}
function renderSourcesInline(md: string, target: HTMLElement) { if (target.querySelector('.message-sources')) return; const urls = extractUrls(md); if (!urls.length) return; const c = document.createElement('div'); c.className = 'message-sources'; c.innerHTML = '<div class="sources-label">Nguồn tham khảo</div>'; const g = document.createElement('div'); g.className = 'sources-grid'; urls.forEach((url, i) => { let d = url; try { d = new URL(url).hostname.replace('www.', ''); } catch { /* */ } const a = document.createElement('a'); a.href = url; a.target = '_blank'; a.rel = 'noopener noreferrer'; a.className = 'source-chip'; a.innerHTML = `<span class="source-chip-index">${i + 1}</span>${d}`; g.appendChild(a); }); c.appendChild(g); target.appendChild(c); if (sourcesPanel) sourcesPanel.style.display = 'block'; if (sourcesList) { sourcesList.innerHTML = ''; urls.forEach((url, i) => { let d = url; try { d = new URL(url).hostname.replace('www.', ''); } catch { /* */ } const a = document.createElement('a'); a.href = url; a.target = '_blank'; a.rel = 'noopener noreferrer'; a.className = 'source-link-item'; a.innerHTML = `<span class="source-link-favicon">${i + 1}</span><span class="source-link-url">${d}</span>`; sourcesList.appendChild(a); }); } }

// ═══ SEND MESSAGE ═══
async function sendMessage(textOverride: string | null = null, isRegenerate = false) {
  const text = textOverride !== null ? textOverride : (chatInput?.value.trim() || '');
  if (!text && pendingAttachments.length === 0) return;
  if (!isRegenerate) { if (!store.getState().currentChatId) await createNewChat(); appendMessageBubble(text, 'user'); }
  const files = [...pendingAttachments]; pendingAttachments = []; renderAttachments();
  if (chatInput) { chatInput.value = ''; chatInput.style.height = '24px'; }
  store.setState({ isGenerating: true }); updateSendButtonState();
  const tc = document.createElement('div'); tc.className = 'thinking-card glass-panel'; tc.innerHTML = '<div class="thinking-header"><div class="thinking-spinner"></div><span class="thinking-label">AI đang xử lý</span></div><div class="thinking-steps"><div class="thinking-step active" data-step="agent"><div class="thinking-step-icon agent">●</div><div class="thinking-step-content"><div class="thinking-step-label">Agent</div><div class="thinking-step-text">Đang nhận dạng...</div></div></div><div class="thinking-step pending" data-step="skill"><div class="thinking-step-icon skill">○</div><div class="thinking-step-content"><div class="thinking-step-label">Skill</div><div class="thinking-step-text">Phân tích intent...</div></div></div><div class="thinking-step pending" data-step="tool"><div class="thinking-step-icon tool">○</div><div class="thinking-step-content"><div class="thinking-step-label">Công cụ</div><div class="thinking-step-text">LLM Routing...</div></div></div><div class="thinking-step pending" data-step="reason"><div class="thinking-step-icon reason">○</div><div class="thinking-step-content"><div class="thinking-step-label">Lý do</div><div class="thinking-step-text">Xác định agent tối ưu...</div></div></div></div>'; chatContent?.appendChild(tc);
  const st = Date.now();
  const { content, footer } = appendMessageBubble('', 'bot', true) as { content: HTMLElement; footer: HTMLElement };
  currentBotContentEl = content; currentBotFooterEl = footer;
  abortController = new AbortController();
  try {
    const att: AttachmentPayload[] = []; for (const f of files) att.push({ name: f.name, type: f.type, data: await readFileAsBase64(f) });
    let full = '';
    await api.streamChat(text, store.getState().currentChatId || undefined, att, abortController.signal, (tok) => { full += tok; if (currentBotContentEl) { currentBotContentEl.innerHTML = renderMarkdown(full); scrollToBottom(); } }, (met) => {
      const el = ((Date.now() - st) / 1000).toFixed(1);
      if (currentBotContentEl) {
        currentBotContentEl.innerHTML = renderMarkdown(full);
        highlightCode(currentBotContentEl);
        renderMath(currentBotContentEl);
      }
      if (currentBotFooterEl) {
        currentBotFooterEl.innerHTML = '';
        const btn = document.createElement('button');
        btn.className = 'metrics-badge';
        btn.innerHTML = `📊 ${el}s · ↑${met.token_in} · ↓${met.token_out} · ${met.ram_mb}`;
        btn.title = 'Xem chi tiết hiệu năng';
        if (!met.latency_ms) {
          met.latency_ms = Date.now() - st;
        }
        btn.addEventListener('click', (e) => {
          e.stopPropagation();
          showMetricsModal(met);
        });
        currentBotFooterEl.appendChild(btn);
      }
      addCopyButtons(currentBotContentEl!);
      tc.remove();
      fetchConversations();
    }, (err) => { throw new Error(err); });
    store.setState({ isGenerating: false }); updateSendButtonState(); abortController = null; currentBotContentEl = null; currentBotFooterEl = null;
  } catch (error) {
    store.setState({ isGenerating: false }); updateSendButtonState(); abortController = null;
    if ((error as Error).name === 'AbortError') return;
    tc.remove(); const el = ((Date.now() - st) / 1000).toFixed(1); const d = diagnoseError((error as Error).name, (error as Error).message);
    if (currentBotContentEl) currentBotContentEl.innerHTML = renderErrorHTML(d);
    if (currentBotFooterEl) currentBotFooterEl.innerHTML = `<span class="msg-timer" style="color:var(--danger)">Thất bại · ${el}s</span>`;
    currentBotContentEl = null; currentBotFooterEl = null;
  } finally { if (sendBtn) (sendBtn as HTMLButtonElement).disabled = false; chatInput?.focus(); scrollToBottom(); }
}
function addCopyButtons(el: HTMLElement) { el.querySelectorAll('pre').forEach(pre => { if (pre.querySelector('.copy-code-btn')) return; const b = document.createElement('button'); b.className = 'copy-code-btn'; b.textContent = '📋 Copy'; b.addEventListener('click', () => { const c = pre.querySelector('code')?.innerText || pre.innerText; navigator.clipboard.writeText(c).then(() => { b.textContent = '✅ Copied!'; setTimeout(() => { b.textContent = '📋 Copy'; }, 2000); }); }); pre.style.position = 'relative'; pre.appendChild(b); }); }
function updateSendButtonState() { if (!sendBtn) return; if (store.getState().isGenerating) { sendBtn.innerHTML = '■'; sendBtn.title = 'Dừng'; sendBtn.classList.add('stop-mode'); } else { sendBtn.innerHTML = '↑'; sendBtn.title = 'Gửi'; sendBtn.classList.remove('stop-mode'); } }
function stopGeneration() { if (abortController) { abortController.abort(); abortController = null; } store.setState({ isGenerating: false }); updateSendButtonState(); if (currentBotContentEl && !currentBotContentEl.innerHTML) currentBotContentEl.innerHTML = '<em>Đã dừng.</em>'; if (currentBotFooterEl) currentBotFooterEl.innerHTML = '<span class="msg-timer">Đã dừng</span>'; currentBotContentEl = null; currentBotFooterEl = null; }

// ═══ ERROR ═══
interface ErrDiag { badge: string; title: string; icon: string; desc: string; suggestions: string[]; name: string; msg: string; }
function diagnoseError(name: string, msg: string): ErrDiag {
  const m = (msg || '').toLowerCase(); let b = '⚙️', t = 'Lỗi xử lý', i = '⚠️', d = msg || 'Lỗi không xác định.'; let s = ['Kiểm tra API Key.', 'Thử lại.'];
  if (m.includes('failed to fetch') || m.includes('network')) { b = '🔌'; t = 'Không kết nối'; i = '📡'; d = 'Không thể kết nối đến máy chủ. Vui lòng thử lại sau.'; s = ['Kiểm tra kết nối mạng.', 'Thử lại sau vài giây.']; }
  else if (m.includes('quota') || m.includes('429')) { b = '📊'; t = 'Hết hạn mức (429)'; i = '⛔'; d = 'Vượt quá giới hạn API.'; s = ['Thêm API Key dự phòng.', 'Chờ 1-2 phút.']; }
  else if (m.includes('thought_signature')) { b = '🤖'; t = 'Lỗi tương thích'; i = '🔧'; s = ['Đổi GEMINI_MODEL.', 'Khởi động lại backend.']; }
  return { badge: b, title: t, icon: i, desc: d, suggestions: s, name, msg };
}
function renderErrorHTML(d: ErrDiag): string { return `<div class="error-container"><div class="error-header"><span class="error-title">${d.icon} ${d.title}</span><span class="error-badge">${d.badge}</span></div><div class="error-body">${d.desc}</div><div class="error-suggestions"><div class="error-suggestions-title">💡 Đề xuất</div>${d.suggestions.map(x => `<div class="error-suggestion-item">${x}</div>`).join('')}</div><details class="error-details"><summary>Chi tiết kỹ thuật</summary><div class="error-details-content">[${d.name}] ${escHtml(d.msg)}</div></details></div>`; }

// ═══ LOG ═══
function addLogEntry(text: string, type = 'info') {
  if (!logStream) return; const e = logStream.querySelector('.log-stream-empty'); if (e) e.remove();
  const el = document.createElement('div'); el.className = `log-entry ${type}`;
  const now = new Date(); const tm = now.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });
  let p = ''; if (type === 'routing') p = '[R]'; else if (type === 'process') p = '[P]'; else if (type === 'tool') p = '[T]'; else if (type === 'success') p = '[✓]'; else if (type === 'error') p = '[✗]';
  el.innerHTML = `<span class="log-time">${tm}</span>${p ? `<span class="log-prefix">${p}</span>` : ''}<span class="log-text">${escHtml(text)}</span>`;
  logStream.appendChild(el); logStream.scrollTop = logStream.scrollHeight;
}

// ═══ UTILS ═══
function scrollToBottom() { requestAnimationFrame(() => { if (chatViewport) chatViewport.scrollTop = chatViewport.scrollHeight; }); }

// ═══ EVENTS ═══
sendBtn?.addEventListener('click', () => { if (store.getState().isGenerating) stopGeneration(); else sendMessage(); });
chatInput?.addEventListener('keypress', (e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); } });
chatInput?.addEventListener('input', function(this: HTMLTextAreaElement) { this.style.height = '24px'; this.style.height = Math.min(this.scrollHeight, 160) + 'px'; });
document.querySelectorAll<HTMLButtonElement>('.welcome-chip[data-quick]').forEach(c => { c.addEventListener('click', () => { if (chatInput) chatInput.value = c.dataset.quick || ''; sendMessage(); }); });
if (scrollToBottomBtn && chatViewport) { chatViewport.addEventListener('scroll', () => { scrollToBottomBtn.classList.toggle('hidden', chatViewport.scrollHeight - chatViewport.scrollTop - chatViewport.clientHeight <= 200); }); scrollToBottomBtn.addEventListener('click', () => { scrollToBottom(); scrollToBottomBtn.classList.add('hidden'); }); }
sidebarBackdrop?.addEventListener('click', () => { conversationsSidebar?.classList.add('collapsed'); sidebarBackdrop.classList.remove('visible'); sidebarBackdrop.classList.add('hidden'); });
document.addEventListener('keydown', (e) => { if (e.ctrlKey && !e.shiftKey && e.key === 'n') { e.preventDefault(); createNewChat(); clearChatUI(); if (welcomeState) welcomeState.style.display = 'flex'; } if (e.ctrlKey && e.shiftKey && e.key === 'S') { e.preventDefault(); conversationsSidebar?.classList.toggle('collapsed'); } if (e.ctrlKey && e.key === 'p') { e.preventDefault(); pipelineSidebar?.classList.toggle('collapsed'); } if (e.ctrlKey && (e.key === '/' || e.key === '?')) { e.preventDefault(); shortcutsPanel?.classList.toggle('hidden'); } if (e.key === 'Escape') { if (settingsModal && !settingsModal.classList.contains('hidden')) hideSettings(); else if (metricsModal && !metricsModal.classList.contains('hidden')) hideMetricsModal(); else if (shortcutsPanel && !shortcutsPanel.classList.contains('hidden')) shortcutsPanel.classList.add('hidden'); else { pipelineSidebar?.classList.add('collapsed'); conversationsSidebar?.classList.add('collapsed'); } } });
closeShortcuts?.addEventListener('click', () => shortcutsPanel?.classList.add('hidden'));
shortcutsPanel?.addEventListener('click', (e) => { if (e.target === shortcutsPanel) shortcutsPanel.classList.add('hidden'); });
newChatBtn?.addEventListener('click', async () => { await createNewChat(); clearChatUI(); if (welcomeState) welcomeState.style.display = 'flex'; });
const tq = ['Tổng tài sản HDB 10 năm?', 'Chi phí dự phòng HDB 2024', 'Tổng quát ngân hàng 2025', 'So sánh HDB và ACB 3 năm'];
runTestBtn?.addEventListener('click', async () => { if (!runTestBtn) return; (runTestBtn as HTMLButtonElement).disabled = true; addLogEntry('--- BẮT ĐẦU TEST ---', 'process'); for (let i = 0; i < tq.length; i++) { addLogEntry(`[Test ${i + 1}/${tq.length}]`, 'process'); await sendMessage(tq[i]); await new Promise(r => setTimeout(r, 3000)); } addLogEntry('--- HOÀN THÀNH ---', 'success'); (runTestBtn as HTMLButtonElement).disabled = false; });

// ═══ INIT ═══
setupEventSource();
await fetchConversations();
if (!store.getState().chats.length) await createNewChat('Cuộc trò chuyện đầu tiên');
else { store.setState({ currentChatId: store.getState().chats[0]!.id }); if (currentChatTitle) currentChatTitle.textContent = store.getState().chats[0]!.title || 'Cuộc trò chuyện mới'; renderConversations(); }
setTimeout(() => chatInput?.focus(), 100);
// Indexium Glass Chat ready.
