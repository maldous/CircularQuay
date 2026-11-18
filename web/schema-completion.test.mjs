import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { parseDocument, dereference, shape, documentsEqual, validate, referenceErrors, prepareGuidedPolicyChange, schemaInventory, publicHelpPointer } from './local/schema-tools.js';

const clone = value => JSON.parse(JSON.stringify(value));
test('JSON parser refuses every malformed structural/primitive boundary without coercion or key loss', () => {
  const accepted = { text: 'quote" slash/ backslash\\ newline\n unicode☃', nested: [true, false, null, -0.2e3, {}, []], '__proto__': 'literal' };
  assert.deepEqual(parseDocument(JSON.stringify(accepted)), accepted);
  for (const source of ['{1:2}', '{"a" 1}', '{"a":1 "b":2}', '[1 2]', '[1,]', '{"a":}', '"unterminated', '"bad\ncontrol"', '1e999', 'true false', 'undefined', '0x20', '{"a":1,"a":2}']) assert.throws(() => parseDocument(source));
  assert.throws(() => parseDocument('['.repeat(66) + '0' + ']'.repeat(66)), /depth/); assert.throws(() => parseDocument(' '.repeat(1048577)), /size/);
  assert.equal(parseDocument(' \r\n\t -12.5e+2 '), -1250); assert.deepEqual(parseDocument('{"__proto__":{"safe":true}}'), Object.fromEntries([['__proto__', { safe: true }]]));
});

test('all supported schema constraints have real accepted and refused controls including branches and escaped pointers', () => {
  const cases = [
    [true, null, 5, false], [false, null, 5, true],
    [{ type: ['integer', 'null'] }, null, 1.5], [{ type: 'integer' }, 1, '1'],
    [{ const: { a: [1, 2] } }, { a: [1, 2] }, { a: [2, 1] }], [{ enum: [null, { b: 1 }] }, { b: 1 }, { b: 2 }],
    [{ allOf: [{ minimum: 1 }, { maximum: 3 }] }, 2, 4], [{ anyOf: [{ const: 'a' }, { const: 'b' }] }, 'a', 'c'],
    [{ oneOf: [{ type: 'integer' }, { minimum: 2 }] }, 1, 2], [{ not: { const: 'bad' } }, 'good', 'bad'],
    [{ if: { const: true }, then: { type: 'boolean' }, else: { type: 'string' } }, 'okay', 7],
    [{ type: 'object', minProperties: 1, maxProperties: 2, required: ['a'], propertyNames: { pattern: '^[a-z]+$' }, additionalProperties: { type: 'number' } }, { a: 1 }, { A: 'x', B: 2, C: 3 }],
    [{ type: 'object', properties: { 'a/b~c': { const: 3 } }, additionalProperties: false }, { 'a/b~c': 3 }, { 'a/b~c': 4, unknown: true }],
    [{ type: 'array', minItems: 1, maxItems: 2, uniqueItems: true, items: { type: 'integer' }, contains: { const: 1 } }, [1, 2], [2, 2, 2]],
    [{ type: 'string', minLength: 2, maxLength: 3, pattern: '^a' }, 'ab', 'z'], [{ type: 'string', format: 'date-time' }, '2026-11-17T00:00:00Z', '2026-11-17'],
    [{ type: 'number', minimum: 0, maximum: 1 }, 0.5, Infinity],
  ];
  for (const [schema, good, bad, alwaysRefuses] of cases) { if (alwaysRefuses) assert.ok(validate(good, schema).length); else { assert.deepEqual(validate(good, schema), []); assert.ok(validate(bad, schema === true ? false : schema).length); } }
  assert.ok(validate([], { minItems: 1 }).length); assert.ok(validate('abcd', { maxLength: 3 }).length); assert.ok(validate(-1, { minimum: 0 }).length);
  assert.ok(validate({}, { minProperties: 1 }).length); assert.ok(validate({ 'a/b~c': 1 }, { properties: { 'a/b~c': false } })[0].startsWith('/a~1b~0c'));
  assert.ok(validate(1, {}, {}, '', 101)[0].includes('depth')); assert.deepEqual(validate(true, { if: { const: true } }), []);
});

