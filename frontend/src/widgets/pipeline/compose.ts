// ═══ Pipeline Widget ═══
// Displays agent pipeline status (agent, skill, tool, reason).

import type { PipelineState } from '../../entities/chat/model/types';

export interface PipelineWidget {
  render: (state: PipelineState) => void;
  destroy: () => void;
}

export function createPipelineWidget(container: HTMLElement): PipelineWidget {
  function render(state: PipelineState) {
    container.innerHTML = '';

    const steps: { key: string; label: string; value: string; status: 'idle' | 'active' }[] = [
      { key: 'agent', label: 'Agent', value: state.agent, status: state.agentStatus },
      { key: 'skill', label: 'Skill', value: state.skill, status: state.skillStatus },
      { key: 'tool', label: 'Tool', value: state.tool, status: state.toolStatus },
      { key: 'reason', label: 'Reason', value: state.reason, status: state.reasonStatus },
    ];

    steps.forEach((step) => {
      const el = document.createElement('div');
      el.className = `pipeline-step pipeline-step-${step.status}`;
      el.innerHTML = `
        <span class="pipeline-step-label">${step.label}</span>
        <span class="pipeline-step-value">${step.value}</span>`;
      container.appendChild(el);
    });
  }

  function destroy() {
    container.innerHTML = '';
  }

  return { render, destroy };
}
