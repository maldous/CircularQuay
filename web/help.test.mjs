import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { createHelpStore } from './help-store.js';
import { readHelpJson, validateHelpManifest, validateHelpCatalog, validateHelpPostings, validateHelpArticle } from './help-contract.js';
import { helpMarkdown } from './help.js';

const root = new URL('.', import.meta.url);
const hash = value => createHash('sha256').update(value).digest('hex');
async function assets() {
  const manifest = JSON.parse(await readFile(new URL('help/generated/manifest.json', root)));
  const catalog = JSON.parse(await readFile(new URL(manifest.catalog.path.slice(1), root)));
  const index = JSON.parse(await readFile(new URL(manifest.search.path.slice(1), root)));
  return { manifest, catalog, index };
}
function fileFetcher(requests, transform = bytes => bytes) {
  return async (path, options) => {
    requests.push({ path, options });
    assert.match(path, /^\/help\/generated\/[a-z0-9.-]+\.json$/);
    const bytes = await readFile(new URL(path.slice(1), root));
    return new Response(transform(bytes, path), { headers: { 'Content-Type': 'application/json' } });
  };
}

test('help loads catalog first, full-text search once and articles on demand without query transport', async () => {
  const requests = [];
  const store = createHelpStore(fileFetcher(requests));
  assert.equal(requests.length, 0);
  const catalog = await store.catalog();
  assert.equal(requests.length, 2);
  assert.ok(requests.every(request => !/article-|search-/.test(request.path)));
  const searches = await Promise.all(Array.from({ length: 20 }, () => store.search('config create')));
  assert.equal(requests.filter(request => /\/search-/.test(request.path)).length, 1);
  assert.equal(searches[0][0].id, 'crud');
  assert.equal((await store.search('TypeScript'))[0].id, 'sdk-ts');
  assert.equal((await store.search('load example'))[0].id, 'policy');
  assert.equal((await store.search('undo'))[0].id, 'policy');
  assert.ok((await store.search('MCP')).some(result => result.id === 'mcp'));
  const beforeUnknown = requests.length;
  assert.deepEqual(await store.search('SYNTHETIC_QUERY_PRIVATE_691E3A'), []);
  assert.deepEqual(await store.search(' '), []);
  assert.equal(requests.length, beforeUnknown);
  const [first, second] = await Promise.all([store.article('policy'), store.article('policy')]);
  assert.equal(first.id, 'policy'); assert.equal(first, second);
  await store.article('policy');
  assert.equal(requests.filter(request => /article-policy-/.test(request.path)).length, 1);
  assert.ok(requests.every(request => !request.path.includes('?') && !request.path.includes('SYNTHETIC_QUERY_PRIVATE')));
  assert.ok(requests.every(request => !request.options.body && !request.options.method));
  assert.equal(catalog.documents.length, 333);
});

test('closed runtime shapes reject unknown, unsafe, mixed-version and mismatched-document artifacts', async () => {
  const { manifest, catalog, index } = await assets();
  validateHelpManifest(manifest); validateHelpCatalog(catalog, manifest.contentVersion); validateHelpPostings(index, manifest.contentVersion, catalog.documents);
  for (const mutate of [value => { value.schemaVersion = 2; }, value => { value.extra = true; }, value => { value.catalog.path = 'https://example.invalid/search'; }, value => { value.catalog.bytes = Infinity; }]) {
    const value = structuredClone(manifest); mutate(value); assert.throws(() => validateHelpManifest(value));
  }
  const duplicate = structuredClone(catalog); duplicate.documents.push(duplicate.documents[0]); assert.throws(() => validateHelpCatalog(duplicate, manifest.contentVersion));
  const unknown = structuredClone(catalog); unknown.documents[0].privateMetadata = 'not public'; assert.throws(() => validateHelpCatalog(unknown, manifest.contentVersion));
  assert.throws(() => validateHelpCatalog(catalog, `sha256:${'0'.repeat(64)}`));
  const reordered = structuredClone(index); reordered.documentIds.reverse(); assert.throws(() => validateHelpPostings(reordered, manifest.contentVersion, catalog.documents));
  const badPosting = structuredClone(index); badPosting.postings['malformed'] = [[100000, 1]]; assert.throws(() => validateHelpPostings(badPosting, manifest.contentVersion, catalog.documents));
  const item = catalog.documents[0]; const article = JSON.parse(await readFile(new URL(item.article.path.slice(1), root)));
  validateHelpArticle(article, manifest.contentVersion, item.id);
  article.canonicalRefs[0].repoPath = '../private/evidence.json'; assert.throws(() => validateHelpArticle(article, manifest.contentVersion, item.id));
});

