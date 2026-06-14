class ApiError extends Error {
    statusCode;
    constructor(message, statusCode) {
        super(message);
        this.statusCode = statusCode;
        this.name = 'ApiError';
    }
}
export class ApiClient {
    baseUrl;
    constructor(baseUrl) {
        this.baseUrl = baseUrl;
    }
    async getSessions() {
        const res = await fetch(`${this.baseUrl}/api/chats`);
        if (!res.ok)
            throw new ApiError('Failed to fetch sessions', res.status);
        const data = await res.json();
        return data.chats || [];
    }
    async createSession(title) {
        const res = await fetch(`${this.baseUrl}/api/chats`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title }),
        });
        if (!res.ok)
            throw new ApiError('Failed to create session', res.status);
        return res.json();
    }
    async deleteSession(chatId) {
        const res = await fetch(`${this.baseUrl}/api/chats?chat_id=${encodeURIComponent(chatId)}`, { method: 'DELETE' });
        if (!res.ok)
            throw new ApiError('Failed to delete session', res.status);
    }
    async getHistory(chatId) {
        const res = await fetch(`${this.baseUrl}/api/history?chat_id=${encodeURIComponent(chatId)}`);
        if (!res.ok)
            throw new ApiError('Failed to fetch history', res.status);
        return res.json();
    }
    async saveConfigKeys(keys) {
        const res = await fetch(`${this.baseUrl}/api/config/keys`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ keys }),
        });
        if (!res.ok)
            throw new ApiError('Failed to save config keys', res.status);
        return res.json();
    }
    async streamChat(message, chatId, attachments, signal, onToken, onDone, onError) {
        const body = { message };
        if (chatId)
            body.chat_id = chatId;
        if (attachments.length > 0)
            body.attachments = attachments;
        const response = await fetch(`${this.baseUrl}/api/chat/stream`, {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal,
        });
        if (!response.ok)
            throw new ApiError(`HTTP ${response.status}`, response.status);
        const reader = response.body?.getReader();
        if (!reader)
            throw new Error('No response body');
        const decoder = new TextDecoder();
        let buffer = '';
        while (true) {
            const { done, value } = await reader.read();
            if (done)
                break;
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() ?? '';
            for (const line of lines) {
                const trimmed = line.trim();
                if (!trimmed.startsWith('data: '))
                    continue;
                let data;
                try {
                    data = JSON.parse(trimmed.substring(6));
                }
                catch {
                    continue;
                }
                if (data.type === 'token' && data.text)
                    onToken(data.text);
                if (data.type === 'done' && data.metrics)
                    onDone(data.metrics);
                if (data.type === 'error' && data.error)
                    onError(data.error);
            }
        }
    }
}
