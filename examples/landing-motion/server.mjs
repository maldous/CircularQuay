import { createReadStream } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { stat } from 'node:fs/promises';
import http from 'node:http';
import { extname, normalize, resolve, relative, sep, isAbsolute } from 'node:path';
import { readFile } from 'node:fs/promises';
import { readSitePage, SITE_PAGE_FILES } from './render-site-header.mjs';

const host = '127.0.0.1';
const port = 5182;
export const root = resolve(import.meta.dirname);
const canonicalHelpRoot = resolve(root, '../../web');
const canonicalHelpNames = Object.freeze(['help-release.js', 'help-search.js', 'help-store.js', 'help-contract.js', 'help.js']);

// The landing preview exposes only these fixed canonical aliases. The landing
// facades remain Node-valid and import these aliases in the browser; the exporter
// materializes both paths for standalone use.
export const canonicalHelpSources = new Map(canonicalHelpNames.map(name => [`web/${name}`, resolve(canonicalHelpRoot, name)]));

// This preview is deliberately a closed, static asset set. Keep documents,
// evidence and repository files outside this allowlist.
export const assets = new Map([
  ['/login', 'login.html'],
  ['/help', 'help.html'],
  ['/help.html', 'help.html'],
  ['/help.css', 'help.css'],
  ['/help-page.js', 'help-page.js'],
  ['/header-search.js', 'header-search.js'],
  ['/header-search.css', 'header-search.css'],
  ['/motion-emphasis.css', 'motion-emphasis.css'],
  ['/assets/flag-en.svg', 'assets/flag-en.svg'],
  ['/assets/flag-ar.svg', 'assets/flag-ar.svg'],
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
  ['/motion-catalog.js', 'motion-catalog.js'],
  ['/explore', 'explore.html'],
  ['/explore.html', 'explore.html'],
  ['/explore.js', 'explore.js'],
  ['/explore.css', 'explore.css'],
  ['/explore-navigation.js', 'explore-navigation.js'],
  ['/explore-navigation.css', 'explore-navigation.css'],
  ['/landing-explore.js', 'landing-explore.js'],
  ['/landing-navigation.js', 'landing-navigation.js'],
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
export function integrationAssetEntries(catalog) {
  const entries = [];
  for (const group of catalog.groups) {
    for (const item of group.items) {
      if (item.asset === null) continue;
      if (!/^assets\/integrations\/[a-z0-9-]+\.svg$/.test(item.asset)) throw new Error('Invalid public integration asset path');
      entries.push([`/${item.asset}`, item.asset]);
    }
  }
  return entries;
}
for (const entry of integrationAssetEntries(catalog)) assets.set(...entry);

// Documentation modules are a revision-bound public snapshot, never a file browser.
const documentationManifest = JSON.parse(await readFile(new URL('explore-content/manifest.json', import.meta.url), 'utf8'));
export function documentationAssetEntries(manifest) {
  const entries = [];
  for (const file of manifest.publicFiles) {
    if (!/^(?:explore-manifest\.js|explore-content\/[a-z0-9-]+\.js|explore-content\/assets\/[a-f0-9]{64}\.(?:svg|png|jpg|webp))$/.test(file.path)) throw new Error('Invalid public documentation asset path');
    entries.push([`/${file.path}`, file.path]);
  }
  return entries;
}
for (const entry of documentationAssetEntries(documentationManifest)) assets.set(...entry);

// Reuse fixed canonical aliases, landing facades, and the generated distribution
// at pinned absolute paths. No directory serving.
const helpDirectory = new URL('client-help/', import.meta.url);
const helpManifest = JSON.parse(await readFile(new URL('generated/manifest.json', helpDirectory), 'utf8'));
const helpCatalog = JSON.parse(await readFile(new URL(`generated/${helpManifest.catalog.path.split('/').pop()}`, helpDirectory), 'utf8'));
for (const name of canonicalHelpNames) {
  assets.set(`/client-help/${name}`, `client-help/${name}`);
  assets.set(`/web/${name}`, `web/${name}`);
}
export function helpAssetEntries(paths) {
  const entries = [];
  for (const path of paths) {
    if (!/^\/help\/generated\/(?:manifest|(?:catalog|search|article-[a-z][a-z0-9.-]{0,100})-[a-f0-9]{64})\.json$/.test(path)) throw new Error('Invalid public help asset path');
    const file = `client-help/generated/${path.split('/').pop()}`;
    entries.push([path, file], [`/${file}`, file]);
  }
  return entries;
}
for (const entry of helpAssetEntries(['/help/generated/manifest.json', helpManifest.catalog.path, helpManifest.search.path, ...helpCatalog.documents.map(item => item.article.path)])) assets.set(...entry);

export const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.svg': 'image/svg+xml; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
  '.xml': 'application/xml; charset=utf-8',
};

export const securityHeaders = {
  'Content-Security-Policy': "default-src 'self'; base-uri 'none'; connect-src 'self'; font-src 'self'; form-action 'none'; frame-ancestors 'none'; img-src 'self'; media-src 'none'; object-src 'none'; script-src 'self'; style-src 'self'",
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

export async function handleRequest(request, response) {
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

  const canonicalFile = canonicalHelpSources.get(asset);
  const file = canonicalFile || resolve(root, normalize(asset));
  const inside = relative(root, file);
  if (!canonicalFile && (inside === '' || inside === '..' || inside.startsWith(`..${sep}`) || isAbsolute(inside))) return reply(response, 403, 'Forbidden');

  try {
    const rendered = SITE_PAGE_FILES.includes(asset) ? await readSitePage(asset) : null;
    const details = rendered ? null : await stat(file);
    if (details && !details.isFile()) return reply(response, 404, 'Not Found');
    const type = contentTypes[extname(file)] || 'application/octet-stream';
    response.writeHead(200, {
      ...securityHeaders,
      'Cache-Control': 'no-store',
      'Content-Length': rendered ? rendered.length : details.size,
      'Content-Type': type,
    });
    if (request.method === 'HEAD') return response.end();
    if (rendered) return response.end(rendered);
    createReadStream(file).pipe(response);
  } catch {
    return reply(response, 404, 'Not Found');
  }
}

export function startLandingPreview({ port: selectedPort = port } = {}) {
  const server = http.createServer(handleRequest);
  server.listen(selectedPort, host, () => {
    process.stdout.write(`CircularQuay landing-motion preview: http://${host}:${selectedPort}/\n`);
  });
  return server;
}

export function stopLandingPreview(server, signal, { exit = code => process.exit(code), schedule = setTimeout, cancel = clearTimeout, write = text => process.stdout.write(text) } = {}) {
  const timeout = schedule(() => exit(1), 5_000);
  timeout.unref();
  server.close(() => { cancel(timeout); exit(0); });
  write(`Received ${signal}; closing landing-motion preview.\n`);
}

// Importing the closed route graph is no longer a request to start a listener.
// Direct CLI still honors the original export-only switch and fixed loopback.
if (process.argv[1] === fileURLToPath(import.meta.url) && !process.env.CQ_EXPORT_ROUTE_MAP) {
  const server = startLandingPreview();
  process.once('SIGINT', () => stopLandingPreview(server, 'SIGINT'));
  process.once('SIGTERM', () => stopLandingPreview(server, 'SIGTERM'));
}
