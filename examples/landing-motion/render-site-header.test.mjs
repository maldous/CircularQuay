import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { extname } from 'node:path';
import test from 'node:test';
import {
  composeSitePage, readSitePage, SITE_HEADER_SLOT, SITE_PAGE_FILES,
  MAX_SITE_TEMPLATE_BYTES, MAX_SITE_HEADER_BYTES,
} from './render-site-header.mjs';

// Complete delivered-page and exact non-header byte pins from the pre-extraction
// 913d2d05 source tree. These are not hashes of a newly generated expectation.
const pageBaselines = [
  {
    file: 'index.html', bytes: 16148,
    sha256: '9e7a7d012f64db3df313b53aeee1a9879c761d833c32083162ea294dacc6c24f',
    prefixBytes: 1673, prefixSha256: 'c601df75d9203791ed329793931e44994d9b83a0b93a3226ce0f2e54cc7e092d',
    suffixBytes: 11650, suffixSha256: '550c7c988d225724f2b860b59361fbdcda7259ffa47a0115b7ea8a5eed481885',
  },
  {
    file: 'about.html', bytes: 4681,
    sha256: '7bd59a3c9c3fddf142848e3042add97f22ddd1d9c2b5c818ca931cb05716faed',
    prefixBytes: 908, prefixSha256: '5c525ea0a8e660513cbd658840f30db17c718778bbd696ba2341f162357c1035',
    suffixBytes: 952, suffixSha256: '4a53e48bfa41ed96ffc613274de4bf583bdb116e2b8627ace62fbf696470c263',
  },
  {
    file: 'accessibility.html', bytes: 4658,
    sha256: '069e6498034d5bd0c66f5b8d5e616214526f043e7b6cdb072249a0dadb922f9b',
    prefixBytes: 898, prefixSha256: 'a7cde441f1c91f1f49ff6dd7a157ad15d18d274457e4ee6daa23471e97335419',
    suffixBytes: 939, suffixSha256: '054e9183317baf272365bbb8bc2c4a815fb6fd8a5ac3f2dfe1e1e4a74ae457bd',
  },
  {
    file: 'contact.html', bytes: 4634,
    sha256: 'ff615c628015e5171e07551b8de73c662251587dfdb6c669000b344f80291f90',
    prefixBytes: 886, prefixSha256: 'a133faf1039f180a8d6403101ce757939bba6066e460b27f4037e8f7543bce4f',
    suffixBytes: 927, suffixSha256: '15cccb11bc3a4a4f59652e39184f255871b6132330881db72eb323131e0bf10e',
  },
  {
    file: 'cookies.html', bytes: 4633,
    sha256: '03b5899823c87ec5244e505959ac31ec54067a63348ec3b4999004b5a1e4595a',
    prefixBytes: 886, prefixSha256: 'ae8f3c3484b595bec7ae7633ff8fa0f550f437661464c6a14c837d5f507e2bd2',
    suffixBytes: 926, suffixSha256: 'f77a98b4fa6fc1d30d21918333eb4c9c1a2272100ba1e5b28a4fa4188bf20ea7',
  },
  {
    file: 'privacy.html', bytes: 4649,
    sha256: 'd5e896e39a84b6ecafb9bfbaa79f6e28fa1bfff6ff0b660ab8bfcb1e934e9315',
    prefixBytes: 886, prefixSha256: '0ab73a1d37e0f791da4217105e24d8f793cddbc6a8d02039b80aeb0a671b464a',
    suffixBytes: 942, suffixSha256: 'f194915f13d22b4b60732cf13ab0fec051f3adf0abfa64568747a4352bb0b135',
  },
  {
    file: 'security.html', bytes: 4629,
    sha256: 'add6f16bda1d82ee437c0b0584c6b6abc897950634cd64be0de4306a61f2a3d2',
    prefixBytes: 888, prefixSha256: '90a95007cf42068c5e9748a01be0a999fb160454082aecf64deffb9e12e6d9a2',
    suffixBytes: 920, suffixSha256: '1b067b44866de2ea4413325762a2c95ac5d4bb5ac4b9992a5c748e51fef5ef21',
  },
  {
    file: 'status.html', bytes: 4621,
    sha256: '02e8c8830af09332352c39929b335bd763536ed87a5404b73785c584607afba0',
    prefixBytes: 884, prefixSha256: '1ad1186c788fadd32621d064e50d4eab3a4d70ba967a3d488b1708b7041a9b63',
    suffixBytes: 916, suffixSha256: '3f9d3c7bf25d0a1ea2f5ff9b7ea48e9b8c800e288c2425aedae1737b06b2e689',
  },
  {
    file: 'terms.html', bytes: 4591,
    sha256: 'f88438375deef215e82fba6a70e075225e697540e70565cfd68ffc66086c1934',
    prefixBytes: 882, prefixSha256: '7052ac5334dceff3cdde534503e1267c239a808aae319d7a2ac3a13fae78f7af',
    suffixBytes: 888, suffixSha256: 'd3790f34e1fe36694b1cd6390a155df2c4f400884527255f437dc7811fc6a4cd',
  },
  {
    file: 'explore.html', bytes: 5935,
    sha256: '83c977beb79290ad9c76802a6e9c552c164486f3ecd8d921331f01aa109a251d',
    prefixBytes: 1156, prefixSha256: '33eb59dc0c5a3e19bc7a212d99f21037fcbd8f02e8ff35d982696fae4fa043ed',
    suffixBytes: 1958, suffixSha256: '4df1b6b41fcc0497238d09a449090e56d8e14acd90eeca4a2d9781bfa6888dae',
  },
  {
    file: 'help.html', bytes: 4735,
    sha256: '0f1f45851bdf78603b5a656e8a832716f358923f946060a81f3519c20dd79a03',
    prefixBytes: 1054, prefixSha256: '4d163265d0492c749dbcdf587d48680f7134aaf77f68e0b8e5769436c3153422',
    suffixBytes: 860, suffixSha256: '7de2451e6e775ee3c885306092d2e9e81c4141913666dff9823f960c7ec2990a',
  },
  {
    file: 'login.html', bytes: 10913,
    sha256: 'd5914c4b614e9abec1c11ba034a153d80bcf5b1696565341adee80aa67e324b0',
    prefixBytes: 1054, prefixSha256: '71826c9a85cc0fd8f85cee5b2b3d4671d947d64135f09507e94e0e731164b8e6',
    suffixBytes: 7038, suffixSha256: '636ac61eea1cd49943c61bca1ac4a38acf3081d476f47753649ccc570af511d3',
  },
];
const expectedPageFiles = pageBaselines.map(page => page.file);
const header = await readFile(new URL('site-header.html', import.meta.url));
const slot = '<!-- cq:site-header -->';
const openingHeader = '<header class="site-header page-width has-help-search">';
const closingHeader = '</header>';
const template = Buffer.from(`<main>before</main>${slot}<footer>after</footer>`);
const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
const compareCodeUnits = (left, right) => left < right ? -1 : left > right ? 1 : 0;

