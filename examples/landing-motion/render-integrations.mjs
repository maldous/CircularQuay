import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';

const directory = new URL('.', import.meta.url);
const catalog = JSON.parse(await readFile(new URL('integrations.json', directory), 'utf8'));
const expectedCounts = { identity: 30, sources: 35, applications: 48 };

// Legacy catalog tooling is intentionally retained as a validator. The full source inventory
// and local marks may be reused later, but the current landing no longer renders the catalog.
async function validateInventory() {
  const ids = new Set();
  const assets = new Set();
  let total = 0;
  for (const group of catalog.groups) {
    if (group.items.length !== expectedCounts[group.id]) throw new Error(`Expected ${expectedCounts[group.id]} ${group.id} entries`);
    for (const item of group.items) {
      if (ids.has(item.id)) throw new Error(`Duplicate integration ID: ${item.id}`);
      ids.add(item.id); total++;
      if (!item.asset || !item.sha256) throw new Error(`Incomplete asset record: ${item.id}`);
      const bytes = await readFile(new URL(item.asset, directory));
      if (createHash('sha256').update(bytes).digest('hex') !== item.sha256) throw new Error(`Asset digest mismatch: ${item.id}`);
      assets.add(item.asset);
    }
  }
  if (total !== 113) throw new Error(`Expected 113 catalog entries, found ${total}`);
  if (assets.size !== 108) throw new Error(`Expected 108 distinct SVG assets, found ${assets.size}`);
  return { total, assets };
}

async function validateLoginProviders() {
  const login = await readFile(new URL('login.html', directory), 'utf8');
  const fieldset = login.match(/<fieldset\b(?=[^>]*\bclass="provider-grid")(?=[^>]*\bdisabled\b)[^>]*>([\s\S]*?)<\/fieldset>/);
  if (!fieldset) throw new Error('Missing disabled provider-grid fieldset in login.html');
  const providers = [...fieldset[1].matchAll(/<button\b[^>]*\bclass="provider-choice"[^>]*\bdata-provider="([^"]+)"[^>]*>\s*<img\b[^>]*\bsrc="([^"]+)"[^>]*>\s*([^<]+?)<\/button>/g)]
    .map(([, name, asset, text]) => ({ name, asset, text: text.trim() }));
  const identity = catalog.groups.find(group => group.id === 'identity').items;
  if (providers.length !== 30) throw new Error(`Expected 30 login providers, found ${providers.length}`);
  identity.forEach((item, index) => {
    const provider = providers[index];
    if (!provider || provider.name !== item.name || provider.text !== item.name || provider.asset !== item.asset) {
      throw new Error(`Login provider ${index + 1} does not match ${item.name}`);
    }
  });
}

const inventory = await validateInventory();
await validateLoginProviders();
process.stdout.write(`${inventory.total} catalog entries/${inventory.assets.size} SVGs, 30 login providers verified.\n`);
