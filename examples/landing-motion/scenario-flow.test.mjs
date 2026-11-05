import test from 'node:test';
import assert from 'node:assert/strict';
import { PERSONA_COUNT, STEP_DURATION, createScenarioSession, sessionStep, staticScenario, storyReservationSessions } from './scenario-flow.js';

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
  assert.deepEqual([...seen].sort(), [
    'check signing', 'limit MCP access', 'log an audit event',
    'read SCIM groups', 'request KMS access', 'use an age key',
  ]);
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

test('a clause stays active across its two one-point-two-second legs and then settles', () => {
  const session = createScenarioSession({ variantPools: variants, random: seeded(29) });
  const vaultStart = STEP_DURATION * 2;
  assert.equal(sessionStep(session, vaultStart + STEP_DURATION / 2).clauseIndex, 1);
  assert.equal(sessionStep(session, vaultStart + STEP_DURATION + 1).clauseIndex, 1);
  assert.equal(sessionStep(session, session.legs.length * STEP_DURATION).settled, true);
});
