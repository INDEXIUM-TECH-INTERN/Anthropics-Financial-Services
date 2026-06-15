// ═══ Chat Page ═══
// Composes all widgets and features into the chat page.

import { ApiClient } from '../../shared/api/client';
import { SSEManager } from '../../shared/api/sse';
import { $, escHtml, readFileAsBase64 } from '../../shared/lib/dom';
import { renderMarkdown } from '../../shared/lib/markdown';
import { showToast } from '../../shared/ui/toast';
import { createConnectionStatus } from '../../shared/ui/connection-status';
import type { ChatSession, AttachmentPayload, TokenMetrics } from '../../shared/api/types';

import { createChatViewWidget } from '../../widgets/chat-view/compose';
import { createSidebarWidget } from '../../widgets/sidebar/compose';
import { createPipelineWidget } from '../../widgets/pipeline/compose';

import {
  $currentChatId,
  $isGenerating,
  $pipeline,
  setChatId,
  setChatTitle,
  setGenerating,
  setPipeline,
} from '../../entities/chat/model/store';

import { $sessions, setSessions, addSession, removeSession } from '../../entities/session/model/store';

import { fetchSessions, createNewSession, removeSession as deleteSession } from '../../entities/session/api/crud';
import { fetchHistory, groupHistoryMessages } from '../../entities/chat/api/history';

import { $sidebarOpen, toggleSidebar, setSidebarOpen } from '../../features/sidebar/toggle/model';
import { $settingsOpen, closeSettings } from '../../features/settings/modal/model';
import { setTheme, type ThemeMode } from '../../features/theme/toggle';

export interface ChatPage {
  init: () => Promise<void>;
  destroy: () => void;
}

