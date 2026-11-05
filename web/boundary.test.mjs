import { test, mock } from 'node:test';
import fs from 'node:fs/promises';
import { syncBuiltinESMExports } from 'node:module';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, symlink, writeFile } from 'node:fs/promises';
import http from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createPreview, reloadablePath, requestURL } from './server.mjs';

function requestPreview(server, path, method = 'GET') {
  return new Promise((resolve, reject) => {
    const request = http.request({ host: '127.0.0.1', port: server.address().port, method, path }, response => {
      const chunks = [];
      response.on('data', chunk => chunks.push(chunk));
      response.once('end', () => resolve({ status: response.statusCode, body: Buffer.concat(chunks) }));
    });
    request.once('error', reject);
    request.end();
  });
}

function openReloadStream(server) {
  return new Promise((resolve, reject) => {
    const request = http.request({ host: '127.0.0.1', port: server.address().port, method: 'GET', path: '/__reload' }, response => {
      response.once('data', () => resolve({ request, response }));
    });
    request.once('error', reject);
    request.end();
  });
}

test('remote preview cannot serve client/config/source files or accept configuration uploads', async () => {
  const server = createPreview('remote');
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  try {
    for (const path of ['/local/model.js', '/local/entry.js', '/local/workbench.js', '/local/configuration.js', '/local/schema-tools.js', '/local/policy-schema.json', '/local/linux-example.json', '/output/playwright/workbench-export.json', '/docs/configuration.md', '/.git/config', '/api/config', '/remote/../local/model.js']) {
      assert.equal((await fetch(base + path)).status, 404, path);
    }
    for (const method of ['POST', 'PUT', 'PATCH', 'DELETE']) {
      const result = await fetch(base + '/api/config', { method, body: 'SYNTHETIC_CONFIG_DO_NOT_TRANSPORT' });
      assert.equal(result.status, 405);
    }
    assert.equal((await fetch(base + '/?config=SYNTHETIC')).status, 400);
    assert.equal((await fetch(base + '/__reload', { method: 'HEAD' })).status, 200);
    assert.equal((await requestPreview(server, 'http://localhost/ui.js')).status, 400);
    assert.equal((await requestPreview(server, 'http://example.invalid/ui.js')).status, 400);
    const response = await fetch(base + '/');
    assert.match(response.headers.get('content-security-policy'), /connect-src 'self'/);
    assert.equal(response.headers.get('cache-control'), 'no-store');
    assert.match(await response.text(), /remote\/entry.js/);
  } finally { await new Promise(resolve => server.close(resolve)); }
});

test('local client is served separately without remote account modules', async () => {
  const server = createPreview('local');
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  try {
    assert.equal((await fetch(base + '/remote/model.js')).status, 404);
    assert.equal((await fetch(base + '/local/model.js')).status, 200);
    assert.match(await (await fetch(base + '/')).text(), /local\/entry.js/);
  } finally { await new Promise(resolve => server.close(resolve)); }
});

test('preview accepts only literal planes', () => {
  assert.throws(() => createPreview('operator'), /preview plane/);
  assert.throws(() => createPreview('../local'), /preview plane/);
});

test('allowlisted assets reject external file and directory symlinks', async () => {
  const fixture = await mkdtemp(join(tmpdir(), 'cq-preview-'));
  const outside = await mkdtemp(join(tmpdir(), 'cq-preview-outside-'));
  let server;
  try {
    await writeFile(join(fixture, 'style.css'), 'body{}');
    await writeFile(join(outside, 'outside.js'), 'outside');
    await symlink(join(outside, 'outside.js'), join(fixture, 'ui.js'), 'file');
    await mkdir(join(outside, 'remote'));
    await writeFile(join(outside, 'remote', 'model.js'), 'outside');
    await symlink(join(outside, 'remote'), join(fixture, 'remote'), process.platform === 'win32' ? 'junction' : 'dir');
    server = createPreview('remote', fixture);
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    const base = `http://127.0.0.1:${server.address().port}`;
    const ordinary = await requestPreview(server, '/style.css');
    assert.equal(ordinary.status, 200);
    assert.equal(ordinary.body.toString('utf8'), 'body{}');
    assert.equal((await fetch(base + '/ui.js')).status, 404);
    assert.equal((await fetch(base + '/remote/model.js')).status, 404);
  } finally {
    if (server) await new Promise(resolve => server.close(resolve));
    await Promise.all([rm(fixture, { recursive: true, force: true }), rm(outside, { recursive: true, force: true })]);
  }
});

