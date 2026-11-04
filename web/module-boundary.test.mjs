import { test } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { createPreview } from './server.mjs';

function requestPreview(server, path) {
  return new Promise((resolve, reject) => {
    const request = http.request({ host: '127.0.0.1', port: server.address().port, path }, response => {
      const chunks = [];
      response.on('data', chunk => chunks.push(chunk));
      response.once('end', () => resolve({ status: response.statusCode, body: Buffer.concat(chunks) }));
    });
    request.once('error', reject);
    request.end();
  });
}

function staticImportSpecifiers(source) {
  return [...source.matchAll(/^\s*import\s+(?:[^'"\n]+\s+from\s+)?['"]([^'"\n]+)['"]\s*;?\s*$/gm)].map(match => match[1]);
}

async function staticModuleClosure(server, entry) {
  const modules = new Map();
  const pending = [entry];
  while (pending.length) {
    const path = pending.pop();
    if (modules.has(path)) continue;
    const response = await requestPreview(server, path);
    assert.equal(response.status, 200, path);
    const source = response.body.toString('utf8');
    modules.set(path, source);
    for (const specifier of staticImportSpecifiers(source)) {
      const target = new URL(specifier, `http://preview.invalid${path}`).pathname;
      if (!modules.has(target)) pending.push(target);
    }
  }
  return modules;
}

test('remote entry static module closure contains no local client module', async () => {
  const server = createPreview('remote');
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  try {
    const modules = await staticModuleClosure(server, '/remote/entry.js');
    assert.deepEqual([...modules.keys()].sort(), ['/remote/entry.js', '/remote/model.js', '/ui.js']);
    assert.equal([...modules.keys()].some(path => path.startsWith('/local/')), false);
    assert.equal((await requestPreview(server, '/local/schema-tools.js')).status, 404);
    assert.equal((await requestPreview(server, '/local/policy-schema.json')).status, 404);
  } finally {
    await new Promise(resolve => server.close(resolve));
  }
});
