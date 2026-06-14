// ═══ Icon Button — Reusable icon-only button ═══
export function createIconButton(options) {
    const btn = document.createElement('button');
    btn.className = `icon-btn ${options.className ?? ''}`.trim();
    btn.title = options.title;
    btn.innerHTML = options.icon;
    if (options.onClick)
        btn.addEventListener('click', options.onClick);
    return btn;
}