test('corrupt help is rejected before use, failed loads can retry and workspace objects are not inputs', async () => {
  const requests = [];
  let corrupt = true;
  const fetcher = fileFetcher(requests, (bytes, path) => {
    if (corrupt && /\/catalog-/.test(path)) { corrupt = false; const altered = Buffer.from(bytes); altered[0] = 32; return altered; }
    return bytes;
  });
  const store = createHelpStore(fetcher);
  await assert.rejects(store.catalog(), /unavailable or incompatible/);
  assert.equal((await store.catalog()).documents.length, 333);
  assert.equal(requests.filter(request => /\/catalog-/.test(request.path)).length, 2);
  assert.ok(requests.every(request => !request.path.includes('/local/') && !request.path.includes('/api/')));
});

test('help payload reader enforces byte bounds, hash and valid UTF-8/JSON before returning content', async () => {
  let cancelled = false;
  const response = new Response(new ReadableStream({ start(controller) { controller.enqueue(new Uint8Array(10)); }, cancel() { cancelled = true; } }), { headers: { 'Content-Type': 'application/json' } });
  await assert.rejects(readHelpJson(response, { bytes: 5, sha256: '0'.repeat(64) })); assert.equal(cancelled, true);
  for (const bytes of [Buffer.from('{bad'), Buffer.from([0xff])]) {
    await assert.rejects(readHelpJson(new Response(bytes, { headers: { 'Content-Type': 'application/json' } }), { bytes: bytes.length, sha256: hash(bytes) }));
  }
  await assert.rejects(readHelpJson(new Response('{}', { headers: { 'Content-Type': 'text/html' } })));
});

test('help Markdown renders hostile HTML, links and code as inert text', () => {
  const source = '## Read safely\n\n<img src=x onerror=alert(1)>\n\n[Open](javascript:alert(1))\n\n```html\n<script>danger()</script>\n```';
  const rendered = helpMarkdown(source);
  assert.equal(rendered.includes('<img'), false); assert.equal(rendered.includes('<script>'), false); assert.equal(rendered.includes('href='), false);
  assert.match(rendered, /&lt;img/); assert.match(rendered, /&lt;script&gt;/); assert.match(rendered, /<pre tabindex="0"><code>/);
});

test('help procedures retain semantic lists and article heading hierarchy', () => {
  const rendered = helpMarkdown('## Procedure\n\n1. Inspect first.\n2. Save second.\n\n- Keep the draft.\n- Retry safely.\n\n### Recovery\n\n4. Resume the next step.');
  assert.match(rendered, /<h3>Procedure<\/h3>/); assert.match(rendered, /<h4>Recovery<\/h4>/);
  assert.match(rendered, /<ol start="1"><li>Inspect first\.<\/li><li>Save second\.<\/li><\/ol>/);
  assert.match(rendered, /<ul><li>Keep the draft\.<\/li>/); assert.match(rendered, /<ol start="4">/);
});

test('public control identifiers and body-only SDK/CLI terms remain searchable', async () => {
  const requests = []; const store = createHelpStore(fileFetcher(requests));
  const source = JSON.parse(await readFile(new URL('help/source-index.json', root)));
  let bindings = 0;
  for (const entry of source.entries) for (const family of entry.controlFamilies) {
    const results = await store.search(family);
    const dedicated = source.entries.find(item => item.id.startsWith('control-') && item.controlFamilies.includes(family));
    assert.equal(results.some(result => result.id === (dedicated?.id || entry.id)), true, family);
    bindings += 1;
  }
  assert.ok(bindings >= 317);
  assert.equal((await store.search('timeout seconds')).some(result => result.id === 'operations'), true);
  assert.equal((await store.search('expect sha256')).some(result => result.id === 'crud'), true);
});

test('static help compression preserves verified bytes, immutable caching and identity fallback', async () => {
  const { createPreview } = await import('./server.mjs');
  const http = await import('node:http');
  const { brotliDecompressSync } = await import('node:zlib');
  const { manifest } = await assets();
  const server = createPreview('remote');
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const request = encoding => new Promise((resolve, reject) => {
    http.get({ host: '127.0.0.1', port: server.address().port, path: manifest.search.path, headers: { 'Accept-Encoding': encoding } }, response => {
      const chunks = []; response.on('data', chunk => chunks.push(chunk));
      response.on('end', () => resolve({ status: response.statusCode, headers: response.headers, bytes: Buffer.concat(chunks) }));
    }).on('error', reject);
  });
  try {
    const compressed = await request('gzip, br');
    assert.equal(compressed.status, 200); assert.equal(compressed.headers['content-encoding'], 'br');
    assert.equal(compressed.headers.vary, 'Accept-Encoding');
    assert.equal(compressed.headers['cache-control'], 'public, max-age=31536000, immutable');
    const decoded = brotliDecompressSync(compressed.bytes);
    assert.equal(hash(decoded), manifest.search.sha256); assert.equal(decoded.length, manifest.search.bytes);
    assert.ok(compressed.bytes.length < decoded.length / 2);
    for (const encoding of ['identity', 'br;q=0', 'br;q=invalid']) {
      const identity = await request(encoding); assert.equal(identity.headers['content-encoding'], undefined); assert.deepEqual(identity.bytes, decoded);
    }
  } finally { await new Promise(resolve => server.close(resolve)); }
});

