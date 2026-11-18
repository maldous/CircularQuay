import test from 'node:test';
import assert from 'node:assert/strict';
import { Writable } from 'node:stream';
import { once } from 'node:events';
import { handleRequest, startLandingPreview, stopLandingPreview } from './server.mjs';
class Response extends Writable { constructor() { super(); this.chunks = []; } _write(bytes, _, callback) { this.chunks.push(Buffer.from(bytes)); callback(); } writeHead(status, headers) { this.status = status; this.headers = headers; } }
async function request(url, method = 'GET') { const res = new Response(); const finished = once(res, 'finish'); await handleRequest({ url, method }, res); await finished; return res; }
test('landing handler preserves empty URL/method, malformed encoding, hash/query, exact aliases and public MIME', async () => {
  for (const [url, method, status] of [['/', undefined, 200], ['', 'GET', 200], ['/%ZZ', 'GET', 400], ['/?x=1', 'GET', 400], ['/#fragment', 'GET', 400], ['/README.md', 'GET', 404], ['/', 'POST', 405], ['/', '', 405]]) { const res = await request(url, method); assert.equal(res.status, status); assert.equal(res.headers['X-Frame-Options'], 'DENY'); }
  assert.equal((await request('/ads.txt')).headers['Content-Type'], 'text/plain; charset=utf-8'); assert.equal((await request('/about', 'HEAD')).chunks.length, 0); assert.equal((await request('/web/help.js', 'HEAD')).status, 200);
});
test('owned ephemeral landing listener is closed normally, not a product or provider service', async t => {
  const server = startLandingPreview({ port: 0 }); t.after(() => server.close()); if (!server.listening) await once(server, 'listening'); assert.equal(server.address().address, '127.0.0.1'); assert.ok(server.address().port > 0); await new Promise(resolve => server.close(resolve));
});
test('landing signal lifecycle retains exact close/exit semantics and cancels its deadline on successful close', () => {
  let deadline, close, cancelled, unrefs = 0; const exits = [], writes = []; const timer = { unref() { unrefs++; } };
  stopLandingPreview({ close(callback) { close = callback; } }, 'SIGTERM', { exit: code => exits.push(code), schedule: (callback, milliseconds) => { assert.equal(milliseconds, 5000); deadline = callback; return timer; }, cancel: value => { cancelled = value; }, write: text => writes.push(text) });
  assert.equal(unrefs, 1); assert.deepEqual(exits, []); assert.match(writes[0], /Received SIGTERM/); close(); assert.equal(cancelled, timer); assert.deepEqual(exits, [0]);
  exits.length = 0; stopLandingPreview({ close() {} }, 'SIGINT', { exit: code => exits.push(code), schedule: callback => { deadline = callback; return timer; }, write() {} }); deadline(); assert.deepEqual(exits, [1]);
});

test('closed route construction retains the exact accepted902-route graph without publishing invalid component inputs', async () => {
  const { assets, integrationAssetEntries, documentationAssetEntries, helpAssetEntries } = await import('./server.mjs');
  const { createHash } = await import('node:crypto');
  const { readFile } = await import('node:fs/promises');
  const golden = '9172a71d3c23106bf178a5597960f2dad6ad97903eeeed5cf3cd5b632b093b0a';
  const before = createHash('sha256').update(JSON.stringify([...assets])).digest('hex');
  assert.equal(assets.size, 902); assert.equal(before, golden);
  const catalog = JSON.parse(await readFile(new URL('./integrations.json', import.meta.url)));
  const entries = integrationAssetEntries(catalog);
  assert.equal(entries.length, 113);
  for (const [route, file] of entries) assert.equal(assets.get(route), file);
  const docs = JSON.parse(await readFile(new URL('./explore-content/manifest.json', import.meta.url)));
  for (const [route, file] of documentationAssetEntries(docs)) assert.equal(assets.get(route), file);
  const manifest = JSON.parse(await readFile(new URL('./client-help/generated/manifest.json', import.meta.url)));
  assert.deepEqual(helpAssetEntries([manifest.catalog.path]), [[manifest.catalog.path, `client-help/generated/${manifest.catalog.path.split('/').pop()}`], [`/client-help/generated/${manifest.catalog.path.split('/').pop()}`, `client-help/generated/${manifest.catalog.path.split('/').pop()}`]]);
  // Component-only malformed metadata: no global Map mutation, request path
  // injection, forged descriptor/pin or claim that an invalid route is public.
  assert.deepEqual(integrationAssetEntries({ groups: [{ items: [{ asset: null }] }] }), []);
  for (const asset of ['../private', '/absolute.svg', 'assets/integrations/UPPER.svg']) assert.throws(() => integrationAssetEntries({ groups: [{ items: [{ asset }] }] }), /Invalid public integration asset/);
  for (const path of ['../private', '/absolute', 'explore-content/assets/not-a-hash.svg']) assert.throws(() => documentationAssetEntries({ publicFiles: [{ path }] }), /Invalid public documentation asset/);
  for (const path of ['../private', '/help/source-index.json', '/help/generated/article-bad.json']) assert.throws(() => helpAssetEntries([path]), /Invalid public help asset/);
  assert.equal(createHash('sha256').update(JSON.stringify([...assets])).digest('hex'), before);
});
