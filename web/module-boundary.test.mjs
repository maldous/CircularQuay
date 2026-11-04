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

function importLiteral(statement) {
  const positions = [statement.indexOf("'"), statement.indexOf('"')].filter(index => index >= 0);
  if (!positions.length) throw new Error('Static import must use a single-line literal.');
  const start = Math.min(...positions);
  const prefix = statement.slice(0, start).trimEnd();
  if (prefix && !prefix.endsWith('from')) throw new Error('Unsupported static import declaration.');
  const end = statement.indexOf(statement[start], start + 1);
  if (end < 0) throw new Error('Unterminated static import literal.');
  const specifier = statement.slice(start + 1, end);
  if (!specifier || specifier.includes('\\')) throw new Error('Static import must use an unescaped literal.');
  let suffix = statement.slice(end + 1).trim();
  if (suffix.startsWith(';')) suffix = suffix.slice(1).trimStart();
  if (suffix && !suffix.startsWith('//')) throw new Error('Unsupported trailing static import syntax.');
  return specifier;
}

function staticImportSpecifiers(source) {
  const specifiers = [];
  for (const raw of source.split('\n')) {
    const line = raw.trim();
    if (!line.startsWith('import') || /^[A-Za-z0-9_$]/.test(line[6] || '')) continue;
    if (line.length > 4096) throw new Error('Static import declaration exceeds the scanner bound.');
    const statement = line.slice(6).trimStart();
    if (statement.startsWith('(') || statement.startsWith('.')) continue;
    specifiers.push(importLiteral(statement));
  }
  return specifiers;
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
      const resolved = new URL(specifier, `http://preview.invalid${path}`);
      assert.equal(resolved.origin, 'http://preview.invalid', 'Static modules must stay on the preview origin.');
      const target = resolved.pathname;
      if (!modules.has(target)) pending.push(target);
    }
  }
  return modules;
}

test('bounded static import scanner covers literal forms without executing dynamic imports', () => {
  const source = [
    "import '../side.js';",
    "  import{ value }from './named.js';",
    "import * as namespace from './namespace.js'; // comment",
    'import defaultValue from "/absolute.js";',
    "import('./dynamic.js');",
    'import.meta.url;',
    "const ignored = 'not an import';",
  ].join('\n');
  assert.deepEqual(staticImportSpecifiers(source), ['../side.js', './named.js', './namespace.js', '/absolute.js']);
});

test('static import scanner rejects unsupported or oversized declarations rather than silently skipping them', () => {
  for (const source of ["import { value }", "import './unterminated", "import './escaped\\x2ejs';", "import './one.js'; import './hidden.js';"]) {
    assert.throws(() => staticImportSpecifiers(source));
  }
  assert.throws(() => staticImportSpecifiers('import ' + ' '.repeat(100000) + "'./oversized.js';"), /scanner bound/);
});

test('remote entry static module closure contains no local client module', async () => {
  const server = createPreview('remote');
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  try {
    const modules = await staticModuleClosure(server, '/remote/entry.js');
    const paths = [...modules.keys()].sort((left, right) => {
      if (left === right) return 0;
      return left < right ? -1 : 1;
    });
    assert.deepEqual(paths, ['/remote/entry.js', '/remote/model.js', '/ui.js']);
    assert.equal([...modules.keys()].some(path => path.startsWith('/local/')), false);
    assert.equal((await requestPreview(server, '/local/schema-tools.js')).status, 404);
    assert.equal((await requestPreview(server, '/local/policy-schema.json')).status, 404);
  } finally {
    await new Promise(resolve => server.close(resolve));
  }
});
