import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import test from 'node:test';
import { readSitePage, SITE_PAGE_FILES } from './render-site-header.mjs';

const here = resolve(import.meta.dirname);
const publicDirectory = resolve(here, '.public-dist');
execFileSync(process.execPath, ['build-public.mjs'], { cwd: here, stdio: 'inherit' });
const { default: worker } = await import('./worker.mjs');

const assetRequests = [];
const assetBinding = {
  async fetch(request) {
    assetRequests.push(request);
    const url = new URL(request.url);
    const file = resolve(publicDirectory, `.${decodeURIComponent(url.pathname)}`);
    if (!file.startsWith(`${publicDirectory}/`)) return new Response('Not Found', { status: 404 });
    try {
      const bytes = await readFile(file);
      return new Response(request.method === 'HEAD' ? null : bytes, { status: 200, headers: { 'Content-Length': String(bytes.length) } });
    } catch {
      return new Response('Not Found', { status: 404 });
    }
  },
};

const dispatch = (path, init = {}) => worker.fetch(new Request(`https://circularquay.aldous.info${path}`, init), { ASSETS: assetBinding });

test('serves only the canonical allowlist with the local server headers', async () => {
  const response = await dispatch('/');
  assert.equal(response.status, 200);
  assert.match(response.headers.get('content-security-policy'), /default-src 'self'/);
  assert.equal(response.headers.get('x-content-type-options'), 'nosniff');
  assert.equal(response.headers.get('x-robots-tag'), 'noindex, nofollow, noarchive');
  assert.equal(response.headers.get('content-type'), 'text/html; charset=utf-8');
});

test('aliases preserve bytes and retain the source MIME type', async () => {
  const [shortRoute, htmlRoute] = await Promise.all([dispatch('/about'), dispatch('/about.html')]);
  assert.equal(shortRoute.status, 200);
  assert.equal(htmlRoute.status, 200);
  assert.deepEqual(new Uint8Array(await shortRoute.arrayBuffer()), new Uint8Array(await htmlRoute.arrayBuffer()));
  assert.equal(shortRoute.headers.get('content-type'), 'text/html; charset=utf-8');
});

test('decoded allowlisted paths rewrite to their canonical public asset', async () => {
  const response = await dispatch('/%6cogin');
  assert.equal(response.status, 200);
  assert.equal(response.headers.get('content-type'), 'text/html; charset=utf-8');
});

test('does not forward visitor headers to the private asset binding', async () => {
  assetRequests.length = 0;
  const response = await dispatch('/login', {
    headers: {
      Authorization: 'Bearer should-not-reach-assets',
      Cookie: 'preview=private',
      'If-None-Match': 'stale',
      Range: 'bytes=0-7',
    },
  });
  assert.equal(response.status, 200);
  const forwarded = assetRequests.at(-1);
  assert.equal(forwarded.method, 'GET');
  for (const name of ['authorization', 'cookie', 'if-none-match', 'range']) {
    assert.equal(forwarded.headers.get(name), null);
  }
});

test('HEAD has headers but no body', async () => {
  const response = await dispatch('/motion-catalog.js', { method: 'HEAD' });
  assert.equal(response.status, 200);
  assert.equal(response.headers.get('content-type'), 'text/javascript; charset=utf-8');
  assert.equal(await response.text(), '');
});

test('rejects methods, query strings, and private source paths', async () => {
  assert.equal((await dispatch('/', { method: 'POST' })).status, 405);
  assert.equal((await dispatch('/?q=preview')).status, 400);
  assert.equal((await dispatch('/README.md')).status, 404);
  assert.equal((await dispatch('/integrations.json')).status, 404);
  assert.equal((await dispatch('/explore-source/README.md')).status, 404);
  assert.equal((await dispatch('/explore-content/manifest.json')).status, 404);
  assert.equal((await dispatch('/evidence/review.json')).status, 404);
  assert.equal((await dispatch('/%')).status, 400);
});


const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
const compareCodeUnits = (left, right) => left < right ? -1 : left > right ? 1 : 0;

test('rendered pages, export manifest and Worker retain exact public parity', async () => {
  const manifest = JSON.parse(await readFile(resolve(here, '.local/public-upload-manifest.json'), 'utf8'));
  assert.equal(manifest.schemaVersion, 1);
  assert.equal(manifest.routeCount, 902);
  assert.equal(manifest.fileCount, 554);
  const routes = manifest.routes.map(({ route, file, mime }) => ({ route, file, mime }))
    .sort((left, right) => compareCodeUnits(left.route, right.route));
  assert.equal(sha256(JSON.stringify(routes)), '6bbc8e3d19bd0ff586303b560f18caf9053e1d741d03563757c382cdfe321e3e');
  const names = manifest.files.map(item => item.file).sort(compareCodeUnits);
  assert.equal(sha256(JSON.stringify(names)), '56a1aa88f034de3994b5a4fefcc670b72d15297be0ae82d12bd7b891b1d6ff48');
  for (const name of SITE_PAGE_FILES) {
    const expected = await readSitePage(name);
    const exported = await readFile(resolve(publicDirectory, name));
    assert.deepEqual(exported, expected, name);
    const recorded = manifest.files.find(item => item.file === name);
    assert.ok(recorded, name);
    assert.equal(recorded.bytes, expected.length, name);
    assert.equal(recorded.sha256, sha256(expected), name);
    const response = await dispatch(`/${name}`);
    assert.equal(response.status, 200, name);
    assert.equal(response.headers.get('content-length'), String(expected.length), name);
    assert.equal(response.headers.get('content-type'), 'text/html; charset=utf-8', name);
    assert.deepEqual(Buffer.from(await response.arrayBuffer()), expected, name);
    const head = await dispatch(`/${name}`, { method: 'HEAD' });
    assert.equal(head.status, 200, name);
    assert.equal(head.headers.get('content-length'), String(expected.length), name);
    assert.equal(await head.text(), '', name);
  }
  for (const privatePath of ['/site-header.html', '/render-site-header.mjs', '/render-site-header.test.mjs']) {
    assert.equal((await dispatch(privatePath)).status, 404, privatePath);
  }
});


test('Worker malformed URLs and missing private assets return fixed nonpromoting refusals', async () => {
  const missing = { ASSETS: { async fetch() { return new Response('missing', { status: 404 }); } } };
  assert.equal((await worker.fetch({ method: 'GET', url: 'not a URL' }, missing)).status, 400);
  assert.equal((await worker.fetch(new Request('https://preview.invalid/'), missing)).status, 404);
  assert.equal((await worker.fetch(new Request('https://preview.invalid/#fragment'), missing)).status, 400);
});