test('Arabic local search normalizes optional marks and letter forms without changing displayed content', async () => {
  const { buildHelpPostings, searchHelp } = await import('./help-search.js');
  const article = { title: 'إعدادات الأمان', summary: 'دليل التهيئة', bodyMarkdown: 'يُحمَّل المحتوى محلياً', keywords: [], audiences: [], controlFamilies: [] };
  const postings = buildHelpPostings([article]);
  for (const query of ['اعدادات الامان', 'إِعْدَادَات الأَمَان', 'المحتوى محليا', 'المحتوي محلياً']) assert.equal(searchHelp(postings, query)[0]?.document, 0, query);
  assert.equal(article.title, 'إعدادات الأمان');
});

test('a cold search starts catalog and postings in parallel, with no query transmission', { timeout: 2000 }, async () => {
  const requests = []; let releaseCatalog;
  const blocked = new Promise(resolve => { releaseCatalog = resolve; });
  const fetcher = fileFetcher(requests);
  const store = createHelpStore(async (path, options) => { const response = await fetcher(path, options); if (/\/catalog-/.test(path)) await blocked; return response; });
  const pending = store.search('TypeScript');
  // Drain manifest/file I/O without a timing-based assertion.
  while (!requests.some(request => /\/search-/.test(request.path))) await new Promise(resolve => setImmediate(resolve));
  assert.ok(requests.some(request => /\/catalog-/.test(request.path)));
  releaseCatalog();
  assert.equal((await pending)[0].id, 'sdk-ts');
});

test('all schema help IDs resolve from schema patterns without including private map keys', async () => {
  const { publicHelpPointer, schemaInventory } = await import('./local/schema-tools.js');
  const { schemaHelpTopic } = await import('./help-launcher.js');
  const schema = JSON.parse(await readFile(new URL('local/policy-schema.json', root)));
  const source = JSON.parse(await readFile(new URL('help/source-index.json', root)));
  for (const item of schemaInventory(schema)) {
    const concrete = item.path === '/' ? '' : item.path.replaceAll('{id}', 'PRIVATE_MAP_KEY_691E3').replaceAll('*', '0');
    const pointer = publicHelpPointer(schema, concrete);
    assert.equal(pointer, item.path);
    assert.equal(pointer.includes('PRIVATE_MAP_KEY'), false);
    const id = await schemaHelpTopic(pointer);
    assert.ok(source.entries.some(entry => entry.id === id), pointer);
  }
  assert.equal(publicHelpPointer(schema, '/sources/private~1key/type'), '/sources/{id}/type');
  assert.equal(publicHelpPointer(schema, '/sources/private~2key/type'), null);
  assert.equal(publicHelpPointer(schema, '/unrecognized_PRIVATE_MAP_KEY'), null);
  assert.equal(await schemaHelpTopic('/sources/PRIVATE_MAP_KEY/type'), null);
});

async function replacementHelpGraph(contentVersion) {
  const { buildHelpPostings } = await import('./help-search.js');
  const source = JSON.parse(await readFile(new URL('help/source-index.json', root)));
  for (const field of ['title', 'summary', 'bodyMarkdown']) source.entries[0][field] = 'X' + source.entries[0][field].slice(1);
  const files = new Map();
  const emit = (name, value) => {
    const bytes = Buffer.from(JSON.stringify(value)); const sha256 = hash(bytes);
    const path = `/help/generated/${name}-${sha256}.json`;
    files.set(path, bytes); return { path, sha256, bytes: bytes.length };
  };
  const documents = source.entries.map(entry => ({ id: entry.id, title: entry.title, summary: entry.summary, audiences: entry.audiences, article: emit(`article-${entry.id}`, { contentVersion, id: entry.id, title: entry.title, bodyMarkdown: entry.bodyMarkdown, capabilityStatus: entry.capabilityStatus, canonicalRefs: entry.canonicalRefs }) }));
  const catalog = emit('catalog', { contentVersion, documents });
  const search = emit('search', { contentVersion, documentIds: source.entries.map(entry => entry.id), postings: buildHelpPostings(source.entries) });
  const current = (await assets()).manifest;
  const manifest = { ...current, contentVersion, catalog, search };
  files.set('/help/generated/manifest.json', Buffer.from(JSON.stringify(manifest)));
  return { files, manifest, documents };
}

