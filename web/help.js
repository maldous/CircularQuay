import { createHelpStore } from './help-store.js';

const escape = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
const store = createHelpStore();

// Markdown is a presentation subset, never HTML. Even code, links, placeholders
// and unsupported markup remain escaped text; no authored URL becomes a fetch.
export function helpMarkdown(source) {
  const lines = source.replaceAll('\r\n', '\n').split('\n');
  const headings = lines.filter(line => /^#{1,4} /.test(line)).map(line => line.indexOf(' '));
  const firstLevel = Math.min(4, ...headings);
  const output = [];
  let code = null;
  let paragraph = [];
  let list = null;
  const inline = text => escape(text).replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  const flushParagraph = () => { if (paragraph.length) output.push(`<p>${inline(paragraph.join(' '))}</p>`); paragraph = []; };
  const flushList = () => {
    if (!list) return;
    const start = list.kind === 'ol' ? ` start="${list.start}"` : '';
    output.push(`<${list.kind}${start}>${list.items.map(item => `<li>${inline(item)}</li>`).join('')}</${list.kind}>`);
    list = null;
  };
  const flush = () => { flushParagraph(); flushList(); };
  for (const line of lines) {
    if (line.startsWith('```')) {
      flush();
      if (code === null) code = [];
      else { output.push(`<pre tabindex="0"><code>${escape(code.join('\n'))}</code></pre>`); code = null; }
      continue;
    }
    if (code !== null) { code.push(line); continue; }
    if (/^#{1,4} /.test(line)) {
      flush();
      const depth = line.indexOf(' ');
      const level = Math.min(6, 3 + depth - firstLevel);
      output.push(`<h${level}>${inline(line.slice(depth + 1))}</h${level}>`);
      continue;
    }
    const ordered = /^\s*(\d{1,4})\. (.*)$/.exec(line);
    const unordered = /^\s*[-*] (.*)$/.exec(line);
    if (ordered || unordered) {
      flushParagraph();
      const kind = ordered ? 'ol' : 'ul';
      if (list && list.kind !== kind) flushList();
      list ||= { kind, start: ordered ? Number(ordered[1]) : 1, items: [] };
      list.items.push(ordered ? ordered[2] : unordered[1]);
    } else if (!line.trim()) flushParagraph();
    else { flushList(); paragraph.push(line); }
  }
  flush();
  if (code !== null) output.push(`<pre tabindex="0"><code>${escape(code.join('\n'))}</code></pre>`);
  return output.join('');
}

export function mountHelp(root, { topic = '', close, standalone = false } = {}) {
  let generation = 0;
  let queryGeneration = 0;
  let selected = topic;
  let timer;
  let disposed = false;
  root.innerHTML = `<header class="help-heading"><div><h1>Help & search</h1><p>Guides, controls and examples</p></div>${close ? '<button type="button" data-help-close class="quiet">Close</button>' : ''}</header><div class="help-query"><label for="help-query">Search the manual<input id="help-query" type="search" maxlength="256" autocomplete="off" spellcheck="false" placeholder="Try “load example” or “TypeScript”"></label><p>Search runs on this device. Articles load as you open them.</p></div><div class="help-status" role="status" aria-live="polite">Loading the help catalog…</div><div class="help-layout"><nav class="help-results" aria-label="Help topics"></nav><article class="help-article" tabindex="-1" aria-label="Help article"><h2>Find a guide</h2><p>Search all available guide text, or choose a topic.</p></article></div>`;
  const input = root.querySelector('#help-query');
  const status = root.querySelector('.help-status');
  const results = root.querySelector('.help-results');
  const article = root.querySelector('.help-article');
  const reportError = () => { status.textContent = 'Help could not load. Your workspace is unchanged. Try again.'; status.setAttribute('role', 'alert'); };
  const list = documents => {
    results.innerHTML = documents.length ? documents.map(item => `<button type="button" class="help-result" data-topic="${escape(item.id)}" aria-label="Open ${escape(item.title)}" ${item.id === selected ? 'aria-current="page"' : ''}><strong>${escape(item.title)}</strong><span>${escape(item.summary)}</span></button>`).join('') : '<div class="help-no-results"><h2>No matching article</h2><p>Try a shorter phrase, a control name, or a CLI/SDK term.</p><button type="button" data-help-clear>Browse all topics</button></div>';
  };
  async function showArticle(id, moveFocus = false) {
    const current = ++generation;
    selected = id;
    for (const button of results.querySelectorAll('[data-topic]')) {
      if (button.dataset.topic === id) button.setAttribute('aria-current', 'page');
      else button.removeAttribute('aria-current');
    }
    article.innerHTML = '<p role="status">Loading article…</p>';
    try {
      const value = await store.article(id);
      if (disposed || current !== generation) return;
      article.innerHTML = `<h2>${escape(value.title)}</h2><p class="help-capability">${escape(value.capabilityStatus)}</p><div class="help-body">${helpMarkdown(value.bodyMarkdown)}</div><details class="help-references"><summary>Source references</summary><ul>${value.canonicalRefs.map(reference => `<li>${escape(reference.repoPath)}</li>`).join('')}</ul></details>${standalone ? '' : `<a class="help-page-link" href="/help.html#${escape(id)}" target="_blank" rel="noopener">Open this guide in a new tab</a>`}`;
      if (moveFocus) { article.focus(); article.scrollTop = 0; }
    } catch { if (!disposed && current === generation) { article.innerHTML = '<h2>Article unavailable</h2><p role="alert">The article could not be loaded or verified. Your workspace is unchanged.</p><button type="button" data-help-retry>Try article again</button>'; } }
  }
  async function updateSearch() {
    const current = ++queryGeneration;
    const query = input.value;
    status.setAttribute('role', 'status');
    status.textContent = query.trim() ? 'Searching locally…' : 'Loading topics…';
    try {
      const documents = query.trim() ? await store.search(query) : (await store.catalog()).documents.filter(item => !item.id.startsWith('control-'));
      if (disposed || current !== queryGeneration) return;
      list(documents); status.textContent = `${documents.length} ${query.trim() ? 'matching articles' : 'guides; search also covers controls'}`;
    } catch { if (!disposed && current === queryGeneration) { reportError(); results.innerHTML = '<button type="button" data-help-reload>Try loading help again</button>'; } }
  }
  input.oninput = () => { clearTimeout(timer); queryGeneration += 1; timer = setTimeout(updateSearch, 100); };
  input.onkeydown = event => {
    if (event.key === 'ArrowDown') { event.preventDefault(); results.querySelector('[data-topic]')?.focus(); }
    if (event.key === 'Enter') { event.preventDefault(); clearTimeout(timer); void updateSearch(); }
  };
  root.onclick = event => {
    if (event.target.closest('[data-help-close]')) close?.();
    const topicButton = event.target.closest('[data-topic]');
    if (topicButton) void showArticle(topicButton.dataset.topic, true);
    if (event.target.closest('[data-help-clear]')) { input.value = ''; input.focus(); void updateSearch(); }
    if (event.target.closest('[data-help-retry]')) void showArticle(selected, true);
    if (event.target.closest('[data-help-reload]')) void updateSearch();
  };
  results.onkeydown = event => {
    if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return;
    const buttons = [...results.querySelectorAll('[data-topic]')];
    const index = buttons.indexOf(event.target);
    if (index < 0) return;
    event.preventDefault();
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1 : (index + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length;
    buttons[next]?.focus();
  };
  void updateSearch().then(() => {
    if (topic && !disposed) return showArticle(topic);
    return undefined;
  }).catch(error => {
    if (!disposed) reportError();
    console.error('Initial help search failed', error);
  });
  return { focus: () => input.focus(), dispose: () => { disposed = true; generation += 1; queryGeneration += 1; clearTimeout(timer); } };
}

let dialog;
let viewer;
let opener;
export function openHelp(topic = '', trigger = document.activeElement) {
  if (!dialog) {
    dialog = document.createElement('dialog');
    dialog.lang = 'en';
    dialog.className = 'help-dialog'; dialog.setAttribute('aria-label', 'Help and search');
    document.body.append(dialog);
    dialog.addEventListener('close', () => { viewer?.dispose(); if (opener?.isConnected) opener.focus(); });
  }
  opener = trigger;
  viewer?.dispose();
  viewer = mountHelp(dialog, { topic, close: () => dialog.close() });
  if (!dialog.open) dialog.showModal();
  viewer.focus();
}