async function loadServerWithoutListening() {
  const previous = process.env.CQ_EXPORT_ROUTE_MAP;
  process.env.CQ_EXPORT_ROUTE_MAP = '1';
  try {
    return await import('./server.mjs');
  } finally {
    if (previous === undefined) delete process.env.CQ_EXPORT_ROUTE_MAP;
    else process.env.CQ_EXPORT_ROUTE_MAP = previous;
  }
}

const server = await loadServerWithoutListening();

function responseFixture() {
  return {
    status: null, headers: null, body: Buffer.alloc(0), endCalls: 0,
    writeHead(status, headers) { this.status = status; this.headers = headers; },
    end(body = '') { this.body = Buffer.from(body); this.endCalls += 1; },
  };
}

async function dispatch(url, method = 'GET') {
  const response = responseFixture();
  await server.handleRequest({ method, url }, response);
  assert.equal(response.endCalls, 1, `${method} ${url} ends once`);
  return response;
}

function assertRenderedPage(bytes, baseline) {
  assert.ok(Buffer.isBuffer(bytes), baseline.file);
  assert.equal(bytes.length, baseline.bytes, baseline.file);
  assert.equal(sha256(bytes), baseline.sha256, baseline.file);
  assert.equal(bytes.includes(Buffer.from(slot)), false, baseline.file);
}

function paddedBuffer(prefix, suffix, size) {
  const first = Buffer.from(prefix), last = Buffer.from(suffix);
  return Buffer.concat([first, Buffer.alloc(size - first.length - last.length, 'x'), last]);
}

