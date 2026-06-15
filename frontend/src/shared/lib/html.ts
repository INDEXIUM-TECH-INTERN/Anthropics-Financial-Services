// ═══════════════════════════════════════════════════
// Shared — HTML Utilities
// ═══════════════════════════════════════════════════

export function h(tag: string, attrs?: Record<string, string>, children?: (HTMLElement | string)[]): HTMLElement {
  const el = document.createElement(tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      el.setAttribute(k, v);
    }
  }
  if (children) {
    for (const child of children) {
      if (typeof child === 'string') {
        el.appendChild(document.createTextNode(child));
      } else {
        el.appendChild(child);
      }
    }
  }
  return el;
}
