import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, mkdir, readFile, readdir, rm, symlink, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { pathToFileURL } from 'node:url';
import { buildMotionCatalog, isBuildEntry } from './build-motion-catalog.mjs';
import { integrationAttribute, providerChoices, validateIntegrations } from './render-integrations.mjs';
import { renderSitePages } from './render-site-pages.mjs';

async function ownedDirectory(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), 'cq-catalog-contract-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  return { root, url: pathToFileURL(`${root}${path.sep}`) };
}

const digest = bytes => createHash('sha256').update(bytes).digest('hex');
const saveJSON = (root, name, value) => writeFile(path.join(root, name), JSON.stringify(value));

async function motionFixture(t) {
  const owned = await ownedDirectory(t);
  await mkdir(path.join(owned.root, 'assets'));
  await writeFile(path.join(owned.root, 'assets', 'mark.svg'), '<svg/>');
  const item = (id, name) => ({ id, name, asset: 'assets/mark.svg' });
  const catalog = { groups: [
    { id: 'identity', items: [item('login', 'Example OAuth'), item('custom', 'Custom identity')] },
    { id: 'sources', items: [item('source', 'Vault Object Storage')] },
    { id: 'applications', items: [item('command', 'Command'), item('postgresql', 'PostgreSQL'), item('other', 'Other')] },
  ] };
  const config = { shortNames: { custom: 'Chosen' }, applicationPools: { shell: ['command'], cloud: ['postgresql', 'other'] } };
  await saveJSON(owned.root, 'integrations.json', catalog);
  await saveJSON(owned.root, 'motion-catalog.config.json', config);
  return { ...owned, catalog, config };
}

function descriptorPayload(output) {
  const rows = output.match(/const descriptors = (\[[\s\S]*?\n\]);/);
  const pools = output.match(/const pools = (\{[\s\S]*?\n\});/);
  assert.ok(rows && pools);
  return { rows: JSON.parse(rows[1]), pools: JSON.parse(pools[1]) };
}

test('motion builder writes deterministic descriptors and check never changes them', async t => {
  const fixture = await motionFixture(t);
  await buildMotionCatalog(fixture.root);
  const target = path.join(fixture.root, 'motion-catalog.js');
  const output = await readFile(target, 'utf8');
  const { rows, pools } = descriptorPayload(output);
  assert.equal(rows.length, 6);
  assert.deepEqual(pools.identity, pools['identity-2']);
  assert.deepEqual(pools.local, pools.remote);
  assert.equal(rows.find(row => row[0] === 'login')[2], 'Example');
  assert.equal(rows.find(row => row[0] === 'custom')[2], 'Chosen');
  assert.equal(rows.find(row => row[0] === 'source')[2], 'Vault');
  assert.equal(rows.find(row => row[0] === 'command')[7], 'run with Command');
  assert.equal(rows.find(row => row[0] === 'postgresql')[6], 'query a database');
  assert.equal(rows.find(row => row[0] === 'other')[7], 'use Other');
  await buildMotionCatalog(fixture.root, { check: true });
  assert.equal(await readFile(target, 'utf8'), output);
  await writeFile(target, 'owned stale output');
  await assert.rejects(buildMotionCatalog(fixture.root, { check: true }), /motion-catalog.js is stale/);
  assert.equal(await readFile(target, 'utf8'), 'owned stale output');
});

test('motion builder rejects unknown and uncovered applications before output', async t => {
  for (const [name, ids, message] of [
    ['unknown', ['absent'], /Unknown application absent/],
    ['uncovered', ['postgresql'], /Catalog coverage is incomplete or duplicated/],
  ]) {
    await t.test(name, async t => {
      const fixture = await motionFixture(t);
      fixture.config.applicationPools.cloud = ids;
      await saveJSON(fixture.root, 'motion-catalog.config.json', fixture.config);
      await assert.rejects(buildMotionCatalog(fixture.root), message);
      assert.ok(!(await readdir(fixture.root)).includes('motion-catalog.js'));
    });
  }
});