export function createChatPage(): ChatPage {
  // ═══ API & SSE ═══
  const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
  const backends: Record<string, string> = {
    gemini: isLocalhost ? 'http://localhost:8080' : window.location.origin,
    claude: 'http://localhost:8081',
  };
  let currentBaseUrl = backends.gemini!;
  let api = new ApiClient(currentBaseUrl);
  const sseManager = new SSEManager();
  let abortController: AbortController | null = null;
  let pendingAttachments: File[] = [];
  let currentBotContentEl: HTMLElement | null = null;
  let currentBotFooterEl: HTMLElement | null = null;
  let fullText = '';

  // ═══ DEBOUNCED RENDER ═══
  // Batches markdown rendering during streaming to avoid O(n²) re-renders.
  let renderScheduled = false;
  let pendingFullText = '';
  const scheduleRender = () => {
    pendingFullText = fullText;
    if (renderScheduled) return;
    renderScheduled = true;
    requestAnimationFrame(() => {
      renderScheduled = false;
      if (currentBotContentEl) {
        currentBotContentEl.innerHTML = renderMarkdown(pendingFullText);
        chatWidget.scrollToBottom();
      }
    });
  };

  // ═══ DOM REFS ═══
  const chatInput = $<HTMLTextAreaElement>('chat-input')!;
  const sendBtn = $<HTMLButtonElement>('send-btn')!;
  const chatContent = $('chat-content')!;
  const chatViewport = $('chat-viewport')!;
  const welcomeState = $('welcome-state')!;
  const conversationsList = $('conversations-list')!;
  const pipelineSidebar = $('pipeline-sidebar');
  const logStream = $('log-stream');
  const currentChatTitleEl = $('current-chat-title');
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
  const closeSettingsBtn = $('close-settings');
  const closeSettingsBtn2 = $('close-settings-btn');
  const saveSettingsBtn = $('save-settings-btn');
  const orKeysInput = $<HTMLTextAreaElement>('or-keys-input');
  const toggleConversations = $('toggle-conversations');
  const togglePipeline = $('toggle-pipeline');
  const closePipeline = $('close-pipeline');
  const themeToggle = $('theme-toggle');
  const newChatBtn = $('newChat-btn');
  const shortcutsPanel = $('shortcuts-panel');
  const closeShortcuts = $('close-shortcuts');
  const metricsModal = $('metrics-modal');
  const closeMetrics = $('close-metrics');

  // ═══ WIDGETS ═══
  const chatWidget = createChatViewWidget(chatContent, chatViewport);
  const sidebarWidget = createSidebarWidget(conversationsList, {
    onSelect: (id) => switchChat(id),
    onDelete: (id) => deleteChat(id),
  });
  const pipelineTarget = pipelineSidebar?.querySelector('.pipeline-steps') as HTMLElement | null;
  const pipelineWidget = pipelineTarget ? createPipelineWidget(pipelineTarget) : null;

  // ═══ CONNECTION STATUS ═══
  const headerEl = document.querySelector('.header-right') as HTMLElement | null;
  let connStatus: ReturnType<typeof createConnectionStatus> | null = null;

  function initConnectionStatus() {
    if (!headerEl) return;
    connStatus = createConnectionStatus(headerEl, {
      onReconnect() {
        connStatus?.setStatus('reconnecting');
        sseManager.disconnect();
        sseManager.connect(currentBaseUrl, {
          onMessage: (d) => handleSSEMessage(d),
          onError: (msg) => {
            addLogEntry(msg, 'error');
            connStatus?.setStatus('disconnected');
          },
        });
        fetch(`${currentBaseUrl}/`, { method: 'HEAD', signal: AbortSignal.timeout(5000) })
          .then((res) => {
            connStatus?.setStatus(res.ok ? 'connected' : 'disconnected');
          })
          .catch(() => {
            connStatus?.setStatus('disconnected');
          });
      },
    });
    fetch(`${currentBaseUrl}/`, { method: 'HEAD', signal: AbortSignal.timeout(5000) })
      .then((res) => {
        connStatus?.setStatus(res.ok ? 'connected' : 'disconnected');
      })
      .catch(() => {
        connStatus?.setStatus('disconnected');
      });
  }

  // ═══ SSE ═══
  function handleSSEMessage(d: Record<string, unknown>) {
    const type = d.type as string;
    const payload = String(d.payload || '');
    const metadata = (d.metadata || {}) as Record<string, string>;
    addLogEntry(payload, type || 'info');
    if (type === 'agent_selected') {
      setPipeline({
        agent: metadata.agent || payload,
        agentStatus: 'active',
        reason: metadata.reason || '',
        reasonStatus: 'active',
      });
    }
    if (type === 'skill_loaded') {
      setPipeline({ skill: metadata.skill || payload, skillStatus: 'active' });
    }
    if (type === 'tool_executed') {
      setPipeline({ tool: metadata.tool || payload, toolStatus: 'active' });
    }
  }

  // ═══ LOG ═══
  function addLogEntry(text: string, type = 'info') {
    if (!logStream) return;
    const empty = logStream.querySelector('.log-stream-empty');
    if (empty) empty.remove();
    const el = document.createElement('div');
    el.className = `log-entry ${type}`;
    const now = new Date();
    const timeStr = now.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });
    const prefixMap: Record<string, string> = {
      routing: '[R]',
      process: '[P]',
      tool: '[T]',
      success: '[✓]',
      error: '[✗]',
    };
    el.innerHTML = `<span class="log-time">${timeStr}</span>${prefixMap[type] ? `<span class="log-prefix">${prefixMap[type]}</span>` : ''}<span class="log-text">${escHtml(text)}</span>`;
    logStream.appendChild(el);
    logStream.scrollTop = logStream.scrollHeight;
  }

  // ═══ THEME ═══
  let currentTheme = (localStorage.getItem('theme') as ThemeMode) ?? 'dark';

  function applyTheme(theme: ThemeMode) {
    currentTheme = theme;
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);
    setTheme(theme);
  }

  applyTheme(currentTheme);

  // ═══ CONVERSATIONS ═══
  async function loadConversations() {
    try {
      const sessions = await fetchSessions(api);
      setSessions(sessions);
      sidebarWidget.render(sessions, $currentChatId.get());
    } catch {
      setSessions([]);
      sidebarWidget.render([], null);
    }
  }

  async function deleteChat(id: string) {
    try {
      await deleteSession(api, id);
      removeSession(id);
      addLogEntry('Đã xóa.', 'success');
      showToast({ message: 'Đã xóa cuộc trò chuyện', type: 'success' });
      if ($currentChatId.get() === id) {
        const remaining = $sessions.get();
        if (remaining.length > 0) {
          await switchChat(remaining[0]!.id);
        } else {
          await createNewChat();
        }
      } else {
        sidebarWidget.render($sessions.get(), $currentChatId.get());
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Lỗi';
      addLogEntry(`Lỗi: ${msg}`, 'error');
      showToast({ message: 'Lỗi xóa cuộc trò chuyện', type: 'error' });
    }
  }

  async function createNewChat(title = 'Cuộc trò chuyện mới') {
    try {
      const c = await createNewSession(api, title);
      addSession(c);
      setChatId(c.id);
      setChatTitle(title);
      chatWidget.clearMessages();
      if (welcomeState) welcomeState.style.display = 'flex';
      if (currentChatTitleEl) currentChatTitleEl.textContent = title;
      sidebarWidget.render($sessions.get(), c.id);
      addLogEntry('Đã tạo mới.', 'success');
      return c;
    } catch {
      const fb: ChatSession = { id: `local_${Date.now()}`, title };
      addSession(fb);
      setChatId(fb.id);
      setChatTitle(title);
      chatWidget.clearMessages();
      if (welcomeState) welcomeState.style.display = 'flex';
      if (currentChatTitleEl) currentChatTitleEl.textContent = title;
      sidebarWidget.render($sessions.get(), fb.id);
      return fb;
    }
  }

  async function switchChat(id: string) {
    setChatId(id);
    const chat = $sessions.get().find((c) => c.id === id);
    if (currentChatTitleEl) currentChatTitleEl.textContent = chat?.title || id;
    chatWidget.clearMessages();
    sidebarWidget.render($sessions.get(), id);

    const skeleton = chatWidget.showSkeleton();
    try {
      const messages = await fetchHistory(api, id);
      chatWidget.removeSkeleton(skeleton);
      const grouped = groupHistoryMessages(messages);
      let hasMessages = false;
      let idx = 0;
      for (const g of grouped) {
        chatWidget.appendMessage(g.content, g.role as 'user' | 'bot', false, g.metrics as TokenMetrics | undefined);
        const msgs = chatContent.querySelectorAll('.message');
        (msgs[msgs.length - 1] as HTMLElement).style.animationDelay = `${idx * 0.06}s`;
        idx++;
        hasMessages = true;
      }
      if (hasMessages && welcomeState) welcomeState.style.display = 'none';
      else if (welcomeState) welcomeState.style.display = 'flex';
      chatWidget.scrollToBottom();
    } catch {
      chatWidget.removeSkeleton(skeleton);
      if (welcomeState) welcomeState.style.display = 'flex';
    }
    addLogEntry(`Chuyển: ${chat?.title || id}`, 'info');
  }

  // ═══ SEND MESSAGE ═══
  async function sendMessage(textOverride: string | null = null) {
    const text = textOverride ?? (chatInput?.value.trim() || '');
    if (!text && pendingAttachments.length === 0) return;

    if (textOverride === null) {
      if (!$currentChatId.get()) await createNewChat();
      chatWidget.appendMessage(text, 'user');
    }

    const files = [...pendingAttachments];
    pendingAttachments = [];
    renderAttachments();

    if (chatInput) {
      chatInput.value = '';
      chatInput.style.height = '24px';
    }

    setGenerating(true);
    updateSendButtonState();

    const thinkingCard = document.createElement('div');
    thinkingCard.className = 'thinking-card';
    chatContent?.appendChild(thinkingCard);

    const typingEl = document.createElement('div');
    typingEl.className = 'typing-indicator';
    typingEl.innerHTML = '<span></span><span></span><span></span>';
    typingEl.style.padding = '4px 0';
    chatContent?.appendChild(typingEl);

    const startTime = Date.now();
    const { content, footer } = chatWidget.appendStreamingBotMessage();
    currentBotContentEl = content;
    currentBotFooterEl = footer;

    abortController = new AbortController();

    try {
      const att: AttachmentPayload[] = [];
      for (const f of files) {
        att.push({ name: f.name, type: f.type, data: await readFileAsBase64(f) });
      }

      fullText = '';
      await api.streamChat(
        text,
        $currentChatId.get() || undefined,
        att,
        abortController.signal,
        (tok) => {
          if (typingEl.parentNode) typingEl.remove();
          fullText += tok;
          scheduleRender();
        },
        (met) => {
          thinkingCard.remove();
          // Final render to ensure all accumulated text is displayed
          if (currentBotContentEl) {
            currentBotContentEl.innerHTML = renderMarkdown(fullText);
            chatWidget.scrollToBottom();
          }
          const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
          if (!met.latency_ms) met.latency_ms = Date.now() - startTime;
          chatWidget.finalizeStreaming(fullText, met, elapsed);
          loadConversations();
        },
        (err) => {
          throw new Error(err);
        },
      );

      setGenerating(false);
      updateSendButtonState();
      abortController = null;
      currentBotContentEl = null;
      currentBotFooterEl = null;
    } catch (error) {
      thinkingCard.remove();
      if (typingEl.parentNode) typingEl.remove();
      setGenerating(false);
      updateSendButtonState();
      abortController = null;

      if ((error as Error).name === 'AbortError') {
        currentBotContentEl = null;
        currentBotFooterEl = null;
        return;
      }

      const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
      chatWidget.showError(error as Error, elapsed);
      chatInput?.focus();
      chatWidget.scrollToBottom();
    }
  }

  function stopGeneration() {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
    setGenerating(false);
    updateSendButtonState();
    if (currentBotFooterEl) currentBotFooterEl.innerHTML = '<span class="msg-timer">Đã dừng</span>';
    currentBotContentEl = null;
    currentBotFooterEl = null;
  }

  function updateSendButtonState() {
    if (!sendBtn) return;
    if ($isGenerating.get()) {
      sendBtn.innerHTML = '■';
      sendBtn.title = 'Dừng';
      sendBtn.classList.add('stop-mode');
    } else {
      sendBtn.innerHTML = '↑';
      sendBtn.title = 'Gửi';
      sendBtn.classList.remove('stop-mode');
    }
  }

  // ═══ ATTACHMENTS ═══
  function renderAttachments() {
    if (!attachmentsPreview) return;
    attachmentsPreview.innerHTML = '';
    pendingAttachments.forEach((file, i) => {
      const chip = document.createElement('div');
      chip.className = 'attachment-chip';
      const icon = file.type.startsWith('image/') ? '🖼️' : '📎';
      chip.innerHTML = `<span>${icon}</span><span class="attachment-chip-name">${escHtml(file.name)}</span><button class="attachment-remove-btn">✕</button>`;
      chip.querySelector('.attachment-remove-btn')?.addEventListener('click', (e) => {
        e.stopPropagation();
        pendingAttachments.splice(i, 1);
        renderAttachments();
      });
      attachmentsPreview.appendChild(chip);
    });
  }

  // ═══ METRICS MODAL ═══
  function hideMetricsModal() {
    metricsModal?.classList.add('hidden');
  }

  // ═══ EVENT LISTENERS ═══
  function setupEventListeners() {
    // Send
    sendBtn?.addEventListener('click', () => {
      if ($isGenerating.get()) stopGeneration();
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

    // Theme
    themeToggle?.addEventListener('click', () => {
      const next = currentTheme === 'dark' ? 'light' : 'dark';
      applyTheme(next);
    });

    // Sidebar
    toggleConversations?.addEventListener('click', () => {
      toggleSidebar();
      if (sidebarBackdrop && window.innerWidth <= 768) {
        const open = $sidebarOpen.get();
        sidebarBackdrop.classList.toggle('hidden', !open);
        sidebarBackdrop.classList.toggle('visible', open);
      }
    });

    togglePipeline?.addEventListener('click', () => {
      pipelineSidebar?.classList.toggle('collapsed');
    });

    closePipeline?.addEventListener('click', () => pipelineSidebar?.classList.add('collapsed'));

    searchInput?.addEventListener('input', (e) => {
      const q = ((e.target as HTMLInputElement).value || '').toLowerCase();
      sidebarWidget.render($sessions.get(), $currentChatId.get(), q);
    });

    sidebarBackdrop?.addEventListener('click', () => {
      setSidebarOpen(false);
      sidebarBackdrop.classList.remove('visible');
      sidebarBackdrop.classList.add('hidden');
    });

    // Settings
    settingsTrigger?.addEventListener('click', () => $settingsOpen.set(true));
    closeSettingsBtn?.addEventListener('click', () => closeSettings());
    closeSettingsBtn2?.addEventListener('click', () => closeSettings());
    settingsModal?.addEventListener('click', (e) => {
      if (e.target === settingsModal) closeSettings();
    });

    saveSettingsBtn?.addEventListener('click', async () => {
      const raw = orKeysInput?.value.trim() || '';
      if (saveSettingsBtn) {
        (saveSettingsBtn as HTMLButtonElement).disabled = true;
        saveSettingsBtn.textContent = 'Đang lưu...';
      }
      try {
        const keys = raw
          .split('\n')
          .map((k) => k.trim())
          .filter(Boolean);
        await api.saveConfigKeys(keys);
        addLogEntry('Đã cập nhật API Keys.', 'success');
        showToast({ message: 'Đã lưu API Keys', type: 'success' });
        closeSettings();
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

    // Metrics
    closeMetrics?.addEventListener('click', hideMetricsModal);
    metricsModal?.addEventListener('click', (e) => {
      if (e.target === metricsModal) hideMetricsModal();
    });

    // Backend
    backendSelect?.addEventListener('change', (e) => {
      const b = (e.target as HTMLSelectElement).value;
      currentBaseUrl = backends[b] ?? backends.gemini!;
      api = new ApiClient(currentBaseUrl);
      sseManager.connect(currentBaseUrl, {
        onMessage: (d) => handleSSEMessage(d),
        onError: (msg) => addLogEntry(msg, 'error'),
      });
      addLogEntry(`Đã chuyển sang ${b}`, 'info');
    });

    // New chat
    newChatBtn?.addEventListener('click', async () => {
      await createNewChat();
      chatWidget.clearMessages();
      if (welcomeState) welcomeState.style.display = 'flex';
    });

    // Run test
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
        await sendMessage(testQueries[i]!);
        await new Promise((r) => setTimeout(r, 3000));
      }
      addLogEntry('--- HOÀN THÀNH ---', 'success');
      (runTestBtn as HTMLButtonElement).disabled = false;
    });

    // Welcome chips
    document.querySelectorAll<HTMLButtonElement>('.welcome-chip[data-quick]').forEach((chip) => {
      chip.addEventListener('click', () => {
        if (chatInput) chatInput.value = chip.dataset.quick || '';
        sendMessage();
      });
    });

    // Scroll FAB
    if (scrollToBottomBtn && chatViewport) {
      chatViewport.addEventListener('scroll', () => {
        const distance = chatViewport.scrollHeight - chatViewport.scrollTop - chatViewport.clientHeight;
        scrollToBottomBtn.classList.toggle('hidden', distance <= 200);
      });
      scrollToBottomBtn.addEventListener('click', () => {
        chatWidget.scrollToBottom();
        scrollToBottomBtn.classList.add('hidden');
      });
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      if (e.ctrlKey && !e.shiftKey && e.key === 'n') {
        e.preventDefault();
        createNewChat();
        chatWidget.clearMessages();
        if (welcomeState) welcomeState.style.display = 'flex';
      }
      if (e.ctrlKey && e.shiftKey && e.key === 'S') {
        e.preventDefault();
        toggleSidebar();
      }
      if (e.ctrlKey && e.key === 'p') {
        e.preventDefault();
        pipelineSidebar?.classList.toggle('collapsed');
      }
      if (e.key === 'Escape') {
        if (settingsModal && !settingsModal.classList.contains('hidden')) closeSettings();
        else if (shortcutsPanel && !shortcutsPanel.classList.contains('hidden')) shortcutsPanel.classList.add('hidden');
      }
    });

    closeShortcuts?.addEventListener('click', () => shortcutsPanel?.classList.add('hidden'));
    shortcutsPanel?.addEventListener('click', (e) => {
      if (e.target === shortcutsPanel) shortcutsPanel.classList.add('hidden');
    });

    // Attachments
    attachBtn?.addEventListener('click', () => fileInput?.click());
    fileInput?.addEventListener('change', () => {
      if (fileInput.files?.length) {
        pendingAttachments = [...pendingAttachments, ...Array.from(fileInput.files)];
        renderAttachments();
        fileInput.value = '';
      }
    });
  }

  // ═══ INIT ═══
  async function init() {
    initConnectionStatus();
    setupEventListeners();

    sseManager.connect(currentBaseUrl, {
      onMessage: (d) => handleSSEMessage(d),
      onError: (msg) => addLogEntry(msg, 'error'),
    });

    await loadConversations();
    const sessions = $sessions.get();
    if (!sessions.length) {
      await createNewChat('Cuộc trò chuyện đầu tiên');
    } else {
      setChatId(sessions[0]!.id);
      if (currentChatTitleEl) currentChatTitleEl.textContent = sessions[0]!.title || 'Cuộc trò chuyện mới';
      sidebarWidget.render(sessions, sessions[0]!.id);
    }

    // Subscribe to pipeline changes for widget updates
    $pipeline.subscribe((_state) => {
      if (pipelineWidget) pipelineWidget.render(_state);
    });

    setTimeout(() => chatInput?.focus(), 100);
  }

  function destroy() {
    chatWidget.destroy();
    sidebarWidget.destroy();
    if (pipelineWidget) pipelineWidget.destroy();
    sseManager.disconnect();
    if (abortController) abortController.abort();
    if (connStatus) connStatus.destroy();
  }

  return { init, destroy };
}
