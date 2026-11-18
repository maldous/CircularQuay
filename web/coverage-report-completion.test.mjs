import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, symlink, mkdir, open, lstat, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import * as V from '../sdk/typescript/test/verify-coverage.mjs';

const source = fileURLToPath(new URL('../sdk/typescript/src/index.ts', import.meta.url));
const report = () => ({ [source]: { path: source, statementMap: { s: { start: { line: 1, column: 0 }, end: { line: 1, column: 1 } } }, fnMap: { f: {} }, branchMap: { b: {} }, s: { s: 1 }, f: { f: 1 }, b: { b: [1, 1] } } });

test('owned coverage report structure is distinct from the unchanged separate100-percent gate', () => {
  assert.doesNotThrow(() => V.validateReport(report()));
  const zero = report(); zero[source].s.s = 0; assert.doesNotThrow(() => V.validateReport(zero));
  assert.equal(V.normalizeCoveragePath(pathToFileURL(source).href), source);
  assert.equal(V.normalizeCoveragePath(source), source);
});

test('every malformed report/source/map/count boundary refuses rather than treating absent hits as a pass', () => {
  for (const value of [null, [], 1, 'text', {}, { unknown: {} }, { [source]: {}, extra: {} }]) assert.throws(() => V.validateReport(value), /coverage validation failed/);
  for (const value of [null, 1, '', 'relative.ts', 'file://[bad', 'https://example.invalid/index.ts']) assert.throws(() => V.normalizeCoveragePath(value), /source identity is invalid/);
  const changes = [
    value => { value[source] = []; }, value => { value[source].path = source + '.other'; },
    ...['statementMap', 'fnMap', 'branchMap', 's', 'f', 'b'].flatMap(name => [value => { value[source][name] = {}; }, value => { value[source][name] = null; }]),
    value => { value[source].s.s = -1; }, value => { value[source].s.s = 1.5; }, value => { value[source].f.f = Number.MAX_SAFE_INTEGER + 1; },
    value => { value[source].b.b = []; }, value => { value[source].b.b = null; }, value => { value[source].b.b = [Infinity]; },
    value => { value[source].s = { unknown: 1 }; }, value => { value[source].f.other = 1; }, value => { value[source].b = { unknown: [1] }; },
  ];
  for (const mutate of changes) { const value = report(); mutate(value); assert.throws(() => V.validateReport(value), /coverage validation failed/); }
  for (const value of [NaN, Infinity, -1, 0.5, '1', null]) assert.equal(V.isCount(value), false);
  for (const value of [0, 1, Number.MAX_SAFE_INTEGER]) assert.equal(V.isCount(value), true);
  for (const value of [null, [], 1]) assert.equal(V.isRecord(value), false);
  assert.equal(V.isRecord({}), true);
});

test('UTF-8/JSON decoding and exact-key correspondence are independent refusal boundaries', () => {
  assert.equal(V.decodeUTF8(Buffer.from('owned')), 'owned'); assert.throws(() => V.decodeUTF8(Buffer.from([255])), /not UTF-8/);
  assert.deepEqual(V.parseReport('{}'), {}); assert.throws(() => V.parseReport('{bad'), /not valid JSON/);
  assert.doesNotThrow(() => V.requireMatchingKeys([['a', 1]], [['a', 2]], 'owned'));
  assert.throws(() => V.requireMatchingKeys([['a', 1]], [], 'owned'), /do not match/);
  assert.throws(() => V.requireMatchingKeys([['a', 1]], [['b', 2]], 'owned'), /do not match/);
});

test('bounded actual owned-file reads refuse directory/symlink/oversize and preserve exact report bytes', async t => {
  const root = await mkdtemp(join(tmpdir(), 'cq-owned-coverage-')); t.after(() => rm(root, { recursive: true, force: true }));
  const file = join(root, 'coverage.json'); const text = JSON.stringify(report()); await writeFile(file, text);
  assert.equal(await V.readBounded(file), text);
  const directory = join(root, 'directory'); await mkdir(directory); await assert.rejects(V.readBounded(directory), /not a regular file/);
  const link = join(root, 'link'); await symlink(directory, link, 'junction'); await assert.rejects(V.readBounded(link), /not a regular file/);
  const large = join(root, 'large'); await writeFile(large, Buffer.alloc(4 * 1024 * 1024 + 1)); await assert.rejects(V.readBounded(large), /byte limit/);
  await writeFile(file, Buffer.from([255])); await assert.rejects(V.readBounded(file), /not UTF-8/);
  assert.equal((await lstat(large)).size, 4 * 1024 * 1024 + 1);
});

