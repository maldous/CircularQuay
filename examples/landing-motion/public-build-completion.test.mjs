import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, readFile, rm, mkdir } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import path from 'node:path';
import os from 'node:os';
import { buildPublic, isPublicBuildEntry } from './build-public.mjs';
async function graph(t) { const root = await mkdtemp(path.join(os.tmpdir(), 'cq-public-export-')); t.after(() => rm(root, { recursive: true, force: true })); await writeFile(path.join(root, 'owned.js'), 'export const owned = true;'); return { root, assets: new Map(Array.from({ length: 902 }, (_, i) => ['/owned-' + i, i === 0 ? 'about.html' : 'owned.js'])), canonicalHelpSources: new Map(), contentTypes: { '.js': 'text/javascript', '.html': 'text/html' } }; }
const sitePageReader = async () => Buffer.from('<h1>Owned static site page</h1>');
test('fixed-count public export retains route/file identity and check mode never rewrites stale owned bytes', async t => {
  const g = await graph(t); const options = { sitePageReader }; assert.match(await buildPublic(g, options), /Exported 902 routes and 2 unique/); const manifestPath = path.join(g.root, '.local/public-upload-manifest.json'); const initial = await readFile(manifestPath); const m = JSON.parse(initial); assert.equal(m.routeCount, 902); assert.equal(m.fileCount, 2); assert.equal(m.files.find(row => row.file === 'owned.js').sha256, createHash('sha256').update(await readFile(path.join(g.root, 'owned.js'))).digest('hex')); assert.equal(m.files.find(row => row.file === 'about.html').routes.length, 1);
  assert.match(await buildPublic(g, { ...options, check: true }), /Verified 902 routes/); await writeFile(path.join(g.root, '.public-dist/owned.js'), 'stale'); await assert.rejects(buildPublic(g, { ...options, check: true }), /Exported bytes differ/); assert.equal(await readFile(path.join(g.root, '.public-dist/owned.js'), 'utf8'), 'stale');
  await buildPublic(g, options); await writeFile(manifestPath, 'stale'); await assert.rejects(buildPublic(g, { ...options, check: true }), /Upload manifest is stale/); await buildPublic(g, options); await writeFile(path.join(g.root, '.local/worker-routes.mjs'), 'stale'); await assert.rejects(buildPublic(g, { ...options, check: true }), /Worker route module is stale/);
  assert.equal(await isPublicBuildEntry(new URL('./build-public.mjs', import.meta.url)), true); assert.equal(await isPublicBuildEntry(''), false); assert.equal(await isPublicBuildEntry(path.join(g.root, 'missing')), false);
});
test('public export refuses count and source escape before erasing existing owned output', async t => {
  const g = await graph(t); await mkdir(path.join(g.root, '.public-dist')); await writeFile(path.join(g.root, '.public-dist/keep'), 'KEEP'); const smaller = { ...g, assets: new Map([['/', 'owned.js']]) }; await assert.rejects(buildPublic(smaller), /Expected 902/);
  g.assets.set('/owned-1', '../outside'); await assert.rejects(buildPublic(g, { sitePageReader }), /Unsafe public source/); assert.equal(await readFile(path.join(g.root, '.public-dist/keep'), 'utf8'), 'KEEP');
});
test('canonical read-only aliases and unknown MIME retain their owned bytes without repository browsing', async t => {
  const g = await graph(t); await writeFile(path.join(g.root, 'canonical'), 'OWNED_CANONICAL'); g.canonicalHelpSources.set('alias.bin', path.join(g.root, 'canonical')); g.assets.set('/owned-2', 'alias.bin'); await buildPublic(g, { sitePageReader }); const m = JSON.parse(await readFile(path.join(g.root, '.local/public-upload-manifest.json'))); assert.equal(m.files.find(row => row.file === 'alias.bin').mime, 'application/octet-stream'); assert.equal(await readFile(path.join(g.root, '.public-dist/alias.bin'), 'utf8'), 'OWNED_CANONICAL');
});
