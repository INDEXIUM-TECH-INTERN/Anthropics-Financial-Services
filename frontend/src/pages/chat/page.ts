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
import type { WorldNewsReport } from '../../shared/api/mock-news';
import { getApiBaseUrl } from '../../shared/lib/api-base';


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
  // API base from /config.js (Render) or same-origin (local dev / docker-compose proxy)
  let currentBaseUrl = getApiBaseUrl();
  let api = new ApiClient(currentBaseUrl);
  const sseManager = new SSEManager();
  let abortController: AbortController | null = null;
  let pendingAttachments: File[] = [];
  let currentBotContentEl: HTMLElement | null = null;
  let currentBotFooterEl: HTMLElement | null = null;
  let currentThinkingCard: HTMLElement | null = null;
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
  const newChatBtn = $('new-chat-btn');
  const shortcutsPanel = $('shortcuts-panel');
  const closeShortcuts = $('close-shortcuts');
  const metricsModal = $('metrics-modal');
  const closeMetrics = $('close-metrics');

  // ═══ TAB CONTROL DOM REFS & STATE ═══
  const tabBtnChat = $('tab-btn-chat');
  const tabBtnNews = $('tab-btn-news');
  const worldNewsContainer = $('world-news-container');
  const conversationsSidebar = $('conversations-sidebar');
  const chatContainerEl = document.querySelector('.chat-container') as HTMLElement | null;
  const inputAreaEl = document.querySelector('.input-area') as HTMLElement | null;
  const toggleConversationsBtn = $('toggle-conversations');
  const closeSidebarBtn = $('close-sidebar');
  const newsDateSelect = $<HTMLSelectElement>('news-date-select');
  const newsLoadingEl = $('world-news-loading');
  const newsDataSourceEl = $('news-data-source');
  const newsErrorEl = $('world-news-error');
  const newsReportTitleEl = $('news-report-title');
  let currentTab = 'chat';
  let worldNewsLoading = false;
  let worldNewsDatesLoaded = false;

  function syncSidebarUI() {
    if (!conversationsSidebar) return;

    const isChatTab = currentTab === 'chat';
    const open = isChatTab && $sidebarOpen.get();
    conversationsSidebar.classList.toggle('active', open);

    if (sidebarBackdrop) {
      sidebarBackdrop.classList.toggle('visible', open);
      sidebarBackdrop.classList.toggle('hidden', !open);
    }

    if (toggleConversationsBtn) {
      toggleConversationsBtn.setAttribute('aria-expanded', String(open));
      toggleConversationsBtn.title = open ? 'Ẩn danh sách' : 'Hiện danh sách';
    }

    conversationsSidebar.setAttribute('aria-hidden', String(!open));
  }

  // ═══ WIDGETS ═══
  const chatWidget = createChatViewWidget(chatContent, chatViewport);
  const sidebarWidget = createSidebarWidget(conversationsList, {
    onSelect: (id) => {
      void switchChat(id);
      if (window.innerWidth <= 768) setSidebarOpen(false);
    },
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
        fetch(`${currentBaseUrl}/health`, { signal: AbortSignal.timeout(5000) })
          .then((res) => {
            connStatus?.setStatus(res.ok ? 'connected' : 'disconnected');
          })
          .catch(() => {
            connStatus?.setStatus('disconnected');
          });
      },
    });
    fetch(`${currentBaseUrl}/health`, { signal: AbortSignal.timeout(5000) })
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

    // Map to inline thinking step
    if (currentThinkingCard) {
      let label = 'Hệ thống';
      let icon = '•';
      let isStep = false;

      if (type === 'process') {
        isStep = true;
        if (payload.includes('phân tích') || payload.includes('chọn Agent')) {
          label = 'Phân tích định hướng';
          icon = '🧭';
        } else if (payload.includes('Khởi tạo')) {
          label = 'Khởi tạo';
          icon = '🚀';
        } else {
          label = 'Xử lý';
          icon = '⚙️';
        }
      } else if (type === 'agent_selected') {
        isStep = true;
        label = 'Chọn Agent';
        icon = '👤';
      } else if (type === 'skill_loaded') {
        isStep = true;
        label = 'Nạp Skill';
        icon = '📚';
      } else if (type === 'tool') {
        isStep = true;
        label = 'Tra cứu';
        icon = '🔍';
      } else if (type === 'tool_executed') {
        isStep = true;
        label = 'Gọi Tool';
        icon = '⚡';
      } else if (type === 'success') {
        isStep = true;
        label = 'Hoàn tất bước';
        icon = '✓';
      } else if (type === 'error') {
        isStep = true;
        label = 'Lỗi';
        icon = '❌';
      }

      if (isStep) {
        addThinkingStep(label, payload, icon);
      }
    }
  }

  function getThinkingStatusText(label: string, payload: string): string {
    const p = payload.toLowerCase();
    if (p.includes('chọn agent') || p.includes('agent tối ưu')) {
      return 'Đang chọn Agent...';
    }
    if (p.includes('nạp cấu hình cho agent')) {
      return 'Đang chuyển giao Agent...';
    }
    if (p.includes('nạp skill')) {
      return 'Đang nạp Skill...';
    }
    if (p.includes('google') || p.includes('tavily') || p.includes('tìm kiếm')) {
      return 'Đang tìm kiếm thông tin...';
    }
    if (p.includes('đọc nội dung từ')) {
      return 'Đang đọc nội dung trang web...';
    }
    if (p.includes('tính toán')) {
      return 'Đang thực hiện tính toán...';
    }
    if (p.includes('tệp nội bộ')) {
      return 'Đang đọc tài liệu hệ thống...';
    }
    if (p.includes('tóm tắt context')) {
      return 'Đang tóm tắt ngữ cảnh...';
    }
    if (p.includes('khởi tạo cuộc hội thoại')) {
      return 'Đang khởi tạo phiên làm việc...';
    }
    
    if (label === 'Chọn Agent') return 'Đang chọn Agent...';
    if (label === 'Nạp Skill') return 'Đang nạp Skill...';
    if (label === 'Tra cứu') return 'Đang tra cứu dữ liệu...';
    if (label === 'Gọi Tool') return 'Đang chạy công cụ...';
    if (label === 'Phân tích định hướng') return 'Đang phân tích định hướng...';
    
    return 'AI đang suy nghĩ...';
  }

  function addThinkingStep(label: string, text: string, _icon = '•') {
    if (!currentThinkingCard) return;
    const stepsContainer = currentThinkingCard.querySelector('.thinking-steps');
    if (!stepsContainer) return;

    // Mark previous steps as done
    stepsContainer.querySelectorAll('.thinking-step.active').forEach((step) => {
      step.classList.remove('active');
      step.classList.add('done');
    });

    const step = document.createElement('div');
    step.className = 'thinking-step active';
    step.innerHTML = `
      <div class="thinking-step-content">
        <div class="thinking-step-label">${escHtml(label)}</div>
        <div class="thinking-step-text">${escHtml(text.length > 60 ? text.slice(0, 60) + '…' : text)}</div>
      </div>
    `;
    stepsContainer.appendChild(step);

    // Update status text on the summary header dynamically
    const summaryText = currentThinkingCard.querySelector('.thinking-summary-text');
    if (summaryText) {
      summaryText.textContent = getThinkingStatusText(label, text);
    }

    chatWidget.scrollToBottom();
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
        chatWidget.appendMessage(g.content, g.role, false, g.metrics as TokenMetrics | undefined);
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
      chatInput.style.height = '36px';
    }

    setGenerating(true);
    updateSendButtonState();

    const startTime = Date.now();
    const { el: botMsgEl, content, footer } = chatWidget.appendStreamingBotMessage();
    currentBotContentEl = content;
    currentBotFooterEl = footer;

    // Create collapsible thinking details block inside the bot message bubble at the top
    const thinkingDetails = document.createElement('details');
    thinkingDetails.className = 'thinking-details';
    thinkingDetails.open = true;

    const summary = document.createElement('summary');
    summary.className = 'thinking-summary';
    summary.innerHTML = `
      <span class="thinking-summary-icon"><div class="thinking-spinner"></div></span>
      <span class="thinking-summary-text">Đang suy nghĩ...</span>
    `;
    thinkingDetails.appendChild(summary);

    const stepsContainer = document.createElement('div');
    stepsContainer.className = 'thinking-steps';
    thinkingDetails.appendChild(stepsContainer);

    const msgBody = botMsgEl.querySelector('.msg-body');
    if (msgBody && content) {
      msgBody.insertBefore(thinkingDetails, content);
    }
    currentThinkingCard = thinkingDetails;

    const typingEl = document.createElement('div');
    typingEl.className = 'typing-indicator';
    typingEl.innerHTML = '<span></span><span></span><span></span>';
    typingEl.style.padding = '4px 0';
    if (content) content.appendChild(typingEl);

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

          // Auto-collapse thinking details when final response text starts streaming
          if (currentThinkingCard && currentThinkingCard.hasAttribute('open')) {
            currentThinkingCard.removeAttribute('open');
            const iconEl = currentThinkingCard.querySelector('.thinking-spinner');
            if (iconEl) iconEl.remove();
            const summaryText = currentThinkingCard.querySelector('.thinking-summary-text');
            if (summaryText) {
              const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
              summaryText.textContent = `Đã suy nghĩ trong ${elapsed}s`;
            }
          }

          fullText += tok;
          scheduleRender();
        },
        (met) => {
          if (currentThinkingCard) {
            currentThinkingCard.removeAttribute('open');
            const iconEl = currentThinkingCard.querySelector('.thinking-spinner');
            if (iconEl) iconEl.remove();
            const summaryText = currentThinkingCard.querySelector('.thinking-summary-text');
            if (summaryText) {
              const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
              summaryText.textContent = `Đã suy nghĩ trong ${elapsed}s`;
            }
            currentThinkingCard = null;
          }

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
      if (currentThinkingCard) {
        currentThinkingCard.removeAttribute('open');
        const iconEl = currentThinkingCard.querySelector('.thinking-spinner');
        if (iconEl) iconEl.remove();
        const summaryText = currentThinkingCard.querySelector('.thinking-summary-text') as HTMLElement | null;
        if (summaryText) summaryText.textContent = 'Lỗi';
        currentThinkingCard = null;
      }

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
    if (currentThinkingCard) {
      currentThinkingCard.removeAttribute('open');
      const iconEl = currentThinkingCard.querySelector('.thinking-spinner');
      if (iconEl) iconEl.remove();
      const summaryText = currentThinkingCard.querySelector('.thinking-summary-text');
      if (summaryText) summaryText.textContent = 'Đã dừng';
      currentThinkingCard = null;
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

  // ═══ SPARKLINE & CHART DRAWING ═══
  function generateSparklinePath(points: number[], width = 100, height = 30): string {
    const min = Math.min(...points);
    const max = Math.max(...points);
    const range = max - min || 1;
    const xStep = width / (points.length - 1);
    return points.map((p, i) => {
      const x = i * xStep;
      const y = height - ((p - min) / range) * (height - 6) - 3;
      return `${i === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`;
    }).join(' ');
  }

  function drawCanvasChart(
    canvasId: string, 
    points: number[], 
    labels: string[], 
    color = '#00f2fe', 
    secondaryPoints?: number[], 
    secondaryColor = '#ffab00'
  ) {
    const canvas = $<HTMLCanvasElement>(canvasId);
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const rect = canvas.getBoundingClientRect();
    const width = rect.width;
    const height = rect.height;
    if (width === 0 || height === 0) return; // Prevent 0-sized crashes

    const dpr = window.devicePixelRatio || 1;
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);

    ctx.clearRect(0, 0, width, height);

    const paddingLeft = 40;
    const paddingRight = 15;
    const paddingTop = 15;
    const paddingBottom = 20;
    const chartWidth = width - paddingLeft - paddingRight;
    const chartHeight = height - paddingTop - paddingBottom;

    let allPoints = [...points];
    if (secondaryPoints) {
      allPoints = [...allPoints, ...secondaryPoints];
    }
    const minVal = Math.min(...allPoints);
    const maxVal = Math.max(...allPoints);
    const valRange = maxVal - minVal || 1;
    
    const yMin = minVal - valRange * 0.1;
    const yMax = maxVal + valRange * 0.1;
    const yRange = yMax - yMin;

    // Gridlines & Y-Axis labels
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
    ctx.lineWidth = 1;
    ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
    ctx.font = '9px var(--font-mono)';
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';

    const steps = 3;
    for (let i = 0; i <= steps; i++) {
      const val = yMin + (yRange * i) / steps;
      const y = paddingTop + chartHeight - (chartHeight * i) / steps;
      
      ctx.beginPath();
      ctx.moveTo(paddingLeft, y);
      ctx.lineTo(width - paddingRight, y);
      ctx.stroke();

      let lbl = val.toFixed(1);
      if (val >= 1000) lbl = Math.round(val).toLocaleString();
      ctx.fillText(lbl, paddingLeft - 8, y);
    }

    // X-Axis labels
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    const xStep = chartWidth / (points.length - 1);
    points.forEach((_, i) => {
      if (i % 2 === 0 || i === points.length - 1) {
        const x = paddingLeft + i * xStep;
        ctx.fillText(labels[i] || '', x, height - paddingBottom + 5);
      }
    });

    const drawLine = (dataPoints: number[], lineColor: string, fillGrad: boolean) => {
      ctx.beginPath();
      dataPoints.forEach((val, i) => {
        const x = paddingLeft + i * xStep;
        const y = paddingTop + chartHeight - ((val - yMin) / yRange) * chartHeight;
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      
      ctx.strokeStyle = lineColor;
      ctx.lineWidth = 2;
      ctx.shadowBlur = 4;
      ctx.shadowColor = lineColor;
      ctx.stroke();
      ctx.shadowBlur = 0;

      if (fillGrad) {
        let r = 0, g = 242, b = 254; // default cyan
        if (lineColor === '#ff5252') { r = 255; g = 82; b = 82; }
        else if (lineColor === '#ffab00') { r = 255; g = 171; b = 0; }
        else if (lineColor === '#00e676') { r = 0; g = 230; b = 118; }

        const grad = ctx.createLinearGradient(0, paddingTop, 0, height - paddingBottom);
        grad.addColorStop(0, `rgba(${r}, ${g}, ${b}, 0.15)`);
        grad.addColorStop(1, `rgba(${r}, ${g}, ${b}, 0)`);
        
        ctx.beginPath();
        ctx.moveTo(paddingLeft, height - paddingBottom);
        dataPoints.forEach((val, i) => {
          const x = paddingLeft + i * xStep;
          const y = paddingTop + chartHeight - ((val - yMin) / yRange) * chartHeight;
          ctx.lineTo(x, y);
        });
        ctx.lineTo(paddingLeft + (dataPoints.length - 1) * xStep, height - paddingBottom);
        ctx.closePath();
        ctx.fillStyle = grad;
        ctx.fill();
      }

      // Dots
      dataPoints.forEach((val, i) => {
        const x = paddingLeft + i * xStep;
        const y = paddingTop + chartHeight - ((val - yMin) / yRange) * chartHeight;
        ctx.beginPath();
        ctx.arc(x, y, 3, 0, 2 * Math.PI);
        ctx.fillStyle = lineColor;
        ctx.fill();
        ctx.lineWidth = 1;
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.8)';
        ctx.stroke();
      });
    };

    drawLine(points, color, !secondaryPoints);
    if (secondaryPoints) {
      drawLine(secondaryPoints, secondaryColor, false);
    }
  }

  function getVietnamTodayISO(): string {
    return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date());
  }

  function formatReportDateLabel(isoDate: string): string {
    const [y, m, d] = isoDate.split('-');
    if (!y || !m || !d) return isoDate;
    return `${d}/${m}/${y}`;
  }

  const DEFAULT_NEWS_HISTORY_DAYS = 90;

  function renderPublisherLogo(logo?: string, source?: string): string {
    if (!logo) return '';
    const alt = escHtml(source || 'Nguồn tin');
    return `<img class="news-publisher-logo" src="${escHtml(logo)}" alt="${alt}" width="20" height="20" loading="lazy" decoding="async" onerror="this.remove()">`;
  }

  function renderArticleThumbnail(thumbnail?: string, title?: string): string {
    if (!thumbnail) return '';
    const alt = escHtml(title || 'Ảnh bài viết');
    return `
      <div class="news-thumb-wrap">
        <img class="news-article-thumb" src="${escHtml(thumbnail)}" alt="${alt}" loading="lazy" decoding="async" onerror="this.closest('.news-thumb-wrap')?.remove()">
      </div>
    `;
  }

  function renderSourceBadge(source: string, logo?: string): string {
    return `
      <span class="news-item-source">
        ${renderPublisherLogo(logo, source)}
        <span>${escHtml(source)}</span>
      </span>
    `;
  }

  function renderNewsCard(n: { title: string; summary: string; source: string; time: string; url?: string; thumbnail?: string; logo?: string }): string {
    const titleText = escHtml(n.title);
    const inner = `
      <h5 class="news-item-title">${titleText}</h5>
      ${n.summary ? `<p class="news-item-desc">${escHtml(n.summary)}</p>` : ''}
      <div class="news-item-footer">
        ${renderSourceBadge(n.source, n.logo)}
        <span class="news-item-time">${escHtml(n.time)}</span>
      </div>
    `;
    if (n.url) {
      return `<a href="${escHtml(n.url)}" target="_blank" rel="noopener noreferrer" class="news-item news-item-card news-item-card-link">${inner}</a>`;
    }
    return `<div class="news-item news-item-card">${inner}</div>`;
  }

  function buildClientDateOptions(dayCount = DEFAULT_NEWS_HISTORY_DAYS): { value: string; label: string }[] {
    const todayIso = getVietnamTodayISO();
    const anchor = new Date(`${todayIso}T12:00:00+07:00`);
    const options: { value: string; label: string }[] = [];
    for (let i = 0; i < dayCount; i++) {
      const d = new Date(anchor);
      d.setDate(d.getDate() - i);
      const value = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(d);
      let label = formatReportDateLabel(value);
      if (value === todayIso) label += ' (Hôm nay)';
      options.push({ value, label });
    }
    return options;
  }

  function clearWorldNewsDashboard() {
    const ids = [
      'quick-highlights-list',
      'key-metrics-row',
      'stock-summary-list',
      'oil-prices-badge',
      'oil-summary-list',
      'gold-usd-summary-list',
      'breaking-news-list',
      'vtv-news-list',
      'vn-finance-news-list',
      'upcoming-watchlist',
    ];
    for (const id of ids) {
      const el = $(id);
      if (el) el.innerHTML = '';
    }
    if (newsDataSourceEl) newsDataSourceEl.textContent = '';
    if (newsReportTitleEl) newsReportTitleEl.textContent = 'Bản tin Tài chính Thế giới sáng';
  }

  function showWorldNewsError(message: string) {
    clearWorldNewsDashboard();
    newsErrorEl?.classList.remove('hidden');
    if (newsErrorEl) {
      newsErrorEl.innerHTML = `${escHtml(message)}<br><br><strong>Gợi ý:</strong> Chạy backend đúng bằng <code>.\run-server.ps1</code> hoặc <code>server.exe -server</code> (không dùng gemini-server demo). Sau đó tải lại trang (Ctrl+Shift+R).`;
    }
  }

  function hideWorldNewsError() {
    newsErrorEl?.classList.add('hidden');
    if (newsErrorEl) newsErrorEl.innerHTML = '';
  }

  // ═══ WORLD NEWS DATE PICKER ═══
  async function loadWorldNewsDates() {
    if (worldNewsDatesLoaded || !newsDateSelect) return;
    try {
      const { dates, defaultDate } = await api.getWorldNewsDates();
      newsDateSelect.innerHTML = dates
        .map((d) => `<option value="${escHtml(d.value)}">${escHtml(d.label)}</option>`)
        .join('');
      if (defaultDate) newsDateSelect.value = defaultDate;
      worldNewsDatesLoaded = true;
    } catch {
      const options = buildClientDateOptions(DEFAULT_NEWS_HISTORY_DAYS);
      newsDateSelect.innerHTML = options
        .map((d) => `<option value="${escHtml(d.value)}">${escHtml(d.label)}</option>`)
        .join('');
      newsDateSelect.value = options[0]?.value ?? getVietnamTodayISO();
    }
  }

  function setWorldNewsLoading(loading: boolean) {
    worldNewsLoading = loading;
    newsLoadingEl?.classList.toggle('hidden', !loading);
    worldNewsContainer?.classList.toggle('is-loading', loading);
  }

  // ═══ RENDER NEWS DASHBOARD ═══
  async function renderWorldNews() {
    if (worldNewsLoading) return;
    await loadWorldNewsDates();
    const selectedDate = newsDateSelect?.value || getVietnamTodayISO();

    hideWorldNewsError();
    setWorldNewsLoading(true);
    let report: WorldNewsReport;
    try {
      report = await api.getWorldNews(selectedDate);
    } catch (err) {
      setWorldNewsLoading(false);
      const msg = err instanceof Error ? err.message : 'Không tải được bản tin';
      addLogEntry(`Lỗi bản tin thế giới: ${msg}`, 'error');
      showToast({ message: msg, type: 'error' });
      showWorldNewsError(`Không tải được bản tin ngày ${formatReportDateLabel(selectedDate)}: ${msg}`);
      return;
    }
    setWorldNewsLoading(false);

    if (newsReportTitleEl) {
      newsReportTitleEl.textContent = `Bản tin Tài chính Thế giới sáng — ${formatReportDateLabel(report.date)}`;
    }
    if (newsDataSourceEl && report.dataSource) {
      const updatedAt = report.generatedAt
        ? ` | Cập nhật: ${new Date(report.generatedAt).toLocaleString('vi-VN', { timeZone: 'Asia/Ho_Chi_Minh' })}`
        : '';
      newsDataSourceEl.textContent = `Nguồn: ${report.dataSource}${updatedAt}`;
    }

    // Add visual transition class
    worldNewsContainer?.classList.remove('news-fade-in');
    void worldNewsContainer?.offsetWidth; // Trigger layout reflow
    worldNewsContainer?.classList.add('news-fade-in');

    // 1. Quick Highlights
    const quickHighlightsList = $('quick-highlights-list');
    if (quickHighlightsList) {
      quickHighlightsList.innerHTML = report.quickHighlights
        .map(h => `<li class="highlights-item">${escHtml(h)}</li>`)
        .join('');
    }

    // 2. Key Metrics Row
    const keyMetricsRow = $('key-metrics-row');
    if (keyMetricsRow) {
      keyMetricsRow.innerHTML = report.keyNumbers
        .map(m => {
          const pathStr = generateSparklinePath(m.sparkline);
          const colorClass = m.isPositive ? 'positive' : 'negative';
          const strokeColor = m.isPositive ? '#00e676' : '#ff5252';
          return `
            <div class="metric-card">
              <div class="metric-card-header">
                <span class="metric-card-label">${escHtml(m.label)}</span>
                <span class="metric-card-change ${colorClass}">
                  ${m.isPositive ? '▲' : '▼'} ${escHtml(m.change)}
                </span>
              </div>
              <div class="metric-card-value-area">
                <span class="metric-card-value">${escHtml(m.value)}</span>
              </div>
              <div class="metric-card-visual">
                <svg viewBox="0 0 100 30" width="100%" height="100%" preserveAspectRatio="none">
                  <path d="${pathStr}" fill="none" stroke="${strokeColor}" stroke-width="1.8" stroke-linecap="round" />
                </svg>
              </div>
            </div>
          `;
        })
        .join('');
    }

    // 3. Stocks detail
    const stockSummaryList = $('stock-summary-list');
    if (stockSummaryList) {
      stockSummaryList.innerHTML = report.stocks.highlights
        .map(h => `<li class="summary-item">${escHtml(h)}</li>`)
        .join('');
    }
    // 4. Oil details WTI & Brent
    const oilPricesBadge = $('oil-prices-badge');
    if (oilPricesBadge) {
      oilPricesBadge.innerHTML = `
        <div class="oil-submetric">
          <span class="oil-submetric-label">WTI (NYMEX):</span>
          <span class="oil-submetric-value">${escHtml(report.oil.wtiPrice)}</span>
          <span class="oil-submetric-change ${report.oil.wtiPositive ? 'positive' : 'negative'}">
            ${report.oil.wtiPositive ? '▲' : '▼'} ${escHtml(report.oil.wtiPercent)}
          </span>
        </div>
        <div class="oil-submetric">
          <span class="oil-submetric-label">Brent (ICE):</span>
          <span class="oil-submetric-value">${escHtml(report.oil.brentPrice)}</span>
          <span class="oil-submetric-change ${report.oil.brentPositive ? 'positive' : 'negative'}">
            ${report.oil.brentPositive ? '▲' : '▼'} ${escHtml(report.oil.brentPercent)}
          </span>
        </div>
      `;
    }
    const oilSummaryList = $('oil-summary-list');
    if (oilSummaryList) {
      oilSummaryList.innerHTML = report.oil.highlights
        .map(h => `<li class="summary-item">${escHtml(h)}</li>`)
        .join('');
    }
    // 5. Gold & USD
    const goldUsdSummaryList = $('gold-usd-summary-list');
    if (goldUsdSummaryList) {
      goldUsdSummaryList.innerHTML = report.goldUsd.highlights
        .map(h => `<li class="summary-item">${escHtml(h)}</li>`)
        .join('');
    }

    // 6. Breaking News
    const breakingNewsListEl = $('breaking-news-list');
    if (breakingNewsListEl) {
      breakingNewsListEl.innerHTML = report.breakingNews
        .map(b => {
          const thumb = renderArticleThumbnail(b.thumbnail, b.content);
          const logo = renderPublisherLogo(b.logo, b.source);
          const metaSource = `<span class="breaking-meta-source">${logo}<span>${escHtml(b.source)}</span></span>`;
          if (b.url) {
            return `
          <a href="${escHtml(b.url)}" target="_blank" rel="noopener noreferrer" class="breaking-news-item breaking-news-item-link ${b.isUrgent ? 'urgent' : ''}">
            ${thumb}
            <div class="breaking-content-area">
              <div class="breaking-meta">
                ${metaSource}
                <span>•</span>
                <span class="breaking-time-inline">${escHtml(b.time)}</span>
                <span>• Breaking</span>
              </div>
              <div class="breaking-text">${escHtml(b.content)}</div>
            </div>
          </a>
        `;
          }
          return `
          <div class="breaking-news-item ${b.isUrgent ? 'urgent' : ''}">
            ${thumb}
            <div class="breaking-content-area">
              <div class="breaking-meta">
                ${metaSource}
                <span>•</span>
                <span class="breaking-time-inline">${escHtml(b.time)}</span>
                <span>• Breaking</span>
              </div>
              <div class="breaking-text">${escHtml(b.content)}</div>
            </div>
          </div>
        `;
        })
        .join('');
    }

    // 7. VTVIndex News
    const vtvNewsList = $('vtv-news-list');
    if (vtvNewsList) {
      if (report.vtvIndexNews && report.vtvIndexNews.length > 0) {
        vtvNewsList.innerHTML = report.vtvIndexNews
          .map((n) => renderNewsCard(n))
          .join('');
      } else {
        vtvNewsList.innerHTML = '<div style="font-size:12px; color:var(--text-tertiary); padding:8px 0;">Không có tin tiêu điểm trong ngày.</div>';
      }
    }

    // 8. Vietnam Finance News
    const vnFinanceNewsList = $('vn-finance-news-list');
    if (vnFinanceNewsList) {
      vnFinanceNewsList.innerHTML = report.vietnamFinanceNews
        .map((n) => renderNewsCard(n))
        .join('');
    }

    // 9. Upcoming Watchlist
    const upcomingWatchlist = $('upcoming-watchlist');
    if (upcomingWatchlist) {
      upcomingWatchlist.innerHTML = report.watchlist
        .map(w => `
          <div class="watchlist-card">
            <div class="watchlist-card-left">
              <div class="watchlist-event" title="${escHtml(w.event)}">${escHtml(w.event)}</div>
              <div class="watchlist-card-meta">
                <span class="watchlist-source">${escHtml(w.source)}</span>
                <span>•</span>
                <span>${escHtml(w.time)}</span>
              </div>
            </div>
            <span class="importance-badge ${w.importance}">${escHtml(w.importance.toUpperCase())}</span>
          </div>
        `)
        .join('');
    }

    // Render Canvas Charts (with a small timeout to let the container size settle)
    setTimeout(() => {
      // Stock chart (S&P 500)
      const stockColor = report.stocks.isPositive ? '#00e676' : '#ff5252';
      drawCanvasChart('stock-chart', report.stocks.chartPoints, report.stocks.chartLabels, stockColor);

      // Oil chart WTI & Brent
      drawCanvasChart(
        'oil-chart',
        report.oil.chartPointsWTI,
        report.oil.chartLabels,
        '#00f2fe',
        report.oil.chartPointsBrent,
        '#ffab00'
      );

      // Gold & DXY Chart
      drawCanvasChart(
        'gold-chart',
        report.goldUsd.chartPointsGold,
        report.goldUsd.chartLabels,
        '#ffe082',
        report.goldUsd.chartPointsDXY,
        '#b0bec5'
      );
    }, 150);
  }

  // ═══ SWITCH TAB FUNCTION ═══
  function switchTab(tab: 'chat' | 'world-news') {
    currentTab = tab;
    if (tab === 'chat') {
      tabBtnChat?.classList.add('active');
      tabBtnNews?.classList.remove('active');
      worldNewsContainer?.classList.add('hidden');
      chatContainerEl?.classList.remove('hidden');
      chatViewport?.classList.remove('hidden');
      if (inputAreaEl) inputAreaEl.classList.remove('hidden');

      if (toggleConversationsBtn) toggleConversationsBtn.style.display = 'flex';
      syncSidebarUI();

      addLogEntry('Đã chuyển sang Trợ lý AI', 'info');
    } else {
      tabBtnChat?.classList.remove('active');
      tabBtnNews?.classList.add('active');
      chatContainerEl?.classList.add('hidden');
      chatViewport?.classList.add('hidden');
      if (inputAreaEl) inputAreaEl.classList.add('hidden');

      if (toggleConversationsBtn) toggleConversationsBtn.style.display = 'none';
      syncSidebarUI();

      worldNewsContainer?.classList.remove('hidden');
      renderWorldNews();
      addLogEntry('Đã chuyển sang Bản tin Thế giới', 'info');
    }
  }

  // ═══ EVENT LISTENERS ═══
  function setupEventListeners() {
    // Tab Switching Events
    tabBtnChat?.addEventListener('click', () => switchTab('chat'));
    tabBtnNews?.addEventListener('click', () => switchTab('world-news'));

    // Date Picker Event
    newsDateSelect?.addEventListener('change', () => {
      renderWorldNews();
    });

    // Window Resize Event to redraw charts responsively
    let resizeTimeout: number;
    window.addEventListener('resize', () => {
      if (currentTab === 'world-news') {
        clearTimeout(resizeTimeout);
        resizeTimeout = window.setTimeout(() => {
          renderWorldNews();
        }, 100);
      }
    });

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
      this.style.height = '36px';
      this.style.height = `${Math.min(this.scrollHeight, 160)}px`;
    });

    // Theme
    themeToggle?.addEventListener('click', () => {
      const next = currentTheme === 'dark' ? 'light' : 'dark';
      applyTheme(next);
    });

    // Sidebar
    $sidebarOpen.subscribe(() => syncSidebarUI());

    toggleConversations?.addEventListener('click', () => {
      toggleSidebar();
    });

    closeSidebarBtn?.addEventListener('click', () => {
      setSidebarOpen(false);
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
    });

    window.addEventListener('resize', () => syncSidebarUI(), { passive: true });

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
      // For now, we only have one backend in Docker setup (Gemini on port 8080 via proxy)
      // In the future, this could map to different backend services
      currentBaseUrl = getApiBaseUrl();
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
    if (scrollToBottomBtn) {
      const updateScrollButton = () => {
        const distance =
          document.documentElement.scrollHeight - window.scrollY - window.innerHeight;
        scrollToBottomBtn.classList.toggle('hidden', distance <= 200);
      };
      window.addEventListener('scroll', updateScrollButton, { passive: true });
      updateScrollButton();
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
        else if ($sidebarOpen.get() && currentTab === 'chat') setSidebarOpen(false);
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
    syncSidebarUI();

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
