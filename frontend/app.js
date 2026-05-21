document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    const consoleLogs = document.getElementById('console-logs');
    
    const API_URL = 'http://localhost:8080/api/chat';
    const EVENTS_URL = 'http://localhost:8080/api/events';

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

    async function sendMessage() {
        const text = chatInput.value.trim();
        if (!text) return;

        addMessage(text, 'user');
        chatInput.value = '';
        chatInput.style.height = '24px';
        sendBtn.disabled = true;

        logToConsole(`New Request: "${text.substring(0, 30)}..."`, 'info');

        const botMsgDiv = addMessage('', 'bot', true);
        const footer = botMsgDiv.querySelector('.msg-footer');
        
        // Start Live Timer
        const startTime = Date.now();
        const timerInterval = setInterval(() => {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            footer.innerHTML = `<span class="timer">⏱ Processing... ${elapsed}s</span>`;
        }, 100);

        const TIMEOUT_MS = 120000; // Tăng lên 120s cho SSE ổn định
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
            }
        } catch (error) {
            clearInterval(timerInterval);
            clearTimeout(timeoutId);
            botMsgDiv.querySelector('.msg-content').textContent = error.name === 'AbortError' ? "❌ Lỗi: Request Timeout (120s)" : "❌ Lỗi: Không thể kết nối server.";
            footer.innerHTML = `<span class="timer" style="color:#ef4444">Error</span>`;
        } finally {
            sendBtn.disabled = false;
            chatInput.focus();
            chatHistory.scrollTop = chatHistory.scrollHeight;
        }
    }

    sendBtn.addEventListener('click', sendMessage);
    chatInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });
});
