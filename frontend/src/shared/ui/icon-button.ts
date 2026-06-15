// ═══ Icon Button — Reusable icon-only button ═══

export interface IconButtonOptions {
  icon: string;
  title: string;
  className?: string;
  onClick?: (e: MouseEvent) => void;
}

export function createIconButton(options: IconButtonOptions): HTMLButtonElement {
  const btn = document.createElement('button');
  btn.className = `icon-btn ${options.className ?? ''}`.trim();
  btn.title = options.title;
  btn.innerHTML = options.icon;
  if (options.onClick) btn.addEventListener('click', options.onClick);
  return btn;
}
