export interface PipelineState {
  agent: string;
  agentStatus: 'idle' | 'active';
  skill: string;
  skillStatus: 'idle' | 'active';
  tool: string;
  toolStatus: 'idle' | 'active';
  reason: string;
  reasonStatus: 'idle' | 'active';
}

export interface ChatEntityState {
  currentChatId: string | null;
  currentChatTitle: string;
  isGenerating: boolean;
  pipeline: PipelineState;
}
