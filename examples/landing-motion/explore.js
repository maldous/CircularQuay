import { documentation } from './explore-manifest.js';
import { navigationGroups, groupForDocument, exploreHomeMarkup } from './explore-navigation.js';

const content = document.querySelector('#document-content');
const status = document.querySelector('#document-status');
const article = document.querySelector('#article');
const shell = document.querySelector('#main');
const documentList = document.querySelector('#document-list');
const contentsList = document.querySelector('#contents-list');
const search = document.querySelector('#document-search');
const topicsBack = document.querySelector('#topics-back');
const mobileTopic = document.querySelector('#mobile-topic-link');
let active = null;
const moduleCache = new Map();
const escapeHtml = value => String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character]);
const repositoryBase = () => String(documentation.repository || '').replace(/\/$/, '').replace(/\.git$/, '').startsWith('http') ? String(documentation.repository).replace(/\/$/, '').replace(/\.git$/, '') : `https://github.com/${String(documentation.repository || '').replace(/^\/+|\/+$/g, '')}`;
const snapshotId = () => String(documentation.sourceFreezeManifestSha256 || documentation.snapshotSha || documentation.snapshotHash || documentation.revision || 'local');
const snapshotShort = () => snapshotId().slice(0, 12);
const repositoryBranch = () => String(documentation.repositoryBranch || documentation.branch || documentation.sourceBranch || 'feature/initial-implementation');
const sourceHref = item => `${repositoryBase()}/blob/${encodeURIComponent(repositoryBranch())}/${item.path}`;
const docHref = item => `#doc=${encodeURIComponent(item.id)}&part=${encodeURIComponent(item.parts[0].id)}`;

function route() {
  const query = new URLSearchParams(location.hash.slice(1));
  const group = navigationGroups.find(item => item.id === query.get('group'));
  const requested = query.get('doc');
  if (!requested) return group ? { type: 'group', group } : { type: 'home' };
  const document = documentation.documents.find(item => item.id === requested) || documentation.documents[0];
  const part = document.parts.find(item => item.id === query.get('part')) || document.parts[0];
  return { type: 'document', document, part, group: groupForDocument(document.id), anchor: query.get('anchor') || '', recovery: document.id !== requested || (query.get('part') && part.id !== query.get('part')) };
}
function matches(item, term) { return !term || [item.title, ...item.parts.flatMap(part => [part.title, ...part.headings.map(heading => heading.title)])].join(' ').toLocaleLowerCase().includes(term.toLocaleLowerCase()); }
function setMode(mode) { shell.dataset.view = mode; }
function resetReader() { content.hidden = false; status.hidden = true; contentsList.replaceChildren(); }

