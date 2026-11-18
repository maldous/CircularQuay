import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { browser, publicHelpFetcher } from './coverage-dom-fixture.test.mjs';
import { helpRelease } from './help-release.js';
import * as contract from './help-contract.js';
import { searchHelp, buildHelpPostings } from './help-search.js';

let sequence = 0;
const load = name => import(new URL(`${name}?owned-component=${++sequence}`, import.meta.url));
const click = (x, node) => x.event(node, 'click', { bubbles: true });
async function ready(x, root) { await x.until(() => /guides;/.test(root.querySelector('.help-status').textContent)); }

test('real help component browses, locally searches, navigates, restores focus and disposes pending work', async t => {
  const x = browser(t); const requests = []; globalThis.fetch = publicHelpFetcher(requests);
  const { mountHelp, openHelp } = await load('help.js'); const root = x.document.querySelector('#app'); let closes = 0;
  const view = mountHelp(root, { close: () => closes++ }); await ready(x, root); view.focus();
  const input = root.querySelector('#help-query'); assert.equal(x.document.activeElement, input);
  const buttons = root.querySelectorAll('[data-topic]'); assert.ok(buttons.length > 10);
  x.event(input, 'keydown', { key: 'ArrowDown' }); assert.equal(x.document.activeElement, buttons[0]);
  for (const key of ['End', 'Home', 'ArrowUp', 'ArrowDown']) x.event(root.querySelector('.help-results'), 'keydown', { target: buttons[0], key });
  x.event(root.querySelector('.help-results'), 'keydown', { target: input, key: 'End' });
  x.event(root.querySelector('.help-results'), 'keydown', { target: buttons[0], key: 'Tab' });
  click(x, buttons[0]); await x.until(() => !!root.querySelector('.help-body'));
  assert.equal(x.document.activeElement, root.querySelector('.help-article')); assert.ok(root.querySelector('.help-page-link'));
  const privateQuery = 'OWNED_QUERY_NOT_SENT'; input.value = privateQuery; x.event(input, 'input'); await x.advance(100); await x.until(() => !!root.querySelector('[data-help-clear]'));
  assert.ok(requests.every(row => !row.path.includes(privateQuery))); click(x, root.querySelector('[data-help-clear]')); await ready(x, root);
  input.value = 'TypeScript'; x.event(input, 'keydown', { key: 'Enter' }); await x.until(() => /matching articles/.test(root.querySelector('.help-status').textContent));
  click(x, root.querySelector('[data-help-close]')); assert.equal(closes, 1);
  input.value = 'pending'; x.event(input, 'input'); view.dispose(); const before = root.innerHTML; await x.advance(500); assert.equal(root.innerHTML, before);
  const opener = x.document.createElement('button'); x.document.body.append(opener); opener.focus(); openHelp('start', opener);
  const dialog = x.document.querySelector('dialog.help-dialog'); assert.equal(dialog.open, true); await x.until(() => !!dialog.querySelector('.help-body'));
  openHelp('policy', opener); await x.until(() => /Policy/i.test(dialog.querySelector('.help-article').textContent)); dialog.close(); assert.equal(x.document.activeElement, opener);
  assert.equal(x.writes.length, 0);
});

test('catalog and article failures are retryable fixed UI states with no workspace mutation', async t => {
  const x = browser(t); const requests = []; let failCatalog = true, failArticle = true;
  globalThis.fetch = publicHelpFetcher(requests, { transform(bytes, path) { if ((failCatalog && path.includes('catalog-')) || (failArticle && path.includes('article-'))) return Buffer.from('INVALID'); return bytes; } });
  const { mountHelp } = await load('help.js'); const root = x.document.querySelector('#app'); const view = mountHelp(root);
  await x.until(() => !!root.querySelector('[data-help-reload]')); assert.equal(root.querySelector('.help-status').getAttribute('role'), 'alert');
  failCatalog = false; click(x, root.querySelector('[data-help-reload]')); await ready(x, root); click(x, root.querySelector('[data-topic]'));
  await x.until(() => !!root.querySelector('[data-help-retry]')); failArticle = false; click(x, root.querySelector('[data-help-retry]')); await x.until(() => !!root.querySelector('.help-body'));
  view.dispose(); assert.equal(x.sources.length, 0); assert.equal(x.writes.length, 0);
});

