import { PUBLIC_ASSET_PATHS, PUBLIC_CONTENT_TYPES, PUBLIC_ROUTES } from './.local/worker-routes.mjs';

const securityHeaders = {
  'Content-Security-Policy': "default-src 'self'; base-uri 'none'; connect-src 'self'; font-src 'self'; form-action 'none'; frame-ancestors 'none'; img-src 'self'; media-src 'none'; object-src 'none'; script-src 'self'; style-src 'self'",
  'Cross-Origin-Resource-Policy': 'same-origin',
  'Referrer-Policy': 'no-referrer',
  'X-Content-Type-Options': 'nosniff',
  'X-Frame-Options': 'DENY',
  'X-Robots-Tag': 'noindex, nofollow, noarchive',
  'Cache-Control': 'no-store',
};

const reply = (status, body) => new Response(body, { status, headers: securityHeaders });

export default {
  async fetch(request, env) {
    if (!['GET', 'HEAD'].includes(request.method)) return reply(405, 'Method Not Allowed');

    let url;
    try {
      url = new URL(request.url);
    } catch {
      return reply(400, 'Bad Request');
    }
    if (url.search || url.hash) return reply(400, 'Bad Request');

    let pathname;
    try {
      pathname = decodeURIComponent(url.pathname);
    } catch {
      return reply(400, 'Bad Request');
    }
    if (!PUBLIC_ROUTES.has(pathname)) return reply(404, 'Not Found');

    const assetPath = PUBLIC_ASSET_PATHS.get(pathname);
    const assetUrl = new URL(request.url);
    assetUrl.pathname = assetPath;
    // The asset binding is private. Do not forward visitor conditionals, ranges,
    // credentials, or other request headers; local preview semantics always return
    // the complete allowlisted asset with no-store.
    const assetRequest = new Request(assetUrl, { method: request.method });
    const response = await env.ASSETS.fetch(assetRequest);
    if (!response.ok) return reply(404, 'Not Found');

    const result = new Response(request.method === 'HEAD' ? null : response.body, response);
    for (const [name, value] of Object.entries(securityHeaders)) result.headers.set(name, value);
    result.headers.set('Content-Type', PUBLIC_CONTENT_TYPES.get(pathname));
    return result;
  },
};
