import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parseDocument, prepareGuidedPolicyChange, replacePolicy } from './local/schema-tools.js';

const schema = JSON.parse(readFileSync(new URL('./local/policy-schema.json', import.meta.url)));
const canonicalPolicy = () => parseDocument(readFileSync(new URL('./local/linux-example.json', import.meta.url), 'utf8'));
const copy = value => structuredClone(value);

const sourceRow = (name, location) => ({ name, type: 'sops', location, environment: 'reference' });
const workloadRow = (name, mode = 'hybrid') => ({ name, mode, environment: 'reference' });

test('guided source rename rejects dangling bindings without changing the active candidate', () => {
  const policy = canonicalPolicy();
  // Keep the map non-empty so deletion reaches reference validation rather
  // than being rejected earlier by the schema's minProperties constraint.
  policy.sources.unreferenced = copy(policy.sources['service-sops']);
  const before = copy(policy);
  const previous = sourceRow('service-sops', 'repo:secrets/reference-api.sops.json');
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'sources', sourceRow('renamed-source', 'repo:secrets/reference-api.sops.json'), previous, schema),
    /unknown source/,
  );
  assert.deepEqual(policy, before);
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'sources', null, previous, schema),
    /unknown source/,
  );
  assert.deepEqual(policy, before);
});

test('guided workload rename and deletion reject dangling grants atomically', () => {
  const policy = canonicalPolicy();
  policy.workloads.unreferenced = copy(policy.workloads['reference-api']);
  const before = copy(policy);
  const previous = workloadRow('reference-api');
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'workloads', workloadRow('renamed-api'), previous, schema),
    /unknown workload/,
  );
  assert.deepEqual(policy, before);
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'workloads', null, previous, schema),
    /unknown workload/,
  );
  assert.deepEqual(policy, before);
});

test('guided source edit preserves untouched provider fields and accepts a full browser-valid candidate', () => {
  const policy = canonicalPolicy();
  const next = prepareGuidedPolicyChange(
    policy,
    'sources',
    sourceRow('service-sops', 'repo:secrets/revised-reference-api.sops.json'),
    sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'),
    schema,
  );
  assert.equal(next.sources['service-sops'].path, 'repo:secrets/revised-reference-api.sops.json');
  assert.equal(next.sources['service-sops'].sops, '/usr/bin/sops');
  assert.equal(next.sources['service-sops'].age_identity_file, '/etc/circularquay/identities/reference-age.txt');
  assert.equal(next.sources['service-sops'].identity, 'none');
  assert.equal(policy.sources['service-sops'].path, 'repo:secrets/reference-api.sops.json');
});

test('guided named maps retain valid special own identifiers without prototype mutation', () => {
  for (const name of ['constructor', 'prototype']) {
    const policy = canonicalPolicy();
    policy.workloads['reference-api'].bindings[0].source = name;
    const next = prepareGuidedPolicyChange(
      policy,
      'sources',
      sourceRow(name, 'repo:secrets/revised-reference-api.sops.json'),
      sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'),
      schema,
    );
    assert.equal(Object.getPrototypeOf(next.sources), Object.prototype);
    assert.equal(Object.hasOwn(next.sources, name), true);
    assert.equal(next.sources[name].path, 'repo:secrets/revised-reference-api.sops.json');
    assert.equal(Object.hasOwn(next.sources, 'service-sops'), false);
  }
});

test('guided __proto__ identifier rejects atomically without prototype mutation', () => {
  const policy = canonicalPolicy();
  policy.workloads['reference-api'].bindings[0].source = '__proto__';
  const before = copy(policy);
  assert.throws(
    () => prepareGuidedPolicyChange(
      policy,
      'sources',
      sourceRow('__proto__', 'repo:secrets/revised-reference-api.sops.json'),
      sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'),
      schema,
    ),
    /does not match required format/,
  );
  assert.deepEqual(policy, before);
  assert.equal(Object.getPrototypeOf(policy.sources), Object.prototype);
  assert.equal({}.polluted, undefined);
});

test('guided source collisions use own map entries, including special identifiers', () => {
  for (const name of ['__proto__', 'constructor', 'prototype']) {
    const policy = canonicalPolicy();
    Object.defineProperty(policy.sources, name, {
      configurable: true,
      enumerable: true,
      value: copy(policy.sources['service-sops']),
      writable: true,
    });
    const before = copy(policy);
    assert.throws(
      () => prepareGuidedPolicyChange(policy, 'sources', sourceRow(name, 'repo:secrets/other.sops.json'), sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'), schema),
      /already exists/,
    );
    assert.deepEqual(policy, before);
  }
});

test('guided candidate rejects an inherited source name as an undeclared binding', () => {
  const policy = canonicalPolicy();
  // This inherited name satisfies the identifier schema, isolating the
  // own-property reference check from the separate mixed-case format denial.
  policy.workloads['reference-api'].bindings[0].source = 'constructor';
  const before = copy(policy);
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'sources', sourceRow('service-sops', 'repo:secrets/revised-reference-api.sops.json'), sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'), schema),
    /unknown source/,
  );
  assert.deepEqual(policy, before);
});

test('guided candidate rejects a mixed-case inherited name at schema validation without mutation', () => {
  const policy = canonicalPolicy();
  policy.workloads['reference-api'].bindings[0].source = 'toString';
  const before = copy(policy);
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'sources', sourceRow('service-sops', 'repo:secrets/revised-reference-api.sops.json'), sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'), schema),
    /does not match required format/,
  );
  assert.deepEqual(policy, before);
});

test('guided identifier collisions and retained invalid drafts never overwrite accepted policy or draft bytes', () => {
  const policy = canonicalPolicy();
  policy.sources.existing = copy(policy.sources['service-sops']);
  const before = copy(policy);
  assert.throws(
    () => prepareGuidedPolicyChange(policy, 'sources', sourceRow('existing', 'repo:secrets/other.sops.json'), sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'), schema),
    /already exists/,
  );
  assert.deepEqual(policy, before);

  delete policy.sources.existing;
  const next = prepareGuidedPolicyChange(policy, 'sources', sourceRow('service-sops', 'repo:secrets/revised-reference-api.sops.json'), sourceRow('service-sops', 'repo:secrets/reference-api.sops.json'), schema);
  const draft = { future: { disabled: false, missing: null, empty: [] } };
  const state = { policy: copy(policy), nativeEditor: { original: copy(draft), document: copy(draft) } };
  assert.equal(replacePolicy(state, next), true);
  assert.deepEqual(state.policy, next);
  assert.deepEqual(state.nativeEditor.original, draft);
  assert.deepEqual(state.nativeEditor.document, draft);
});
