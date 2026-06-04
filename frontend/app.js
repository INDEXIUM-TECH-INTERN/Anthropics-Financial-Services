document.addEventListener('DOMContentLoaded', async () => {
    // Configure marked to open links in new tab
    const renderer = new marked.Renderer();
    const linkRenderer = renderer.link;
    renderer.link = (arg1, title, text) => {
        let href = '';
        let linkText = '';
        let linkTitle = '';
        let isObj = typeof arg1 === 'object';
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
            return `<a href="#" title="${linkTitle || ''}">${linkText}</a>`;
        }
        const localLink = href.startsWith(`${window.location.protocol}//${window.location.host}`) || href.startsWith('/') || href.startsWith('#');
        const html = isObj ? linkRenderer.call(renderer, arg1) : linkRenderer.call(renderer, href, title, text);
        if (!localLink) {
            return html.replace(/^<a /, '<a target="_blank" rel="noopener noreferrer" ');
        }
        return html;
    };
    marked.setOptions({ renderer });

    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    const consoleLogs = document.getElementById('console-logs');
    const runTestBtn = document.getElementById('run-test-btn');
    const themeToggleBtn = document.getElementById('theme-toggle');
    const welcomeSection = document.getElementById('welcome-section');
    
    // Settings Modal Elements
    const settingsTrigger = document.getElementById('settings-trigger');
    const settingsModal = document.getElementById('settings-modal');
    const closeSettings = document.getElementById('close-settings');
    const saveSettingsBtn = document.getElementById('save-settings-btn');
    const geminiKeyInput = document.getElementById('gemini-key-input');
    const orKeysInput = document.getElementById('or-keys-input');
    
    // New Chat / Reset Element
    const newChatBtn = document.getElementById('new-chat-btn');

    // --- Load saved API keys from localStorage ---
    if (localStorage.getItem('gemini_key')) {
        geminiKeyInput.value = localStorage.getItem('gemini_key');
    }
    if (localStorage.getItem('openrouter_keys')) {
        orKeysInput.value = localStorage.getItem('openrouter_keys');
    }

    // --- Theme Switcher Logic ---
    const savedTheme = localStorage.getItem('theme') || 'dark';
    document.documentElement.setAttribute('data-theme', savedTheme);

    if (themeToggleBtn) {
        themeToggleBtn.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        });
    }
    
    // Tự động xác định Base URL dựa trên URL hiện tại của trình duyệt
    const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
    const currentOrigin = window.location.origin;

    const backends = {
        gemini: isLocalhost ? "http://localhost:8080" : currentOrigin,
        claude: "http://localhost:8081" // Vẫn giữ localhost cho các port khác nếu chưa tunnel
    };

    let currentBackend = 'gemini';
    let currentBaseUrl = backends[currentBackend];
    let eventSource = null;

    // --- Multi-chat sessions backed by Redis (server) ---
    let allChats = []; // [{id, title, ...} from /api/chats]
    let currentChatId = null;

    const chatsList = document.getElementById('chats-list');

    function updateSidebarList() {
        if (!chatsList) return;
        chatsList.innerHTML = '';
        if (allChats.length === 0) {
            chatsList.innerHTML = '<div style="padding: 12px; font-size:13px; opacity:0.6; text-align:center;">Chưa có đoạn chat nào.</div>';
            return;
        }
        allChats.forEach(chat => {
            const isCurrent = chat.id === currentChatId;
            const chatItem = document.createElement('div');
            chatItem.className = `chat-item ${isCurrent ? 'active' : ''}`;
            
            // Title span
            const titleSpan = document.createElement('span');
            titleSpan.className = 'chat-item-title';
            titleSpan.textContent = chat.title || 'Cuộc trò chuyện mới';
            titleSpan.addEventListener('click', () => {
                switchToChat(chat.id);
            });
            chatItem.appendChild(titleSpan);

            // Delete button
            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'delete-chat-btn';
            deleteBtn.title = 'Xóa đoạn chat';
            deleteBtn.innerHTML = '<i data-lucide="trash-2"></i>';
            deleteBtn.addEventListener('click', async (e) => {
                e.stopPropagation();
                if (confirm('Bạn có chắc chắn muốn xóa đoạn chat này?')) {
                    await deleteChatFromServer(chat.id);
                }
            });
            chatItem.appendChild(deleteBtn);

            chatsList.appendChild(chatItem);
        });

        if (window.lucide) {
            lucide.createIcons();
        }
    }

    async function deleteChatFromServer(chatId) {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats?chat_id=${encodeURIComponent(chatId)}`, {
                method: 'DELETE'
            });
            if (!res.ok) throw new Error('delete failed');
            
            allChats = allChats.filter(c => c.id !== chatId);
            logToConsole('Đã xóa đoạn chat thành công.', 'success');
            
            if (currentChatId === chatId) {
                if (allChats.length > 0) {
                    await switchToChat(allChats[0].id);
                } else {
                    await createNewChatOnServer();
                }
            } else {
                updateSidebarList();
            }
        } catch (e) {
            console.error(e);
            logToConsole('Không thể xóa đoạn chat: ' + e.message, 'error');
        }
    }

    async function fetchChatsFromServer() {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats`);
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            allChats = Array.isArray(data.chats) ? data.chats : (Array.isArray(data) ? data : []);
            updateSidebarList();
            return allChats;
        } catch (e) {
            console.warn('[UI] fetch /api/chats failed, falling back to empty', e);
            allChats = [];
            updateSidebarList();
            return [];
        }
    }

    async function createNewChatOnServer(title = 'Cuộc trò chuyện mới') {
        try {
            const res = await fetch(`${currentBaseUrl}/api/chats`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title })
            });
            if (!res.ok) throw new Error('create failed');
            const created = await res.json();
            allChats.unshift(created);
            currentChatId = created.id;

            // clear UI for new chat
            const existing = chatHistory.querySelectorAll('.message');
            existing.forEach(el => el.remove());
            if (welcomeSection) welcomeSection.style.display = 'flex';

            logToConsole('Đã tạo đoạn chat mới (lưu Redis).', 'success');
            updateSidebarList();
            return created;
        } catch (e) {
            console.error(e);
            // local fallback
            const fb = { id: 'local_' + Date.now(), title, messages: [] };
            allChats.unshift(fb);
            currentChatId = fb.id;
            const existing = chatHistory.querySelectorAll('.message');
            existing.forEach(el => el.remove());
            if (welcomeSection) welcomeSection.style.display = 'flex';
            updateSidebarList();
            return fb;
        }
    }

    async function switchToChat(chatId) {
        currentChatId = chatId;
        updateSidebarList();
        const chatMeta = allChats.find(c => c.id === chatId);

        // clear UI
        const existing = chatHistory.querySelectorAll('.message');
        existing.forEach(el => el.remove());

        // Fetch full history for this chat_id from Redis via server for immediate display
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
                        text.includes('=== TÓM TẮT NGỮ CẢNH')) {
                        return;
                    }
                    const sender = (role === 'user') ? 'user' : 'bot';
                    addMessage(text, sender);
                    hasContent = true;
                });
                if (hasContent) {
                    if (welcomeSection) welcomeSection.style.display = 'none';
                    chatHistory.scrollTop = chatHistory.scrollHeight;
                    return;
                }
            }
        } catch (e) {
            console.warn('Failed to fetch history for switch', e);
        }

        if (welcomeSection) welcomeSection.style.display = 'flex';

        logToConsole(`Đã chuyển sang đoạn chat: ${chatMeta ? chatMeta.title : chatId}`, 'info');
    }

    function getCurrentChat() {
        if (!currentChatId) return null;
        return allChats.find(c => c.id === currentChatId);
    }

    // Load and render chat segments from server (source of truth for current session)
    // This makes F5 / refresh restore the "đoạn chat" from backend history
    async function loadAndRenderHistoryFromServer() {
        try {
            const res = await fetch(`${currentBaseUrl}/api/history`);
            if (!res.ok) return;
            const data = await res.json();
            const hist = data.history || data.History || [];
            if (!Array.isArray(hist) || hist.length === 0) return;

            const current = getCurrentChat();
            if (!current) return;

            // Remove any existing message bubbles (keep welcome for now)
            const existing = chatHistory.querySelectorAll('.message');
            existing.forEach(el => el.remove());

            let hasVisible = false;
            hist.forEach(h => {
                const role = String(h.role || h.Role || '').toLowerCase();
                if (role === 'tool') return; // internal, don't show in main chat

                let text = h.content || h.Content || '';
                // Skip internal bootstrap / huge context injections
                if (!text || text.length > 4000 || 
                    text.includes('ANTHROPIC AGENT CONFIG') || 
                    text.includes('SKILL MARKDOWN') ||
                    text.includes('=== TÓM TẮT NGỮ CẢNH')) {
                    return;
                }

                const sender = (role === 'user') ? 'user' : 'bot';
                addMessage(text, sender);
                hasVisible = true;

                // add to current chat
                current.messages.push({sender, text});
            });

            if (hasVisible) {
                if (welcomeSection) welcomeSection.style.display = 'none';
                chatHistory.scrollTop = chatHistory.scrollHeight;
            }

            console.log('[UI] Restored', hist.length, 'history items from server into current chat');
        } catch (e) {
            console.warn('[UI] Could not load history from server:', e);
        }
    }

    function setupEventSource() {
        if (eventSource) {
            eventSource.close();
        }
        
        const eventsUrl = `${currentBaseUrl}/api/events`;
        eventSource = new EventSource(eventsUrl);
        
        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                logToConsole(data); // Pass the full structured event
            } catch (e) {}
        };
        
        eventSource.onerror = () => {
            logToConsole(`Mất kết nối SSE tới backend ${currentBackend}. Đang thử kết nối lại...`, "error");
        };
        
        logToConsole(`Đã kết nối tới backend ${currentBackend} (SSE)`, "success");
    }

    const backendSelect = document.getElementById('backend-select');
    if (backendSelect) {
        backendSelect.addEventListener('change', (e) => {
            currentBackend = e.target.value;
            currentBaseUrl = backends[currentBackend];
            logToConsole(`Đang chuyển đổi sang backend ${currentBackend}...`, "process");
            setupEventSource();
        });
    }

    // Settings Modal Handlers
    if (settingsTrigger && settingsModal) {
        settingsTrigger.addEventListener('click', () => {
            settingsModal.style.display = 'flex';
        });
    }

    if (closeSettings && settingsModal) {
        closeSettings.addEventListener('click', () => {
            settingsModal.style.display = 'none';
        });
        
        // Click outside modal content to close
        settingsModal.addEventListener('click', (e) => {
            if (e.target === settingsModal) {
                settingsModal.style.display = 'none';
            }
        });
    }

    if (saveSettingsBtn) {
        saveSettingsBtn.addEventListener('click', async () => {
            const geminiKey = geminiKeyInput.value.trim();
            const orKeysValue = orKeysInput.value.trim();
            
            // Save locally
            localStorage.setItem('gemini_key', geminiKey);
            localStorage.setItem('openrouter_keys', orKeysValue);
            
            // Parse openrouter keys by line
            const keysArray = orKeysValue.split('\n').map(k => k.trim()).filter(k => k !== '');
            
            saveSettingsBtn.disabled = true;
            saveSettingsBtn.textContent = 'Đang lưu...';
            
            try {
                // Post OpenRouter keys to backend config endpoint
                const configUrl = `${currentBaseUrl}/api/config/keys`;
                const response = await fetch(configUrl, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ keys: keysArray })
                });
                
                const resData = await response.json();
                if (response.ok && resData.status === 'success') {
                    logToConsole('Đã cập nhật cấu hình API Keys thành công lên backend.', 'success');
                    alert('Lưu cấu hình và đồng bộ hóa thành công!');
                } else {
                    logToConsole('Có lỗi xảy ra khi cập nhật API Keys lên backend: ' + (resData.message || 'Lỗi không xác định'), 'error');
                    alert('Lưu cấu hình lỗi: ' + (resData.message || 'Kiểm tra log.'));
                }
            } catch (err) {
                logToConsole('Không thể kết nối đến backend để lưu API Keys: ' + err.message, 'error');
                alert('Không thể kết nối đến backend để lưu API Keys.');
            } finally {
                saveSettingsBtn.disabled = false;
                saveSettingsBtn.textContent = 'Lưu cấu hình';
                settingsModal.style.display = 'none';
            }
        });
    }

    // Reset Chat / New Chat handler
    if (newChatBtn) {
        newChatBtn.addEventListener('click', async () => {
            logToConsole('Đang tạo cuộc trò chuyện mới...', 'process');
            
            try {
                await createNewChatOnServer(); // creates a new session on Redis and clears active UI viewport
                
                sourcesList.innerHTML = '<div class="empty-state">Chưa có tài liệu nào trong ngữ cảnh hiện tại.</div>';
                consoleLogs.innerHTML = '<div class="pipeline-empty">Sẵn sàng phân tích yêu cầu khi có câu hỏi.</div>';
                
                // Reset Live Status Board
                document.getElementById('current-agent').textContent = "Chưa hoạt động";
                document.getElementById('current-skill').textContent = "Chưa nạp";
                document.getElementById('current-tool').textContent = "Đang chờ câu hỏi...";
                document.getElementById('current-reason').textContent = "Chưa có dữ liệu phân tích";
                
                logToConsole('Đã tạo cuộc trò chuyện mới.', 'success');
            } catch (err) {
                logToConsole('Lỗi khi khởi tạo cuộc trò chuyện mới.', 'error');
            }
        });
    }

    // Initial setup
    setupEventSource();

    // Wire the "Tất cả đoạn chat" button (folder-kanban) to toggle the sidebar
    const allChatsBtn = document.getElementById('all-chats-btn');
    const chatsSidebar = document.getElementById('chats-sidebar');
    if (allChatsBtn && chatsSidebar) {
        allChatsBtn.addEventListener('click', () => {
            chatsSidebar.classList.toggle('collapsed');
        });
    }

    const testQueries = [
        "Tổng tài sản của HDB trong 10 năm qua thay đổi thế nào?",
        "Chỉ tiêu 'Chi phí dự phòng rủi ro tín dụng' nằm ở trang nào trong báo cáo tài chính năm 2023 của HDB?",
        "Trích dẫn báo cáo nói về chi phí dự phòng của HDB năm 2024",
        "Chi phí dự phòng rủi ro của HDB năm 2024",
        "Báo cáo tài chính 6 tháng đầu năm 2025?",
        "HDB có những lợi thế gì trong những năm gần đây để bức tốc phát triển trong các năm sắp tới?",
        "Ban lãnh đạo hiện tại của HDB gồm những ai?",
        "Tổng quát ngành ngân hàng năm 2025",
        "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
    ];

    // Auto-resize textarea
    chatInput.addEventListener('input', function() {
        this.style.height = '24px';
        this.style.height = (this.scrollHeight) + 'px';
    });

    function logToConsole(message, type = 'info') {
        // Clear empty pipeline state if present
        const emptyState = consoleLogs.querySelector('.pipeline-empty');
        if (emptyState) {
            consoleLogs.innerHTML = '';
        }

        // Handle structured event object
        if (message && typeof message === 'object') {
            type = message.type || type;
            message = message.payload || '';
        }

        const logDiv = document.createElement('div');
        logDiv.className = `log-entry ${type}`;
        
        // Define prefix & Icon based on type
        let prefix = '';
        let iconName = 'info';
        
        if (type === 'routing') {
            prefix = '[ROUTING] ';
            iconName = 'git-commit';
        } else if (type === 'process') {
            prefix = '[PROCESS] ';
            iconName = 'activity';
        } else if (type === 'tool') {
            prefix = '[TOOL] ';
            iconName = 'tool';
        } else if (type === 'success') {
            prefix = '[SUCCESS] ';
            iconName = 'check-circle2';
        } else if (type === 'error') {
            prefix = '[ERROR] ';
            iconName = 'alert-triangle';
        }

        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });
        
        logDiv.innerHTML = `
            <div style="display: flex; align-items: flex-start; gap: 8px;">
                <i data-lucide="${iconName}" style="width: 14px; height: 14px; margin-top: 2px; flex-shrink: 0;"></i>
                <div style="flex: 1;">
                    <span style="opacity: 0.5; font-size: 10px; margin-right: 6px; font-family: var(--font-mono);">${timeString}</span>
                    <span class="log-payload">${prefix}${message}</span>
                </div>
            </div>
        `;
        
        consoleLogs.appendChild(logDiv);
        
        // Render lucide icon in log entry
        if (window.lucide) {
            lucide.createIcons({
                attrs: {
                    class: 'status-icon-mini'
                },
                nameAttr: 'data-lucide'
            });
        }
        
        // Auto-scroll to bottom
        consoleLogs.scrollTop = consoleLogs.scrollHeight;

        // Parse log message to update Structured Live Status Board
        updateLiveStatusBoard(message, type);
    }

    function updateLiveStatusBoard(message, type) {
        const currentAgentEl = document.getElementById('current-agent');
        const currentSkillEl = document.getElementById('current-skill');
        const currentToolEl = document.getElementById('current-tool');
        const currentReasonEl = document.getElementById('current-reason');

        if (!currentAgentEl || !currentSkillEl || !currentToolEl || !currentReasonEl) return;

        // 1. Reset Board when a new analysis starts
        if (message.includes("Đang phân tích yêu cầu để chọn Agent tối ưu...")) {
            currentAgentEl.innerHTML = '<span style="color: var(--accent);"><i data-lucide="loader-2" class="animate-spin" style="width: 12px; height: 12px; display: inline-block; vertical-align: middle; margin-right: 4px;"></i> System Router</span>';
            currentSkillEl.textContent = "Analyzing Intent...";
            currentToolEl.textContent = "LLM Routing...";
            currentReasonEl.textContent = "Determining the best financial agent...";
            if (window.lucide) lucide.createIcons();
            return;
        }

        // 2. Agent Selection and Reason
        const agentMatch = message.match(/Đã chọn Agent:\s*([a-zA-Z0-9\-_]+)/i);
        if (agentMatch) {
            const agentName = agentMatch[1];
            currentAgentEl.innerHTML = `<i data-lucide="bot" style="width: 12px; height: 12px; display: inline-block; vertical-align: middle; margin-right: 4px; color: var(--accent);"></i> ${agentName}`;
            
            // Extract reason if present
            const reasonMatch = message.match(/\((?:Lý do|Reason):\s*(.+)\)/i);
            if (reasonMatch) {
                currentReasonEl.textContent = reasonMatch[1];
            }
            if (window.lucide) lucide.createIcons();
            return;
        }

        // 3. Skill Loading
        if (message.includes("Đang nạp skill chuyên biệt:")) {
            const skillName = message.replace("Đang nạp skill chuyên biệt:", "").trim();
            currentSkillEl.innerHTML = `<i data-lucide="award" style="width: 12px; height: 12px; display: inline-block; vertical-align: middle; margin-right: 4px; color: var(--primary);"></i> ${skillName}`;
            if (window.lucide) lucide.createIcons();
            return;
        }

        // 4. Tool Execution
        if (type === 'tool' || message.includes("Thực thi Tool:") || message.includes("Tra cứu dữ liệu:")) {
            let toolDisplay = message;
            if (message.includes("Thực thi Tool:")) {
                const toolName = message.replace("Thực thi Tool:", "").replace("...", "").trim();
                toolDisplay = `Running: ${toolName}`;
            } else if (message.includes("Tra cứu dữ liệu:")) {
                const query = message.replace("Tra cứu dữ liệu:", "").trim();
                toolDisplay = `Searching: ${query}`;
            } else if (message.includes("Đang đọc nội dung từ:")) {
                const url = message.replace("Đang đọc nội dung từ:", "").trim();
                try {
                    const domain = new URL(url).hostname.replace('www.', '');
                    toolDisplay = `Scraping: ${domain}`;
                } catch(e) {
                    toolDisplay = `Scraping Document`;
                }
            } else if (message.includes("Thực hiện tính toán:")) {
                const expr = message.replace("Thực hiện tính toán:", "").trim();
                toolDisplay = `Calculating: ${expr}`;
            }
            
            currentToolEl.innerHTML = `<i data-lucide="play" style="width: 10px; height: 10px; display: inline-block; vertical-align: middle; margin-right: 4px; color: var(--warning);"></i> ${toolDisplay}`;
            if (window.lucide) lucide.createIcons();
            return;
        }

        // 5. Transfer / Handoff
        if (message.includes("Yêu cầu chuyển giao sang:")) {
            const target = message.replace("Yêu cầu chuyển giao sang:", "").trim();
            currentAgentEl.innerHTML = `<i data-lucide="git-pull-request" style="width: 12px; height: 12px; display: inline-block; vertical-align: middle; margin-right: 4px; color: var(--accent);"></i> ${target}`;
            currentToolEl.textContent = `Handing off...`;
            if (window.lucide) lucide.createIcons();
            return;
        }
    }

    function renderSources(markdownText, targetElement) {
        const urlRegex = /(https?:\/\/[^\s\)\]]+)/g;
        const urls = markdownText.match(urlRegex) || [];
        const uniqueUrls = [...new Set(urls)];
        
        if (uniqueUrls.length > 0) {
            // 1. Update Global Sidebar
            sourcesList.innerHTML = '';
            uniqueUrls.forEach((url, index) => {
                let domain = url;
                try { domain = new URL(url).hostname; } catch(e) {}
                const card = document.createElement('div');
                card.className = 'source-card';
                card.innerHTML = `
                    <div style="display: flex; align-items: center; gap: 8px;">
                        <i data-lucide="file-text" style="width: 12px; height: 12px; color: var(--primary);"></i>
                        <span style="color: var(--on-surface-variant); font-size: 11px; font-weight: 700;">SOURCE [${index+1}]</span>
                    </div>
                    <a href="${url}" target="_blank" rel="noopener noreferrer">${domain}</a>
                `;
                sourcesList.appendChild(card);
            });

            // 2. Add Inline Sources at the END of message
            const sourcesContainer = document.createElement('div');
            sourcesContainer.className = 'message-sources';
            sourcesContainer.innerHTML = '<div class="sources-label">Nguồn tài liệu</div>';
            
            const grid = document.createElement('div');
            grid.className = 'sources-grid';
            
            uniqueUrls.forEach((url, index) => {
                let domain = url;
                try { domain = new URL(url).hostname.replace('www.', ''); } catch(e) {}
                const card = document.createElement('a');
                card.href = url;
                card.target = "_blank";
                card.rel = "noopener noreferrer";
                card.className = 'inline-source-card';
                card.style.textDecoration = 'none';
                card.innerHTML = `
                    <div class="source-index">${index+1}</div>
                    <span class="source-link">${domain}</span>
                `;
                grid.appendChild(card);
            });
            
            sourcesContainer.appendChild(grid);
            targetElement.appendChild(sourcesContainer);
            
            if (window.lucide) lucide.createIcons();
        }
    }

    function addMessage(text, sender, isLoading = false) {
        console.log('[UI] Creating chat segment:', sender, text ? text.substring(0, 50) + '...' : '(loading)');
        const msgDiv = document.createElement('div');
        msgDiv.className = `message ${sender}`;
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'msg-content';
        
        if (isLoading) {
            contentDiv.innerHTML = `
                <div class="typing-dots">
                    <div class="dot"></div>
                    <div class="dot"></div>
                    <div class="dot"></div>
                </div>
            `;
        } else if (sender === 'bot') {
            contentDiv.innerHTML = marked.parse(text);
            renderSources(text, contentDiv);
        } else {
            contentDiv.textContent = text;
        }
        
        msgDiv.appendChild(contentDiv);

        if (sender === 'bot') {
            const footer = document.createElement('div');
            footer.className = 'msg-footer';
            msgDiv.appendChild(footer);
        }

        chatHistory.appendChild(msgDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
        return msgDiv;
    }

    function diagnoseError(errorName, errorMessage, errorStack = '') {
        const name = String(errorName || '').trim();
        const msg = String(errorMessage || '').toLowerCase();
        
        let badge = 'Hệ thống';
        let title = 'Lỗi xử lý hệ thống';
        let icon = 'alert-triangle';
        let description = errorMessage || 'Đã xảy ra lỗi không xác định khi Agent xử lý yêu cầu của bạn.';
        let suggestions = [
            'Kiểm tra lại cấu hình API Key trong mục Cài đặt (biểu tượng thanh trượt ở góc dưới cùng bên trái).',
            'Thử lại sau ít phút hoặc nhấn nút Reset cuộc trò chuyện (biểu tượng cộng ở thanh bên trái).'
        ];

        // 1. Connection Refused / Fetch failure
        if (msg.includes('failed to fetch') || msg.includes('connection refused') || msg.includes('network error') || msg.includes('net::err_connection_refused')) {
            badge = 'Kết nối mạng';
            title = 'Lỗi kết nối máy chủ';
            icon = 'wifi-off';
            description = 'Không thể kết nối đến máy chủ Backend (địa chỉ ' + currentBaseUrl + '). Vui lòng kiểm tra kết nối mạng hoặc trạng thái hoạt động của Server.';
            suggestions = [
                'Chắc chắn rằng bạn đã khởi chạy backend Go (bằng lệnh <code>go run cmd/gemini-cli/main.go -server</code> hoặc chạy file script <code>run-server.ps1</code>).',
                'Kiểm tra xem cổng dịch vụ <code>8080</code> có đang bị chặn bởi tường lửa hoặc ứng dụng khác chiếm dụng không.',
                'Nếu chạy trên môi trường Docker hoặc máy chủ ảo, hãy xác minh cấu hình port forwarding.'
            ];
        }
        // 2. Rate limits or Quota exceeded
        else if (msg.includes('quota') || msg.includes('rate') || msg.includes('429') || msg.includes('limit') || msg.includes('exceeded')) {
            badge = 'Giới hạn dùng';
            title = 'Hết hạn mức truy cập (429)';
            icon = 'alert-octagon';
            description = 'Tài khoản của bạn đã vượt quá giới hạn cuộc gọi cho phép (Quota Exceeded hoặc Rate Limit) từ nhà cung cấp API (Gemini / OpenRouter).';
            suggestions = [
                'Mở mục <b>Cài đặt Workspace</b> (biểu tượng thanh trượt ở góc dưới bên trái) và điền thêm các API Key dự phòng (mỗi dòng một khóa) để kích hoạt cơ chế luân phiên (key rotation) tự động của backend.',
                'Sử dụng một API Key khác có hạn mức cao hơn.',
                'Vui lòng chờ khoảng 1-2 phút trước khi gửi yêu cầu tiếp theo để giới hạn lượt dùng được khôi phục.'
            ];
        }
        // 3. Model signature issue
        else if (msg.includes('thought_signature') || msg.includes('thought signature') || msg.includes('missing a thought_signature') || msg.includes('function call is missing')) {
            badge = 'Tương thích Model';
            title = 'Lỗi chữ ký mô hình';
            icon = 'shield-alert';
            description = 'Mô hình Gemini Preview mới (ví dụ: gemini-3.1-flash-lite) yêu cầu chữ ký tư duy đặc biệt (thought_signature) mà backend Go hiện tại chưa tương thích đầy đủ khi chạy tool calling.';
            suggestions = [
                'Mở file cấu hình <code>.env</code> ở thư mục gốc của dự án và thay đổi giá trị thành <code>GEMINI_MODEL=gemini-1.5-flash</code> hoặc <code>gemini-1.5-pro</code> (các bản ổn định không bắt buộc thought_signature).',
                'Khởi động lại backend server sau khi cập nhật file cấu hình .env.'
            ];
        }
        // 4. Timeout error
        else if (name === 'AbortError' || msg.includes('timeout') || msg.includes('quá thời gian')) {
            badge = 'Thời gian chờ';
            title = 'Yêu cầu quá thời gian phản hồi';
            icon = 'clock';
            description = 'Thời gian xử lý của Agent đã vượt quá giới hạn tối đa cho phép (300 giây). Hệ thống đang xử lý quá nhiều tài liệu lớn hoặc dịch vụ AI phản hồi quá chậm.';
            suggestions = [
                'Kiểm tra console/terminal của Backend Go để xem Agent đang bị treo ở công cụ tìm kiếm hay scraping nào.',
                'Rút ngắn câu hỏi hoặc chia nhỏ tác vụ để giảm tải khối lượng xử lý cho AI.',
                'Kiểm tra lại đường truyền mạng Internet của bạn.'
            ];
        }
        // 5. Frontend TypeError or code bug
        else if (name.includes('Type') || name.includes('Syntax') || name.includes('Reference') || msg.includes('starts-with') || msg.includes('is not a function')) {
            badge = 'Mã nguồn Giao diện';
            title = 'Lỗi xử lý logic giao diện';
            icon = 'code';
            description = 'Đã xảy ra lỗi lập trình Javascript không mong muốn ở trình duyệt khi hiển thị tin nhắn.';
            suggestions = [
                'Vui lòng làm mới trang (nhấn <b>F5</b> hoặc Ctrl+F5) để khôi phục lại trạng thái giao diện sạch và thử lại.',
                'Sao chép chi tiết ngăn xếp lỗi (stack trace) ở phần dưới và gửi cho lập trình viên để vá lỗi trong file <code>app.js</code>.'
            ];
        }

        return { badge, title, icon, description, suggestions };
    }

    async function sendMessage(textOverride = null) {
        const text = textOverride !== null ? textOverride : chatInput.value.trim();
        if (!text) return false;

        // ensure we have a current chat session (server-backed)
        if (!getCurrentChat()) {
            await createNewChatOnServer();
        }

        // Hide welcome section on first query
        if (welcomeSection && welcomeSection.style.display !== 'none') {
            welcomeSection.style.display = 'none';
        }

        addMessage(text, 'user');

        chatInput.value = '';
        chatInput.style.height = '24px';
        sendBtn.disabled = true;
        if (runTestBtn) runTestBtn.disabled = true;

        logToConsole(`Yêu cầu mới: "${text.substring(0, 45)}..."`, 'info');

        // Reset Live Status Board on new submit
        const currentAgentEl = document.getElementById('current-agent');
        const currentSkillEl = document.getElementById('current-skill');
        const currentToolEl = document.getElementById('current-tool');
        const currentReasonEl = document.getElementById('current-reason');
        if (currentAgentEl && currentSkillEl && currentToolEl && currentReasonEl) {
            currentAgentEl.innerHTML = '<span style="color: var(--accent);"><i data-lucide="loader-2" class="animate-spin" style="width: 12px; height: 12px; display: inline-block; vertical-align: middle; margin-right: 4px;"></i> Đang nhận dạng...</span>';
            currentSkillEl.textContent = "Đang chờ...";
            currentToolEl.textContent = "Bắt đầu tiến trình...";
            currentReasonEl.textContent = "Đang phân tích yêu cầu...";
            if (window.lucide) lucide.createIcons();
        }

        const botMsgDiv = addMessage('', 'bot', true);
        const footer = botMsgDiv.querySelector('.msg-footer');
        
        // Start Live Timer
        const startTime = Date.now();
        const timerInterval = setInterval(() => {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            footer.innerHTML = `<span class="timer">⏱ Đang xử lý... ${elapsed}s</span>`;
        }, 100);

        const TIMEOUT_MS = 300000;
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), TIMEOUT_MS);

        try {
            const apiUrl = `${currentBaseUrl}/api/chat`;
            const payload = { message: text };
            if (currentChatId) {
                payload.chat_id = currentChatId;
            }
            const response = await fetch(apiUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
                signal: controller.signal
            });
            const data = await response.json();
            const finalDuration = ((Date.now() - startTime) / 1000).toFixed(1);

            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            
            const contentDiv = botMsgDiv.querySelector('.msg-content');
            botMsgDiv.classList.remove('loading');

            if (data.error) {
                botMsgDiv.classList.add('error-message');
                const diag = diagnoseError('BackendError', data.error);
                contentDiv.innerHTML = `
                    <div class="error-container">
                        <div class="error-header-row">
                            <div class="error-badge-title">
                                <i data-lucide="${diag.icon}"></i>
                                <span>${diag.title}</span>
                            </div>
                            <span class="error-class-badge">${diag.badge}</span>
                        </div>
                        <div class="error-explanation-box">
                            <div class="error-body-text">${diag.description}</div>
                        </div>
                        <div class="error-action-panel">
                            <div class="error-action-title">
                                <i data-lucide="check-square" style="width: 14px; height: 14px;"></i>
                                Đề xuất khắc phục
                            </div>
                            <ul class="error-action-list">
                                ${diag.suggestions.map(s => `
                                    <li class="error-action-item">
                                        <i data-lucide="arrow-right-circle"></i>
                                        <span>${s}</span>
                                    </li>
                                `).join('')}
                            </ul>
                        </div>
                        <div class="error-details-accordion">
                            <button class="details-toggle-btn">
                                <span>Chi tiết kỹ thuật (Developer)</span>
                                <i data-lucide="chevron-down"></i>
                            </button>
                            <div class="details-content" style="display: none;">
                                <div style="font-weight: 700; margin-bottom: 6px; color: #fca5a5; font-family: var(--font-mono); font-size: 12px;">[BackendError]</div>
                                <div style="margin-bottom: 10px; font-family: var(--font-mono); font-size: 12px; color: #fecaca; line-height: 1.5;">${data.error}</div>
                            </div>
                        </div>
                    </div>
                `;
                
                const toggleBtn = contentDiv.querySelector('.details-toggle-btn');
                const detailsContent = contentDiv.querySelector('.details-content');
                if (toggleBtn && detailsContent) {
                    toggleBtn.addEventListener('click', () => {
                        const isHidden = detailsContent.style.display === 'none';
                        detailsContent.style.display = isHidden ? 'block' : 'none';
                        toggleBtn.querySelector('i').setAttribute('data-lucide', isHidden ? 'chevron-up' : 'chevron-down');
                        if (window.lucide) lucide.createIcons();
                    });
                }
                
                footer.innerHTML = `<span class="timer" style="color:var(--danger)">Thất bại sau ${finalDuration}s</span>`;
                if (window.lucide) lucide.createIcons();
                return false;
            } else {
                contentDiv.innerHTML = marked.parse(data.reply);
                renderSources(data.reply, contentDiv);
                
                // no local persist needed, server is source of truth
                
                // Add copy buttons to code blocks
                contentDiv.querySelectorAll('pre').forEach(pre => {
                    const copyBtn = document.createElement('button');
                    copyBtn.className = 'copy-code-btn';
                    copyBtn.innerHTML = '<i data-lucide="copy" style="width: 12px; height: 12px;"></i> Copy';
                    copyBtn.style.position = 'absolute';
                    copyBtn.style.top = '8px';
                    copyBtn.style.right = '8px';
                    copyBtn.style.background = 'rgba(255,255,255,0.1)';
                    copyBtn.style.border = 'none';
                    copyBtn.style.color = '#fff';
                    copyBtn.style.borderRadius = '4px';
                    copyBtn.style.padding = '4px 8px';
                    copyBtn.style.fontSize = '11px';
                    copyBtn.style.cursor = 'pointer';
                    copyBtn.style.display = 'flex';
                    copyBtn.style.alignItems = 'center';
                    copyBtn.style.gap = '4px';
                    
                    pre.appendChild(copyBtn);
                    
                    copyBtn.addEventListener('click', () => {
                        const code = pre.querySelector('code').innerText;
                        navigator.clipboard.writeText(code).then(() => {
                            copyBtn.innerHTML = '<i data-lucide="check" style="width: 12px; height: 12px; color: var(--success);"></i> Copied!';
                            if (window.lucide) lucide.createIcons();
                            setTimeout(() => {
                                copyBtn.innerHTML = '<i data-lucide="copy" style="width: 12px; height: 12px;"></i> Copy';
                                if (window.lucide) lucide.createIcons();
                            }, 2000);
                        });
                    });
                });
                
                footer.innerHTML = `
                    <div class="metrics-modern">
                        <span class="metric-item">Độ trễ: ${finalDuration}s</span>
                        <span class="metric-divider"></span>
                        <span class="metric-item">RAM: ${data.metrics.ram_mb}</span>
                        <span class="metric-divider"></span>
                        <span class="metric-item">Tải: ${data.metrics.cpu_load.split(' ')[0]} Goroutines</span>
                        <span class="metric-divider"></span>
                        <span class="metric-item">Tokens: ${data.metrics.token_in}/${data.metrics.token_out}</span>
                    </div>
                `;
                if (window.lucide) lucide.createIcons();

                // refresh chat list in background (for title/count updates from server)
                fetchChatsFromServer().catch(() => {});
                return true;
            }
        } catch (error) {
            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            
            const errorName = error.name || 'Error';
            const errorDetails = error.message || String(error);
            const errorStack = error.stack || '';
            const diag = diagnoseError(errorName, errorDetails, errorStack);

            botMsgDiv.classList.add('error-message');
            const contentDiv = botMsgDiv.querySelector('.msg-content');
            contentDiv.innerHTML = `
                <div class="error-container">
                    <div class="error-header-row">
                        <div class="error-badge-title">
                            <i data-lucide="${diag.icon}"></i>
                            <span>${diag.title}</span>
                        </div>
                        <span class="error-class-badge">${diag.badge}</span>
                    </div>
                    <div class="error-explanation-box">
                        <div class="error-body-text">${diag.description}</div>
                    </div>
                    <div class="error-action-panel">
                        <div class="error-action-title">
                            <i data-lucide="check-square" style="width: 14px; height: 14px;"></i>
                            Đề xuất khắc phục
                        </div>
                        <ul class="error-action-list">
                            ${diag.suggestions.map(s => `
                                <li class="error-action-item">
                                    <i data-lucide="arrow-right-circle"></i>
                                    <span>${s}</span>
                                </li>
                            `).join('')}
                        </ul>
                    </div>
                    <div class="error-details-accordion">
                        <button class="details-toggle-btn">
                            <span>Chi tiết kỹ thuật (Developer)</span>
                            <i data-lucide="chevron-down"></i>
                        </button>
                        <div class="details-content" style="display: none;">
                            <div style="font-weight: 700; margin-bottom: 6px; color: #fca5a5; font-family: var(--font-mono); font-size: 12px;">[${errorName}]</div>
                            <div style="margin-bottom: 10px; font-family: var(--font-mono); font-size: 12px; color: #fecaca; line-height: 1.5;">${errorDetails}</div>
                            ${errorStack ? `<pre style="margin: 0; padding: 10px; background: rgba(0,0,0,0.4); border-radius: var(--radius-sm); border: 1px solid rgba(255,255,255,0.05); overflow-x: auto; max-height: 150px;"><code style="font-size: 11px; color: #fca5a5; white-space: pre;">${errorStack}</code></pre>` : ''}
                        </div>
                    </div>
                </div>
            `;

            const toggleBtn = contentDiv.querySelector('.details-toggle-btn');
            const detailsContent = contentDiv.querySelector('.details-content');
            if (toggleBtn && detailsContent) {
                toggleBtn.addEventListener('click', () => {
                    const isHidden = detailsContent.style.display === 'none';
                    detailsContent.style.display = isHidden ? 'block' : 'none';
                    toggleBtn.querySelector('i').setAttribute('data-lucide', isHidden ? 'chevron-up' : 'chevron-down');
                    if (window.lucide) lucide.createIcons();
                });
            }

            footer.innerHTML = `<span class="timer" style="color:var(--danger)">Lỗi kết nối</span>`;
            if (window.lucide) lucide.createIcons();
            return false;
        } finally {
            sendBtn.disabled = false;
            if (runTestBtn) runTestBtn.disabled = false;
            chatInput.focus();
            chatHistory.scrollTop = chatHistory.scrollHeight;
        }
    }

    async function runAutoTest() {
        if (runTestBtn) runTestBtn.disabled = true;
        logToConsole(`--- BẮT ĐẦU CHẠY BỘ KIỂM THỬ TỰ ĐỘNG ---`, 'process');
        
        for (let i = 0; i < testQueries.length; i++) {
            logToConsole(`[Test ${i+1}/${testQueries.length}] Đang chạy câu hỏi...`, 'process');
            await sendMessage(testQueries[i]);
            
            logToConsole(`[Test ${i+1}/${testQueries.length}] Hoàn thành. Nghỉ 3 giây...`, 'info');
            await new Promise(r => setTimeout(r, 3000));
        }
        
        logToConsole(`--- HOÀN THÀNH BỘ KIỂM THỬ TỰ ĐỘNG ---`, 'success');
        if (runTestBtn) runTestBtn.disabled = false;
    }

    if (runTestBtn) {
        runTestBtn.addEventListener('click', runAutoTest);
    }

    sendBtn.addEventListener('click', () => sendMessage());
    chatInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    // Attach quick suggestion chips (data-quick)
    document.querySelectorAll('.discovery-chips .chip[data-quick]').forEach(chip => {
        chip.addEventListener('click', () => {
            const text = chip.getAttribute('data-quick');
            if (text) {
                chatInput.value = text;
                sendMessage();
            }
        });
    });

    // Initialize multi-chat from Redis-backed server on load
    await fetchChatsFromServer();
    if (allChats.length === 0) {
        await createNewChatOnServer('Cuộc trò chuyện đầu tiên');
    } else {
        // load the most recent one
        currentChatId = allChats[0].id;
        // Messages will be loaded by backend when sending with this chat_id.
        // Show welcome for a clean start.
        if (welcomeSection) welcomeSection.style.display = 'flex';
        updateSidebarList();
    }

    // Note: Server history hydration disabled to avoid conflicting with client-side multi-chat sessions.
    // The /api/history can still be used manually if needed for single-session backend state.
    // loadAndRenderHistoryFromServer();

    // Ensure input is focusable and ready
    if (chatInput) {
        setTimeout(() => chatInput.focus(), 100);
    }

    if (window.lucide) lucide.createIcons();

    console.log('[UI] Chat interface initialized and ready for interaction. Try typing or clicking a chip.');
});
