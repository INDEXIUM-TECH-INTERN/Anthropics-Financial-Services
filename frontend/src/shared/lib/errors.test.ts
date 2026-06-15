import { describe, it, expect } from 'vitest';
import { diagnoseError, renderErrorHTML } from './errors';

describe('diagnoseError', () => {
  it('should diagnose abort errors', () => {
    const result = diagnoseError('AbortError', 'The operation was aborted');
    expect(result.title).toBe('Đã hủy');
    expect(result.badge).toBe('🛑');
  });

  it('should diagnose network errors', () => {
    const result = diagnoseError('TypeError', 'Failed to fetch');
    expect(result.title).toBe('Không kết nối');
    expect(result.badge).toBe('🔌');
  });

  it('should diagnose quota errors', () => {
    const result = diagnoseError('Error', '429 Too Many Requests');
    expect(result.title).toBe('Hết hạn mức (429)');
    expect(result.badge).toBe('📊');
  });

  it('should diagnose auth errors', () => {
    const result = diagnoseError('Error', '401 Unauthorized');
    expect(result.title).toBe('Xác thực thất bại');
    expect(result.badge).toBe('🔑');
  });

  it('should diagnose thought signature errors', () => {
    const result = diagnoseError('Error', 'thought_signature mismatch');
    expect(result.title).toBe('Lỗi tương thích');
    expect(result.badge).toBe('🤖');
  });

  it('should return fallback for unknown errors', () => {
    const result = diagnoseError('Error', 'something weird happened');
    expect(result.title).toBe('Lỗi xử lý');
    expect(result.name).toBe('Error');
    expect(result.msg).toBe('something weird happened');
  });

  it('should include suggestions for each error type', () => {
    const network = diagnoseError('TypeError', 'network error');
    expect(network.suggestions.length).toBeGreaterThan(0);
    expect(network.suggestions[0]).toContain('kết nối');
  });
});

describe('renderErrorHTML', () => {
  it('should render error HTML with all sections', () => {
    const diag = diagnoseError('TypeError', 'Failed to fetch');
    const html = renderErrorHTML(diag);
    expect(html).toContain('error-container');
    expect(html).toContain('error-header');
    expect(html).toContain('error-body');
    expect(html).toContain('error-suggestions');
    expect(html).toContain('error-details');
  });

  it('should escape HTML in error message', () => {
    const diag = diagnoseError('Error', '<script>alert("xss")</script>');
    const html = renderErrorHTML(diag);
    expect(html).not.toContain('<script>');
  });
});
