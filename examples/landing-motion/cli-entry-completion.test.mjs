import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { runCli, assertCliPassed, previewPreload } from '../../web/cli-process-fixture.test.mjs';

const here = fileURLToPath(new URL('.', import.meta.url));

test('required catalogue CLI checks enter actual modules and validate tracked public assets without generation', async t => {
  const cases = [
    ['build-motion-catalog.mjs', '', ['--check']],
    ['render-integrations.mjs', /catalog entries\/.* SVGs, 30 login providers verified/, ['--check']],
    ['render-site-pages.mjs', /Checked \d+ static site files\./, ['--check']],
  ];
  for (const [name, expected, args] of cases) {
    await t.test(name, async () => {
      const result = await runCli(fileURLToPath(new URL(name, import.meta.url)), args, { cwd: here });
      assertCliPassed(result, name);
      if (expected === '') assert.equal(result.stdout, '');
      else assert.match(result.stdout, expected);
    });
  }
});

test('landing default CLI serves its real fixed loopback port and both registered graceful callbacks', async t => {
  for (const signal of ['SIGINT', 'SIGTERM']) {
    await t.test(signal, async t => {
      const preload = await previewPreload(t, [5182], signal);
      const result = await runCli(fileURLToPath(new URL('./server.mjs', import.meta.url)), [], { cwd: here, preload, timeout: 30_000 });
      assertCliPassed(result, 'landing default CLI');
      assert.match(result.stdout, /CircularQuay landing-motion preview: http:\/\/127\.0\.0\.1:5182/);
      assert.match(result.stdout, /Owned default preview served all fixed ports/);
      assert.ok(result.stdout.includes(`Received ${signal}; closing landing-motion preview.`));
    });
  }
});

test('actual syntax checker CLI uses its default tracked tree without installation or generation', async () => {
  const { fileURLToPath } = await import('node:url');
  const { runCli, assertCliPassed } = await import('../../web/cli-process-fixture.test.mjs');
  const result = await runCli(fileURLToPath(new URL('../../web/check.mjs', import.meta.url)), [], { cwd: fileURLToPath(new URL('../../web/', import.meta.url)), descendants: true });
  assertCliPassed(result, 'syntax checker CLI');
  assert.match(result.stdout, /All frontend JavaScript syntax checks passed\. No bundler or install required\./);
});

test('owned POSIX CLI group termination also closes the inherited pipe held by its ignoring descendant', { timeout: 10_000 }, async t => {
  const { mkdtemp, writeFile, rm } = await import('node:fs/promises');
  const { tmpdir } = await import('node:os');
  const { join } = await import('node:path');
  const root = await mkdtemp(join(tmpdir(), 'cq-owned-cli-tree-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  const file = join(root, 'owned-tree.mjs');
  const child = "process.on('SIGTERM',()=>{});process.stdout.write('owned-descendant-ready\\n');setInterval(()=>{},1000);";
  await writeFile(file, `import {spawn} from 'node:child_process';\nprocess.on('SIGTERM',()=>{});spawn(process.execPath,['-e',${JSON.stringify(child)}],{stdio:['ignore','inherit','inherit']});setInterval(()=>{},1000);\n`);
  const result = await runCli(file, [], { cwd: root, timeout: 2_000, descendants: true });
  assert.equal(result.error?.code, 'CQ_CLI_TIMEOUT');
  assert.equal(result.treeStrategy, 'owned-posix-group');
  assert.equal(result.closed, true); assert.equal(result.reaped, true);
  assert.match(result.stdout, /owned-descendant-ready/);
  assert.equal(result.signal, 'SIGKILL');
});