test('held descriptor identity rejects every changed/nonregular/unsafe-integer variant using owned names', async t => {
  const root = await mkdtemp(join(tmpdir(), 'cq-owned-descriptor-')); let ownedHandle; t.after(async () => { await ownedHandle?.close(); await rm(root, { recursive: true, force: true }); });
  const file = join(root, 'profile'); await writeFile(file, 'owned'); const original = await lstat(file); const handle = ownedHandle = await open(file, 'r');
  assert.equal(await V.requireStableReport(file, original, handle).then(value => value.ino), original.ino);
  const info = changes => ({ isFile: () => true, isSymbolicLink: () => false, dev: 1, ino: 2, ...changes });
  assert.equal(V.sameRegularFile(info({}), info({})), true);
  for (const changed of [{ isFile: () => false }, { isSymbolicLink: () => true }, { dev: 3 }, { ino: 3 }, { dev: NaN }, { ino: 0.5 }]) {
    assert.equal(V.sameRegularFile(info({}), info(changed)), false);
    assert.equal(V.sameRegularFile(info(changed), info({})), false);
  }
  await assert.rejects(V.requireStableReport(file, original, { stat: async () => ({ ...original, isFile: () => true, isSymbolicLink: () => false, ino: original.ino + 1 }) }), /changed while it was read/);
});

test('readAtMost enforces true4MiB including one sentinel byte and repeated partial reads', async () => {
  const bytes = Buffer.from('owned'); let position = 0;
  const small = { async read(target, offset, count) { const size = Math.min(2, count, bytes.length - position); bytes.copy(target, offset, position, position + size); position += size; return { bytesRead: size }; } };
  assert.deepEqual(await V.readAtMost(small), bytes);
  const limit = 4 * 1024 * 1024;
  assert.equal((await V.readAtMost({ async read(target, offset, count) { const bytesRead = Math.min(count, limit - offset); target.fill(0, offset, offset + bytesRead); return { bytesRead }; } })).length, limit);
  await assert.rejects(V.readAtMost({ async read(target, offset, count) { target.fill(0, offset, offset + count); return { bytesRead: count }; } }), /byte limit/);
});

test('actual CLI main keeps exactly-one-path semantics while imported validation has no side effects', async t => {
  const root = await mkdtemp(join(tmpdir(), 'cq-cli-coverage-')); t.after(() => rm(root, { recursive: true, force: true })); const file = join(root, 'coverage.json'); await writeFile(file, JSON.stringify(report()));
  const previous = process.argv;
  try {
    for (const args of [[], [''], [file, 'extra']]) { process.argv = [process.execPath, 'verify-coverage.mjs', ...args]; await assert.rejects(V.main(), /expected one coverage report path/); }
    process.argv = [process.execPath, 'verify-coverage.mjs', file]; await V.main();
    await writeFile(file, '{bad'); await assert.rejects(V.main(), /not valid JSON/);
  } finally { process.argv = previous; }
});

test('entry detection recognizes the actual verifier through an owned symlink rather than silently skipping validation', async t => {
  const root = await mkdtemp(join(tmpdir(), 'cq-coverage-entry-')); t.after(() => rm(root, { recursive: true, force: true }));
  const entry = fileURLToPath(new URL('../sdk/typescript/test/verify-coverage.mjs', import.meta.url));
  const aliasDirectory = join(root, 'test-alias'); await symlink(fileURLToPath(new URL('../sdk/typescript/test/', import.meta.url)), aliasDirectory, 'junction'); const alias = join(aliasDirectory, 'verify-coverage.mjs');
  assert.equal(await V.isMain(entry), true); assert.equal(await V.isMain(alias), true);
  assert.equal(await V.isMain(''), false); assert.equal(await V.isMain(join(root, 'missing')), false);
  assert.equal(await V.isMain(fileURLToPath(import.meta.url)), false);
});

test('actual owned-report verifier CLI enters its tracked module with success and refusal exit evidence', async t => {
  const { runCli, assertCliPassed } = await import('./cli-process-fixture.test.mjs');
  const root = await mkdtemp(join(tmpdir(), 'cq-owned-cli-report-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  const file = join(root, 'coverage.json');
  const entry = fileURLToPath(new URL('../sdk/typescript/test/verify-coverage.mjs', import.meta.url));
  await writeFile(file, JSON.stringify(report()));
  const accepted = await runCli(entry, [file], { cwd: root });
  assertCliPassed(accepted, 'owned report CLI');
  assert.equal(accepted.stdout, '');
  for (const args of [[], [file, 'extra']]) {
    const refused = await runCli(entry, args, { cwd: root });
    assert.equal(refused.error, undefined); assert.equal(refused.signal, null); assert.notEqual(refused.status, 0);
    assert.match(refused.stderr, /coverage validation failed: expected one coverage report path/);
  }
  await writeFile(file, '{bad');
  const malformed = await runCli(entry, [file], { cwd: root });
  assert.equal(malformed.error, undefined); assert.equal(malformed.signal, null); assert.notEqual(malformed.status, 0);
  assert.match(malformed.stderr, /coverage validation failed: coverage report is not valid JSON/);
});