test('site-header renderer exposes only the fixed page and byte-bound contract', () => {
  assert.equal(SITE_HEADER_SLOT, slot);
  assert.deepEqual(SITE_PAGE_FILES, expectedPageFiles);
  assert.equal(Object.isFrozen(SITE_PAGE_FILES), true);
  assert.equal(MAX_SITE_TEMPLATE_BYTES, 65_536);
  assert.equal(MAX_SITE_HEADER_BYTES, 8_192);
  assert.equal(header.length, 2825);
  assert.equal(sha256(header), 'fd81f00c24b18653789e23d19e39f569c8c2b370dbedc999889ef94042979097');
});

test('all twelve composed pages preserve every old delivered byte and non-header region', async () => {
  for (const baseline of pageBaselines) {
    const source = await readFile(new URL(baseline.file, import.meta.url));
    const sourceCopy = Buffer.from(source), headerCopy = Buffer.from(header);
    const marker = source.indexOf(slot);
    assert.equal(marker, baseline.prefixBytes, baseline.file);
    assert.equal(source.lastIndexOf(slot), marker, baseline.file);
    const prefix = source.subarray(0, marker);
    const suffix = source.subarray(marker + Buffer.byteLength(slot));
    assert.equal(sha256(prefix), baseline.prefixSha256, baseline.file);
    assert.equal(suffix.length, baseline.suffixBytes, baseline.file);
    assert.equal(sha256(suffix), baseline.suffixSha256, baseline.file);
    const composed = composeSitePage(baseline.file, source, header);
    assertRenderedPage(composed, baseline);
    assert.notEqual(composed, source);
    assert.notEqual(composed, header);
    assert.deepEqual(composed.subarray(0, prefix.length), prefix, baseline.file);
    assert.deepEqual(composed.subarray(composed.length - suffix.length), suffix, baseline.file);
    assert.deepEqual(await readSitePage(baseline.file), composed, baseline.file);
    assert.deepEqual(source, sourceCopy, baseline.file);
    assert.deepEqual(header, headerCopy, baseline.file);
  }
});

test('composition applies only the three fixed nonlanding href substitutions', () => {
  const literal = Buffer.from(`${openingHeader}<a href="#main">Home</a><a href="#scene">Flow</a><a href="#ecosystem">Sources</a><span data-value="#scene">#main</span>${closingHeader}`);
  const source = Buffer.from(`unchanged:#main:${slot}:unchanged:#ecosystem`);
  const index = composeSitePage('index.html', source, literal);
  assert.deepEqual(index, Buffer.concat([Buffer.from('unchanged:#main:'), literal, Buffer.from(':unchanged:#ecosystem')]));
  const otherHeader = Buffer.from(`${openingHeader}<a href="/">Home</a><a href="/#scene">Flow</a><a href="/#ecosystem">Sources</a><span data-value="#scene">#main</span>${closingHeader}`);
  for (const file of expectedPageFiles.slice(1)) {
    assert.deepEqual(composeSitePage(file, source, literal), Buffer.concat([Buffer.from('unchanged:#main:'), otherHeader, Buffer.from(':unchanged:#ecosystem')]), file);
  }
});

test('composition returns an independent buffer without sharing mutable input bytes', () => {
  const source = Buffer.from(`before${slot}after`), fragment = Buffer.from(header);
  const sourceCopy = Buffer.from(source), fragmentCopy = Buffer.from(fragment);
  const expected = Buffer.concat([Buffer.from('before'), fragmentCopy, Buffer.from('after')]);
  const first = composeSitePage('index.html', source, fragment);
  assert.deepEqual(first, expected);
  first.fill(0x78);
  assert.deepEqual(source, sourceCopy);
  assert.deepEqual(fragment, fragmentCopy);
  const second = composeSitePage('index.html', source, fragment);
  assert.notEqual(second, first);
  assert.deepEqual(second, expected);
});

test('unknown names are refused without coercion before input validation or file selection', async () => {
  const unknownNames = [null, undefined, 1, {}, [], Buffer.from('index.html'), 'index', 'INDEX.html', '/index.html', '../index.html', 'index.html?x', 'index.html#x', 'site-header.html', 'render-site-header.mjs'];
  for (const name of unknownNames) {
    assert.throws(() => composeSitePage(name, null, null), { name: 'Error', message: 'Unknown site page' });
    await assert.rejects(readSitePage(name), { name: 'Error', message: 'Unknown site page' });
  }
  const noCoercion = { toString() { assert.fail('Page names must not be coerced'); } };
  assert.throws(() => composeSitePage(noCoercion, template, header), { name: 'Error', message: 'Unknown site page' });
});