function renderRail(group, selected) {
  const term = search.value.trim();
  const items = term ? documentation.documents.filter(item => matches(item, term)) : (group?.documents || []).map(id => documentation.documents.find(item => item.id === id)).filter(Boolean);
  documentList.replaceChildren();
  if (group && !term) { const label = document.createElement('p'); label.className = 'rail-topic'; label.textContent = group.title; documentList.append(label); }
  items.forEach(item => { const link = document.createElement('a'); link.className = 'document-link'; link.href = docHref(item); link.textContent = item.title; if (item.id === selected?.id) link.setAttribute('aria-current', 'page'); documentList.append(link); });
  if (!items.length) documentList.innerHTML = '<p class="rail-revision">No documents match this search.</p>';
}
function home() { active = null; setMode('home'); resetReader(); content.innerHTML = `<section class="explore-home"><h1>Explore CircularQuay</h1><p class="home-intro">Find the documentation that matches the work in front of you. Start with the product shape, a workload, a connection, an access decision, an operational task, or the evidence behind a release.</p>${exploreHomeMarkup()}</section>`; article.scrollTop = 0; }
function groupOverview(group) { active = null; setMode('group'); resetReader(); const docs = group.documents.map(id => documentation.documents.find(item => item.id === id)).filter(Boolean); content.innerHTML = `<section class="group-overview"><nav class="breadcrumbs" aria-label="Breadcrumb"><a href="/explore">Explore</a><span aria-hidden="true">/</span><span>${escapeHtml(group.title)}</span></nav><h1>${escapeHtml(group.title)}</h1><p class="group-intro">${escapeHtml(group.description)}</p><div class="group-documents">${docs.map(item => `<a href="${docHref(item)}"><strong>${escapeHtml(item.title)}</strong><span>${escapeHtml(item.parts.length > 1 ? `${item.parts.length} chapters` : 'Read the guide')}</span></a>`).join('')}</div></section>`; renderRail(group, null); article.scrollTop = 0; }
async function loadPart(item, part) { if (!moduleCache.has(item.id)) moduleCache.set(item.id, import(item.module)); const module = await moduleCache.get(item.id); if (!module.default || module.default.id !== item.id) throw new Error('The local document module does not match its manifest entry.'); return module.default.parts.find(candidate => candidate.id === part.id) || module.default.parts[0]; }
function guideContents(item, activePart) { if (item.parts.length < 2) return ''; return `<nav class="guide-contents" aria-label="In this guide"><p>In this guide</p><ol>${item.parts.map(part => `<li><a${part.id === activePart.id ? ' aria-current="page"' : ''} href="#doc=${encodeURIComponent(item.id)}&part=${encodeURIComponent(part.id)}">${escapeHtml(part.title.replaceAll('`', ''))}</a></li>`).join('')}</ol></nav>`; }
function renderContents(item, part) { contentsList.replaceChildren(); if (item.parts.length > 1) { const title = document.createElement('p'); title.className = 'contents-heading'; title.textContent = 'In this guide'; contentsList.append(title); item.parts.forEach(chapter => { const link = document.createElement('a'); link.href = `#doc=${encodeURIComponent(item.id)}&part=${encodeURIComponent(chapter.id)}`; link.textContent = chapter.title.replaceAll('`', ''); if (chapter.id === part.id) link.setAttribute('aria-current', 'page'); contentsList.append(link); }); } const title = document.createElement('p'); title.className = 'contents-heading'; title.textContent = 'On this page'; contentsList.append(title); part.headings.filter(heading => content.querySelector(`#${CSS.escape(heading.id)}`)).forEach(heading => { const link = document.createElement('a'); link.href = `#doc=${encodeURIComponent(item.id)}&part=${encodeURIComponent(part.id)}&anchor=${encodeURIComponent(heading.id)}`; link.dataset.level = heading.level; link.textContent = heading.title; contentsList.append(link); }); }
function neighbors(item, part) { const entries = documentation.documents.flatMap(doc => doc.parts.map(chapter => ({ doc, chapter }))); const index = entries.findIndex(entry => entry.doc.id === item.id && entry.chapter.id === part.id); return [entries[index - 1], entries[index + 1]]; }
function partNavigation(item, part) { const [previous, next] = neighbors(item, part); const link = (entry, label) => entry ? `<a href="#doc=${encodeURIComponent(entry.doc.id)}&part=${encodeURIComponent(entry.chapter.id)}">${label}: ${escapeHtml(entry.chapter.title)}</a>` : ''; return `<nav class="part-navigation" aria-label="Document navigation">${link(previous, 'Previous')}${link(next, 'Next')}</nav>`; }
function scroll(anchor) { article.scrollTop = 0; if (anchor) requestAnimationFrame(() => content.querySelector(`#${CSS.escape(anchor)}`)?.scrollIntoView({ block: 'start', inline: 'nearest' })); }
async function documentView(next, focus = false) { active = next; setMode('document'); renderRail(next.group, next.document); const group = next.group; const groupHref = `/explore#group=${encodeURIComponent(group?.id || '')}`; topicsBack.href = groupHref; mobileTopic.href = groupHref; mobileTopic.textContent = group?.title || 'Topic'; status.hidden = false; status.dataset.state = ''; status.textContent = 'Loading documentation…'; content.hidden = true; try { const loaded = await loadPart(next.document, next.part); if (active !== next) return; const title = (next.document.parts.length > 1 ? next.part.title : next.document.title).replaceAll('`', ''); content.innerHTML = `<nav class="breadcrumbs" aria-label="Breadcrumb"><a href="/explore">Explore</a><span aria-hidden="true">/</span><a href="${groupHref}">${escapeHtml(group?.title || 'Documentation')}</a><span aria-hidden="true">/</span><span>${escapeHtml(next.document.title)}</span></nav>${guideContents(next.document, next.part)}${loaded.html}`; if (!content.querySelector('h1')) content.insertAdjacentHTML('afterbegin', `<h1 tabindex="-1">${escapeHtml(title)}</h1>`); if (next.recovery) content.insertAdjacentHTML('afterbegin', '<p class="document-recovery" role="status">That document location was not found. Showing the nearest available document instead.</p>'); content.insertAdjacentHTML('beforeend', `<p class="document-provenance"><a href="${sourceHref(next.document)}" rel="noreferrer">Repository source (may differ from this local snapshot)</a></p>${partNavigation(next.document, next.part)}`); resetReader(); renderContents(next.document, next.part); scroll(next.anchor); if (focus) content.querySelector('h1, h2, h3, h4')?.focus({ preventScroll: true }); } catch (error) { status.hidden = false; status.dataset.state = 'error'; status.textContent = `This document could not be loaded. Choose another document or return to the topic overview. (${error.message})`; contentsList.replaceChildren(); } }
function render(focus = false) { const next = route(); if (next.type === 'home') home(); else if (next.type === 'group') groupOverview(next.group); else documentView(next, focus); }
function initialise() { if (!documentation?.documents?.length) throw new Error('Documentation manifest is empty.'); const assigned = new Set(navigationGroups.flatMap(group => group.documents)); if (assigned.size !== documentation.documents.length || documentation.documents.some(item => !assigned.has(item.id))) throw new Error('Documentation navigation is incomplete.'); document.querySelector('#rail-revision').textContent = `Local working-tree snapshot · ${documentation.snapshotDate} · ${snapshotShort()}`; document.querySelector('#revision-stamp').textContent = `Local working-tree snapshot ${snapshotShort()} · ${documentation.snapshotDate}`; document.querySelector('#repository-link').href = `${repositoryBase()}/tree/${encodeURIComponent(repositoryBranch())}`; search.addEventListener('input', () => renderRail(active?.group || route().group, active?.document)); addEventListener('hashchange', () => render(true)); render(false); }
initialise();
