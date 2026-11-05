import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';
import http from 'node:http';
import { extname, normalize, resolve } from 'node:path';

const host = '127.0.0.1';
const port = 5182;
const root = resolve(import.meta.dirname);

// This preview is deliberately a closed, static asset set. Keep documents,
// evidence and repository files outside this allowlist.
const assets = new Map([
  ['/', 'index.html'],
  ['/index.html', 'index.html'],
  ['/style.css', 'style.css'],
  ['/motion.js', 'motion.js'],
  ['/preference-bootstrap.js', 'preference-bootstrap.js'],
  ['/assets/logo-light.svg', 'assets/logo-light.svg'],
  ['/assets/logo-dark.svg', 'assets/logo-dark.svg'],
]);

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.svg': 'image/svg+xml; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
};

const securityHeaders = {
  'Content-Security-Policy': "default-src 'self'; base-uri 'none'; connect-src 'none'; font-src 'self'; form-action 'none'; frame-ancestors 'none'; img-src 'self'; media-src 'none'; object-src 'none'; script-src 'self'; style-src 'self'",
  'Cross-Origin-Resource-Policy': 'same-origin',
  'Referrer-Policy': 'no-referrer',
  'X-Content-Type-Options': 'nosniff',
  'X-Frame-Options': 'DENY',
};

function reply(response, status, body = '') {
  response.writeHead(status, securityHeaders);
  response.end(body);
}

const server = http.createServer(async (request, response) => {
  if (!['GET', 'HEAD'].includes(request.method || '')) return reply(response, 405, 'Method Not Allowed');

  let pathname;
  try {
    const url = new URL(request.url || '/', `http://${host}:${port}`);
    if (url.search || url.hash) return reply(response, 400, 'Query strings are not supported');
    pathname = decodeURIComponent(url.pathname);
  } catch {
    return reply(response, 400, 'Bad Request');
  }

  const asset = assets.get(pathname);
  if (!asset) return reply(response, 404, 'Not Found');

  const file = resolve(root, normalize(asset));
  if (!file.startsWith(`${root}/`)) return reply(response, 403, 'Forbidden');

  try {
    const details = await stat(file);
    if (!details.isFile()) return reply(response, 404, 'Not Found');
    const type = contentTypes[extname(file)] || 'application/octet-stream';
    response.writeHead(200, {
      ...securityHeaders,
      'Cache-Control': 'no-store',
      'Content-Length': details.size,
      'Content-Type': type,
    });
    if (request.method === 'HEAD') return response.end();
    createReadStream(file).pipe(response);
  } catch {
    return reply(response, 404, 'Not Found');
  }
});

server.listen(port, host, () => {
  process.stdout.write(`CircularQuay landing-motion preview: http://${host}:${port}/\n`);
});

function stop(signal) {
  server.close(() => process.exit(0));
  setTimeout(() => process.exit(1), 5_000).unref();
  process.stdout.write(`Received ${signal}; closing landing-motion preview.\n`);
}

process.once('SIGINT', () => stop('SIGINT'));
process.once('SIGTERM', () => stop('SIGTERM'));
