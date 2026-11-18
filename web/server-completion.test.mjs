import test from 'node:test';
import assert from 'node:assert/strict';
import { EventEmitter } from 'node:events';
import { Writable } from 'node:stream';
import { mkdtemp, writeFile, mkdir, rm, symlink } from 'node:fs/promises';
import { brotliDecompressSync } from 'node:zlib';
import os from 'node:os';
import path from 'node:path';
import { createPreview, requestURL, reloadablePath } from './server.mjs';
class Response extends Writable { constructor() { super(); this.chunks = []; } _write(bytes, _encoding, callback) { this.chunks.push(Buffer.from(bytes)); callback(); } writeHead(status, headers) { this.status = status; this.headers = headers; } get body() { return Buffer.concat(this.chunks); } }
async function owned(t) { const root = await mkdtemp(path.join(os.tmpdir(), 'cq-preview-owned-')); t.after(() => rm(root, { recursive: true, force: true })); await mkdir(path.join(root, 'local')); return root; }
async function request(server, url = '/', method = 'GET', headers = {}) { const req = Object.assign(new EventEmitter(), { url, method, headers }); const res = new Response(); await server.listeners('request')[0](req, res); return { req, res }; }

test('preview request syntax and reload filters reject ambiguous values without a network listener', () => {
  for (const value of [null, 7, 'relative', '//outside', '/path\\file', '/path?secret', '/path#fragment', '/bad%z0']) assert.equal(requestURL(value), null);
  assert.equal(requestURL('/ui.js').pathname, '/ui.js'); for (const value of ['output/owned.js', 'README.md', 'own.JS']) assert.equal(reloadablePath(value), false); assert.equal(reloadablePath('local\\owned.json'), true); assert.throws(() => createPreview('unknown'), /preview plane/);
});
test('actual preview request handler serves only selected plane bytes and fixed security headers, including compression grammar', async t => {
  const root = await owned(t); const bytes = Buffer.from('export const synthetic = "' + 'owned'.repeat(1000) + '";'); await writeFile(path.join(root, 'ui.js'), bytes); await writeFile(path.join(root, 'local/entry.js'), 'export {};');
  const local = createPreview('local', root), remote = createPreview('remote', root); t.after(() => { local.close(); remote.close(); });
  for (const header of ['br;q=0', 'br;q=0.000', 'br;q=2', 'br;q=1.1', 'br;q=0.0001', 'br;bad=1', 'br;q=0.5;other=1', 'gzip', 'br;q=.5']) { const { res } = await request(local, '/ui.js', 'GET', { 'accept-encoding': header }); assert.equal(res.status, 200); assert.equal(res.headers['Content-Encoding'], undefined); assert.deepEqual(res.body, bytes); }
  for (const header of ['br', 'BR;Q=1', 'gzip, br;q=0.001', 'br;q=1.000']) { const { res } = await request(local, '/ui.js', 'GET', { 'accept-encoding': header }); assert.equal(res.headers['Content-Encoding'], 'br'); assert.deepEqual(brotliDecompressSync(res.body), bytes); }
  assert.equal((await request(local, '/local/entry.js')).res.status, 200); assert.equal((await request(remote, '/local/entry.js')).res.status, 404); assert.equal((await request(local, '/ui.js', 'HEAD')).res.body.length, 0); assert.equal((await request(local, '/ui.js', 'POST')).res.status, 405); assert.equal((await request(local, '/ui.js?x')).res.status, 400);
  for (const plane of [local, remote]) { const { res } = await request(plane); assert.equal(res.status, 200); assert.match(res.body.toString(), /synthetic|demo|client/); assert.match(res.headers['Content-Security-Policy'], /form-action 'none'/); }
});
test('actual protected owned-file paths reject missing/nonfiles/linked parents/oversize instead of exposing other data', async t => {
  const root = await owned(t); const server = createPreview('local', root); t.after(() => server.close()); assert.equal((await request(server, '/ui.js')).res.status, 404); await mkdir(path.join(root, 'ui.js')); assert.equal((await request(server, '/ui.js')).res.status, 404); await rm(path.join(root, 'ui.js'), { recursive: true });
  await writeFile(path.join(root, 'ui.js'), Buffer.alloc(4 * 1024 * 1024 + 1)); assert.equal((await request(server, '/ui.js')).res.status, 404);
  const other = await owned(t); await writeFile(path.join(other, 'entry.js'), 'OWNED_NOT_EXPOSED'); await rm(path.join(root, 'local'), { recursive: true }); await symlink(other, path.join(root, 'local'), 'junction'); assert.equal((await request(server, '/local/entry.js')).res.status, 404);
});
test('reload SSE bound is exact, HEAD never subscribes and closed originals release capacity', async t => {
  const root = await owned(t); const server = createPreview('local', root); t.after(() => server.close()); const active = [];
  try { assert.equal((await request(server, '/__reload', 'HEAD')).res.status, 200); for (let n = 0; n < 32; n++) { const row = await request(server, '/__reload'); assert.equal(row.res.status, 200); active.push(row); } assert.equal((await request(server, '/__reload')).res.status, 503); active[0].req.emit('close'); active[0].res.end(); const next = await request(server, '/__reload'); assert.equal(next.res.status, 200); active.push(next); }
  finally { for (const row of active) { row.req.emit('close'); row.res.end(); } }
});

