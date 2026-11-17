import { readFile } from 'node:fs/promises';

export const SITE_HEADER_SLOT = '<!-- cq:site-header -->';
export const MAX_SITE_TEMPLATE_BYTES = 64 * 1024;
export const MAX_SITE_HEADER_BYTES = 8 * 1024;
export const SITE_PAGE_FILES = Object.freeze([
  'index.html', 'about.html', 'accessibility.html', 'contact.html',
  'cookies.html', 'privacy.html', 'security.html', 'status.html',
  'terms.html', 'explore.html', 'help.html', 'login.html',
]);

const pageNames = new Set(SITE_PAGE_FILES);
const slotBytes = Buffer.from(SITE_HEADER_SLOT);

function requirePageName(name) {
  if (!pageNames.has(name)) throw new Error('Unknown site page');
}

function decodeInputs(template, header) {
  if (!Buffer.isBuffer(template) || !Buffer.isBuffer(header)) {
    throw new TypeError('Site page inputs must be buffers');
  }
  if (template.length > MAX_SITE_TEMPLATE_BYTES || header.length > MAX_SITE_HEADER_BYTES) {
    throw new RangeError('Site page source exceeds size limit');
  }
  let templateText, headerText;
  try {
    const decoder = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true });
    templateText = decoder.decode(template);
    headerText = decoder.decode(header);
  } catch {
    throw new Error('Site page inputs must be UTF-8');
  }
  if (templateText.includes('\0') || headerText.includes('\0')) {
    throw new Error('Site page inputs contain NUL');
  }
  return { templateText, headerText };
}

function requireLiteralHeader(header) {
  const oneOpening = [...header.matchAll(/<header\b/gi)].length === 1;
  const oneClosing = [...header.matchAll(/<\/header>/gi)].length === 1;
  if (!/^<header class="site-header page-width has-help-search">/.test(header)
    || !header.endsWith('</header>') || !oneOpening || !oneClosing
    || header.includes(SITE_HEADER_SLOT)) {
    throw new Error('Invalid literal site header');
  }
}

// Source composition is limited to these twelve maintained templates and one
// literal fragment. This is not an HTML evaluator or a request-selected include.
export function composeSitePage(name, template, header) {
  requirePageName(name);
  const { templateText, headerText } = decodeInputs(template, header);
  const start = template.indexOf(slotBytes);
  if (start < 0 || template.indexOf(slotBytes, start + slotBytes.length) !== -1) {
    throw new Error('Expected one site-header slot');
  }
  if (/<header\b/i.test(templateText)) throw new Error('Template contains a site header');
  requireLiteralHeader(headerText);
  const renderedHeader = name === 'index.html' ? headerText : headerText
    .replace('href="#main"', 'href="/"')
    .replaceAll('href="#scene"', 'href="/#scene"')
    .replaceAll('href="#ecosystem"', 'href="/#ecosystem"');
  // Byte offsets preserve the surrounding UTF-8 source exactly, including
  // comments and indentation. The returned buffer never aliases either input.
  return Buffer.concat([
    template.subarray(0, start), Buffer.from(renderedHeader),
    template.subarray(start + slotBytes.length),
  ]);
}

// Fixed local inputs only; callers cannot supply an include path or URL.
export async function readSitePage(name) {
  requirePageName(name);
  const [template, header] = await Promise.all([
    readFile(new URL(name, import.meta.url)),
    readFile(new URL('site-header.html', import.meta.url)),
  ]);
  return composeSitePage(name, template, header);
}
