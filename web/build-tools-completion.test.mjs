import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, readdir, rm, copyFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { brotliDecompressSync } from 'node:zlib';
import path from 'node:path';
import os from 'node:os';
import { buildHelp, isHelpBuildEntry } from './tools/build-help.mjs';
import { checkFrontend, isCheckEntry, runFrontendCheck } from './check.mjs';
const digest = bytes => createHash('sha256').update(bytes).digest('hex');
async function owned(t) { const root = await mkdtemp(path.join(os.tmpdir(), 'cq-help-build-')); t.after(() => rm(root, { recursive: true, force: true })); return root; }
async function helpFixture(t) {
  const root = await owned(t); await mkdir(path.join(root, 'help/generated'), { recursive: true });
  await copyFile(new URL('./help/source-contract.schema.json', import.meta.url), path.join(root, 'help/source-contract.schema.json'));
  const source = JSON.parse(await readFile(new URL('./help/source-index.json', import.meta.url))); source.entries = source.entries.slice(0, 2);
  await writeFile(path.join(root, 'help/source-index.json'), JSON.stringify(source)); return { root, source };
}
test('help builder publishes only closed authored fields with exact identity, compression, pins and stale cleanup in owned roots', async t => {
  const { root, source } = await helpFixture(t); const stale = `article-stale-${'0'.repeat(64)}.json`; await writeFile(path.join(root, 'help/generated', stale), 'old'); await writeFile(path.join(root, 'help/generated/keep.txt'), 'retained');
  const message = await buildHelp(root); assert.match(message, /Built 2 public help articles/); const manifestBytes = await readFile(path.join(root, 'help/generated/manifest.json')); const manifest = JSON.parse(manifestBytes);
  assert.equal(manifest.contentVersion, 'sha256:' + digest(await readFile(path.join(root, 'help/source-index.json'))));
  const catalogBytes = await readFile(path.join(root, manifest.catalog.path.slice(1))); assert.equal(digest(catalogBytes), manifest.catalog.sha256); assert.equal(catalogBytes.length, manifest.catalog.bytes);
  const catalog = JSON.parse(catalogBytes); assert.equal(catalog.documents.length, 2);
  for (const [index, row] of catalog.documents.entries()) { const bytes = await readFile(path.join(root, row.article.path.slice(1))); const article = JSON.parse(bytes); assert.deepEqual(Object.keys(article), ['contentVersion', 'id', 'title', 'bodyMarkdown', 'capabilityStatus', 'canonicalRefs']); assert.equal(article.bodyMarkdown, source.entries[index].bodyMarkdown); assert.deepEqual(brotliDecompressSync(await readFile(path.join(root, row.article.path.slice(1) + '.br'))), bytes); }
  const names = await readdir(path.join(root, 'help/generated')); assert.ok(!names.includes(stale)); assert.equal(await readFile(path.join(root, 'help/generated/keep.txt'), 'utf8'), 'retained');
  const release = await readFile(path.join(root, 'help-release.js'), 'utf8'); assert.ok(release.includes(digest(manifestBytes))); assert.match(await readFile(path.join(root, 'help/assets.js'), 'utf8'), /helpAssetPaths/); assert.equal(await buildHelp(root), message); assert.deepEqual(await readFile(path.join(root, 'help/generated/manifest.json')), manifestBytes);
});
test('help builder rejects contract/hash/source limits, malformed records and duplicate public IDs before output', async t => {
  const { root, source } = await helpFixture(t); const file = path.join(root, 'help/source-index.json');
  for (const text of ['{', ' '.repeat(1048577), JSON.stringify({ artifactType: 'different' }), JSON.stringify({ ...source, entries: [source.entries[0], source.entries[0]] }), JSON.stringify({ ...source, entries: [{ ...source.entries[0], id: '../private' }] })]) { await writeFile(file, text); await assert.rejects(buildHelp(root)); assert.deepEqual(await readdir(path.join(root, 'help/generated')), []); }
  await writeFile(file, JSON.stringify(source)); await writeFile(path.join(root, 'help/source-contract.schema.json'), '{}'); await assert.rejects(buildHelp(root), /contract changed/); assert.deepEqual(await readdir(path.join(root, 'help/generated')), []);
});
test('syntax checker runs only fixed pinned-Node --check over owned source files, preserving failure status and successful no-install path', async t => {
  const root = await owned(t); for (const dir of ['local', 'remote', 'help', 'tools']) await mkdir(path.join(root, dir)); await writeFile(path.join(root, 'ignored.txt'), 'not JavaScript'); await writeFile(path.join(root, 'owned.js'), 'const safelyParsed = 1;');
  assert.deepEqual(await checkFrontend(root), { status: 0, stderr: '' }); await writeFile(path.join(root, 'local/invalid.mjs'), 'export const ='); const failed = await checkFrontend(root); assert.ok(failed.status !== 0); assert.match(failed.stderr, /SyntaxError/); await assert.rejects(checkFrontend(path.join(root, 'missing')));
  assert.equal(await isHelpBuildEntry(new URL('./tools/build-help.mjs', import.meta.url)), true); assert.equal(await isCheckEntry(new URL('./check.mjs', import.meta.url)), true); assert.equal(await isCheckEntry(''), false); assert.equal(await isHelpBuildEntry(path.join(root, 'missing')), false); assert.equal(await isCheckEntry(path.join(root, 'ignored.txt')), false);
});