test('owned development watcher keeps original two-plane lifecycle, filters output changes and coalesces reloads', async t => {
  const root = await owned(t); const { startDevelopmentPreview } = await import('./server.mjs'); const preview = startDevelopmentPreview({ sourceRoot: root, ports: [0, 0] }); t.after(() => preview.close());
  for (const server of preview.servers) if (!server.listening) await new Promise(resolve => server.once('listening', resolve));
  const original = await request(preview.servers[0], '/__reload'); const before = original.res.body.toString();
  preview.watcher.emit('change', 'change', 'output/ignored.js'); preview.watcher.emit('change', 'change', 'README.md');
  preview.watcher.emit('change', 'change', 'owned.js'); preview.watcher.emit('change', 'change', 'local/owned.json');
  await new Promise(resolve => setTimeout(resolve, 170)); assert.equal(original.res.body.toString(), before + 'data: reload\n\n');
  original.req.emit('close'); original.res.end(); preview.close(); preview.close();
});

test('default preview CLI serves both declared loopback ports then closes its real watcher and servers', async t => {
  const { fileURLToPath } = await import('node:url');
  const { runCli, assertCliPassed, previewPreload } = await import('./cli-process-fixture.test.mjs');
  const preload = await previewPreload(t, [5180, 5181]);
  const result = await runCli(fileURLToPath(new URL('./server.mjs', import.meta.url)), [], { cwd: fileURLToPath(new URL('.', import.meta.url)), preload, timeout: 30_000 });
  assertCliPassed(result, 'default development preview CLI');
  assert.match(result.stdout, /Account\/operator synthetic preview: http:\/\/127\.0\.0\.1:5180/);
  assert.match(result.stdout, /LOCAL client synthetic preview: http:\/\/127\.0\.0\.1:5181/);
  assert.match(result.stdout, /Owned default preview served all fixed ports/);
});

test('real landing public paths use native containment on every declared web platform without opening a listener', async () => {
  const landing = await import('../examples/landing-motion/server.mjs');
  for (const [url, expected, body] of [['/style.css', 200, ''], ['/web/help.js', 200, ''], ['/README.md', 404, 'Not Found']]) {
    const response = new Response();
    const finished = new Promise(resolve => response.once('finish', resolve));
    await landing.handleRequest({ method: 'HEAD', url }, response);
    await finished;
    assert.equal(response.status, expected);
    // The direct Writable preserves reply()'s error body; only successful HEAD
    // responses take the handler's empty-body branch.
    assert.deepEqual(response.body, Buffer.from(body));
    assert.equal(response.headers['X-Frame-Options'], 'DENY');
  }
});

test('asset identity component rejects a real owned file replacement without pretending it is an HTTP race', async t => {
  const { lstat, rename } = await import('node:fs/promises');
  const { matchesAssetIdentityRecord, isContainedAssetPath } = await import('./server.mjs');
  const root = await owned(t), file = path.join(root, 'ui.js'), held = path.join(root, 'held-owned.js');
  await writeFile(file, 'owned-original');
  const original = await lstat(file, { bigint: true });
  assert.equal(matchesAssetIdentityRecord(original, original), true);
  await rename(file, held); await writeFile(file, 'owned-replaced');
  const replacement = await lstat(file, { bigint: true });
  assert.equal(original.size, replacement.size);
  assert.equal(matchesAssetIdentityRecord(original, replacement), false);
  assert.equal(await import('node:fs/promises').then(fs => fs.readFile(held, 'utf8')), 'owned-original');
  assert.equal(isContainedAssetPath(root, file), true);
  assert.equal(isContainedAssetPath(root, root), false);
  assert.equal(isContainedAssetPath(root, path.dirname(root)), false);
  assert.equal(isContainedAssetPath(root, root + '-sibling'), false);
});

test('identity and read-count component contracts reject malformed typed values without fabricating native Stats or read results', async () => {
  const { matchesAssetIdentityRecord, isValidAssetReadCount } = await import('./server.mjs');
  // Plain identity DTO fields only, no isFile/stat/read mock and no claim that
  // a native filesystem returned these malformed inode values.
  const record = { dev: 1n, ino: 2n };
  for (const ino of [undefined, 0n, -1n, 2, '2']) {
    assert.equal(matchesAssetIdentityRecord(record, { dev: 1n, ino }), false);
    assert.equal(matchesAssetIdentityRecord({ dev: 1n, ino }, record), false);
  }
  assert.equal(matchesAssetIdentityRecord(record, { dev: 3n, ino: 2n }), false);
  assert.equal(matchesAssetIdentityRecord(record, { dev: 1n, ino: 3n }), false);
  for (const value of [-1, NaN, Infinity, 0.5, '1', 1n, undefined]) assert.equal(isValidAssetReadCount(value), false);
  for (const value of [0, 1, Number.MAX_SAFE_INTEGER]) assert.equal(isValidAssetReadCount(value), true);
});
