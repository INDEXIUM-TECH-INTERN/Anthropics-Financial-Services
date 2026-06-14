const INITIAL_PIPELINE = {
    agent: 'Chưa hoạt động', agentStatus: 'idle',
    skill: 'Chưa nạp', skillStatus: 'idle',
    tool: 'Đang chờ...', toolStatus: 'idle',
    reason: 'Chưa có dữ liệu', reasonStatus: 'idle',
};
class Store {
    state;
    listeners = new Set();
    constructor() {
        this.state = {
            currentChatId: null,
            currentChatTitle: 'Cuộc trò chuyện mới',
            chats: [],
            isGenerating: false,
            currentBackend: 'gemini',
            theme: localStorage.getItem('theme') ?? 'dark',
            pipeline: { ...INITIAL_PIPELINE },
        };
    }
    getState() { return this.state; }
    setState(partial) {
        this.state = { ...this.state, ...partial };
        this.notify();
    }
    setPipeline(partial) {
        this.state = { ...this.state, pipeline: { ...this.state.pipeline, ...partial } };
        this.notify();
    }
    resetPipeline() {
        this.state = { ...this.state, pipeline: { ...INITIAL_PIPELINE } };
        this.notify();
    }
    subscribe(listener) {
        this.listeners.add(listener);
        return () => { this.listeners.delete(listener); };
    }
    notify() { for (const l of this.listeners)
        l(); }
}
export const store = new Store();
