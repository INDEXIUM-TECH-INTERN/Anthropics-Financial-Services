// ═══ Thinking Card — Shows AI processing steps inline in chat ═══

export function createThinkingCard(): HTMLElement {
  const el = document.createElement('div');
  el.className = 'thinking-card glass-panel';
  el.innerHTML = `
    <div class="thinking-header">
      <div class="thinking-spinner"></div>
      <span class="thinking-label">AI đang xử lý</span>
    </div>
    <div class="thinking-steps">
      <div class="thinking-step active" data-step="agent">
        <div class="thinking-step-icon agent">●</div>
        <div class="thinking-step-content">
          <div class="thinking-step-label">Agent</div>
          <div class="thinking-step-text">Đang nhận dạng...</div>
        </div>
      </div>
      <div class="thinking-step pending" data-step="skill">
        <div class="thinking-step-icon skill">○</div>
        <div class="thinking-step-content">
          <div class="thinking-step-label">Skill</div>
          <div class="thinking-step-text">Phân tích intent...</div>
        </div>
      </div>
      <div class="thinking-step pending" data-step="tool">
        <div class="thinking-step-icon tool">○</div>
        <div class="thinking-step-content">
          <div class="thinking-step-label">Công cụ</div>
          <div class="thinking-step-text">LLM Routing...</div>
        </div>
      </div>
      <div class="thinking-step pending" data-step="reason">
        <div class="thinking-step-icon reason">○</div>
        <div class="thinking-step-content">
          <div class="thinking-step-label">Lý do</div>
          <div class="thinking-step-text">Xác định agent tối ưu...</div>
        </div>
      </div>
    </div>`;
  return el;
}