test('motion builder preserves actual missing asset and JSON read refusals', async t => {
  const fixture = await motionFixture(t);
  await rm(path.join(fixture.root, 'assets', 'mark.svg'));
  await assert.rejects(buildMotionCatalog(fixture.root), { code: 'ENOENT' });
  await writeFile(path.join(fixture.root, 'motion-catalog.config.json'), '{');
  await assert.rejects(buildMotionCatalog(fixture.root), SyntaxError);
  assert.ok(!(await readdir(fixture.root)).includes('motion-catalog.js'));
});

function providerMarkup(items) {
  return `<fieldset class="provider-grid" disabled>${items.map(item => `<button class="provider-choice" data-provider="${item.name}"><img src="${item.asset}">${item.name}</button>`).join('')}</fieldset>`;
}

async function inventoryFixture(t) {
  const owned = await ownedDirectory(t);
  await mkdir(path.join(owned.root, 'assets'));
  const assets = [];
  for (let index = 0; index < 108; index += 1) {
    const asset = `assets/mark-${index}.svg`;
    const bytes = `<svg>owned-${index}</svg>`;
    await writeFile(path.join(owned.root, asset), bytes);
    assets.push({ asset, sha256: digest(bytes) });
  }
  let next = 0;
  const catalog = { groups: Object.entries({ identity: 30, sources: 35, applications: 48 }).map(([id, count]) => ({
    id, items: Array.from({ length: count }, () => {
      const number = next++;
      return { id: `item-${number}`, name: `Provider ${number}`, ...assets[number % assets.length] };
    }),
  })) };
  await saveJSON(owned.root, 'integrations.json', catalog);
  await writeFile(path.join(owned.root, 'login.html'), providerMarkup(catalog.groups[0].items));
  return { ...owned, catalog };
}

test('integration validator verifies owned113 entries108 actual hashes30 providers', async t => {
  const fixture = await inventoryFixture(t);
  const before = await readFile(path.join(fixture.root, 'integrations.json'));
  assert.equal(await validateIntegrations(fixture.url), '113 catalog entries/108 SVGs, 30 login providers verified.\n');
  assert.deepEqual(await readFile(path.join(fixture.root, 'integrations.json')), before);
});

test('integration inventory refuses every structural and byte-integrity boundary', async t => {
  const changes = [
    ['count', catalog => catalog.groups[0].items.pop(), /Expected 30 identity entries/],
    ['duplicate', catalog => { catalog.groups[0].items[1].id = catalog.groups[0].items[0].id; }, /Duplicate integration ID/],
    ['incomplete', catalog => { delete catalog.groups[0].items[0].sha256; }, /Incomplete asset record/],
    ['digest', catalog => { catalog.groups[0].items[0].sha256 = '0'.repeat(64); }, /Asset digest mismatch/],
    ['total', catalog => { catalog.groups.pop(); }, /Expected 113 catalog entries, found 65/],
    ['distinct', catalog => { catalog.groups[2].items[42] = { ...catalog.groups[2].items[42], ...catalog.groups[0].items[0], id: 'item-107', name: 'Provider107' }; }, /Expected 108 distinct SVG assets, found 107/],
  ];
  for (const [name, edit, message] of changes) {
    await t.test(name, async t => {
      const fixture = await inventoryFixture(t);
      edit(fixture.catalog);
      await saveJSON(fixture.root, 'integrations.json', fixture.catalog);
      await assert.rejects(validateIntegrations(fixture.url), message);
    });
  }
});