test('schema control inventory preserves base choices and maps only public structure, never concrete map IDs', () => {
  const schema = { type: 'object', properties: { rows: { type: 'array', items: { $ref: '#/definitions/item' } }, secrets: { type: 'object', additionalProperties: { type: 'boolean' } }, 'a/b~c': { const: 1 } }, definitions: { item: { type: 'object', properties: { id: { type: 'string' }, choice: { enum: [true, false] } }, allOf: [{ properties: { number: { type: 'integer' } }, required: ['number'] }] } } };
  assert.equal(dereference({ $ref: '#/definitions/item', title: 'Override' }, schema).title, 'Override'); assert.deepEqual(dereference(null, schema), {});
  const s = shape({ properties: { enabled: { type: 'boolean' } }, anyOf: [{ properties: { enabled: { const: true }, alt: { const: 'x' } } }], then: { type: 'object', items: { type: 'string' }, additionalProperties: false, minimum: 0 } }, schema);
  assert.deepEqual(s.properties.enabled, { type: 'boolean' }); assert.equal(s.type, 'object'); assert.equal(s.additionalProperties, false); assert.equal(s.minimum, 0);
  const inventory = schemaInventory(schema); assert.ok(inventory.some(row => row.path === '/rows/*/number')); assert.ok(inventory.some(row => row.path === '/secrets/{id}' && row.control === 'boolean select'));
  for (const [pointer, result] of [['', '/'], ['/rows/0/id', '/rows/*/id'], ['/secrets/OWNED_PRIVATE_ID', '/secrets/{id}'], ['/a~1b~0c', '/a/b~c'], ['/rows/01/id', null], ['/rows/0/unknown', null], ['/bad~2key', null], ['relative', null], [null, null], ['/' + 'x'.repeat(4097), null]]) assert.equal(publicHelpPointer(schema, pointer), result);
  assert.equal(documentsEqual({ a: [null, { b: 1 }] }, { a: [null, { b: 1 }] }), true); assert.equal(documentsEqual([1], [1, 2]), false); assert.equal(documentsEqual({ a: 1 }, { b: 1 }), false);
  const cycle = { type: 'object', properties: {} }; cycle.properties.self = cycle; assert.equal(schemaInventory(cycle).length, 1);
});

test('real guided variable API replaces literal/source bindings atomically and skips only nonmatching workload rows', () => {
  const policy = { sources: { safe: { type: 'json', path: './owned.json' } }, workloads: { app: { mode: 'online', environment: { KEEP: 'x', OLD: 'old' }, bindings: [{ name: 'OLD', source: 'safe', field: 'old' }, { name: 'KEEP_BINDING', source: 'safe', field: 'k' }] }, other: { mode: 'online', environment: {} } } };
  const rows = [{ name: 'missing', environment: 'production' }, { name: 'app', environment: 'production' }, { name: 'other', environment: 'different' }]; const original = clone(policy);
  const literal = prepareGuidedPolicyChange(policy, 'variables', { name: 'NEW', environment: 'production', kind: 'Non-secret literal', value: '$(NOT_EXECUTED)' }, { name: 'OLD', environment: 'production' }, true, rows);
  assert.deepEqual(literal.workloads.app.environment, { KEEP: 'x', NEW: '$(NOT_EXECUTED)' }); assert.equal(literal.workloads.app.bindings.length, 1); assert.deepEqual(policy, original);
  const bound = prepareGuidedPolicyChange(literal, 'variables', { name: 'NEW', environment: 'production', kind: 'Source binding', source: 'safe', value: 'field' }, { name: 'NEW', environment: 'production' }, true, rows);
  assert.equal(Object.hasOwn(bound.workloads.app.environment, 'NEW'), false); assert.deepEqual(bound.workloads.app.bindings.at(-1), { name: 'NEW', source: 'safe', field: 'field' });
  const removed = prepareGuidedPolicyChange(bound, 'variables', null, { name: 'NEW', environment: 'production' }, true, rows); assert.equal(removed.workloads.app.bindings.length, 1);
  assert.throws(() => prepareGuidedPolicyChange(policy, 'variables', { name: 'BAD', environment: 'production', source: 'missing', value: 'field' }, null, true, rows), /unknown source/);
  const absent = { workloads: { app: { mode: 'online' } } }; assert.deepEqual(prepareGuidedPolicyChange(absent, 'variables', null, { name: 'OLD', environment: 'production' }, true, rows).workloads.app.bindings, []);
});

test('native reference checks keep own-entry semantics and composition only defers appropriate source/workload resolution', () => {
  const policy = { sources: {}, workloads: { app: { mode: 'online', snapshot: { bad: true }, bindings: [null, { name: 'Name', source: 'missing' }, { name: 'NAME', source: 'other' }] } }, grants: [null, { workload: 'missing', authority: 'missing' }], mcp: { authorities: {} } };
  const errors = referenceErrors(policy); assert.ok(errors.some(value => value.includes('duplicate output'))); assert.ok(errors.some(value => value.includes('online cannot'))); assert.ok(errors.some(value => value.includes('unknown authority')));
  const composition = clone(policy); composition.composition = { sources: [] }; assert.ok(referenceErrors(composition).every(value => !value.includes('unknown source') && !value.includes('unknown workload')));
  const empty = prepareGuidedPolicyChange({}, 'unknown', { name: 'ignored' }, null, false); assert.deepEqual(empty, {});
  assert.throws(() => prepareGuidedPolicyChange({ workloads: { taken: {} } }, 'workloads', { name: 'taken', mode: 'online' }, null, true), /already exists/);
  assert.throws(() => prepareGuidedPolicyChange({ sources: { taken: {} } }, 'sources', { name: 'taken', type: 'json' }, null, true), /already exists/);
  assert.throws(() => prepareGuidedPolicyChange({}, 'workloads', { name: 'new', mode: 'online' }, null, false), /Guided change/);
});
