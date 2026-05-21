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
        
        return logDiv; // Trả về phần tử để có thể update text
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
        
        // Tạo dòng log riêng cho Timer
        const timerLog = logToConsole(`> Waiting for agent response... [00:00]`, 'process');
        
        const loadingMsg = addMessage('', 'bot', true);

        // Khởi tạo Live Timer
        const startTime = Date.now();
        const timerInterval = setInterval(() => {
            const elapsed = Math.floor((Date.now() - startTime) / 1000);
            const minutes = String(Math.floor(elapsed / 60)).padStart(2, '0');
            const seconds = String(elapsed % 60).padStart(2, '0');
            
            const now = new Date();
            const timeString = now.toLocaleTimeString('en-US', { hour12: false });
            timerLog.textContent = `[${timeString}] > Waiting for agent response... [${minutes}:${seconds}]`;
        }, 1000);

        // Khởi tạo AbortController cho Timeout (90 giây)
        const TIMEOUT_MS = 90000;
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
            const duration = Date.now() - startTime;

            clearTimeout(timeoutId);
            clearInterval(timerInterval);
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
            clearTimeout(timeoutId);
            clearInterval(timerInterval);
            loadingMsg.remove();
            
            if (error.name === 'AbortError') {
                logToConsole(`> Timeout Error: Request exceeded ${TIMEOUT_MS / 1000}s limit.`, 'error');
                addMessage(`❌ Request Timeout: Agent mất quá nhiều thời gian để phản hồi.`, 'bot');
            } else {
                logToConsole(`> Network error: Could not reach the Go Backend.`, 'error');
                addMessage(`❌ Không thể kết nối tới server. Vui lòng đảm bảo Backend đang chạy.`, 'bot');
            }
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
