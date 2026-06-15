import { marked } from 'marked';
import DOMPurify from 'isomorphic-dompurify';
import type { Tokens } from 'marked';
import Prism from 'prismjs';
import 'prismjs/components/prism-typescript';
import 'prismjs/components/prism-go';
import 'prismjs/components/prism-python';
import 'prismjs/components/prism-json';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-sql';
import 'prismjs/components/prism-markdown';
import type { DataCard, ChartData, TableData, MetricData, ComparisonData } from '../api/types';

const renderer = new marked.Renderer();

renderer.link = function (this: typeof renderer, token: Tokens.Link): string {
  const href = token.href;
  const title = token.title ?? undefined;
  if (!href || typeof href !== 'string') {
    const text = token.tokens?.length ? (this.parser?.parseInline(token.tokens) ?? '') : '';
    return `<a href="#">${text}</a>`;
  }
  const local =
    href.startsWith(`${window.location.protocol}//${window.location.host}`) ||
    href.startsWith('/') ||
    href.startsWith('#');
  const text = token.tokens?.length ? (this.parser?.parseInline(token.tokens) ?? '') : '';
  const linkHtml = local
    ? `<a href="${href}"${title ? ` title="${title}"` : ''}>${text}</a>`
    : `<a href="${href}"${title ? ` title="${title}"` : ''} target="_blank" rel="noopener noreferrer">${text}</a>`;
  return linkHtml;
};

renderer.code = function (this: typeof renderer, token: Tokens.Code): string {
  const language = token.lang;
  const code = token.text;
  // Support data card code blocks: ```data-chart, ```data-table, etc.
  if (
    language === 'data-chart' ||
    language === 'data-table' ||
    language === 'data-metric' ||
    language === 'data-comparison'
  ) {
    return `<div class="data-card-placeholder" data-card-type="${language.replace('data-', '')}" data-card-code="${encodeAttr(code)}"></div>`;
  }
  const lang = language && Prism.languages[language] ? language : 'markup';
  const highlighted = Prism.highlight(code, Prism.languages[lang]!, lang);
  return `<pre class="language-${lang}"><code class="language-${lang}">${highlighted}</code></pre>`;
};

marked.setOptions({ renderer, breaks: true });

function encodeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function decodeAttr(str: string): string {
  return str
    .replace(/&quot;/g, '"')
    .replace(/&gt;/g, '>')
    .replace(/&lt;/g, '<')
    .replace(/&amp;/g, '&');
}

export function renderMarkdown(text: string): string {
  const rawHtml = marked.parse(text) as string;
  const cleanHtml = DOMPurify.sanitize(rawHtml, {
    ALLOWED_TAGS: [
      'p',
      'br',
      'strong',
      'em',
      'code',
      'pre',
      'ul',
      'ol',
      'li',
      'a',
      'h1',
      'h2',
      'h3',
      'h4',
      'h5',
      'h6',
      'blockquote',
      'table',
      'thead',
      'tbody',
      'tr',
      'th',
      'td',
      'img',
      'span',
      'div',
      'hr',
      'sub',
      'sup',
      'del',
      'input',
    ],
    ALLOWED_ATTR: [
      'href',
      'class',
      'src',
      'alt',
      'title',
      'target',
      'rel',
      'checked',
      'type',
      'disabled',
      'data-card-type',
      'data-card-code',
    ],
  });
  return cleanHtml;
}
export function highlightCode(el: HTMLElement): void {
  Prism.highlightAllUnder(el);
}
export function extractUrls(text: string): string[] {
  return [...new Set(text.match(/https?:\/\/[^\s\)\]]+/g) || [])];
}

// ═══ Retry Utility ═══

export interface RetryOptions {
  maxRetries?: number;
  initialDelay?: number;
  maxDelay?: number;
  backoffFactor?: number;
}

export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {}
): Promise<T> {
  const {
    maxRetries = 3,
    initialDelay = 1000,
    maxDelay = 30000,
    backoffFactor = 2,
  } = options;

  let lastError: Error;
  let delay = initialDelay;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      if (attempt === maxRetries) {
        break;
      }

      await new Promise(resolve => setTimeout(resolve, delay));
      delay = Math.min(delay * backoffFactor, maxDelay);
    }
  }

  throw lastError!;
}

// ═══ Entity Detection ═══

const ENTITY_MAP: Record<string, 'company' | 'index' | 'metric' | 'sector'> = {
  HDB: 'company',
  ACB: 'company',
  VCB: 'company',
  TCB: 'company',
  MBB: 'company',
  BID: 'company',
  CTG: 'company',
  VPB: 'company',
  STB: 'company',
  EIB: 'company',
  VNINDEX: 'index',
  HNXINDEX: 'index',
  UPCOMINDEX: 'index',
  NIM: 'metric',
  ROE: 'metric',
  ROA: 'metric',
  NPL: 'metric',
  CAR: 'metric',
  LDR: 'metric',
  CIR: 'metric',
  PPOP: 'metric',
};

export function detectEntities(text: string): string[] {
  const found: string[] = [];
  const seen = new Set<string>();
  const upper = text.toUpperCase();
  for (const name of Object.keys(ENTITY_MAP)) {
    if (upper.includes(name) && !seen.has(name)) {
      found.push(name);
      seen.add(name);
    }
  }
  return found;
}

// ═══ Data Card Parsing ═══

export function parseDataCards(html: string): DataCard[] {
  const cards: DataCard[] = [];
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, 'text/html');
  const placeholders = doc.querySelectorAll('.data-card-placeholder');

  placeholders.forEach((ph) => {
    const rawType = ph.getAttribute('data-card-type');
    const rawCode = ph.getAttribute('data-card-code');
    if (!rawType || !rawCode) return;

    const type = rawType as DataCard['type'];
    try {
      const json = JSON.parse(decodeAttr(rawCode));
      switch (type) {
        case 'chart':
          cards.push({ type: 'chart', title: json.title, data: json as ChartData });
          break;
        case 'table':
          cards.push({ type: 'table', title: json.title, data: json as TableData });
          break;
        case 'metric':
          cards.push({ type: 'metric', data: json as MetricData });
          break;
        case 'comparison':
          cards.push({ type: 'comparison', title: json.title, data: json as ComparisonData });
          break;
      }
    } catch {
      /* skip invalid cards */
    }
  });

  return cards;
}
