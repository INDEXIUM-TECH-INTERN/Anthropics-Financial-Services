import type { ChatSession } from '../types/api';

interface PipelineState {
  agent: string;
  agentStatus: 'idle' | 'active';
  skill: string;
  skillStatus: 'idle' | 'active';
  tool: string;
  toolStatus: 'idle' | 'active';
  reason: string;
  reasonStatus: 'idle' | 'active';
}

interface AppState {
  currentChatId: string | null;
  currentChatTitle: string;
  chats: ChatSession[];
  isGenerating: boolean;
  currentBackend: 'gemini' | 'claude';
  theme: 'dark' | 'light';
  pipeline: PipelineState;
}

type Listener = () => void;

const INITIAL_PIPELINE: PipelineState = {
  agent: 'Chưa hoạt động', agentStatus: 'idle',
  skill: 'Chưa nạp', skillStatus: 'idle',
  tool: 'Đang chờ...', toolStatus: 'idle',
  reason: 'Chưa có dữ liệu', reasonStatus: 'idle',
};

class Store {
  private state: AppState;
  private listeners: Set<Listener> = new Set();

  constructor() {
    this.state = {
      currentChatId: null,
      currentChatTitle: 'Cuộc trò chuyện mới',
      chats: [],
      isGenerating: false,
      currentBackend: 'gemini',
      theme: (localStorage.getItem('theme') as AppState['theme']) ?? 'dark',
      pipeline: { ...INITIAL_PIPELINE },
    };
  }

  getState(): Readonly<AppState> { return this.state; }

  setState(partial: Partial<AppState>): void {
    this.state = { ...this.state, ...partial };
    this.notify();
  }

  setPipeline(partial: Partial<PipelineState>): void {
    this.state = { ...this.state, pipeline: { ...this.state.pipeline, ...partial } };
    this.notify();
  }

  resetPipeline(): void {
    this.state = { ...this.state, pipeline: { ...INITIAL_PIPELINE } };
    this.notify();
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  }

  private notify(): void { for (const l of this.listeners) l(); }
}

export const store = new Store();
