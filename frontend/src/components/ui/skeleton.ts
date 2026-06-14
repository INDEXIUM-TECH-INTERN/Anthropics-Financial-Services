// ═══ Skeleton Loading Components ═══

export function createMessageSkeleton(): HTMLElement {
  const el = document.createElement('div');
  el.className = 'skeleton-loading';
  el.innerHTML = `
    <div class="skeleton-message">
      <div class="skeleton-avatar"></div>
      <div class="skeleton-body">
        <div class="skeleton-line short"></div>
        <div class="skeleton-line"></div>
      </div>
    </div>`;
  return el;
}
