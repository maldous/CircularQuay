import http from 'node:http';
import { lstat, open, realpath } from 'node:fs/promises';
import { constants, watch } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { extname, isAbsolute, join, relative, resolve, sep } from 'node:path';

const root = fileURLToPath(new URL('.', import.meta.url));
const shared = new Set(['ui.js', 'style.css', 'logo.png']);
const planes = new Set(['local', 'remote']);
const assets = {
  local: new Set(['local/entry.js', 'local/model.js', 'local/workbench.js', 'local/configuration.js', 'local/schema-tools.js', 'local/policy-schema.json', 'local/linux-example.json']),
  remote: new Set(['remote/entry.js', 'remote/model.js']),
};
const types = { '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.json': 'application/json' };
const listeners = new Set();
const maximumAssetBytes = 4 * 1024 * 1024;
const maximumReloadListeners = 32;

function isContained(directory, path) {
  const remainder = relative(directory, path);
  return remainder !== '' && remainder !== '..' && !remainder.startsWith(`..${sep}`) && !isAbsolute(remainder);
}

async function trustedAssetPath(directory, name) {
  const path = resolve(directory, name);
  if (!isContained(directory, path)) return null;
  const rootInfo = await lstat(directory);
  if (!rootInfo.isDirectory() || rootInfo.isSymbolicLink()) return null;
  let current = directory;
  const components = name.split('/');
  for (const [index, component] of components.entries()) {
    current = join(current, component);
    const info = await lstat(current);
    if (info.isSymbolicLink() || (index < components.length - 1 && !info.isDirectory())) return null;
    if (index === components.length - 1 && !info.isFile()) return null;
  }
  const [canonicalDirectory, canonicalPath] = await Promise.all([realpath(directory), realpath(path)]);
  return isContained(canonicalDirectory, canonicalPath) ? path : null;
}

async function readTrustedAsset(directory, name) {
  const path = await trustedAssetPath(directory, name);
  if (!path) return null;
  const flags = constants.O_RDONLY | (process.platform === 'win32' ? 0 : constants.O_NOFOLLOW);
  const file = await open(path, flags);
  try {
    const [opened, named] = await Promise.all([file.stat(), lstat(path)]);
    if (!opened.isFile() || opened.size > maximumAssetBytes || !named.isFile() || named.isSymbolicLink() || opened.dev !== named.dev || opened.ino !== named.ino) return null;
    const bytes = await readBoundedAsset(file, opened.size);
    const [after, namedAfter] = await Promise.all([file.stat(), lstat(path)]);
    if (!bytes || after.size !== opened.size || !namedAfter.isFile() || namedAfter.isSymbolicLink() || after.dev !== namedAfter.dev || after.ino !== namedAfter.ino) return null;
    return bytes;
  } finally {
    await file.close();
  }
}

async function readBoundedAsset(file, expectedSize) {
  if (!Number.isSafeInteger(expectedSize) || expectedSize < 0 || expectedSize > maximumAssetBytes) return null;
  const buffer = Buffer.alloc(Math.min(expectedSize + 1, maximumAssetBytes + 1));
  let read = 0;
  while (read < buffer.length) {
    const result = await file.read(buffer, read, buffer.length - read, read);
    if (!Number.isSafeInteger(result.bytesRead) || result.bytesRead < 0) return null;
    if (result.bytesRead === 0) break;
    read += result.bytesRead;
  }
  return read === expectedSize ? buffer.subarray(0, read) : null;
}

export function reloadablePath(name) {
  const path = String(name).replaceAll('\\', '/');
  return !path.startsWith('output/') && /\.(?:js|mjs|css|json)$/.test(path);
}

export function requestURL(value) {
  try {
    if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//') || value.includes('\\') || value.includes('#') || /%(?![0-9A-Fa-f]{2})/.test(value)) return null;
    const url = new URL(value, 'http://localhost');
    return url.origin === 'http://localhost' && !url.search ? url : null;
  } catch { return null; }
}

export function createPreview(plane, sourceRoot = root) {
  if (!planes.has(plane)) throw new TypeError('preview plane');
  const directory = resolve(sourceRoot);
  return http.createServer(async (request, response) => {
    const headers = {
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
      'Referrer-Policy': 'no-referrer',
      'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'",
    };
    const reject = (status) => { response.writeHead(status, headers); response.end(); };
    if (!['GET', 'HEAD'].includes(request.method)) return reject(405);
    const url = requestURL(request.url);
    if (!url) return reject(400);
    if (url.pathname === '/__reload') {
      if (request.method === 'HEAD') {
        response.writeHead(200, { ...headers, 'Content-Type': 'text/event-stream' });
        return response.end();
      }
      if (listeners.size >= maximumReloadListeners) return reject(503);
      response.writeHead(200, { ...headers, 'Content-Type': 'text/event-stream', Connection: 'keep-alive' });
      response.write(': development hot reload\n\n');
      listeners.add(response);
      request.on('close', () => listeners.delete(response));
      return;
    }
    const name = url.pathname.slice(1);
    if (!name) {
      response.writeHead(200, { ...headers, 'Content-Type': 'text/html' });
      response.end(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>CircularQuay · ${plane === 'local' ? 'Local client' : 'Managed service demo'}</title><link rel="stylesheet" href="/style.css"></head><body><div id="app"></div><script type="module" src="/${plane}/entry.js"></script></body></html>`);
      return;
    }
    // Serving roots are explicit and disjoint: no repository browsing or local-client module on remote.
    if (!shared.has(name) && !assets[plane].has(name)) return reject(404);
    try {
      const bytes = await readTrustedAsset(directory, name);
      if (!bytes) return reject(404);
      response.writeHead(200, { ...headers, 'Content-Type': types[extname(name)] || 'application/octet-stream' });
      response.end(request.method === 'HEAD' ? undefined : bytes);
    } catch { reject(404); }
  });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  let reloadTimer;
  const watcher = watch(root, { recursive: true }, (_event, name) => {
    if (!reloadablePath(name)) return;
    clearTimeout(reloadTimer);
    reloadTimer = setTimeout(() => {
      for (const response of listeners) response.write('data: reload\n\n');
    }, 120);
  });
  const servers = [createPreview('remote'), createPreview('local')];
  servers.forEach((server, index) => server.listen(5180 + index, '127.0.0.1', () => {
    console.log(`${index ? 'LOCAL client' : 'Account/operator'} synthetic preview: http://127.0.0.1:${5180 + index}`);
  }));
  for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => {
    for (const response of listeners) response.end();
    servers.forEach(server => server.close());
    watcher.close();
    clearTimeout(reloadTimer);
  });
}