test('the bundled release pins the approved source, exact manifest bytes and distribution version', async () => {
  const { helpRelease } = await import('./help-release.js');
  const source = await readFile(new URL('help/source-index.json', root));
  const manifestBytes = await readFile(new URL('help/generated/manifest.json', root));
  assert.equal(hash(source), 'e0c00966bf7781d52c22e3a1a9f1b7da79fdf061764100da2dab97adf7803124');
  assert.ok(source.byteLength < 1024 * 1024, 'Reviewed public help source must stay below the existing first-party source identity cap.');
  assert.equal(helpRelease.sourceSha256, hash(source));
  assert.equal(helpRelease.contentVersion, `sha256:${hash(source)}`);
  assert.equal(helpRelease.manifest.bytes, manifestBytes.length);
  assert.equal(helpRelease.manifest.sha256, hash(manifestBytes));
  assert.ok(Object.isFrozen(helpRelease)); assert.ok(Object.isFrozen(helpRelease.manifest));
  const decoded = JSON.parse(manifestBytes);
  const reordered = Buffer.from(JSON.stringify(Object.fromEntries(Object.entries(decoded).reverse())));
  assert.deepEqual(JSON.parse(reordered), decoded); assert.equal(reordered.length, manifestBytes.length);
  const requests = [];
  const store = createHelpStore(async path => { requests.push(path); return new Response(reordered, { headers: { 'Content-Type': 'application/json' } }); });
  await assert.rejects(store.catalog(), /unavailable or incompatible/);
  assert.deepEqual(requests, ['/help/generated/manifest.json']);
});

for (const preserveSourceLabel of [true, false]) test(`a self-consistent substituted help graph is rejected before catalog use (${preserveSourceLabel ? 'copied source label' : 'new source label'})`, async () => {
  const { helpRelease } = await import('./help-release.js');
  const version = preserveSourceLabel ? helpRelease.contentVersion : `sha256:${'0'.repeat(64)}`;
  const { files, manifest, documents } = await replacementHelpGraph(version);
  const manifestBytes = files.get('/help/generated/manifest.json');
  // The attack matches the pinned size and supplies matching subordinate hashes;
  // neither the stream bound nor an internally consistent graph is enough.
  assert.equal(manifestBytes.length, helpRelease.manifest.bytes);
  assert.notEqual(hash(manifestBytes), helpRelease.manifest.sha256);
  validateHelpCatalog(JSON.parse(files.get(manifest.catalog.path)), version);
  validateHelpPostings(JSON.parse(files.get(manifest.search.path)), version, documents);
  validateHelpArticle(JSON.parse(files.get(documents[0].article.path)), version, documents[0].id);
  for (const descriptor of [manifest.catalog, manifest.search, documents[0].article]) {
    assert.equal(hash(files.get(descriptor.path)), descriptor.sha256);
    assert.equal(files.get(descriptor.path).length, descriptor.bytes);
  }
  if (!preserveSourceLabel) assert.throws(() => validateHelpManifest(manifest));
  const requests = []; let substitute = true;
  const approved = fileFetcher([]);
  const store = createHelpStore(async (path, options) => {
    requests.push(path);
    return substitute ? new Response(files.get(path), { headers: { 'Content-Type': 'application/json' } }) : approved(path, options);
  });
  await assert.rejects(store.catalog(), /unavailable or incompatible/);
  assert.deepEqual(requests, ['/help/generated/manifest.json']);
  substitute = false;
  assert.equal((await store.catalog()).documents[0].title, 'Install and first run');
  assert.equal(requests.length, 3);
});


test('compact public help stays below the existing source cap with approved parsed semantics', async () => {
  const bytes = await readFile(new URL('help/source-index.json', root));
  const value = JSON.parse(bytes);
  const canonical = item => Array.isArray(item) ? item.map(canonical) : item && typeof item === 'object' ? Object.fromEntries(Object.keys(item).sort().map(key => [key, canonical(item[key])])) : item;
  assert.ok(bytes.length < 1048576);
  assert.equal(value.entries.length, 333);
  assert.equal(value.contentVersion, 'sha256:5162f1e0d999e62af1b5bad6e033e7618d2ddddaaceca8c8e944c794979caa63');
  assert.equal(hash(Buffer.from(JSON.stringify(canonical(value)))), 'e342234a10a5a55292ca40b3bbe8398f28616907d7249c081f3bdeb9cc197716');
});
