// ═══ Premium Code Block — ChatGPT-style with header bar + copy button ═══

import { escHtml } from '../../utils/dom';

export function createCodeBlock(code: string, language: string): HTMLElement {
  const container = document.createElement('div');
  container.className = 'code-block-container';

  const header = document.createElement('div');
  header.className = 'code-block-header';
  header.innerHTML = `<span class="code-block-lang">${escHtml(language || 'code')}</span>`;

  const copyBtn = document.createElement('button');
  copyBtn.className = 'code-block-copy-btn';
  copyBtn.textContent = '📋 Copy';
  copyBtn.addEventListener('click', () => {
    navigator.clipboard.writeText(code).then(() => {
      copyBtn.textContent = '✅ Copied!';
      setTimeout(() => { copyBtn.textContent = '📋 Copy'; }, 2000);
    });
  });
  header.appendChild(copyBtn);

  const pre = document.createElement('pre');
  pre.className = `language-${escHtml(language || 'markup')}`;
  const codeEl = document.createElement('code');
  codeEl.className = `language-${escHtml(language || 'markup')}`;
  codeEl.textContent = code;
  pre.appendChild(codeEl);

  container.appendChild(header);
  container.appendChild(pre);

  return container;
}