test('allowlisted assets reject content larger than the preview bound', async () => {
  const fixture = await mkdtemp(join(tmpdir(), 'cq-preview-large-'));
  let server;
  try {
    await writeFile(join(fixture, 'style.css'), Buffer.alloc(4 * 1024 * 1024 + 1));
    server = createPreview('remote', fixture);
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    const base = `http://127.0.0.1:${server.address().port}`;
    assert.equal((await fetch(base + '/style.css')).status, 404);
  } finally {
    if (server) await new Promise(resolve => server.close(resolve));
    await rm(fixture, { recursive: true, force: true });
  }
});

test('request target parsing and watcher path filtering are bounded and portable', () => {
  assert.equal(requestURL('http://[::1'), null);
  assert.equal(requestURL('http://localhost/ui.js'), null);
  assert.equal(requestURL('http://example.invalid/ui.js'), null);
  assert.equal(requestURL('/%invalid'), null);
  assert.equal(requestURL('/ui.js')?.pathname, '/ui.js');
  for (const target of ['/logo-light.svg?', '/ui.js?x=1', '/__reload?', '/?']) assert.equal(requestURL(target), null, target);
  assert.equal(requestURL('/logo-light.svg')?.pathname, '/logo-light.svg');
  assert.equal(reloadablePath('output\\evidence.json'), false);
  assert.equal(reloadablePath('local\\entry.js'), true);
});

test('reload admission is finite and HEAD leaves no held stream', { timeout: 5000 }, async () => {
  const server = createPreview('remote');
  const streams = [];
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  try {
    for (let index = 0; index < 32; index += 1) streams.push(await openReloadStream(server));
    assert.equal((await requestPreview(server, '/__reload')).status, 503);
    const head = await requestPreview(server, '/__reload', 'HEAD');
    assert.equal(head.status, 200);
    assert.equal(head.body.length, 0);
  } finally {
    for (const stream of streams) {
      stream.response.destroy();
      stream.request.destroy();
    }
    await new Promise(resolve => server.close(resolve));
  }
});

for (const component of ['root', 'intermediate']) {
  test(`asset open rejects a deterministic ${component} replacement before reading another file`, async () => {
    const fixture = await mkdtemp(join(tmpdir(), 'cq-preview-race-'));
    const directory = join(fixture, 'current');
    const outside = join(fixture, 'outside');
    const backup = join(fixture, 'original-component');
    const name = component === 'root' ? 'style.css' : 'remote/model.js';
    const target = join(directory, name);
    const swappedPath = component === 'root' ? directory : join(directory, 'remote');
    const replacement = component === 'root' ? outside : join(outside, 'remote');
    let server;
    let openMock;
    let armed = false;
    let swapped = false;
    let attackReads = 0;
    try {
      await Promise.all([mkdir(join(directory, 'remote'), { recursive: true }), mkdir(join(outside, 'remote'), { recursive: true })]);
      await writeFile(target, 'SYNTHETIC_EXPECTED_ASSET');
      await writeFile(join(outside, name), 'SYNTHETIC_EXTERNAL_ASSET');
      const nativeOpen = fs.open;
      openMock = mock.method(fs, 'open', async (path, ...args) => {
        const attack = armed && path === target;
        if (attack) {
          armed = false;
          await fs.rename(swappedPath, backup);
          await symlink(replacement, swappedPath, process.platform === 'win32' ? 'junction' : 'dir');
          swapped = true;
        }
        const file = await nativeOpen(path, ...args);
        return attack ? {
          stat: (...options) => file.stat(...options),
          read: (...options) => { attackReads += 1; return file.read(...options); },
          close: () => file.close(),
        } : file;
      });
      syncBuiltinESMExports();
      server = createPreview('remote', directory);
      await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
      const ordinary = await requestPreview(server, '/' + name);
      assert.equal(ordinary.status, 200);
      assert.equal(ordinary.body.toString(), 'SYNTHETIC_EXPECTED_ASSET');
      armed = true;
      const attacked = await requestPreview(server, '/' + name);
      assert.equal(swapped, true, 'the replacement happened at the actual open boundary');
      assert.equal(attacked.status, 404);
      assert.equal(attacked.body.length, 0);
      assert.equal(attackReads, 0, 'reject the unexpected descriptor before reading its payload');
      await fs.unlink(swappedPath);
      await fs.rename(backup, swappedPath);
      const restored = await requestPreview(server, '/' + name);
      assert.equal(restored.status, 200);
      assert.equal(restored.body.toString(), 'SYNTHETIC_EXPECTED_ASSET');
      for (const query of ['?', '?x=1']) assert.equal((await requestPreview(server, '/' + name + query)).status, 400);
    } finally {
      openMock?.mock.restore();
      syncBuiltinESMExports();
      if (server) await new Promise(resolve => server.close(resolve));
      await rm(fixture, { recursive: true, force: true });
    }
  });
}