test('integration login requires disabled grid exact count order text and asset', async t => {
  for (const [name, edit, message] of [
    ['enabled', html => html.replace(' disabled', ''), /Missing disabled provider-grid/],
    ['count', html => html.replace(/<button[\s\S]*?<\/button>/, ''), /Expected 30 login providers, found 29/],
    ['name', html => html.replace('data-provider="Provider 0"', 'data-provider="Wrong"'), /Login provider 1 does not match/],
    ['text', html => html.replace('>Provider 0</button>', '>Wrong</button>'), /Login provider 1 does not match/],
    ['asset', html => html.replace('src="assets/mark-0.svg"', 'src="assets/other.svg"'), /Login provider 1 does not match/],
  ]) {
    await t.test(name, async t => {
      const fixture = await inventoryFixture(t);
      const html = providerMarkup(fixture.catalog.groups[0].items);
      await writeFile(path.join(fixture.root, 'login.html'), edit(html));
      await assert.rejects(validateIntegrations(fixture.url), message);
    });
  }
});

test('integration markup parser declines incomplete tags and attribute delimiters', () => {
  for (const html of ['plain text', '<button', '<button class="other"><img src="x">Text', '<button class="provider-choice" data-provider="A"><img', '<button class="provider-choice" data-provider="A">A', '<button class="provider-choice"><img src="x">A', '<button class="provider-choice" data-provider="A"><img>A']) {
    assert.deepEqual(providerChoices(html), []);
  }
  assert.equal(integrationAttribute('<img>', 'src'), undefined);
  assert.equal(integrationAttribute('<img src="unterminated>', 'src'), undefined);
  assert.equal(integrationAttribute('<img src="owned.svg">', 'src'), 'owned.svg');
  assert.deepEqual(providerChoices('<button class="provider-choice" data-provider="A"><img src="a.svg"> A </button>'), [{ name: 'A', asset: 'a.svg', text: 'A' }]);
});

test('site renderer refuses invalid origins before creating any owned output', async t => {
  const owned = await ownedDirectory(t);
  for (const originValue of ['not a URL', 'http://example.test', 'https://user:password@example.test', 'https://example.test/path', 'https://example.test/?query', 'https://example.test/#fragment']) {
    await assert.rejects(renderSitePages(owned.url, { originValue }));
    assert.deepEqual(await readdir(owned.root), []);
  }
});

test('site renderer preserves exact tracked default bytes and detects stale output', async t => {
  const owned = await ownedDirectory(t);
  assert.equal(await renderSitePages(owned.url), 'Rendered 12 static site files.\n');
  const names = await readdir(owned.root);
  assert.equal(names.length, 12);
  for (const name of names) {
    assert.equal(digest(await readFile(new URL(name, owned.url))), defaultPageHashes[name]);
  }
  assert.equal(await renderSitePages(owned.url, { check: true }), 'Checked 12 static site files.\n');
  await writeFile(new URL('footer.html', owned.url), 'owned stale footer');
  await assert.rejects(renderSitePages(owned.url, { check: true }), /footer.html does not match/);
  assert.equal(await readFile(new URL('footer.html', owned.url), 'utf8'), 'owned stale footer');
});

test('site sitemap uses explicit valid HTTPS origin without changing page bytes', async t => {
  const owned = await ownedDirectory(t);
  await renderSitePages(owned.url, { originValue: 'https://example.test:8443/' });
  const xml = await readFile(new URL('sitemap.xml', owned.url), 'utf8');
  assert.equal((xml.match(/<loc>/g) || []).length, 9);
  assert.ok(xml.includes('<loc>https://example.test:8443/</loc>'));
  assert.ok(xml.includes('<loc>https://example.test:8443/status</loc>'));
  assert.equal(digest(await readFile(new URL('about.html', owned.url))), defaultPageHashes['about.html']);
  await renderSitePages(owned.url, { originValue: 'https://example.test:8443/', check: true });
});

test('builders retain actual owned filesystem refusal without replacing directories', async t => {
  const motion = await motionFixture(t);
  await mkdir(path.join(motion.root, 'motion-catalog.js'));
  await assert.rejects(buildMotionCatalog(motion.root));
  assert.deepEqual(await readdir(path.join(motion.root, 'motion-catalog.js')), []);
  const site = await ownedDirectory(t);
  await assert.rejects(renderSitePages(site.url, { check: true }), { code: 'ENOENT' });
  await mkdir(new URL('footer.html', site.url));
  await assert.rejects(renderSitePages(site.url));
  assert.deepEqual(await readdir(site.root), ['footer.html']);
});

