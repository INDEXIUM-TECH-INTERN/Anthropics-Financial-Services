document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    const consoleLogs = document.getElementById('console-logs');
    const runTestBtn = document.getElementById('run-test-btn');
    
    const API_URL = 'http://localhost:8080/api/chat';
    const EVENTS_URL = 'http://localhost:8080/api/events';

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

    // --- Real-time SSE Logic ---
    const eventSource = new EventSource(EVENTS_URL);
    eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            logToConsole(data.payload, data.type);
        } catch (e) {}
    };
    eventSource.onerror = () => {
        logToConsole("SSE Connection lost. Reconnecting...", "error");
    };

    // Auto-resize textarea
    chatInput.addEventListener('input', function() {
        this.style.height = '24px';
        this.style.height = (this.scrollHeight) + 'px';
    });

    function logToConsole(message, type = 'info') {
        const logDiv = document.createElement('div');
        logDiv.className = `log-entry ${type}`;
        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', { hour12: false });
        logDiv.textContent = `[${timeString}] ${message}`;
        consoleLogs.appendChild(logDiv);
        consoleLogs.scrollTop = consoleLogs.scrollHeight;
    }

    function extractSources(markdownText) {
        const urlRegex = /(https?:\/\/[^\s\)]+)/g;
        const urls = markdownText.match(urlRegex) || [];
        const uniqueUrls = [...new Set(urls)];
        
        if (uniqueUrls.length > 0) {
            sourcesList.innerHTML = '';
            uniqueUrls.forEach((url, index) => {
                let domain = url;
                try { domain = new URL(url).hostname; } catch(e) {}
                const card = document.createElement('div');
                card.className = 'source-card';
                card.innerHTML = `
                    <span style="color: #6b7280; font-size: 11px;">Source [${index+1}]</span>
                    <a href="${url}" target="_blank">${domain}</a>
                `;
                sourcesList.appendChild(card);
            });
        }
    }

    function addMessage(text, sender, isLoading = false) {
        const msgDiv = document.createElement('div');
        msgDiv.className = `message ${sender}`;
        
        const contentDiv = document.createElement('div');
        contentDiv.className = 'msg-content';
        
        if (isLoading) {
            contentDiv.innerHTML = '<div class="loading-indicator"></div><div class="loading-indicator" style="width: 70%; margin-top: 8px;"></div>';
        } else if (sender === 'bot') {
            contentDiv.innerHTML = marked.parse(text);
            extractSources(text);
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

        const botMsgDiv = addMessage('', 'bot', true);
        const footer = botMsgDiv.querySelector('.msg-footer');
        
        // Start Live Timer
        const startTime = Date.now();
        const timerInterval = setInterval(() => {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            footer.innerHTML = `<span class="timer">⏱ Processing... ${elapsed}s</span>`;
        }, 100);

        const TIMEOUT_MS = 120000;
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), TIMEOUT_MS);

        try {
            const response = await fetch(API_URL, {
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
                extractSources(data.reply);
                
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
