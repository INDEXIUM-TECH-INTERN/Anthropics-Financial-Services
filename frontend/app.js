document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    const sourcesList = document.getElementById('sources-list');
    
    const metricLatency = document.getElementById('metric-latency');
    const metricRam = document.getElementById('metric-ram');
    const metricCpu = document.getElementById('metric-cpu');
    const metricTokens = document.getElementById('metric-tokens');

    const API_URL = 'http://localhost:8080/api/chat';

    // Auto-resize textarea
    chatInput.addEventListener('input', function() {
        this.style.height = '24px';
        this.style.height = (this.scrollHeight) + 'px';
    });

    function extractSources(markdownText) {
        // Tìm các mẫu dạng [1] Tên - URL hoặc [URL: https://...]
        const urlRegex = /(https?:\/\/[^\s\)]+)/g;
        const urls = markdownText.match(urlRegex) || [];
        
        // Loại bỏ trùng lặp
        const uniqueUrls = [...new Set(urls)];
        
        if (uniqueUrls.length > 0) {
            sourcesList.innerHTML = ''; // Xóa chữ 'Chưa có nguồn'
            uniqueUrls.forEach((url, index) => {
                // Rút gọn domain để hiển thị đẹp
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
            extractSources(text); // Bóc tách nguồn cập nhật Sidebar
        } else {
            contentDiv.textContent = text;
        }
        
        msgDiv.appendChild(contentDiv);
        chatHistory.appendChild(msgDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
        return msgDiv;
    }

    function updateMetrics(metrics) {
        if (!metrics) return;
        metricLatency.textContent = `${metrics.latency_ms} ms`;
        metricRam.textContent = metrics.ram_mb;
        metricCpu.textContent = metrics.cpu_load.replace(' Goroutines (Active)', '');
        metricTokens.textContent = `${metrics.token_in} / ${metrics.token_out}`;
    }

    async function sendMessage() {
        const text = chatInput.value.trim();
        if (!text) return;

        addMessage(text, 'user');
        chatInput.value = '';
        chatInput.style.height = '24px';
        sendBtn.disabled = true;

        const loadingMsg = addMessage('', 'bot', true);

        try {
            const response = await fetch(API_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: text })
            });
            const data = await response.json();

            loadingMsg.remove();
            
            if (data.error) {
                addMessage(`❌ Lỗi: ${data.error}`, 'bot');
            } else {
                addMessage(data.reply, 'bot');
                updateMetrics(data.metrics);
            }
        } catch (error) {
            loadingMsg.remove();
            addMessage(`❌ Không thể kết nối tới server.`, 'bot');
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
