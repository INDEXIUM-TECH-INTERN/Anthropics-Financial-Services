document.addEventListener('DOMContentLoaded', () => {
    const chatInput = document.getElementById('chat-input');
    const sendBtn = document.getElementById('send-btn');
    const chatHistory = document.getElementById('chat-history');
    
    const metricLatency = document.getElementById('metric-latency');
    const metricRam = document.getElementById('metric-ram');
    const metricCpu = document.getElementById('metric-cpu');
    const metricTokens = document.getElementById('metric-tokens');

    const API_URL = 'http://localhost:8080/api/chat';

    function addMessage(text, sender, isLoading = false) {
        const msgDiv = document.createElement('div');
        msgDiv.className = `message ${sender}`;
        if (isLoading) msgDiv.classList.add('loading');
        
        if (sender === 'bot' && !isLoading) {
            msgDiv.innerHTML = marked.parse(text); // Dùng marked để parse markdown sang html
        } else {
            msgDiv.textContent = text;
        }
        
        chatHistory.appendChild(msgDiv);
        chatHistory.scrollTop = chatHistory.scrollHeight;
        return msgDiv;
    }

    function updateMetrics(metrics) {
        if (!metrics) return;
        metricLatency.textContent = `${metrics.latency_ms} ms`;
        metricRam.textContent = metrics.ram_mb;
        metricCpu.textContent = metrics.cpu_load;
        metricTokens.textContent = `${metrics.token_in} / ${metrics.token_out}`;
    }

    async function sendMessage() {
        const text = chatInput.value.trim();
        if (!text) return;

        addMessage(text, 'user');
        chatInput.value = '';
        sendBtn.disabled = true;

        const loadingMsg = addMessage('Agent is researching and analyzing...', 'loading', true);

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
