// ═══ Sidebar Widget ═══
// Composes conversation list, search, and sidebar toggle.

import { formatTime } from '../../shared/lib/dom';
import { createIconButton } from '../../shared/ui/icon-button';
import type { ChatSession } from '../../shared/api/types';

export interface SidebarWidget {
  render: (sessions: ChatSession[], activeId: string | null, query?: string) => void;
  destroy: () => void;
}

export function createSidebarWidget(
  container: HTMLElement,
  callbacks: {
    onSelect: (id: string) => void;
    onDelete: (id: string) => void;
  },
): SidebarWidget {
  function render(sessions: ChatSession[], activeId: string | null, query = '') {
    container.innerHTML = '';

    const filtered = query ? sessions.filter((s) => s.title.toLowerCase().includes(query.toLowerCase())) : sessions;

    if (!filtered.length) {
      const empty = document.createElement('div');
      empty.className = 'sidebar-empty';
      empty.textContent = query ? 'Không tìm thấy.' : 'Chưa có cuộc trò chuyện.';
      container.appendChild(empty);
      return;
    }

    filtered.forEach((session) => {
      const item = document.createElement('div');
      item.className = `conv-item ${session.id === activeId ? 'active' : ''}`;

      const titleEl = document.createElement('div');
      titleEl.className = 'conv-item-title';
      titleEl.textContent = session.title;

      const timeEl = document.createElement('div');
      timeEl.className = 'conv-item-meta';
      timeEl.textContent = formatTime(session.updated_at);

      const delBtn = createIconButton({
        icon: '✕',
        title: 'Xóa',
        className: 'conv-item-delete',
        onClick: (e) => {
          e.stopPropagation();
          if (confirm('Xóa cuộc trò chuyện này?')) callbacks.onDelete(session.id);
        },
      });

      item.appendChild(titleEl);
      item.appendChild(timeEl);
      item.appendChild(delBtn);
      item.addEventListener('click', () => callbacks.onSelect(session.id));
      container.appendChild(item);
    });
  }

  function destroy() {
    container.innerHTML = '';
  }

  return { render, destroy };
}
