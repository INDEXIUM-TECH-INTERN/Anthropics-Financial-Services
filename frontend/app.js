document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    const consoleLogs = document.getElementById('console-logs');
    
    const API_URL = 'http://localhost:8080/api/chat';

    // Auto-resize textarea
    chatInput.addEventListener('input', function() {
        this.style.height = '24px';
        this.style.height = (this.scrollHeight) + 'px';
    });

    function logToConsole(message, type = 'info') {
        const logDiv = document.createElement('div');
        logDiv.className = `log-entry ${type}`;
        
        // Lấy thời gian hiện tại HH:MM:SS
        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', { hour12: false });
        
        logDiv.textContent = `[${timeString}] ${message}`;
        consoleLogs.appendChild(logDiv);
        consoleLogs.scrollTop = consoleLogs.scrollHeight;
    }

    function extractSources(markdownText) {
        // Tìm các mẫu dạng [1] Tên - URL hoặc [URL: https://...]
        const urlRegex = /(https?:\/\/[^\s\)]+)/g;
        const urls = markdownText.match(urlRegex) || [];
        
        // Loại bỏ trùng lặp
        const uniqueUrls = [...new Set(urls)];
        
        if (uniqueUrls.length > 0) {
            sourcesList.innerHTML = ''; // Xóa chữ 'Chưa có nguồn'
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
        chatHistory.appendChild(msgDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
        return msgDiv;
    }

    function appendMetricsToMessage(msgDiv, metrics) {
        if (!metrics) return;
        
        const metricsDiv = document.createElement('div');
        metricsDiv.className = `msg-metrics`;
        
        metricsDiv.innerHTML = `
            <span>⏱ ${metrics.latency_ms}ms</span>
            <span>🧠 ${metrics.ram_mb}</span>
            <span>⚡ ${metrics.cpu_load.replace(' Goroutines (Active)', '')} Threads</span>
            <span>🪙 ${metrics.token_in}/${metrics.token_out} Tokens</span>
        `;
        
        msgDiv.appendChild(metricsDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
    }

    async function sendMessage() {
        const text = chatInput.value.trim();
        if (!text) return;

        addMessage(text, 'user');
        chatInput.value = '';
        chatInput.style.height = '24px';
        sendBtn.disabled = true;

        logToConsole(`> User input received: "${text.substring(0, 30)}..."`, 'info');
        logToConsole(`> Routing request to optimal financial agent...`, 'process');

        const loadingMsg = addMessage('', 'bot', true);

        // Giả lập tiến trình console
        let simulationInterval = null;
        let step = 0;
        const simulationSteps = [
            "> Fetching agent configurations from GitHub repository...",
            "> Bootstrapping specialized financial skills context...",
            "> Orchestrator initiating analysis (this may take up to 20s)...",
            "> Querying MCP tools (financial_research / market_data)...",
            "> Validating and parsing real-time tool outputs..."
        ];
        
        simulationInterval = setInterval(() => {
            if (step < simulationSteps.length) {
                logToConsole(simulationSteps[step], 'process');
                step++;
            } else {
                clearInterval(simulationInterval);
            }
        }, 3000);

        try {
            const startTime = Date.now();
            const response = await fetch(API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: text })
            });
            const data = await response.json();
            const duration = Date.now() - startTime;

            clearInterval(simulationInterval);
            loadingMsg.remove();
            
            if (data.error) {
                logToConsole(`> Backend execution error: ${data.error}`, 'error');
                addMessage(`❌ Lỗi: ${data.error}`, 'bot');
            } else {
                logToConsole(`> Response generation complete in ${duration}ms.`, 'success');
                const finalMsgDiv = addMessage(data.reply, 'bot');
                appendMetricsToMessage(finalMsgDiv, data.metrics);
                
                const sourcesCount = sourcesList.querySelectorAll('.source-card').length;
                if (sourcesCount > 0) {
                    logToConsole(`> Extracted and validated ${sourcesCount} dynamic sources.`, 'success');
                }
            }
        } catch (error) {
            clearInterval(simulationInterval);
            loadingMsg.remove();
            logToConsole(`> Network error: Could not reach the Go Backend.`, 'error');
            addMessage(`❌ Không thể kết nối tới server. Vui lòng đảm bảo Backend đang chạy.`, 'bot');
        } finally {
            sendBtn.disabled = false;
            chatInput.focus();
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