test('help check mode validates existing artifacts without publication or stale-file deletion', async t => {
  const { root } = await helpFixture(t);
  await buildHelp(root);
  const files = ['help-release.js', 'help/assets.js', ...(await readdir(path.join(root, 'help/generated'))).map(name => `help/generated/${name}`)];
  const { stat, utimes } = await import('node:fs/promises');
  const old = new Date('2001-01-01T00:00:00Z');
  for (const file of files) await utimes(path.join(root, file), old, old);
  const before = await Promise.all(files.map(async file => [file, await readFile(path.join(root, file)), (await stat(path.join(root, file))).mtimeMs]));
  assert.match(await buildHelp(root, { check: true }), /Verified 2 public help articles/);
  for (const [file, bytes, mtime] of before) { assert.deepEqual(await readFile(path.join(root, file)), bytes); assert.equal((await stat(path.join(root, file))).mtimeMs, mtime); }
  const manifest = path.join(root, 'help/generated/manifest.json');
  await writeFile(manifest, 'stale owned bytes');
  await assert.rejects(buildHelp(root, { check: true }), /Generated help artifact is stale/);
  assert.equal(await readFile(manifest, 'utf8'), 'stale owned bytes');
  await buildHelp(root);
  const stale = path.join(root, 'help/generated', `article-stale-${'0'.repeat(64)}.json`);
  await writeFile(stale, 'retained owned stale file');
  await assert.rejects(buildHelp(root, { check: true }), /contains a stale artifact/);
  assert.equal(await readFile(stale, 'utf8'), 'retained owned stale file');
});

test('actual help CLI check enters the tracked module and validates without rewriting public generated assets', async () => {
  const { fileURLToPath } = await import('node:url');
  const { runCli, assertCliPassed } = await import('./cli-process-fixture.test.mjs');
  const result = await runCli(fileURLToPath(new URL('./tools/build-help.mjs', import.meta.url)), ['--check'], { cwd: fileURLToPath(new URL('.', import.meta.url)) });
  assertCliPassed(result, 'help check CLI');
  assert.match(result.stdout, /Verified \d+ public help articles/);
});


test('CLI syntax-result adapter preserves real owned invalid-source stderr and exit without global process mutation', async t => {
  const root = await owned(t);
  for (const directory of ['local', 'remote', 'help', 'tools']) await mkdir(path.join(root, directory));
  const file = path.join(root, 'owned.mjs');
  await writeFile(file, 'export const =');
  const errors = [], lines = [], exits = [];
  const io = { error: value => errors.push(value), line: value => lines.push(value), setExit: value => exits.push(value) };
  const status = await runFrontendCheck(root, io);
  assert.notEqual(status, 0); assert.deepEqual(exits, [status]); assert.equal(lines.length, 0);
  assert.match(errors.join(''), /SyntaxError/);
  await writeFile(file, 'export const valid = 1;');
  errors.length = 0; exits.length = 0;
  assert.equal(await runFrontendCheck(root, io), 0);
  assert.deepEqual(errors, []); assert.deepEqual(exits, []);
  assert.deepEqual(lines, ['All frontend JavaScript syntax checks passed. No bundler or install required.']);
});

test('owned CLI supervisor force-terminates a leaf that installs an ignoring SIGTERM handler and observes reaping', { timeout: 10_000 }, async t => {
  const { runCli } = await import('./cli-process-fixture.test.mjs');
  const root = await owned(t); const file = path.join(root, 'ignoring-leaf.mjs');
  await writeFile(file, "process.on('SIGTERM',()=>{});process.stdout.write('owned-ready\\n');setInterval(()=>{},1000);\n");
  const result = await runCli(file, [], { cwd: root, timeout: 2_000 });
  assert.equal(result.error?.code, 'CQ_CLI_TIMEOUT');
  assert.equal(result.closed, true); assert.equal(result.reaped, true);
  assert.match(result.stdout, /owned-ready/);
  assert.ok(result.status !== 0 || result.signal !== null);
});

test('owned CLI supervisor captures bounded output and refuses stdout or stderr overflow before success', { timeout: 10_000 }, async t => {
  const { runCli, assertCliPassed } = await import('./cli-process-fixture.test.mjs');
  const root = await owned(t); const file = path.join(root, 'bounded-leaf.mjs');
  await writeFile(file, "process.stdout.write('owned');\n");
  const passed = await runCli(file, [], { cwd: root, outputLimit: 32 });
  assertCliPassed(passed, 'bounded owned leaf'); assert.equal(passed.stdout, 'owned');
  for (const stream of ['stdout', 'stderr']) {
    await writeFile(file, `process.${stream}.write('X'.repeat(1024));setInterval(()=>{},1000);\n`);
    const refused = await runCli(file, [], { cwd: root, outputLimit: 32 });
    assert.equal(refused.error?.code, 'CQ_CLI_OUTPUT_LIMIT');
    assert.equal(refused.closed, true); assert.equal(refused.reaped, true);
    assert.ok(Buffer.byteLength(refused.stdout) <= 32); assert.ok(Buffer.byteLength(refused.stderr) <= 32);
  }
});