test('both composition inputs must be buffers before size or text validation', () => {
  for (const value of [null, undefined, '', slot, [], new Uint8Array([1])]) {
    assert.throws(() => composeSitePage('index.html', value, header), { name: 'TypeError', message: 'Site page inputs must be buffers' });
    assert.throws(() => composeSitePage('index.html', template, value), { name: 'TypeError', message: 'Site page inputs must be buffers' });
  }
  const oversized = Buffer.alloc(MAX_SITE_TEMPLATE_BYTES + 1);
  assert.throws(() => composeSitePage('index.html', oversized, null), { name: 'TypeError', message: 'Site page inputs must be buffers' });
});

test('exact UTF-8 byte limits are accepted and one extra byte fails before decoding', () => {
  const maximumTemplate = paddedBuffer(slot, '€', MAX_SITE_TEMPLATE_BYTES);
  const maximumHeader = paddedBuffer(openingHeader, `€${closingHeader}`, MAX_SITE_HEADER_BYTES);
  const beforeTemplate = Buffer.from(maximumTemplate), beforeHeader = Buffer.from(maximumHeader);
  const result = composeSitePage('index.html', maximumTemplate, maximumHeader);
  assert.deepEqual(result, Buffer.concat([maximumHeader, maximumTemplate.subarray(Buffer.byteLength(slot))]));
  assert.equal(result.length, MAX_SITE_TEMPLATE_BYTES - Buffer.byteLength(slot) + MAX_SITE_HEADER_BYTES);
  assert.deepEqual(maximumTemplate, beforeTemplate);
  assert.deepEqual(maximumHeader, beforeHeader);
  for (const [source, fragment] of [
    [Buffer.concat([maximumTemplate, Buffer.from('x')]), header],
    [template, Buffer.concat([maximumHeader, Buffer.from('x')])],
    [Buffer.alloc(MAX_SITE_TEMPLATE_BYTES + 1, 0xff), Buffer.alloc(0)],
    [template, Buffer.alloc(MAX_SITE_HEADER_BYTES + 1, 0xff)],
  ]) {
    assert.throws(() => composeSitePage('index.html', source, fragment), { name: 'RangeError', message: 'Site page source exceeds size limit' });
  }
});

test('malformed UTF-8 and NUL bytes fail before slot and header interpretation', () => {
  const invalid = Buffer.from([0xc3, 0x28, 0]);
  assert.throws(() => composeSitePage('index.html', invalid, header), { name: 'Error', message: 'Site page inputs must be UTF-8' });
  assert.throws(() => composeSitePage('index.html', template, invalid), { name: 'Error', message: 'Site page inputs must be UTF-8' });
  const withNul = Buffer.concat([template, Buffer.from([0])]);
  const headerWithNul = Buffer.from(`${openingHeader}\0${closingHeader}`);
  for (const [source, fragment] of [[withNul, header], [template, headerWithNul]]) {
    const sourceCopy = Buffer.from(source), fragmentCopy = Buffer.from(fragment);
    assert.throws(() => composeSitePage('index.html', source, fragment), { name: 'Error', message: 'Site page inputs contain NUL' });
    assert.deepEqual(source, sourceCopy);
    assert.deepEqual(fragment, fragmentCopy);
  }
});

test('templates require one literal slot and no pre-existing header before fragment validation', () => {
  for (const source of ['<main>missing</main>', `${slot}${slot}`, `${slot}${slot}${slot}`]) {
    assert.throws(() => composeSitePage('index.html', Buffer.from(source), Buffer.alloc(0)), { name: 'Error', message: 'Expected one site-header slot' });
  }
  for (const tag of ['<header>', '<HEADER class="existing">']) {
    assert.throws(() => composeSitePage('index.html', Buffer.from(`${slot}${tag}</header>`), Buffer.alloc(0)), { name: 'Error', message: 'Template contains a site header' });
  }
});

test('literal fragments refuse surrounding bytes, invalid wrappers, multiple headers and recursion', () => {
  for (const fragment of [
    '', '<header>wrong class</header>', ` ${header}`, `${header}\n`, `${header}tail`,
    `${openingHeader}<header>nested</header>${closingHeader}`,
    `${openingHeader}${closingHeader}${closingHeader}`,
    `${openingHeader}${slot}${closingHeader}`,
  ]) {
    assert.throws(() => composeSitePage('index.html', template, Buffer.from(fragment)), { name: 'Error', message: 'Invalid literal site header' });
  }
});

