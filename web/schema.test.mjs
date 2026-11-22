import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { parseDocument, validate, referenceErrors, schemaInventory, replacePolicy, shape, reindexArrayInputs } from './local/schema-tools.js';
import { model } from './local/model.js';
const schema = JSON.parse(readFileSync(new URL('./local/policy-schema.json', import.meta.url)));
const text = readFileSync(new URL('./local/linux-example.json', import.meta.url), 'utf8');

test('fresh local demo policy satisfies the pinned browser schema without fabricated provider options', () => {
  assert.deepEqual(validate(model.policy, schema), []);
  assert.deepEqual(referenceErrors(model.policy), []);
});

test('a saved policy edit updates the typed editor without aliasing its untouched source', () => {
  const original = parseDocument(text);
  const state = { policy: structuredClone(original), nativeEditor: { original: structuredClone(original), document: structuredClone(original) } };
  const edited = structuredClone(original);
  edited.updates.minimum_version = '2.0.0';
  assert.equal(replacePolicy(state, edited), false);
  assert.deepEqual(state.policy, edited);
  assert.deepEqual(state.nativeEditor.document, edited);
  state.nativeEditor.document.updates.minimum_version = '3.0.0';
  assert.equal(state.policy.updates.minimum_version, '2.0.0');
  assert.equal(state.nativeEditor.original.updates.minimum_version, '2.0.0');
});

test('a separate invalid editor draft survives an edit to the saved policy', () => {
  const original = parseDocument(text);
  const draft = { ...structuredClone(original), future: { disabled: false, missing: null, empty: [] } };
  const state = { policy: structuredClone(original), nativeEditor: { original: structuredClone(draft), document: structuredClone(draft) } };
  const edited = structuredClone(original);
  edited.updates.minimum_version = '2.0.0';
  assert.equal(replacePolicy(state, edited), true);
  assert.deepEqual(state.policy, edited);
  assert.deepEqual(state.nativeEditor.document, draft);
  assert.deepEqual(state.nativeEditor.original, draft);
});

test('alternative telemetry branches retain boolean controls while validation enforces the chosen shape', () => {
  for (const name of ['observability', 'metricsSignal', 'tracesSignal']) {
    const fields = shape(schema.$defs[name], schema).properties;
    const enabled = shape(fields.enabled, schema);
    assert.equal(enabled.type, 'boolean');
    assert.equal(Object.hasOwn(enabled, 'const'), false);
    assert.deepEqual(validate({ enabled: false }, schema.$defs[name], schema), []);
  }
  const active = parseDocument(text);
  assert.deepEqual(validate(active, schema), []);
  active.observability.enabled = false;
  assert.ok(validate(active, schema).length, 'disabled telemetry cannot retain enabled-only fields');
  active.observability = { enabled: false };
  assert.deepEqual(validate(active, schema), []);
});

test('invalid numeric draft paths follow moved items and shift after an earlier item is removed', () => {
  const inputs = new Map([['/values/0', '-'], ['/values/1/nested/2', '1e'], ['/other/0', 'unchanged']]);
  reindexArrayInputs(inputs, '/values', index => index === 0 ? 1 : index === 1 ? 0 : index);
  assert.deepEqual(Object.fromEntries(inputs), { '/values/1': '-', '/values/0/nested/2': '1e', '/other/0': 'unchanged' });
  reindexArrayInputs(inputs, '/values', index => index === 0 ? null : index - 1);
  assert.deepEqual(Object.fromEntries(inputs), { '/values/0': '-', '/other/0': 'unchanged' });
});

test('canonical Linux fixture retains all nested values through parse/save/export', () => {
  const original = JSON.parse(text);
  const document = parseDocument(text);
  assert.deepEqual(document, original);
  assert.deepEqual(validate(document, schema), []);
  assert.deepEqual(referenceErrors(document), []);
  document.observability.metrics.interval_seconds = 45;
  document.workloads['reference-api'].targets['systemd-credential'].activation = 'reload';
  const edited = parseDocument(JSON.stringify(document));
  assert.equal(edited.observability.metrics.interval_seconds, 45);
  assert.equal(edited.workloads['reference-api'].targets['systemd-credential'].activation, 'reload');
  assert.deepEqual(edited.workloads['reference-api'].snapshot, original.workloads['reference-api'].snapshot);
  assert.deepEqual(edited.mcp, original.mcp);
});

test('absent, false, empty, null and unknown fields are preserved without accepting unsupported config', () => {
  const raw = '{"version":1,"sources":{},"workloads":{},"future":{"disabled":false,"empty":"","null":null,"items":[]}}';
  const document = parseDocument(raw);
  assert.equal(JSON.stringify(document), raw);
  assert.ok(validate(document, schema).some(error => error.includes('/future')));
  assert.equal(Object.hasOwn(document, 'grants'), false);
});

test('duplicate keys, malformed numbers/strings and unsafe JSON depth reject before replacement', () => {
  for (const input of ['{"version":1,"version":2}', '{"a":{"x":1,"x":2}}', '{"x":01}', '{"x":NaN}', '{"x":1e999}', '{"x":"unterminated}', '[1,]', '{"x":1}\u00a0']) assert.throws(() => parseDocument(input), input);
  assert.throws(() => parseDocument('['.repeat(66) + '0' + ']'.repeat(66)));
  assert.deepEqual(parseDocument('{"__proto__":{"polluted":true}}'), JSON.parse('{"__proto__":{"polluted":true}}'));
  assert.equal({}.polluted, undefined);
});

test('provider branch conditions, nested types, bounds, references and duplicate arrays deny', () => {
  const document = parseDocument(text);
  document.sources['service-sops'].type = 'aws-sm';
  assert.ok(validate(document, schema).length);
  const invalid = parseDocument(text);
  invalid.observability.metrics.interval_seconds = '30';
  assert.ok(validate(invalid, schema).some(error => error.includes('interval_seconds')));
  invalid.observability.metrics.interval_seconds = 30;
  invalid.workloads['reference-api'].bindings[0].source = 'missing';
  assert.ok(referenceErrors(invalid).some(error => error.includes('unknown source')));
  invalid.mcp.origins.push(invalid.mcp.origins[0]);
  assert.ok(validate(invalid, schema).some(error => error.includes('duplicate array')));
});

test('schema control inventory reaches provider, target, identity, telemetry, include and optional fields', () => {
  const paths = new Set(schemaInventory(schema).map(row => row.path));
  for (const path of ['/sources/{id}/age_identity_file', '/sources/{id}/vault_allowlist/*', '/workloads/{id}/targets/{id}/activation', '/workloads/{id}/bindings/*/allow_publish', '/mcp/authorities/{id}/permissions/{id}/scopes/*', '/composition/includes/*/sha256', '/observability/metrics/interval_seconds', '/updates/minimum_version']) assert.ok(paths.has(path), path);
});
