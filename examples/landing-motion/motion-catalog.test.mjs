import { readFile, access } from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
import test from 'node:test';
import { motionCatalog } from './motion-catalog.js';
import { createScenarioSession, createVariantRotation } from './scenario-flow.js';
const root = path.dirname(new URL(import.meta.url).pathname);
const integrations = JSON.parse(await readFile(path.join(root, 'integrations.json'), 'utf8'));
const expected = integrations.groups.flatMap(group => group.items.map(item => `${group.id}:${item.id}`));
const actual = [...motionCatalog.identity, ...motionCatalog.local, ...Object.values(['shell','container','cloud','ci']).flatMap(slot => motionCatalog[slot])].map(item => `${item.group}:${item.id}`);
test('paired identity and source pools retain matching values without shared mutable entries', () => {
  for (const [left, right] of [['identity', 'identity-2'], ['local', 'remote']]) {
    assert.deepEqual(motionCatalog[left], motionCatalog[right]);
    assert.notStrictEqual(motionCatalog[left], motionCatalog[right]);
    for (let index = 0; index < motionCatalog[left].length; index += 1) {
      assert.notStrictEqual(motionCatalog[left][index], motionCatalog[right][index]);
    }
  }
});
if (expected.length !== 113 || new Set(actual).size !== 113 || expected.some(id => !actual.includes(id))) throw new Error('All 113 catalog entries must appear exactly once across the canonical pools');
for (const [slot, entries] of Object.entries(motionCatalog)) for (const entry of entries) { await access(path.join(root, entry.asset)); if (!entry.role || !entry.intent || entry.intent.includes('unidentified') || !entry.action.includes(entry.name)) throw new Error(`Invalid ${slot}/${entry.id}`); }
const fixed = Object.fromEntries(Object.entries(motionCatalog).map(([slot, entries]) => [slot, entries[0]]));
const personaFor = slot => ({ shell: 'new-hire', container: 'backend-developer', cloud: 'frontend-developer', ci: 'release-engineer' }[slot]);
for (const slot of ['shell', 'container', 'cloud', 'ci']) for (const entry of motionCatalog[slot]) {
  const selection = { ...fixed, [slot]: entry };
  const session = createScenarioSession({ variantPools: motionCatalog, variantSelection: selection, storyChoice: { personaId: personaFor(slot), context: 'on a test run' }, optionalSlots: [], random: () => 0 });
  if (session.selection[slot].id !== entry.id || !session.clauses.some(clause => clause.text.includes(entry.action))) throw new Error(`Forced application variant failed: ${slot}/${entry.id}`);
}
for (const [slot, entries] of [['identity', motionCatalog.identity], ['local', motionCatalog.local]]) for (const entry of entries) {
  const selection = { ...fixed, [slot]: entry };
  if (slot === 'identity' && selection['identity-2'].asset === entry.asset) selection['identity-2'] = motionCatalog['identity-2'][1];
  if (slot === 'local' && selection.remote.asset === entry.asset) selection.remote = motionCatalog.remote[1];
  const session = createScenarioSession({ variantPools: motionCatalog, variantSelection: selection, storyChoice: { personaId: 'new-hire', context: 'on a test run' }, optionalSlots: [], random: () => 0 });
  if (session.selection[slot].id !== entry.id || !session.clauses.some(clause => clause.text.includes(entry.name))) throw new Error(`Forced ${slot} variant failed: ${entry.id}`);
}
let seed = 7;
const rotate = createVariantRotation({ variantPools: motionCatalog, random: () => { seed = (seed * 48271) % 2147483647; return seed / 2147483647; } });
const observed = new Set();
for (let index = 0; index < 140; index += 1) Object.entries(rotate()).forEach(([slot, entry]) => observed.add(`${slot}:${entry.id}`));
for (const [slot, entries] of Object.entries(motionCatalog)) if (entries.some(entry => !observed.has(`${slot}:${entry.id}`))) throw new Error(`Rotation did not expose every ${slot} entry`);
console.log('motion catalog: 113 entries, assets and task language verified');
