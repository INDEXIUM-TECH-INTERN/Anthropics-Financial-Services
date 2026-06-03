document.addEventListener('DOMContentLoaded', () => {
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
    
    const backends = {
        gemini: "http://localhost:8080",
        claude: "http://localhost:8081"
    };

    let currentBackend = 'gemini';
    let currentBaseUrl = backends[currentBackend];
    let eventSource = null;

    function setupEventSource() {
        if (eventSource) {
            eventSource.close();
        }
        
        const eventsUrl = `${currentBaseUrl}/api/events`;
        eventSource = new EventSource(eventsUrl);
        
        eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                logToConsole(data.payload, data.type);
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
            const confirmReset = confirm("Bạn có chắc chắn muốn xóa lịch sử trò chuyện và bắt đầu cuộc hội thoại mới?");
            if (!confirmReset) return;

            logToConsole('Đang yêu cầu reset cuộc hội thoại...', 'process');
            
            try {
                const resetUrl = `${currentBaseUrl}/api/reset`;
                const response = await fetch(resetUrl);
                const data = await response.json();
                
                if (response.ok && data.status === 'reset') {
                    // Clear UI states
                    chatHistory.innerHTML = '';
                    chatHistory.appendChild(welcomeSection);
                    welcomeSection.style.display = 'flex';
                    
                    sourcesList.innerHTML = '<div class="empty-state">Chưa có tài liệu nào trong ngữ cảnh hiện tại.</div>';
                    consoleLogs.innerHTML = '<div class="pipeline-empty">Sẵn sàng phân tích yêu cầu khi có câu hỏi.</div>';
                    
                    // Reset Live Status Board
                    document.getElementById('current-agent').textContent = "Chưa hoạt động";
                    document.getElementById('current-skill').textContent = "Chưa nạp";
                    document.getElementById('current-tool').textContent = "Đang chờ câu hỏi...";
                    document.getElementById('current-reason').textContent = "Chưa có dữ liệu phân tích";
                    
                    logToConsole('Reset cuộc hội thoại thành công. Hệ thống đã sẵn sàng.', 'success');
                    alert('Khởi tạo cuộc hội thoại mới thành công!');
                } else {
                    logToConsole('Reset cuộc hội thoại thất bại.', 'error');
                }
            } catch (err) {
                logToConsole('Không thể kết nối đến server để reset cuộc hội thoại: ' + err.message, 'error');
            }
        });
    }

    // Initial setup
    setupEventSource();

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
        const urlRegex = /(https?:\/\/[^\s\)]+)/g;
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
                    <a href="${url}" target="_blank">${domain}</a>
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
                const card = document.createElement('div');
                card.className = 'inline-source-card';
                card.innerHTML = `
                    <div class="source-index">${index+1}</div>
                    <a href="${url}" target="_blank" class="source-link">${domain}</a>
                `;
                grid.appendChild(card);
            });
            
            sourcesContainer.appendChild(grid);
            targetElement.appendChild(sourcesContainer);
            
            if (window.lucide) lucide.createIcons();
        }
    }

    function addMessage(text, sender, isLoading = false) {
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

    async function sendMessage(textOverride = null) {
        const text = textOverride !== null ? textOverride : chatInput.value.trim();
        if (!text) return false;

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
            const response = await fetch(apiUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: text }),
                signal: controller.signal
            });
            const data = await response.json();
            const finalDuration = ((Date.now() - startTime) / 1000).toFixed(1);

            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            
            const contentDiv = botMsgDiv.querySelector('.msg-content');
            botMsgDiv.classList.remove('loading');

            if (data.error) {
                contentDiv.textContent = `❌ Lỗi: ${data.error}`;
                footer.innerHTML = `<span class="timer" style="color:var(--danger)">Thất bại sau ${finalDuration}s</span>`;
                return false;
            } else {
                contentDiv.innerHTML = marked.parse(data.reply);
                renderSources(data.reply, contentDiv);
                
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
                return true;
            }
        } catch (error) {
            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            botMsgDiv.querySelector('.msg-content').textContent = error.name === 'AbortError' ? "❌ Lỗi: Yêu cầu quá thời gian phản hồi (300s)" : "❌ Lỗi: Không thể kết nối server.";
            footer.innerHTML = `<span class="timer" style="color:var(--danger)">Lỗi kết nối</span>`;
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
});
