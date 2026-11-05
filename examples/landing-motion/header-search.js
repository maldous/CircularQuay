import { createHelpStore } from './client-help/help-store.js';
import { buildHelpPostings, searchHelp } from './client-help/help-search.js';
import { documentation } from './explore-manifest.js';

export const maximumSuggestions = 8;

export function documentationSuggestions(query, documents = documentation.documents) {
  const entries = new Map(documents.map((document, index) => [index, {
    title: document.title,
    summary: document.category || 'Documentation',
    keywords: [document.path, ...document.parts.flatMap(part => [part.title, ...part.headings.map(heading => heading.title)])],
    audiences: [], bodyMarkdown: '', controlFamilies: [],
  }]));
  const postings = buildHelpPostings(entries);
  return searchHelp(postings, query, maximumSuggestions).map(hit => {
    const document = documents[hit.document];
    const title = document.id === 'sdk-python-readme' ? `${document.title} · Python` : document.title;
    return { id: document.id, title, summary: document.category || 'Documentation', label: 'Documentation', href: `/explore#doc=${encodeURIComponent(document.id)}&part=${encodeURIComponent(document.parts[0]?.id || 'part-1')}` };
  });
}

export function mergeSuggestions(help, guides) {
  const unique = (items, key) => items.filter((item, index) => items.findIndex(candidate => key(candidate) === key(item)) === index);
  const guideResults = unique(help.map(item => ({ id: item.id, title: item.title, summary: item.summary || '', label: 'Guide', href: `/help#${encodeURIComponent(item.id)}` })), item => `${item.label}:${item.id}`);
  const documentation = unique(guides, item => `${item.label}:${item.id}`);
  const reservedDocumentation = Math.min(2, documentation.length);
  const results = [...guideResults.slice(0, maximumSuggestions - reservedDocumentation), ...documentation.slice(0, reservedDocumentation)];
  for (const item of [...guideResults.slice(maximumSuggestions - reservedDocumentation), ...documentation.slice(reservedDocumentation)]) {
    if (results.length < maximumSuggestions) results.push(item);
  }
  return results.slice(0, maximumSuggestions);
}

function initialise() {
  const form = document.querySelector('form.header-search');
  const input = document.querySelector('#site-search');
  const popup = document.querySelector('#site-search-popup');
  const list = document.querySelector('#site-search-results');
  const status = document.querySelector('#site-search-status');
  if (!form || !input || !popup || !list || !status || form.dataset.searchReady) return;
  form.dataset.searchReady = 'true';
  input.disabled = false; form.querySelector('button[type="submit"]')?.removeAttribute('disabled');
  const store = createHelpStore();
  let timer = 0, request = 0, active = -1, results = [], tabCompletion = false, completed = null;
  const close = ({ invalidate = false } = {}) => { if (invalidate) { request++; clearTimeout(timer); } popup.hidden = true; input.setAttribute('aria-expanded', 'false'); active = -1; input.removeAttribute('aria-activedescendant'); };
  const setActive = index => {
    active = index < 0 ? -1 : (index + results.length) % results.length;
    if (active < 0) input.removeAttribute('aria-activedescendant');
    else input.setAttribute('aria-activedescendant', `site-search-option-${active}`);
    [...list.children].forEach((item, position) => item.setAttribute('aria-selected', String(position === active)));
    if (active >= 0) list.children[active]?.scrollIntoView({ block: 'nearest' });
  };
  const render = () => {
    list.replaceChildren(...results.map((item, index) => {
      const option = document.createElement('li'); option.id = `site-search-option-${index}`; option.setAttribute('role', 'option'); option.dataset.index = String(index);
      const title = document.createElement('strong'); title.textContent = item.title;
      const label = document.createElement('span'); label.className = 'search-result-label'; label.textContent = item.label;
      const summary = document.createElement('small'); summary.textContent = item.summary;
      option.append(title, label, summary); return option;
    }));
    popup.hidden = false; input.setAttribute('aria-expanded', 'true'); setActive(results.length ? 0 : -1);
    status.textContent = results.length ? `${results.length} suggestions available.` : 'No matching guides or documentation.';
  };
  const search = async () => {
    const query = input.value.trim(); const current = ++request;
    if (!query) { results = []; status.textContent = ''; close(); return; }
    status.textContent = 'Searching locally…';
    list.replaceChildren(); popup.hidden = false; input.setAttribute('aria-expanded', 'true');
    try {
      const [help, guides] = await Promise.all([store.search(query), Promise.resolve(documentationSuggestions(query))]);
      if (current !== request) return;
      results = mergeSuggestions(help, guides); render();
    } catch {
      if (current !== request) return;
      results = []; list.replaceChildren(); popup.hidden = false; input.setAttribute('aria-expanded', 'true'); status.textContent = 'Search is unavailable. Try typing again.';
    }
  };
  const open = index => { const item = typeof index === 'object' ? index : results[index < 0 ? 0 : index]; if (item) location.href = item.href; };
  const invalidateForInput = () => { request++; clearTimeout(timer); results = []; completed = null; tabCompletion = false; close(); };
  input.addEventListener('input', () => { invalidateForInput(); if (input.value.trim()) timer = setTimeout(search, 100); else status.textContent = ''; });
  input.addEventListener('keydown', event => {
    if (event.key === 'Escape') { event.preventDefault(); results = []; completed = null; close({ invalidate: true }); status.textContent = ''; return; }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') { event.preventDefault(); if (results.length && !popup.hidden) setActive(active + (event.key === 'ArrowDown' ? 1 : -1)); else void search().then(() => { if (results.length) setActive(event.key === 'ArrowDown' ? 0 : results.length - 1); }).catch(() => {}); return; }
    if (event.key === 'Tab' && !event.shiftKey && !popup.hidden && results.length && !tabCompletion) { event.preventDefault(); completed = results[active < 0 ? 0 : active]; input.value = completed.title; tabCompletion = true; close(); return; }
    if (event.key === 'Enter') { event.preventDefault(); clearTimeout(timer); if (completed) open(completed); else if (results.length) open(active); else void search().then(() => { if (results.length) open(0); }).catch(() => {}); }
  });
  form.addEventListener('submit', event => { event.preventDefault(); if (completed) open(completed); else if (results.length) open(active); else void search().then(() => { if (results.length) open(0); }).catch(() => {}); });
  list.addEventListener('mousemove', event => { const option = event.target.closest('[data-index]'); if (option) setActive(Number(option.dataset.index)); });
  list.addEventListener('pointerdown', event => { if (event.target.closest('[data-index]')) event.preventDefault(); });
  list.addEventListener('click', event => { const option = event.target.closest('[data-index]'); if (option) open(Number(option.dataset.index)); });
  form.addEventListener('focusout', () => setTimeout(() => { if (!form.contains(document.activeElement)) { results = []; close({ invalidate: true }); } }, 0));
  document.addEventListener('pointerdown', event => { if (!form.contains(event.target) && !popup.contains(event.target)) { results = []; close({ invalidate: true }); } });
}

if (typeof document !== 'undefined') initialise();
