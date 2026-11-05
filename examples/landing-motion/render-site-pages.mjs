import { readFile, writeFile } from 'node:fs/promises';

const directory = new URL('.', import.meta.url);
const originValue = process.env.PUBLIC_SITE_ORIGIN;
const origin = originValue ? new URL(originValue) : null;
if (origin && (origin.protocol !== 'https:' || origin.username || origin.password || origin.search || origin.hash || origin.pathname !== '/')) {
  throw new Error('PUBLIC_SITE_ORIGIN must be an https origin with no path, query, fragment, or credentials');
}

const links = [['About', '/about'], ['Contact', '/contact'], ['Privacy', '/privacy'], ['Terms', '/terms'], ['Cookies', '/cookies'], ['Accessibility', '/accessibility'], ['Security', '/security'], ['Status', '/status']];
const pages = {
  about: ['About CircularQuay', 'About CircularQuay — TBC', 'CircularQuay is a local design concept for bringing identity, encrypted sources and applications together through device policy. Product, company and service copy remain under review.'],
  contact: ['Contact', 'Contact — TBC', 'A production contact route, responsible contact details and response process are still to be confirmed. This local preview publishes no email address or intake endpoint.'],
  privacy: ['Privacy', 'Privacy information — TBC', 'A production privacy notice needs approved data, controller, retention and jurisdiction details. This local preview has no connected service, account or telemetry endpoint.'],
  terms: ['Terms', 'Terms — TBC', 'Terms of use, service scope and jurisdiction require approved production copy. This local design preview is not an offer of service.'],
  cookies: ['Cookies', 'Cookie information — TBC', 'Cookie choices and any production consent requirements are pending review. This preview does not present a consent banner or a functional preference manager.'],
  accessibility: ['Accessibility', 'Accessibility statement — TBC', 'A production accessibility statement requires an approved scope, standard, assessment and contact path. Device and assistive-technology qualification remain pending.'],
  security: ['Security', 'Security information — TBC', 'Security commitments, reporting routes and response expectations require approved production details. This preview makes no security-service promise.'],
  status: ['Status', 'Service status — TBC', 'A production status service and incident communications process have not been approved. This local preview has no operational service status to report.'],
};

const footer = `<footer class="site-footer page-width" lang="en">\n  <p class="footer-brand">© CircularQuay</p>\n  <nav aria-label="Footer navigation — scroll horizontally for more links">${links.map(([label, route]) => `<a href="${route}">${label}</a>`).join('')}<a href="/cookies">Privacy settings</a></nav>\n</footer>\n`;

const landing = await readFile(new URL('index.html', directory), 'utf8');
const headerMatch = landing.match(/<header class="site-header page-width">[\s\S]*?<\/header>/);
if (!headerMatch) throw new Error('Could not find the current landing-page header');
const header = `<!-- site-header:start -->\n${headerMatch[0]
  .replace('href="#main"', 'href="/"')
  .replaceAll('href="#scene"', 'href="/#scene"')
  .replaceAll('href="#ecosystem"', 'href="/#ecosystem"')}\n<!-- site-header:end -->`;

function page(slug, [label, heading, description]) {
  return `<!doctype html>\n<html lang="en" dir="ltr" data-theme="light">\n<head>\n  <meta charset="utf-8">\n  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">\n  <meta name="robots" content="noindex,nofollow">\n  <meta name="description" content="${label} information for the CircularQuay local design preview. Content pending production review.">\n  <title>${label} | CircularQuay</title>\n  <link rel="icon" href="assets/logo-light.svg" type="image/svg+xml">\n  <script src="preference-bootstrap.js"></script>\n  <link rel="stylesheet" href="style.css">\n  <script type="module" src="appearance.js"></script>\n  <script type="module" src="locale.js"></script>\n</head>\n<body>\n  <a class="skip-link" href="#main">Skip to content</a>\n  ${header}\n  <main id="main" class="information-page page-width" lang="en" dir="ltr">\n    <h1>${heading}</h1>\n    <p>To be completed before production.</p>\n    <p>${description}</p>\n    <p>Content, ownership and publication approval are pending.</p>\n  </main>\n  ${footer}\n</body>\n</html>\n`;
}

const outputs = {
  'footer.html': footer,
  ...Object.fromEntries(Object.entries(pages).map(([slug, content]) => [`${slug}.html`, page(slug, content)])),
  'robots.txt': 'User-agent: *\nDisallow: /\n# Local design preview only; public crawl policy is pending domain and content review.\n',
  'sitemap.xml': origin ? `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n  <url><loc>${origin.origin}/</loc></url>\n${Object.keys(pages).map(slug => `  <url><loc>${origin.origin}/${slug}</loc></url>`).join('\n')}\n</urlset>\n` : '<?xml version="1.0" encoding="UTF-8"?>\n<!-- Sitemap pending a reviewed public HTTPS origin. -->\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>\n',
  'ads.txt': '# ads.txt pending a reviewed publisher relationship; no advertising records are declared for this local preview.\n',
};

for (const [name, content] of Object.entries(outputs)) {
  const target = new URL(name, directory);
  if (process.argv.includes('--check')) {
    if (await readFile(target, 'utf8') !== content) throw new Error(`${name} does not match render-site-pages.mjs`);
  } else await writeFile(target, content);
}
process.stdout.write(`${process.argv.includes('--check') ? 'Checked' : 'Rendered'} ${Object.keys(outputs).length} static site files.\n`);
