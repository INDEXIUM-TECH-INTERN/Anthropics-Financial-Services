import { escHtml } from './dom';

export interface ErrorDiag {
  badge: string;
  title: string;
  icon: string;
  desc: string;
  suggestions: string[];
  name: string;
  msg: string;
}

const FALLBACK_ERROR: ErrorDiag = {
  badge: '⚙️',
  title: 'Lỗi xử lý',
  icon: '⚠️',
  desc: 'Lỗi không xác định.',
  suggestions: ['Kiểm tra API Key.', 'Thử lại.'],
  name: 'Error',
  msg: 'Unknown error',
};

export function diagnoseError(name: string, msg: string): ErrorDiag {
  const m = (msg || '').toLowerCase();

  if (m.includes('aborted') || m.includes('aborterror')) {
    return {
      badge: '🛑',
      title: 'Đã hủy',
      icon: '🚫',
      desc: 'Yêu cầu đã bị hủy bởi người dùng.',
      suggestions: ['Thử gửi lại tin nhắn.'],
      name,
      msg,
    };
  }

  if (m.includes('failed to fetch') || m.includes('network') || m.includes('fetch')) {
    return {
      badge: '🔌',
      title: 'Không kết nối',
      icon: '📡',
      desc: 'Không thể kết nối đến máy chủ. Vui lòng thử lại sau.',
      suggestions: ['Kiểm tra kết nối mạng.', 'Thử lại sau vài giây.'],
      name,
      msg,
    };
  }

  if (m.includes('quota') || m.includes('429')) {
    return {
      badge: '📊',
      title: 'Hết hạn mức (429)',
      icon: '⛔',
      desc: 'Vượt quá giới hạn API.',
      suggestions: ['Thêm API Key dự phòng.', 'Chờ 1-2 phút.'],
      name,
      msg,
    };
  }

  if (m.includes('401') || m.includes('403') || m.includes('unauthorized')) {
    return {
      badge: '🔑',
      title: 'Xác thực thất bại',
      icon: '🔒',
      desc: 'API Key không hợp lệ hoặc đã hết hạn.',
      suggestions: ['Kiểm tra lại API Key.', 'Thử với Key khác.'],
      name,
      msg,
    };
  }

  if (m.includes('thought_signature') || m.includes('thought signature')) {
    return {
      badge: '🤖',
      title: 'Lỗi tương thích',
      icon: '🔧',
      desc: 'Lỗi tương thích mô hình AI.',
      suggestions: ['Đổi GEMINI_MODEL.', 'Khởi động lại backend.'],
      name,
      msg,
    };
  }

  return { ...FALLBACK_ERROR, name, msg };
}

export function renderErrorHTML(d: ErrorDiag): string {
  return `<div class="error-container">
    <div class="error-header">
      <span class="error-title">${d.icon} ${d.title}</span>
      <span class="error-badge">${d.badge}</span>
    </div>
    <div class="error-body">${d.desc}</div>
    <div class="error-suggestions">
      <div class="error-suggestions-title">💡 Đề xuất</div>
      ${d.suggestions.map((s) => `<div class="error-suggestion-item">${s}</div>`).join('')}
    </div>
    <details class="error-details">
      <summary>Chi tiết kỹ thuật</summary>
      <div class="error-details-content">[${d.name}] ${escHtml(d.msg)}</div>
    </details>
  </div>`;
}
