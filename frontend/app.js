document.addEventListener('DOMContentLoaded', async () => {

    // ── Marked config ──
    const renderer = new marked.Renderer();
    const origLink = renderer.link;
    renderer.link = (arg1, title, text) => {
        let href, linkText, linkTitle;
        const isObj = typeof arg1 === 'object';
        if (isObj) {
            href = arg1.href || '';
            linkText = arg1.text || '';
            linkTitle = arg1.title || '';
        } else {
            href = arg1 || '';
            linkText = text || '';
            linkTitle = title || '';
        }
        if (!href || typeof href !== 'string') {
            return `<a href="#">${linkText}</a>`;
        }
        const local = href.startsWith(`${window.location.protocol}//${window.location.host}`) || href.startsWith('/') || href.startsWith('#');
        const html = isObj ? origLink.call(renderer, arg1) : origLink.call(renderer, href, title, text);
        return local ? html : html.replace(/^<a /, '<a target="_blank" rel="noopener noreferrer" ');
    };
    marked.setOptions({ renderer, breaks: true });

    // ── DOM refs ──
    const $ = id => document.getElementById(id);
    const chatInput = $('chat-input');
    const sendBtn = $('send-btn');
    const chatContent = $('chat-content');
    const chatViewport = $('chat-viewport');
    const welcomeState = $('welcome-state');
    const conversationsList = $('conversations-list');
    const conversationsSidebar = $('conversations-sidebar');
    const pipelineSidebar = $('pipeline-sidebar');
    const logStream = $('log-stream');
    const sourcesPanel = $('sources-panel');
    const sourcesList = $('sources-list');
    const runTestBtn = $('run-test-btn');
    const currentChatTitle = $('current-chat-title');

    // Settings
    const settingsTrigger = $('settings-trigger');
    const settingsModal = $('settings-modal');
    const closeSettings = $('close-settings');
    const closeSettingsBtn = $('close-settings-btn');
    const saveSettingsBtn = $('save-settings-btn');
    const geminiKeyInput = $('gemini-key-input');
    const orKeysInput = $('or-keys-input');

    // Pipeline cards
    const pipeAgent = $('pipe-agent');
    const pipeSkill = $('pipe-skill');
    const pipeTool = $('pipe-tool');
    const pipeReason = $('pipe-reason');

    // ── State ──
    const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
    const currentOrigin = window.location.origin;
    const backends = {
        gemini: isLocalhost ? 'http://localhost:8080' : currentOrigin,
        claude: 'http://localhost:8081'
    };
    let currentBackend = 'gemini';
    let currentBaseUrl = backends[currentBackend];
    let eventSource = null;
    let allChats = [];
    let currentChatId = null;

    // Streaming / abort state
    let activeAbortController = null;
    let isGenerating = false;
    let currentBotMessageEl = null;
    let currentBotContentEl = null;
    let currentBotFooterEl = null;

    // ── Theme ──
    const savedTheme = localStorage.getItem('theme') || 'dark';
    document.documentElement.setAttribute('data-theme', savedTheme);
    $('theme-toggle').addEventListener('click', () => {
        const cur = document.documentElement.getAttribute('data-theme');
        const next = cur === 'dark' ? 'light' : 'dark';
        document.documentElement.setAttribute('data-theme', next);
        localStorage.setItem('theme', next);
    });

    // ── Sidebar toggles ──
    $('toggle-conversations').addEventListener('click', () => {
        conversationsSidebar.classList.toggle('collapsed');
    });
    $('toggle-pipeline').addEventListener('click', () => {
        pipelineSidebar.classList.toggle('collapsed');
        if (!pipelineSidebar.classList.contains('collapsed')) {
            logStream.scrollTop = logStream.scrollHeight;
        }
    });
    $('close-pipeline').addEventListener('click', () => {
        pipelineSidebar.classList.add('collapsed');
    });

    // ── Settings modal ──
    const openSettings = () => { settingsModal.style.display = 'flex'; };
    const hideSettings = () => { settingsModal.style.display = 'none'; };
    settingsTrigger.addEventListener('click', openSettings);
    closeSettings.addEventListener('click', hideSettings);
    closeSettingsBtn.addEventListener('click', hideSettings);
    settingsModal.addEventListener('click', e => { if (e.target === settingsModal) hideSettings(); });

    // Load saved keys
    if (localStorage.getItem('gemini_key')) geminiKeyInput.value = localStorage.getItem('gemini_key');
    if (localStorage.getItem('openrouter_keys')) orKeysInput.value = localStorage.getItem('openrouter_keys');

    saveSettingsBtn.addEventListener('click', async () => {
        const geminiKey = geminiKeyInput.value.trim();
        const orKeysValue = orKeysInput.value.trim();
        localStorage.setItem('gemini_key', geminiKey);
        localStorage.setItem('openrouter_keys', orKeysValue);
        const keysArray = orKeysValue.split('\n').map(k => k.trim()).filter(k => k);
        saveSettingsBtn.disabled = true;
        saveSettingsBtn.textContent = 'Đang lưu...';
        try {
            const res = await fetch(`${currentBaseUrl}/api/config/keys`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ keys: keysArray })
            });
            const data = await res.json();
            if (res.ok && data.status === 'success') {
                addLogEntry('Đã cập nhật API Keys.', 'success');
                hideSettings();
            } else {
                addLogEntry('Lỗi: ' + (data?.message || 'Không xác định'), 'error');
            }
        } catch (err) {
            addLogEntry('Không thể kết nối backend: ' + err.message, 'error');
        } finally {
            saveSettingsBtn.disabled = false;
            saveSettingsBtn.textContent = 'Lưu cấu hình';
        }
    });

    // ── Backend selector ──
    const backendSelect = $('backend-select');
    backendSelect.addEventListener('change', e => {
        currentBackend = e.target.value;
        currentBaseUrl = backends[currentBackend];
        setupEventSource();
        addLogEntry(`Đã chuyển sang ${currentBackend}`, 'info');
    });

    // ── SSE (pipeline events) ──
    function setupEventSource() {
        if (eventSource) eventSource.close();
        eventSource = new EventSource(`${currentBaseUrl}/api/events`);
        eventSource.onmessage = e => {
            try {
                const data = JSON.parse(e.data);
                addLogEntry(data.payload || '', data.type || 'info');
                updatePipelineFromEvent(data);
            } catch (err) { /* ignore */ }
        };
        eventSource.onerror = () => {
            addLogEntry('Mất kết nối SSE. Đang thử lại...', 'error');
        };
    }

    // ── Conversations ──
    async function fetchConversations() {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats`);
            if (!res.ok) throw new Error();
            const data = await res.json();
            allChats = Array.isArray(data.chats) ? data.chats : [];
        } catch {
            allChats = [];
        }
        renderConversations();
    }

    function renderConversations() {
        conversationsList.innerHTML = '';
        if (allChats.length === 0) {
            conversationsList.innerHTML = '<div style="padding:16px;font-size:13px;color:var(--text-quaternary);text-align:center;">Chưa có cuộc trò chuyện nào.</div>';
            return;
        }
        allChats.forEach(chat => {
            const isActive = chat.id === currentChatId;
            const el = document.createElement('div');
            el.className = `conv-item ${isActive ? 'active' : ''}`;
            el.innerHTML = `
                <span class="conv-item-title">${escHtml(chat.title || 'Cuộc trò chuyện mới')}</span>
                <span class="conv-item-meta">${formatTime(chat.updated_at)}</span>
                <button class="conv-item-delete" title="Xóa"><i data-lucide="trash-2"></i></button>
            `;
            el.querySelector('.conv-item-title').addEventListener('click', () => switchChat(chat.id));
            el.querySelector('.conv-item-delete').addEventListener('click', async e => {
                e.stopPropagation();
                if (confirm('Xóa cuộc trò chuyện này?')) {
                    await deleteChat(chat.id);
                }
            });
            conversationsList.appendChild(el);
        });
        if (window.lucide) lucide.createIcons();
    }

    async function deleteChat(chatId) {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats?chat_id=${encodeURIComponent(chatId)}`, { method: 'DELETE' });
            if (!res.ok) throw new Error();
            allChats = allChats.filter(c => c.id !== chatId);
            addLogEntry('Đã xóa cuộc trò chuyện.', 'success');
            if (currentChatId === chatId) {
                if (allChats.length > 0) {
                    await switchChat(allChats[0].id);
                } else {
                    await createNewChat();
                }
            } else {
                renderConversations();
            }
        } catch (err) {
            addLogEntry('Không thể xóa: ' + err.message, 'error');
        }
    }

    async function createNewChat(title = 'Cuộc trò chuyện mới') {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title })
            });
            if (!res.ok) throw new Error();
            const created = await res.json();
            allChats.unshift(created);
            currentChatId = created.id;
            clearChatUI();
            welcomeState.style.display = 'flex';
            currentChatTitle.textContent = title;
            renderConversations();
            addLogEntry('Đã tạo cuộc trò chuyện mới.', 'success');
            return created;
        } catch {
            const fb = { id: 'local_' + Date.now(), title, messages: [] };
            allChats.unshift(fb);
            currentChatId = fb.id;
            clearChatUI();
            welcomeState.style.display = 'flex';
            currentChatTitle.textContent = title;
            renderConversations();
            return fb;
        }
    }

    async function switchChat(chatId) {
        currentChatId = chatId;
        const chat = allChats.find(c => c.id === chatId);
        currentChatTitle.textContent = chat ? chat.title : chatId;
        clearChatUI();
        renderConversations();

        try {
            const res = await fetch(`${currentBaseUrl}/api/history?chat_id=${encodeURIComponent(chatId)}`);
            if (res.ok) {
                const data = await res.json();
                const hist = data.history || [];
                let hasContent = false;
                hist.forEach(h => {
                    const role = String(h.role || h.Role || '').toLowerCase();
                    if (role === 'tool') return;
                    let text = h.content || h.Content || '';
                    if (!text || text.length > 4000 ||
                        text.includes('ANTHROPIC AGENT CONFIG') ||
                        text.includes('SKILL MARKDOWN') ||
                        text.includes('=== TÓM TẮT NGỮ CẢNH')) return;
                    appendMessageBubble(text, role === 'user' ? 'user' : 'bot');
                    hasContent = true;
                });
                if (hasContent) {
                    welcomeState.style.display = 'none';
                } else {
                    welcomeState.style.display = 'flex';
                }
                scrollToBottom();
            }
        } catch {
            welcomeState.style.display = 'flex';
        }
        addLogEntry(`Đã chuyển sang: ${chat ? chat.title : chatId}`, 'info');
    }

    function clearChatUI() {
        const msgs = chatContent.querySelectorAll('.message');
        msgs.forEach(m => m.remove());
        logStream.innerHTML = '<div class="log-stream-empty">Sẵn sàng phân tích yêu cầu khi có câu hỏi.</div>';
        sourcesPanel.style.display = 'none';
        sourcesList.innerHTML = '';
        resetPipeline();
    }

    // ── Chat ──
    function appendMessageBubble(text, sender, isStreaming = false) {
        welcomeState.style.display = 'none';
        const el = document.createElement('div');
        el.className = `message ${sender}`;

        const avatar = document.createElement('div');
        avatar.className = 'msg-avatar';

        const body = document.createElement('div');
        body.className = 'msg-body';

        const senderLabel = document.createElement('div');
        senderLabel.className = 'msg-sender';
        senderLabel.textContent = sender === 'user' ? 'Bạn' : 'Indexium AI';

        const content = document.createElement('div');
        content.className = 'msg-content';

        if (sender === 'bot') {
            if (isStreaming) {
                content.innerHTML = '';
            } else {
                content.innerHTML = marked.parse(text);
                renderSourcesInline(text, content);
            }
        } else {
            content.textContent = text;
        }

        body.appendChild(senderLabel);
        body.appendChild(content);

        const footer = document.createElement('div');
        footer.className = 'msg-footer';

        if (sender === 'bot') {
            if (isStreaming) {
                footer.innerHTML = '<span class="msg-timer streaming">Đang trả lời…</span>';
            } else {
                footer.innerHTML = '<span class="msg-timer">—</span>';
            }
            body.appendChild(footer);
        }

        // Action buttons for bot messages (not streaming)
        if (sender === 'bot' && !isStreaming) {
            const actions = document.createElement('div');
            actions.className = 'msg-actions';

            const copyBtn = document.createElement('button');
            copyBtn.className = 'msg-action-btn';
            copyBtn.title = 'Copy';
            copyBtn.innerHTML = '<i data-lucide="copy" style="width:13px;height:13px;"></i>';
            copyBtn.addEventListener('click', () => {
                const plainText = content.innerText || content.textContent;
                navigator.clipboard.writeText(plainText).then(() => {
                    copyBtn.innerHTML = '<i data-lucide="check" style="width:13px;height:13px;color:var(--success);"></i>';
                    if (window.lucide) lucide.createIcons();
                    setTimeout(() => {
                        copyBtn.innerHTML = '<i data-lucide="copy" style="width:13px;height:13px;"></i>';
                        if (window.lucide) lucide.createIcons();
                    }, 2000);
                });
            });
            actions.appendChild(copyBtn);

            const regenBtn = document.createElement('button');
            regenBtn.className = 'msg-action-btn';
            regenBtn.title = 'Tạo lại câu trả lời';
            regenBtn.innerHTML = '<i data-lucide="refresh-cw" style="width:13px;height:13px;"></i>';
            regenBtn.addEventListener('click', () => {
                const allMsgs = chatContent.querySelectorAll('.message');
                let prevUserMsg = null;
                for (let i = allMsgs.length - 1; i >= 0; i--) {
                    if (allMsgs[i] === el) continue;
                    if (allMsgs[i].classList.contains('user')) {
                        prevUserMsg = allMsgs[i].querySelector('.msg-content')?.textContent;
                        break;
                    }
                }
                if (prevUserMsg) {
                    el.remove();
                    sendMessage(prevUserMsg, true);
                }
            });
            actions.appendChild(regenBtn);

            body.appendChild(actions);
        }

        el.appendChild(avatar);
        el.appendChild(body);
        chatContent.appendChild(el);
        scrollToBottom();

        if (window.lucide) lucide.createIcons();
        return { el, content, footer };
    }

    function scrollToBottom() {
        requestAnimationFrame(() => {
            chatViewport.scrollTop = chatViewport.scrollHeight;
        });
    }

    // ── Sources ──
    function renderSourcesInline(markdownText, targetEl) {
        const urlRegex = /(https?:\/\/[^\s\)\]]+)/g;
        const urls = [...new Set(markdownText.match(urlRegex) || [])];
        if (!urls.length) return;

        const container = document.createElement('div');
        container.className = 'message-sources';
        container.innerHTML = '<div class="sources-label">Nguồn tham khảo</div>';

        const grid = document.createElement('div');
        grid.className = 'sources-grid';

        urls.forEach((url, i) => {
            let domain = url;
            try { domain = new URL(url).hostname.replace('www.', ''); } catch {}
            const a = document.createElement('a');
            a.href = url;
            a.target = '_blank';
            a.rel = 'noopener noreferrer';
            a.className = 'source-chip';
            a.innerHTML = `<span class="source-chip-index">${i + 1}</span>${domain}`;
            grid.appendChild(a);
        });

        container.appendChild(grid);
        targetEl.appendChild(container);

        sourcesPanel.style.display = 'block';
        sourcesList.innerHTML = '';
        urls.forEach((url, i) => {
            let domain = url;
            try { domain = new URL(url).hostname.replace('www.', ''); } catch {}
            const a = document.createElement('a');
            a.href = url;
            a.target = '_blank';
            a.rel = 'noopener noreferrer';
            a.className = 'source-link-item';
            a.innerHTML = `
                <span class="source-link-favicon">${i + 1}</span>
                <span class="source-link-url">${domain}</span>
            `;
            sourcesList.appendChild(a);
        });
    }

    // ── Stop generation ──
    function stopGeneration() {
        if (activeAbortController) {
            activeAbortController.abort();
            activeAbortController = null;
        }
        isGenerating = false;
        updateSendButtonState();
        if (currentBotContentEl && currentBotContentEl.innerHTML === '') {
            currentBotContentEl.innerHTML = '<em style="color:var(--text-quaternary)">Đã dừng.</em>';
        }
        if (currentBotFooterEl) {
            currentBotFooterEl.innerHTML = '<span class="msg-timer" style="color:var(--warning)">Đã dừng</span>';
        }
        currentBotMessageEl = null;
        currentBotContentEl = null;
        currentBotFooterEl = null;
    }

    function updateSendButtonState() {
        if (isGenerating) {
            sendBtn.innerHTML = '<i data-lucide="square" style="width:16px;height:16px;"></i>';
            sendBtn.title = 'Dừng';
            sendBtn.classList.add('stop-mode');
            sendBtn.disabled = false;
            if (runTestBtn) runTestBtn.disabled = true;
        } else {
            sendBtn.innerHTML = '<i data-lucide="arrow-up" style="width:16px;height:16px;"></i>';
            sendBtn.title = 'Gửi';
            sendBtn.classList.remove('stop-mode');
            sendBtn.disabled = false;
            if (runTestBtn) runTestBtn.disabled = false;
        }
        if (window.lucide) lucide.createIcons();
    }

    // ── Send message (STREAMING) ──
    async function sendMessage(textOverride = null, isRegenerate = false) {
        const text = textOverride !== null ? textOverride : chatInput.value.trim();
        if (!text) return false;

        if (!isRegenerate) {
            if (!currentChatId) await createNewChat();
            welcomeState.style.display = 'none';
            appendMessageBubble(text, 'user');
        }

        chatInput.value = '';
        chatInput.style.height = '24px';

        isGenerating = true;
        updateSendButtonState();

        const startTime = Date.now();

        const { el: msgEl, content, footer } = appendMessageBubble('', 'bot', true);
        currentBotMessageEl = msgEl;
        currentBotContentEl = content;
        currentBotFooterEl = footer;

        activeAbortController = new AbortController();

        try {
            const payload = { message: text };
            if (currentChatId) payload.chat_id = currentChatId;

            const response = await fetch(`${currentBaseUrl}/api/chat/stream`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
                signal: activeAbortController.signal
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';
            let fullText = '';

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop();

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (!trimmed.startsWith('data: ')) continue;
                    const dataStr = trimmed.substring(6);

                    try {
                        const data = JSON.parse(dataStr);
                        if (data.type === 'token' && data.text) {
                            fullText += data.text;
                            content.innerHTML = marked.parse(fullText);
                            scrollToBottom();
                        }
                        if (data.type === 'done') {
                            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                            content.innerHTML = marked.parse(fullText);
                            renderSourcesInline(fullText, content);
                            addCopyButtons(content);

                            footer.innerHTML = `
                                <span class="msg-timer">${elapsed}s</span>
                                <span class="msg-metrics">
                                    <span class="msg-metric">↑ ${data.metrics?.token_in || 0}</span>
                                    <span class="msg-metric-sep">·</span>
                                    <span class="msg-metric">↓ ${data.metrics?.token_out || 0}</span>
                                    <span class="msg-metric-sep">·</span>
                                    <span class="msg-metric">${data.metrics?.ram_mb || ''}</span>
                                </span>
                            `;

                            // Add action buttons
                            const actions = document.createElement('div');
                            actions.className = 'msg-actions';

                            const copyBtn = document.createElement('button');
                            copyBtn.className = 'msg-action-btn';
                            copyBtn.title = 'Copy';
                            copyBtn.innerHTML = '<i data-lucide="copy" style="width:13px;height:13px;"></i>';
                            copyBtn.addEventListener('click', () => {
                                navigator.clipboard.writeText(fullText).then(() => {
                                    copyBtn.innerHTML = '<i data-lucide="check" style="width:13px;height:13px;color:var(--success);"></i>';
                                    if (window.lucide) lucide.createIcons();
                                    setTimeout(() => {
                                        copyBtn.innerHTML = '<i data-lucide="copy" style="width:13px;height:13px;"></i>';
                                        if (window.lucide) lucide.createIcons();
                                    }, 2000);
                                });
                            });
                            actions.appendChild(copyBtn);

                            const regenBtn = document.createElement('button');
                            regenBtn.className = 'msg-action-btn';
                            regenBtn.title = 'Tạo lại câu trả lời';
                            regenBtn.innerHTML = '<i data-lucide="refresh-cw" style="width:13px;height:13px;"></i>';
                            regenBtn.addEventListener('click', () => {
                                const allMsgs = chatContent.querySelectorAll('.message');
                                let prevUserMsg = null;
                                for (let i = allMsgs.length - 1; i >= 0; i--) {
                                    if (allMsgs[i] === msgEl) continue;
                                    if (allMsgs[i].classList.contains('user')) {
                                        prevUserMsg = allMsgs[i].querySelector('.msg-content')?.textContent;
                                        break;
                                    }
                                }
                                if (prevUserMsg) {
                                    msgEl.remove();
                                    sendMessage(prevUserMsg, true);
                                }
                            });
                            actions.appendChild(regenBtn);

                            msgEl.querySelector('.msg-body').appendChild(actions);
                            if (window.lucide) lucide.createIcons();
                        }
                        if (data.type === 'error') {
                            throw new Error(data.error || 'Unknown streaming error');
                        }
                    } catch (parseErr) {
                        if (parseErr.message && !parseErr.message.includes('JSON')) {
                            throw parseErr;
                        }
                    }
                }
            }

            isGenerating = false;
            updateSendButtonState();
            activeAbortController = null;
            currentBotMessageEl = null;
            currentBotContentEl = null;
            currentBotFooterEl = null;
            fetchConversations();
            return true;

        } catch (error) {
            isGenerating = false;
            updateSendButtonState();
            activeAbortController = null;

            if (error.name === 'AbortError') {
                return false;
            }

            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            const diag = diagnoseError(error.name || 'Error', error.message || String(error), error.stack || '');
            content.innerHTML = renderErrorHTML(diag);
            footer.innerHTML = `<span class="msg-timer" style="color:var(--danger)">Thất bại · ${elapsed}s</span>`;
            setupErrorAccordion(msgEl);
            if (window.lucide) lucide.createIcons();

            currentBotMessageEl = null;
            currentBotContentEl = null;
            currentBotFooterEl = null;
            return false;
        } finally {
            sendBtn.disabled = false;
            if (runTestBtn) runTestBtn.disabled = false;
            chatInput.focus();
            scrollToBottom();
        }
    }

    // ── Error handling ──
    function diagnoseError(name, msg, stack = '') {
        const m = String(msg || '').toLowerCase();
        let badge = 'Hệ thống', title = 'Lỗi xử lý', icon = 'alert-triangle', desc = msg || 'Đã xảy ra lỗi không xác định.';
        let suggestions = [
            'Kiểm tra cấu hình API Key trong Cài đặt.',
            'Thử lại hoặc tạo cuộc trò chuyện mới.'
        ];

        if (m.includes('failed to fetch') || m.includes('connection refused') || m.includes('network error')) {
            badge = 'Kết nối'; title = 'Không thể kết nối máy chủ'; icon = 'wifi-off';
            desc = `Không thể kết nối đến backend tại ${currentBaseUrl}.`;
            suggestions = [
                'Chạy backend Go: <code>go run cmd/gemini-cli/main.go -server</code>',
                'Kiểm tra cổng 8080 có bị chiếm dụng không.'
            ];
        } else if (m.includes('quota') || m.includes('rate') || m.includes('429') || m.includes('exceeded')) {
            badge = 'Giới hạn'; title = 'Hết hạn mức truy cập (429)'; icon = 'alert-octagon';
            desc = 'Tài khoản đã vượt quá giới hạn cuộc gọi API.';
            suggestions = [
                'Thêm API Key dự phòng trong Cài đặt để kích hoạt key rotation.',
                'Chờ 1-2 phút trước khi thử lại.'
            ];
        } else if (m.includes('thought_signature') || m.includes('thought signature')) {
            badge = 'Model'; title = 'Lỗi tương thích mô hình'; icon = 'shield-alert';
            desc = 'Mô hình Gemini yêu cầu thought_signature mà backend chưa hỗ trợ đầy đủ.';
            suggestions = [
                'Đổi GEMINI_MODEL trong .env thành gemini-1.5-flash hoặc gemini-2.0-flash.',
                'Khởi động lại backend sau khi cập nhật.'
            ];
        } else if (name === 'AbortError' || m.includes('timeout')) {
            badge = 'Timeout'; title = 'Yêu cầu quá thời gian'; icon = 'clock';
            desc = 'Thời gian xử lý vượt quá 5 phút.';
            suggestions = ['Rút ngắn câu hỏi hoặc chia nhỏ tác vụ.', 'Kiểm tra kết nối mạng.'];
        }

        return { badge, title, icon, desc, suggestions, name, msg, stack };
    }

    function renderErrorHTML(diag) {
        return `
            <div class="error-container">
                <div class="error-header">
                    <span class="error-title"><i data-lucide="${diag.icon}"></i>${diag.title}</span>
                    <span class="error-badge">${diag.badge}</span>
                </div>
                <div class="error-body">${diag.desc}</div>
                <div class="error-suggestions">
                    <div class="error-suggestions-title"><i data-lucide="check-square"></i>Đề xuất khắc phục</div>
                    ${diag.suggestions.map(s => `<div class="error-suggestion-item">${s}</div>`).join('')}
                </div>
                <button class="error-details-toggle">
                    <span>Chi tiết kỹ thuật</span>
                    <i data-lucide="chevron-down"></i>
                </button>
                <div class="error-details-content" style="display:none;">
                    <div style="font-weight:700;margin-bottom:4px;color:var(--danger);font-family:var(--font-mono);font-size:11px;">[${diag.name}]</div>
                    <div style="font-family:var(--font-mono);font-size:11px;color:var(--text-tertiary);margin-bottom:8px;">${escHtml(diag.msg)}</div>
                    ${diag.stack ? `<pre style="margin:0;font-size:10px;color:var(--text-quaternary);white-space:pre-wrap;word-break:break-all;">${escHtml(diag.stack)}</pre>` : ''}
                </div>
            </div>`;
    }

    function setupErrorAccordion(el) {
        const toggle = el.querySelector('.error-details-toggle');
        const content = el.querySelector('.error-details-content');
        if (!toggle || !content) return;
        toggle.addEventListener('click', () => {
            const hidden = content.style.display === 'none';
            content.style.display = hidden ? 'block' : 'none';
            toggle.querySelector('i').setAttribute('data-lucide', hidden ? 'chevron-up' : 'chevron-down');
            if (window.lucide) lucide.createIcons();
        });
    }

    // ── Copy buttons for code blocks ──
    function addCopyButtons(contentEl) {
        contentEl.querySelectorAll('pre').forEach(pre => {
            pre.style.position = 'relative';
            const btn = document.createElement('button');
            btn.className = 'copy-code-btn';
            btn.innerHTML = '<i data-lucide="copy" style="width:11px;height:11px;"></i> Copy';
            btn.addEventListener('click', () => {
                const code = pre.querySelector('code')?.innerText || pre.innerText;
                navigator.clipboard.writeText(code).then(() => {
                    btn.innerHTML = '<i data-lucide="check" style="width:11px;height:11px;color:var(--success);"></i> Copied!';
                    if (window.lucide) lucide.createIcons();
                    setTimeout(() => {
                        btn.innerHTML = '<i data-lucide="copy" style="width:11px;height:11px;"></i> Copy';
                        if (window.lucide) lucide.createIcons();
                    }, 2000);
                });
            });
            pre.appendChild(btn);
        });
    }

    // ── Pipeline ──
    function resetPipeline() {
        [pipeAgent, pipeSkill, pipeTool, pipeReason].forEach(el => {
            el.textContent = el.id === 'pipe-agent' ? 'Chưa hoạt động' :
                             el.id === 'pipe-skill' ? 'Chưa nạp' :
                             el.id === 'pipe-tool' ? 'Đang chờ...' : 'Chưa có dữ liệu';
            el.className = 'pipeline-card-value idle';
        });
    }

    function setPipelineAnalyzing() {
        pipeAgent.innerHTML = '<span style="display:inline-flex;align-items:center;gap:6px;"><i data-lucide="loader-2" style="width:12px;height:12px;animation:spin 1s linear infinite;"></i> Đang nhận dạng...</span>';
        pipeAgent.className = 'pipeline-card-value';
        pipeSkill.textContent = 'Phân tích intent...';
        pipeSkill.className = 'pipeline-card-value';
        pipeTool.textContent = 'LLM Routing...';
        pipeTool.className = 'pipeline-card-value';
        pipeReason.textContent = 'Xác định agent tối ưu...';
        pipeReason.className = 'pipeline-card-value';
        if (window.lucide) lucide.createIcons();
    }

    function updatePipelineFromEvent(data) {
        const msg = String(data.payload || '');
        const type = data.type || '';

        if (type === 'agent_selected') {
            const meta = data.metadata || {};
            pipeAgent.textContent = meta.agent || msg;
            pipeAgent.className = 'pipeline-card-value';
            pipeReason.textContent = meta.reason || '';
            pipeReason.className = 'pipeline-card-value';
        }
        if (type === 'skill_loaded') {
            const meta = data.metadata || {};
            pipeSkill.textContent = meta.skill || msg;
            pipeSkill.className = 'pipeline-card-value';
        }
        if (type === 'tool_executed') {
            const meta = data.metadata || {};
            pipeTool.textContent = meta.tool || msg;
            pipeTool.className = 'pipeline-card-value';
        }
    }

    // ── Log entries ──
    function addLogEntry(text, type = 'info') {
        const empty = logStream.querySelector('.log-stream-empty');
        if (empty) empty.remove();

        const el = document.createElement('div');
        el.className = `log-entry ${type}`;

        const now = new Date();
        const time = now.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });

        let prefix = '';
        if (type === 'routing') prefix = '[ROUTER]';
        else if (type === 'process') prefix = '[PROC]';
        else if (type === 'tool') prefix = '[TOOL]';
        else if (type === 'success') prefix = '[OK]';
        else if (type === 'error') prefix = '[ERR]';

        el.innerHTML = `
            <span class="log-time">${time}</span>
            ${prefix ? `<span class="log-prefix">${prefix}</span>` : ''}
            <span class="log-text">${escHtml(text)}</span>
        `;

        logStream.appendChild(el);
        logStream.scrollTop = logStream.scrollHeight;
    }

    // ── Auto-resize textarea ──
    chatInput.addEventListener('input', function () {
        this.style.height = '24px';
        this.style.height = Math.min(this.scrollHeight, 160) + 'px';
    });

    // ── Event listeners ──
    sendBtn.addEventListener('click', () => {
        if (isGenerating) {
            stopGeneration();
        } else {
            sendMessage();
        }
    });

    chatInput.addEventListener('keypress', e => {
        if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); }
    });

    // Quick chips
    document.querySelectorAll('.welcome-chip[data-quick]').forEach(chip => {
        chip.addEventListener('click', () => {
            chatInput.value = chip.getAttribute('data-quick');
            sendMessage();
        });
    });

    // New chat button
    $('new-chat-btn').addEventListener('click', async () => {
        await createNewChat();
        clearChatUI();
        welcomeState.style.display = 'flex';
    });

    // ── Auto-test ──
    const testQueries = [
        "Tổng tài sản của HDB trong 10 năm qua thay đổi thế nào?",
        "Chi phí dự phòng rủi ro tín dụng của HDB năm 2024",
        "Tổng quát ngành ngân hàng năm 2025",
        "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
    ];

    runTestBtn.addEventListener('click', async () => {
        runTestBtn.disabled = true;
        addLogEntry('--- BẮT ĐẦU BỘ KIỂM THỬ ---', 'process');
        for (let i = 0; i < testQueries.length; i++) {
            addLogEntry(`[Test ${i + 1}/${testQueries.length}]`, 'process');
            await sendMessage(testQueries[i]);
            await new Promise(r => setTimeout(r, 3000));
        }
        addLogEntry('--- HOÀN THÀNH BỘ KIỂM THỬ ---', 'success');
        runTestBtn.disabled = false;
    });

    // ── Helpers ──
    function escHtml(s) {
        const d = document.createElement('div');
        d.textContent = s;
        return d.innerHTML;
    }

    function formatTime(iso) {
        if (!iso) return '';
        try {
            const d = new Date(iso);
            const now = new Date();
            const diff = (now - d) / 1000;
            if (diff < 60) return 'vừa xong';
            if (diff < 3600) return `${Math.floor(diff / 60)}p`;
            if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
            return d.toLocaleDateString('vi-VN', { day: 'numeric', month: 'short' });
        } catch { return ''; }
    }

    // ── Init ──
    setupEventSource();
    await fetchConversations();
    if (allChats.length === 0) {
        await createNewChat('Cuộc trò chuyện đầu tiên');
    } else {
        currentChatId = allChats[0].id;
        currentChatTitle.textContent = allChats[0].title || 'Cuộc trò chuyện mới';
        renderConversations();
    }

    setTimeout(() => chatInput.focus(), 100);
    if (window.lucide) lucide.createIcons();

    const style = document.createElement('style');
    style.textContent = `@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`;
    document.head.appendChild(style);

    console.log('[Indexium] AI Financial Agent ready.');
});
