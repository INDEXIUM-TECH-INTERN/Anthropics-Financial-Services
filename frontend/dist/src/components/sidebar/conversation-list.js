// ═══ Conversation List — Renders sidebar conversation history ═══
import { escHtml, formatTime } from '../../utils/dom';
function groupByDate(chats) {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    const weekAgo = new Date(today);
    weekAgo.setDate(weekAgo.getDate() - 7);
    const monthAgo = new Date(today);
    monthAgo.setDate(monthAgo.getDate() - 30);
    const groups = new Map();
    const labels = ['Hôm nay', 'Hôm qua', '7 ngày trước', '30 ngày trước', 'Cũ hơn'];
    for (const l of labels)
        groups.set(l, []);
    for (const chat of chats) {
        const d = chat.updated_at ? new Date(chat.updated_at) : new Date(0);
        if (d >= today)
            (groups.get('Hôm nay') ?? []).push(chat);
        else if (d >= yesterday)
            (groups.get('Hôm qua') ?? []).push(chat);
        else if (d >= weekAgo)
            (groups.get('7 ngày trước') ?? []).push(chat);
        else if (d >= monthAgo)
            (groups.get('30 ngày trước') ?? []).push(chat);
        else
            (groups.get('Cũ hơn') ?? []).push(chat);
    }
    return groups;
}
export function renderConversationList(container, chats, currentChatId, onSwitchChat, onDeleteChat, searchQuery) {
    container.innerHTML = '';
    if (!chats.length) {
        container.innerHTML = '<div style="padding:16px;font-size:13px;opacity:0.5;text-align:center;">Chưa có cuộc trò chuyện.</div>';
        return;
    }
    const query = searchQuery.toLowerCase();
    const filtered = query
        ? chats.filter(c => (c.title || '').toLowerCase().includes(query))
        : chats;
    const groups = groupByDate(filtered);
    for (const [label, groupChats] of groups) {
        if (!groupChats.length)
            continue;
        // Sticky date header
        const header = document.createElement('div');
        header.className = 'sidebar-section-label';
        header.textContent = label;
        container.appendChild(header);
        for (const chat of groupChats) {
            const isActive = chat.id === currentChatId;
            const item = document.createElement('div');
            item.className = `conv-item ${isActive ? 'active' : ''}`;
            item.innerHTML = `
        <span class="conv-item-title">${escHtml(chat.title || 'Cuộc trò chuyện mới')}</span>
        <span class="conv-item-meta">${formatTime(chat.updated_at)}</span>
        <button class="conv-item-delete" title="Xóa">✕</button>
      `;
            // Switch chat on title click
            item.querySelector('.conv-item-title')?.addEventListener('click', () => onSwitchChat(chat.id));
            // Delete on ✕ click
            item.querySelector('.conv-item-delete')?.addEventListener('click', async (e) => {
                e.stopPropagation();
                if (confirm('Xóa?'))
                    onDeleteChat(chat.id);
            });
            // Double-click to rename
            const titleEl = item.querySelector('.conv-item-title');
            titleEl.addEventListener('dblclick', (e) => {
                e.stopPropagation();
                const input = document.createElement('input');
                input.type = 'text';
                input.className = 'conv-rename-input';
                input.value = chat.title || '';
                titleEl.style.display = 'none';
                titleEl.parentNode?.insertBefore(input, titleEl);
                input.focus();
                input.select();
                const finish = (save) => {
                    const t = input.value.trim();
                    input.remove();
                    titleEl.style.display = '';
                    if (save && t && t !== chat.title) {
                        chat.title = t;
                        titleEl.textContent = escHtml(t);
                    }
                };
                input.addEventListener('blur', () => finish(true));
                input.addEventListener('keydown', (ke) => {
                    if (ke.key === 'Enter') {
                        ke.preventDefault();
                        input.blur();
                    }
                    else if (ke.key === 'Escape') {
                        ke.preventDefault();
                        finish(false);
                    }
                });
            });
            container.appendChild(item);
        }
    }
}
