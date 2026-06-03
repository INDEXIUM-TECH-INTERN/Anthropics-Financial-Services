document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    const consoleLogs = document.getElementById('console-logs');
    const runTestBtn = document.getElementById('run-test-btn');
    const themeToggleBtn = document.getElementById('theme-toggle');
    
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
            logToConsole(`SSE Connection to ${currentBackend} lost. Reconnecting...`, "error");
        };
        
        logToConsole(`Connected to ${currentBackend} backend (SSE)`, "success");
    }

    const backendSelect = document.getElementById('backend-select');
    if (backendSelect) {
        backendSelect.addEventListener('change', (e) => {
            currentBackend = e.target.value;
            currentBaseUrl = backends[currentBackend];
            logToConsole(`Switching to ${currentBackend} backend...`, "process");
            setupEventSource();
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
        const logDiv = document.createElement('div');
        logDiv.className = `log-entry ${type}`;
        
        // Define prefix based on type
        let prefix = '> ';
        if (type === 'routing') prefix = '🧭 [ROUTING] ';
        if (type === 'process') prefix = '⚙️ [PROCESS] ';
        if (type === 'tool') prefix = '🛠️ [TOOL] ';
        if (type === 'success') prefix = '✅ [SUCCESS] ';
        if (type === 'error') prefix = '❌ [ERROR] ';

        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', { hour12: false, minute: '2-digit', second: '2-digit' });
        
        logDiv.innerHTML = `<span style="opacity: 0.5; margin-right: 8px;">${timeString}</span> <span class="log-payload">${prefix}${message}</span>`;
        
        consoleLogs.appendChild(logDiv);
        
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
            currentAgentEl.textContent = "Đang phân tích...";
            currentSkillEl.textContent = "Đang phân tích...";
            currentToolEl.textContent = "Đang kết nối LLM...";
            currentReasonEl.textContent = "Đang phân tích bối cảnh câu hỏi...";
            return;
        }

        // 2. Agent Selection and Reason
        const agentMatch = message.match(/Đã chọn Agent:\s*([a-zA-Z0-9\-_]+)(?:\s*\((?:Lý do|Reason):\s*(.+)\))?/i);
        if (agentMatch) {
            currentAgentEl.textContent = agentMatch[1];
            if (agentMatch[2]) {
                currentReasonEl.textContent = agentMatch[2];
            }
            return;
        }

        // 3. Skill Loading
        const skillMatch = message.match(/Đang nạp skill chuyên biệt:\s*(.+)/i);
        if (skillMatch) {
            currentSkillEl.textContent = skillMatch[1];
            return;
        }

        // 4. Tool/Scraping/Calculating
        if (type === 'tool') {
            if (message.includes("Thực thi Tool:")) {
                const toolName = message.replace("Thực thi Tool:", "").replace("...", "").trim();
                currentToolEl.textContent = `Chạy Tool: ${toolName}`;
            } else if (message.includes("Tra cứu dữ liệu:")) {
                const query = message.replace("Tra cứu dữ liệu:", "").trim();
                currentToolEl.textContent = `Tìm kiếm: "${query}"`;
            } else if (message.includes("Đang đọc nội dung từ:")) {
                const url = message.replace("Đang đọc nội dung từ:", "").trim();
                try {
                    const domain = new URL(url).hostname;
                    currentToolEl.textContent = `Scraping: ${domain}`;
                } catch(e) {
                    currentToolEl.textContent = `Scraping tài liệu`;
                }
            } else if (message.includes("Thực hiện tính toán:")) {
                const expr = message.replace("Thực hiện tính toán:", "").trim();
                currentToolEl.textContent = `Tính toán: ${expr}`;
            } else {
                currentToolEl.textContent = message;
            }
            return;
        }

        // 5. Transfer / Handoff
        const handoffMatch = message.match(/Yêu cầu chuyển giao sang:\s*(.+)/i);
        if (handoffMatch) {
            currentAgentEl.textContent = `${handoffMatch[1]} (Đang chuyển giao)`;
            currentToolEl.textContent = `Chuyển giao Agent...`;
            return;
        }

        // 6. Nạp tài liệu
        const docMatch = message.match(/Nạp tài liệu:\s*(.+)/i);
        if (docMatch) {
            currentToolEl.textContent = `Đọc cấu hình: ${docMatch[1]}`;
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
                card.className = 'source-card'; // Using existing sidebar style
                card.innerHTML = `
                    <div style="display: flex; align-items: center; gap: 8px;">
                        <i data-lucide="file-text" style="width: 12px; height: 12px; color: #60a5fa;"></i>
                        <span style="color: #94a3b8; font-size: 11px; font-weight: 700;">SOURCE [${index+1}]</span>
                    </div>
                    <a href="${url}" target="_blank">${domain}</a>
                `;
                sourcesList.appendChild(card);
            });

            // 2. Add Inline Sources at the END of message
            const sourcesContainer = document.createElement('div');
            sourcesContainer.className = 'message-sources';
            sourcesContainer.innerHTML = '<div class="sources-label">Sources</div>';
            
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
            // Render sources directly inside the bot message bubble or right below it
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

        addMessage(text, 'user');
        chatInput.value = '';
        chatInput.style.height = '24px';
        sendBtn.disabled = true;
        if (runTestBtn) runTestBtn.disabled = true;

        logToConsole(`New Request: "${text.substring(0, 30)}..."`, 'info');

        // Reset Live Status Board on new submit
        const currentAgentEl = document.getElementById('current-agent');
        const currentSkillEl = document.getElementById('current-skill');
        const currentToolEl = document.getElementById('current-tool');
        const currentReasonEl = document.getElementById('current-reason');
        if (currentAgentEl && currentSkillEl && currentToolEl && currentReasonEl) {
            currentAgentEl.textContent = "Đang nhận dạng...";
            currentSkillEl.textContent = "Đang chờ...";
            currentToolEl.textContent = "Bắt đầu tiến trình...";
            currentReasonEl.textContent = "Đang phân tích yêu cầu...";
        }

        const botMsgDiv = addMessage('', 'bot', true);
        const footer = botMsgDiv.querySelector('.msg-footer');
        
        // Start Live Timer
        const startTime = Date.now();
        const timerInterval = setInterval(() => {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            footer.innerHTML = `<span class="timer">⏱ Processing... ${elapsed}s</span>`;
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
                footer.innerHTML = `<span class="timer" style="color:#ef4444">Failed after ${finalDuration}s</span>`;
                return false;
            } else {
                contentDiv.innerHTML = marked.parse(data.reply);
                renderSources(data.reply, contentDiv);
                
                footer.innerHTML = `
                    <span class="timer">✅ ${finalDuration}s</span>
                    <div class="metrics-inline">
                        <span>🧠 ${data.metrics.ram_mb}</span>
                        <span>⚡ ${data.metrics.cpu_load.split(' ')[0]} Threads</span>
                        <span>🪙 ${data.metrics.token_in}/${data.metrics.token_out}</span>
                    </div>
                `;
                return true;
            }
        } catch (error) {
            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            botMsgDiv.querySelector('.msg-content').textContent = error.name === 'AbortError' ? "❌ Lỗi: Request Timeout (120s)" : "❌ Lỗi: Không thể kết nối server.";
            footer.innerHTML = `<span class="timer" style="color:#ef4444">Error</span>`;
            return false;
        } finally {
            sendBtn.disabled = false;
            chatInput.focus();
            chatHistory.scrollTop = chatHistory.scrollHeight;
        }
    }

    async function runAutoTest() {
        if (runTestBtn) runTestBtn.disabled = true;
        logToConsole(`--- STARTING AUTO-TEST SUITE ---`, 'process');
        
        for (let i = 0; i < testQueries.length; i++) {
            logToConsole(`[Test ${i+1}/${testQueries.length}] Running...`, 'process');
            await sendMessage(testQueries[i]);
            
            logToConsole(`[Test ${i+1}/${testQueries.length}] Finished. Resting 3s...`, 'info');
            await new Promise(r => setTimeout(r, 3000));
        }
        
        logToConsole(`--- AUTO-TEST SUITE COMPLETED ---`, 'success');
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