test('stale/disposed catalog, search and article results cannot replace a newer help view', async t => {
  const x = browser(t); const requests = []; let release; let blocked = true;
  const gate = new Promise(resolve => { release = resolve; });
  globalThis.fetch = publicHelpFetcher(requests, { before: path => blocked && path.includes('article-') ? gate : Promise.resolve() });
  const { mountHelp } = await load('help.js'); const root = x.document.querySelector('#app'); const view = mountHelp(root); await ready(x, root);
  let completedDigests = 0;
  const actualDigest = crypto.subtle.digest.bind(crypto.subtle);
  t.mock.method(crypto.subtle, 'digest', async (...args) => {
    const value = await actualDigest(...args);
    completedDigests++;
    return value;
  });
  click(x, root.querySelector('[data-topic]'));
  await x.until(() => requests.some(row => row.path.includes('article-')));
  view.dispose(); root.innerHTML = '<p>New independent page</p>'; blocked = false; release();
  await x.until(() => completedDigests > 0);
  assert.equal(root.textContent, 'New independent page');
  const abandoned = x.document.createElement('section'); x.document.body.append(abandoned); const next = mountHelp(abandoned, { topic: 'not-in-catalog', standalone: true }); await x.until(() => !!abandoned.querySelector('[data-help-retry]'));
  assert.equal(abandoned.querySelector('.help-page-link'), null); next.dispose();
});

test('actual launcher topic/schema keyboard routes use public assets only and retain a single launcher', async t => {
  const x = browser(t, { html: '<button id="trigger" data-help-topic="policy" data-help-schema="/sources/{id}">Context</button>', hash: '#/configuration' }); const requests = []; globalThis.fetch = publicHelpFetcher(requests);
  const launcher = await load('help-launcher.js'); launcher.installHelp(); launcher.installHelp(); assert.equal(x.document.querySelectorAll('#help-launcher').length, 1);
  for (const bad of [null, '', '/private key', '/sources/SECRET', '/a//b', '/a~1b']) assert.equal(await launcher.schemaHelpTopic(bad), null);
  assert.match(await launcher.schemaHelpTopic('/'), /^control-schema--[a-f0-9]{12}$/); assert.match(await launcher.schemaHelpTopic('/sources/{id}'), /^control-schema-sources-id-/);
  const trigger = x.document.querySelector('#trigger'); trigger.focus(); x.event(x.document, 'keydown', { key: 'F1' }); await x.until(() => !!x.document.querySelector('.help-dialog'));
  const dialog = x.document.querySelector('.help-dialog'); assert.equal(dialog.open, true); dialog.close(); trigger.click(); await x.drain(); assert.equal(dialog.open, true); dialog.close();
  x.document.querySelector('#help-launcher').click(); await x.drain(); assert.equal(dialog.open, true); assert.ok(requests.every(row => !row.path.includes('SECRET'))); dialog.close();
});

test('standalone help entry binds only validated topics and client-help adapter preserves static fetch options', async t => {
  const x = browser(t, { html: '<main id="help-page"></main>', hash: '#policy' }); const requests = []; globalThis.fetch = publicHelpFetcher(requests);
  await load('help-page.js'); const root = x.document.querySelector('#help-page'); await x.until(() => !!root.querySelector('.help-body')); assert.equal(root.lang, 'en'); assert.equal(root.querySelector('.help-page-link'), null);
  const adapter = await import('../examples/landing-motion/client-help/help-catalog.js'); assert.equal(adapter.previewHelpAssetPath(new URL('https://preview.invalid/help/generated/manifest.json')), '/client-help/generated/manifest.json'); assert.equal(adapter.previewHelpAssetPath('/unrelated'), '/unrelated');
  const store = adapter.createPreviewHelpStore(publicHelpFetcher(requests)); assert.ok((await store.catalog()).documents.length > 0); assert.ok(requests.some(row => row.path.startsWith('/client-help/'))); assert.ok(requests.every(row => row.options.credentials === 'same-origin'));
});

