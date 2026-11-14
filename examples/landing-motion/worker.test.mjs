import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import test from 'node:test';

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
      return new Response(request.method === 'HEAD' ? null : bytes, { status: 200 });
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
