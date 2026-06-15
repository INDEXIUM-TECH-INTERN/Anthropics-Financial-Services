import { atom, computed } from 'nanostores';
import type { PipelineState } from './types';

const INITIAL_PIPELINE: PipelineState = {
  agent: 'Chua hoat dong',
  agentStatus: 'idle',
  skill: 'Chua nap',
  skillStatus: 'idle',
  tool: 'Dang cho...',
  toolStatus: 'idle',
  reason: 'Chua co du lieu',
  reasonStatus: 'idle',
};

export const $currentChatId = atom<string | null>(null);
export const $currentChatTitle = atom<string>('Cuoc tro chuyen moi');
export const $isGenerating = atom<boolean>(false);
export const $pipeline = atom<PipelineState>({ ...INITIAL_PIPELINE });
export const $hasActiveChat = computed($currentChatId, (id) => id !== null);

export function setChatId(id: string | null) {
  $currentChatId.set(id);
}
export function setChatTitle(title: string) {
  $currentChatTitle.set(title);
}
export function setGenerating(g: boolean) {
  $isGenerating.set(g);
}
export function setPipeline(partial: Partial<PipelineState>) {
  $pipeline.set({ ...$pipeline.get(), ...partial });
}
export function resetPipeline() {
  $pipeline.set({ ...INITIAL_PIPELINE });
}
export function resetChat() {
  $currentChatId.set(null);
  $currentChatTitle.set('Cuoc tro chuyen moi');
  $isGenerating.set(false);
  $pipeline.set({ ...INITIAL_PIPELINE });
}
