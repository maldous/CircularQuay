import test from 'node:test';
import assert from 'node:assert/strict';
import { PERSONA_COUNT, STEP_DURATION, createOptionalSchedule, createScenarioSession, createStoryRotation, createVariantRotation, sessionStep, staticScenario, storyReservationSessions } from './scenario-flow.js';

const variants = {
  identity: [{ name: 'GitHub', asset: 'github.svg' }, { name: 'Okta', asset: 'okta.svg' }],
  'identity-2': [{ name: 'GitHub', asset: 'github.svg' }, { name: 'Auth0', asset: 'auth0.svg' }],
  local: [{ name: 'Local file', asset: 'local.svg' }],
  remote: [{ name: 'Google Drive', asset: 'drive.svg' }, { name: 'Dropbox', asset: 'dropbox.svg' }],
  shell: [{ name: 'Bash', asset: 'bash.svg' }, { name: 'PowerShell', asset: 'pwsh.svg' }],
  container: [{ name: 'Docker', asset: 'docker.svg' }, { name: 'Kubernetes', asset: 'k8s.svg' }],
  cloud: [{ name: 'AWS', asset: 'aws.svg' }, { name: 'Netlify', asset: 'netlify.svg' }],
  ci: [{ name: 'GitHub Actions', asset: 'actions.svg' }, { name: 'Jenkins', asset: 'jenkins.svg' }],
};

function seeded(seed) {
  let value = seed;
  return () => ((value = value * 16807 % 2147483647) - 1) / 2147483646;
}

test('the catalog offers at least twenty-four contexts with both user goals and the exact alternative', () => {
  const session = createScenarioSession({ variantPools: variants, random: seeded(3) });
  assert.ok(PERSONA_COUNT >= 24);
  assert.match(session.prefix, /^As .+, (I need to|my human operator needs to) .+ and .+,$/);
  assert.equal(session.suffix, 'or I can use CircularQuay.');
});

test('each session has two fully connected workloads with distinct identity, vault, and target choices', () => {
  const session = createScenarioSession({ variantPools: variants, random: seeded(11) });
  assert.equal(session.tasks.length, 2);
  assert.ok(session.tasks.every(task => task.length >= 3 && task.length <= 4));
  assert.equal(session.legs.length, session.clauses.length * 2);
  assert.deepEqual(session.legs.map(leg => leg.clauseIndex), session.clauses.flatMap((_, index) => [index, index]));

  const [first, second] = session.tasks;
  assert.match(first[0].text, /^(all I have to do is|all my human operator has to do is) sign in to /);
  assert.match(second[0].text, /^then sign in to /);
  assert.match(first[1].text, /^open the .+ vault$/);
  assert.match(second[1].text, /^open the .+ vault$/);
  assert.notEqual(first[0].references[0], second[0].references[0]);
  assert.notEqual(first[1].references[0], second[1].references[0]);
  assert.notEqual(first.at(-1).references[0], second.at(-1).references[0]);

  for (const task of session.tasks) {
    for (const item of task) {
      assert.equal(item.legs.length, 2);
      assert.notEqual(item.legs[0].direction, item.legs[1].direction);
    }
    assert.deepEqual(task[0].legs.map(leg => leg.direction), ['reverse', 'forward']);
    assert.deepEqual(task[1].legs.map(leg => leg.direction), ['reverse', 'forward']);
    assert.deepEqual(task.at(-1).legs.map(leg => leg.direction), ['forward', 'reverse']);
  }
});

test('two IDP tiles never receive the same selected mark when alternatives exist', () => {
  for (let seed = 1; seed < 40; seed++) {
    const session = createScenarioSession({ variantPools: variants, random: seeded(seed) });
    assert.notEqual(session.selection.identity.asset, session.selection['identity-2'].asset);
  }
});

test('optional steps are sparse, can be absent, and all feature copies remain reachable', () => {
  const seen = new Set();
  const random = seeded(17);
  let optionalWorkloads = 0;
  for (let index = 1; index <= 300; index++) {
    const session = createScenarioSession({ variantPools: variants, random });
    for (const item of session.clauses.filter(clause => clause.legs[0].route.startsWith('optional-'))) {
      optionalWorkloads++;
      seen.add(item.text);
      assert.equal(item.legs.length, 2);
      assert.notEqual(item.legs[0].direction, item.legs[1].direction);
    }
  }
  assert.ok(optionalWorkloads / 600 < .5);
  const none = createScenarioSession({ variantPools: variants, random: () => .9 });
  assert.equal(none.clauses.some(clause => clause.legs[0].route.startsWith('optional-')), false);
  assert.deepEqual([...seen].sort((left, right) => left.localeCompare(right, 'en')), [
    'check signing', 'limit MCP access', 'log an audit event',
    'read SCIM groups', 'request KMS access', 'use an age key',
  ]);
});

test('the balanced optional schedule leads with one visible feature and covers each four-scene mask', () => {
  const next = createOptionalSchedule({ random: seeded(41) });
  const first = [next(), next(), next(), next()];
  assert.ok(first[0].length === 1);
  assert.deepEqual(first.map(mask => mask.join()).sort((left, right) => left.localeCompare(right, 'en')), ['', '0', '0,1', '1']);
  const following = Array.from({ length: 8 }, next);
  assert.equal([...first, ...following].flat().length, 12);
  const sequence = seed => Array.from({ length: 12 }, createOptionalSchedule({ random: seeded(seed) }), mask => mask.join());
  assert.deepEqual(sequence(91), sequence(91));
});

