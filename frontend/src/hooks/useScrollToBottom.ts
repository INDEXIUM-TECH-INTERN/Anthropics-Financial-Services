/**
 * Attach scroll-to-bottom FAB behavior.
 * Returns { scrollToBottom, cleanup }.
 */
export function useScrollToBottom(
  viewport: HTMLElement | null,
  fab: HTMLElement | null,
  threshold = 200,
): { scrollToBottom: () => void; cleanup: () => void } {
  if (!viewport || !fab) {
    return { scrollToBottom: () => undefined, cleanup: () => undefined };
  }

  const handleScroll = () => {
    const distance = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    fab.classList.toggle('hidden', distance <= threshold);
  };

  viewport.addEventListener('scroll', handleScroll, { passive: true });

  const scrollToBottom = () => {
    viewport.scrollTo({ top: viewport.scrollHeight, behavior: 'smooth' });
    fab.classList.add('hidden');
  };

  return {
    scrollToBottom,
    cleanup: () => viewport.removeEventListener('scroll', handleScroll),
  };
}
