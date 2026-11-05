import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';
import http from 'node:http';
import { extname, normalize, resolve } from 'node:path';
import { readFile } from 'node:fs/promises';

const host = '127.0.0.1';
const port = 5182;
const root = resolve(import.meta.dirname);

// This preview is deliberately a closed, static asset set. Keep documents,
// evidence and repository files outside this allowlist.
const assets = new Map([
  ['/login', 'login.html'],
  ['/login.html', 'login.html'],
  ['/about', 'about.html'],
  ['/about.html', 'about.html'],
  ['/contact', 'contact.html'],
  ['/contact.html', 'contact.html'],
  ['/privacy', 'privacy.html'],
  ['/privacy.html', 'privacy.html'],
  ['/terms', 'terms.html'],
  ['/terms.html', 'terms.html'],
  ['/cookies', 'cookies.html'],
  ['/cookies.html', 'cookies.html'],
  ['/accessibility', 'accessibility.html'],
  ['/accessibility.html', 'accessibility.html'],
  ['/security', 'security.html'],
  ['/security.html', 'security.html'],
  ['/status', 'status.html'],
  ['/status.html', 'status.html'],
  ['/', 'index.html'],
  ['/index.html', 'index.html'],
  ['/style.css', 'style.css'],
  ['/motion.js', 'motion.js'],
  ['/scenario-flow.js', 'scenario-flow.js'],
  ['/explore', 'explore.html'],
  ['/explore.html', 'explore.html'],
  ['/explore.js', 'explore.js'],
  ['/explore.css', 'explore.css'],
  ['/explore-navigation.js', 'explore-navigation.js'],
  ['/explore-navigation.css', 'explore-navigation.css'],
  ['/landing-explore.js', 'landing-explore.js'],
  ['/appearance.js', 'appearance.js'],
  ['/locale.js', 'locale.js'],
  ['/browser-services.js', 'browser-services.js'],
  ['/service-config.js', 'service-config.js'],
  ['/site-services.js', 'site-services.js'],
  ['/login.js', 'login.js'],
  ['/login.css', 'login.css'],
  ['/robots.txt', 'robots.txt'],
  ['/sitemap.xml', 'sitemap.xml'],
  ['/ads.txt', 'ads.txt'],
  ['/preference-bootstrap.js', 'preference-bootstrap.js'],
  ['/assets/logo-light.svg', 'assets/logo-light.svg'],
  ['/assets/logo-dark.svg', 'assets/logo-dark.svg'],
]);

// Only declared local integration SVGs are public; catalogs, credits and
// evidence remain outside the HTTP asset surface.
const catalog = JSON.parse(await readFile(new URL('integrations.json', import.meta.url), 'utf8'));
for (const group of catalog.groups) {
  for (const item of group.items) {
    if (item.asset === null) continue;
    if (!/^assets\/integrations\/[a-z0-9-]+\.svg$/.test(item.asset)) {
      throw new Error('Invalid public integration asset path');
    }
    assets.set(`/${item.asset}`, item.asset);
  }
}

// Documentation modules are a revision-bound public snapshot, never a file browser.
const documentationManifest = JSON.parse(await readFile(new URL('explore-content/manifest.json', import.meta.url), 'utf8'));
for (const file of documentationManifest.publicFiles) {
  if (!/^(?:explore-manifest\.js|explore-content\/[a-z0-9-]+\.js|explore-content\/assets\/[a-f0-9]{64}\.(?:svg|png|jpg|webp))$/.test(file.path)) {
    throw new Error('Invalid public documentation asset path');
  }
  assets.set(`/${file.path}`, file.path);
}

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.svg': 'image/svg+xml; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
  '.xml': 'application/xml; charset=utf-8',
};

const securityHeaders = {
  'Content-Security-Policy': "default-src 'self'; base-uri 'none'; connect-src 'none'; font-src 'self'; form-action 'none'; frame-ancestors 'none'; img-src 'self'; media-src 'none'; object-src 'none'; script-src 'self'; style-src 'self'",
  'Cross-Origin-Resource-Policy': 'same-origin',
  'Referrer-Policy': 'no-referrer',
  'X-Content-Type-Options': 'nosniff',
  'X-Frame-Options': 'DENY',
  'X-Robots-Tag': 'noindex, nofollow, noarchive',
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
