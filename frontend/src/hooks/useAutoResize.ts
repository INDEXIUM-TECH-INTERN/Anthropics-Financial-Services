/**
 * Attach auto-resize behavior to a textarea element.
 * Caps height at maxRows × lineHeight.
 * Returns a cleanup function.
 */
export function useAutoResize(
  textarea: HTMLTextAreaElement | null,
  maxRows = 8,
): () => void {
  if (!textarea) return () => undefined;

  const computed = getComputedStyle(textarea);
  const lineHeight = Number.parseFloat(computed.lineHeight) || 24;
  const maxHeight = lineHeight * maxRows;

  const resize = () => {
    textarea.style.height = '0px';
    textarea.style.height = `${Math.min(textarea.scrollHeight, maxHeight)}px`;
  };

  textarea.addEventListener('input', resize);
  resize();

  return () => textarea.removeEventListener('input', resize);
}