test('rendered pages retain complete no-JS navigation and all original locale message keys', async () => {
  const index = (await readSitePage('index.html')).toString('utf8');
  const keys = [...new Set([...index.matchAll(/data-i18n="([^"]+)"/g)].map(match => match[1]))].sort();
  assert.deepEqual(keys, ['download', 'explore', 'localFile', 'login', 'navFlow', 'policyScope', 'roleAuth', 'roleCI', 'roleCloud', 'roleContainer', 'roleLocal', 'roleRemote', 'roleShell', 'title1', 'title2']);
  for (const file of expectedPageFiles) {
    const html = (await readSitePage(file)).toString('utf8');
    assert.match(html, /<nav aria-label="Main navigation"><a href="\/explore#doc=docs-architecture&amp;part=part-1"/);
    assert.match(html, /<a class="explore-action" href="\/explore">/);
    assert.match(html, /<a class="download-action" href="https:\/\/github.com\/maldous\/CircularQuay\/releases\/latest">/);
    assert.match(html, /<a href="\/login" id="login"/);
    assert.match(html, /<input id="site-search" type="text"[^>]*role="combobox"/);
    assert.match(html, /<ul id="site-search-results"[^>]*role="listbox"/);
  }
});

test('shared composition preserves all public route, MIME and security-header identities', () => {
  const triplets = [...server.assets].map(([route, file]) => [route, file, server.contentTypes[extname(file)] || 'application/octet-stream']);
  triplets.sort(([left], [right]) => compareCodeUnits(left, right));
  assert.equal(triplets.length, 902);
  assert.equal(new Set(triplets.map(([, file]) => file)).size, 554);
  assert.equal(sha256(JSON.stringify(triplets)), '33246fe6f2167bd30a5363593cf1eb16cc860a09e1880baf9ff51acb59d8c4c9');
  const htmlFiles = [...new Set(triplets.map(([, file]) => file).filter(file => file.endsWith('.html')))].sort();
  assert.deepEqual(htmlFiles, [...expectedPageFiles].sort());
  const headers = Object.entries(server.securityHeaders).sort(([left], [right]) => compareCodeUnits(left, right));
  assert.equal(sha256(JSON.stringify(headers)), '9fc74fa776bc9ffee54c1df6b93448d0cc69bce229105820a8600ed869d25522');
});

test('preview GET and HEAD use complete rendered bytes and lengths on every HTML alias', async () => {
  const byFile = new Map(pageBaselines.map(page => [page.file, page]));
  for (const [route, file] of server.assets) {
    if (!byFile.has(file)) continue;
    const get = await dispatch(route), head = await dispatch(route, 'HEAD');
    assert.equal(get.status, 200, route);
    assert.equal(head.status, 200, route);
    assertRenderedPage(get.body, byFile.get(file));
    assert.equal(head.body.length, 0, route);
    assert.equal(get.headers['Content-Length'], byFile.get(file).bytes, route);
    assert.equal(head.headers['Content-Length'], get.headers['Content-Length'], route);
    assert.equal(get.headers['Content-Type'], 'text/html; charset=utf-8', route);
    assert.equal(get.headers['Cache-Control'], 'no-store', route);
    assert.deepEqual(head.headers, get.headers, route);
    for (const [name, value] of Object.entries(server.securityHeaders)) assert.equal(get.headers[name], value, route);
  }
});

test('preview still refuses private composition sources, methods and invalid request URLs', async () => {
  for (const path of ['/site-header.html', '/render-site-header.mjs', '/render-site-header.test.mjs', '/README.md', '/integrations.json', '/explore-source/README.md', '/.local/public-upload-manifest.json']) {
    assert.equal((await dispatch(path)).status, 404, path);
  }
  assert.equal((await dispatch('/', 'POST')).status, 405);
  assert.equal((await dispatch('/?query=header')).status, 400);
  assert.equal((await dispatch('/%')).status, 400);
  assert.equal((await dispatch('/#fragment')).status, 400);
  const encoded = await dispatch('/%69ndex.html');
  assert.equal(encoded.status, 200);
  assertRenderedPage(encoded.body, pageBaselines[0]);
});