test('optional slot masks compose none, one, or both optional workload clauses', () => {
  for (const [slots, clauseCount] of [[[], 6], [[0], 7], [[1], 7], [[0, 1], 8]]) {
    const session = createScenarioSession({ variantPools: variants, random: seeded(23), optionalSlots: slots });
    const optionalClauses = session.clauses.filter(item => item.legs[0].route.startsWith('optional-'));
    assert.equal(session.clauses.length, clauseCount);
    assert.deepEqual(optionalClauses.map(item => item.workloadIndex), slots);
    assert.ok(optionalClauses.every(item => item.legs.length === 2 && item.legs[0].direction !== item.legs[1].direction));
    assert.deepEqual(session.legs.map(leg => leg.clauseIndex), session.clauses.flatMap((_, index) => [index, index]));
  }
});

test('the static reduced-motion example stays on the basic connected flow', () => {
  const session = staticScenario(variants);
  assert.equal(session.clauses.some(clause => clause.legs[0].route.startsWith('optional-')), false);
  assert.deepEqual(session.clauses.map((_, index) => index + 1), [1, 2, 3, 4, 5, 6]);
});

test('story reservation uses a bounded set covering every persona, task order, and longest optional pair', () => {
  const candidates = storyReservationSessions(variants);
  assert.equal(candidates.length, PERSONA_COUNT * 4);
  assert.ok(candidates.every(candidate => candidate.tasks.length === 2 && candidate.clauses.length >= 6));
  assert.ok(candidates.some(candidate => candidate.clauses.length === 8));
});

test('reservation envelopes independently retain widest context, name, intent, and action text', () => {
  const rich = structuredClone(variants);
  rich.cloud = [
    { id: 'wide-name', name: 'Extraordinarily Wide Platform', asset: 'wide.svg', intent: 'short intent', action: { template: 'run {name}' } },
    { id: 'wide-copy', name: 'CQ', asset: 'copy.svg', intent: 'the widest intent phrase', action: { template: 'perform the widest action through {name}' } },
  ];
  const measure = item => String(item.name || item.text).length;
  const candidates = storyReservationSessions(rich, measure);
  assert.ok(candidates.some(candidate => candidate.prefix.includes('the widest intent phrase')));
  assert.ok(candidates.some(candidate => candidate.clauses.some(clause => clause.text.includes('perform the widest action through Extraordinarily Wide Platform'))));
});

test('a clause stays active across its two one-point-two-second legs and then settles', () => {
  const session = createScenarioSession({ variantPools: variants, random: seeded(29) });
  const vaultStart = STEP_DURATION * 2;
  assert.equal(sessionStep(session, vaultStart + STEP_DURATION / 2).clauseIndex, 1);
  assert.equal(sessionStep(session, vaultStart + STEP_DURATION + 1).clauseIndex, 1);
  assert.equal(sessionStep(session, session.legs.length * STEP_DURATION).settled, true);
});

test('shared rotation bags expose every identity and source before reuse while keeping each frame distinct', () => {
  const item = (id, node) => ({ id, name: id, asset: `${id}.svg`, role: node });
  const pools = {
    identity: ['i1', 'i2', 'i3'].map(id => item(id, 'identity')),
    'identity-2': ['i1', 'i2', 'i3'].map(id => item(id, 'identity')),
    local: ['s1', 's2', 's3'].map(id => item(id, 'source')),
    remote: ['s1', 's2', 's3'].map(id => item(id, 'source')),
    shell: [item('shell', 'app')], container: [item('container', 'app')], cloud: [item('cloud', 'app')], ci: [item('ci', 'app')],
  };
  const next = createVariantRotation({ variantPools: pools, random: seeded(7) });
  const selections = [next(), next(), next()];
  assert.equal(new Set(selections.flatMap(selection => [selection.identity.id, selection['identity-2'].id])).size, 3);
  assert.equal(new Set(selections.flatMap(selection => [selection.local.id, selection.remote.id])).size, 3);
  assert.ok(selections.every(selection => selection.identity.id !== selection['identity-2'].id && selection.local.id !== selection.remote.id));
});

test('catalog variant intent and action override the generic persona language', () => {
  const item = (id, node, extra = {}) => ({ id, name: id, asset: `${id}.svg`, role: node, ...extra });
  const pools = {
    identity: [item('id-one', 'identity')], 'identity-2': [item('id-two', 'identity')], local: [item('local-one', 'source')], remote: [item('remote-one', 'source')],
    shell: [item('Terminal', 'application', { intent: 'open a custom terminal', action: { template: 'launch {name}' } })],
    container: [item('Container', 'application')], cloud: [item('Cloud', 'application')], ci: [item('Pipeline', 'application', { intent: 'run custom delivery', action: { template: 'deliver through {name}' } })],
  };
  const selection = Object.fromEntries(Object.entries(pools).map(([node, values]) => [node, values[0]]));
  const session = createScenarioSession({ variantPools: pools, variantSelection: selection, random: () => 0, optionalSlots: [] });
  assert.match(session.prefix, /open a custom terminal and run custom delivery/);
  assert.ok(session.clauses.some(clause => clause.text === 'launch Terminal'));
  assert.ok(session.clauses.some(clause => clause.text === 'deliver through Pipeline'));
});

test('story rotation independently cycles actors and contexts before repeating', () => {
  const next = createStoryRotation({ random: seeded(73) });
  const choices = Array.from({ length: PERSONA_COUNT }, next);
  assert.equal(new Set(choices.map(choice => choice.personaId)).size, PERSONA_COUNT);
  assert.notEqual(choices[0].context, choices[1].context);
  const session = createScenarioSession({ variantPools: variants, storyChoice: { personaId: 'new-hire', context: 'during a Friday release' }, random: () => 0, optionalSlots: [] });
  assert.match(session.prefix, /^As a new hire during a Friday release, I need to .+ and .+,$/);
});