test('remaining closed help contract discriminants reject malformed bounded records rather than accept inferred metadata', async () => {
  const manifest = JSON.parse(await readFile(new URL('./help/generated/manifest.json', import.meta.url)));
  const read = async descriptor => JSON.parse(await readFile(new URL('.' + descriptor.path, import.meta.url)));
  const catalog = await read(manifest.catalog); const postings = await read(manifest.search); const item = catalog.documents[0]; const article = await read(item.article);
  const clone = value => structuredClone(value);
  for (const value of [null, [], {}, { ...manifest, unexpected: 1 }, { ...manifest, contentVersion: 'bad' }, { ...manifest, authoringSchemaSha256: '0'.repeat(64) }]) assert.throws(() => contract.validateHelpManifest(value));
  for (const edit of [x => { x.documents = []; }, x => { x.documents[0].audiences = [null]; }, x => { x.documents[0].article.path = x.documents[1].article.path; }, x => { x.documents[0].summary = ''; }]) { const value = clone(catalog); edit(value); assert.throws(() => contract.validateHelpCatalog(value, manifest.contentVersion)); }
  for (const edit of [x => { x.documentIds = ['wrong']; }, x => { x.postings = []; }, x => { x.postings = { '!': [] }; }, x => { x.postings = { valid: [[0]] }; }, x => { x.postings = { valid: [[0, 1], [0, 2]] }; }, x => { x.postings = { valid: [[0, Infinity]] }; }]) { const value = clone(postings); edit(value); assert.throws(() => contract.validateHelpPostings(value, manifest.contentVersion, catalog.documents)); }
  for (const edit of [x => { x.bodyMarkdown = ''; }, x => { x.canonicalRefs = [{ repoPath: '../private', revision: 'a'.repeat(40), sha256: 'a'.repeat(64) }]; }, x => { x.canonicalRefs = [{ repoPath: '/private', revision: 'a'.repeat(40), sha256: 'a'.repeat(64) }]; }, x => { x.canonicalRefs = [{ repoPath: 'safe', revision: 'x', sha256: 'bad' }]; }]) { const value = clone(article); edit(value); assert.throws(() => contract.validateHelpArticle(value, manifest.contentVersion, item.id)); }
  assert.equal(manifest.contentVersion, helpRelease.contentVersion);
});

test('postings fallback fields, exact/prefix intersections and result caps retain deterministic public ranking', () => {
  const base = { title: 'Owned alpha', summary: '', keywords: [], audiences: [], bodyMarkdown: '' };
  const postings = buildHelpPostings(new Map([[0, base], [1, { ...base, controlFamilies: ['alpha'], title: 'beta' }]]));
  assert.equal(searchHelp(postings, 'alpha', 0).length, 1); assert.equal(searchHelp(postings, 'al').length, 2); assert.deepEqual(searchHelp(postings, 'unknown alpha'), []); assert.deepEqual(searchHelp(postings, '!!!'), []);
  assert.equal(searchHelp(postings, 'alpha', 1000).length, 2); assert.deepEqual(searchHelp({ alpha: [[0, 1]], alphabet: [[0, 5], [1, 1]] }, 'alph').map(row => row.document), [0, 1]);
});

test('payload transport grammar refuses null bodies, failed HTTP, misleading lengths and malformed evidence without a descriptor', async t => {
  browser(t); const response = (body, headers = {}, status = 200) => new Response(body, { status, headers: { 'Content-Type': 'application/json', ...headers } });
  assert.deepEqual(await contract.readHelpJson(response('{}')), {});
  for (const res of [response(null), response('{}', {}, 404), response('{}', { 'Content-Length': 'NaN' }), response('{}', { 'Content-Length': '4194305' }), response('{}', { 'Content-Type': '' }), response('{')]) await assert.rejects(contract.readHelpJson(res), /unavailable or incompatible/);
  await assert.rejects(contract.readHelpJson(response('{}'), { bytes: 3, sha256: '0'.repeat(64) }), /unavailable or incompatible/);
  for (const value of [null, [], { path: '/help/generated/article-a-' + '0'.repeat(64) + '.json', sha256: '0'.repeat(64), bytes: 0 }, { path: '/help/generated/catalog-' + '0'.repeat(64) + '.json', sha256: '0'.repeat(64), bytes: Number.MAX_SAFE_INTEGER }]) assert.throws(() => contract.validateHelpAsset(value));
});