test('integration validator propagates owned file reads and malformed catalog refusals', async t => {
  const fixture = await inventoryFixture(t);
  await rm(path.join(fixture.root, 'login.html'));
  await assert.rejects(validateIntegrations(fixture.url), { code: 'ENOENT' });
  await rm(path.join(fixture.root, 'assets', 'mark-0.svg'));
  await assert.rejects(validateIntegrations(fixture.url), { code: 'ENOENT' });
  await writeFile(path.join(fixture.root, 'integrations.json'), '{');
  await assert.rejects(validateIntegrations(fixture.url), SyntaxError);
});

// Exact tracked bbf/tree4d754141 defaults, not generated by this test.
const defaultPageHashes = {
  "footer.html": "11a900bb019bb54c39f978aaa73bdfce151fceeb2f6f2915e7a7b2b965082d85",
  "about.html": "056a571b7e14163a3f587b03453727400b4524b1ccf7a5ab9e3f63b04d2dfb84",
  "contact.html": "31c5b8bef6222113ade524442a963f5130a3ce8084abc9fcbab1139537a1263b",
  "privacy.html": "f57a8a12414aaaa2c57daed5e91509150721de8b0f19552222573bdf09a8a838",
  "terms.html": "9e3b1e88fdcab614122a8fe65fb44d6ab07e293602ccc184523ff321df71a784",
  "cookies.html": "6a558e6ee921226cadd1a8e18168996ae38ef027701caf183fa42a5e7118986c",
  "accessibility.html": "56587d44f227a555aaf96c33646f603150f02a36cf1919a4ca57c73960fabd5a",
  "security.html": "28e5b3d46b06353717b0ae13a4c63e8abfa3136d2bfcb22b33092ca16b5f3ddd",
  "status.html": "5f4f173e36ee31496caafec12c9acfc040363030a0dfc4cf1cf1e5e69f4cc7c8",
  "robots.txt": "a6f7515dfbbcf4fb2edaa4d7576e10b0ab75fb5e3f7d1c8773439dbd25a796d2",
  "sitemap.xml": "e6a6c4b19a138957ff1b5e3034744f941ffaa0cf26523ff0da99da103cd9ce1a",
  "ads.txt": "70c42f3c31712f3d0498651555b20f0b1aa3b138080041bc8a5d97b3efc1063b"
};


test('build entry compares actual owned files, not lexical argv', async t => {
  const { root } = await ownedDirectory(t);
  const entry = path.join(root, 'entry.mjs');
  const imported = path.join(root, 'imported.mjs');
  await writeFile(entry, '// owned entry, never executed\n');
  await writeFile(imported, '// owned import, never executed\n');
  assert.equal(await isBuildEntry(pathToFileURL(entry), entry), true);
  assert.equal(await isBuildEntry(pathToFileURL(imported), entry), false);
  assert.equal(await isBuildEntry(pathToFileURL(entry), ''), false);
  assert.equal(await isBuildEntry(pathToFileURL(entry), path.join(root, 'absent.mjs')), false);
});

test('build entry follows an owned directory symlink in either identity', async t => {
  const { root } = await ownedDirectory(t);
  const target = path.join(root, 'target');
  const alias = path.join(root, 'alias');
  await mkdir(target);
  const entry = path.join(target, 'entry.mjs');
  await writeFile(entry, '// owned entry, never executed\n');
  // Junctions avoid requiring Windows file-symlink privilege; on POSIX this
  // remains an ordinary directory symlink. No child script is executed.
  await symlink(target, alias, 'junction');
  const linkedEntry = path.join(alias, 'entry.mjs');
  assert.equal(await isBuildEntry(pathToFileURL(entry), linkedEntry), true);
  assert.equal(await isBuildEntry(pathToFileURL(linkedEntry), entry), true);
});
