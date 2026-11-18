import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { browser } from './coverage-dom-fixture.test.mjs';
let sequence = 0;
async function editor(t, policy, schema) {
  const x = browser(t, { hash: '#/configuration' }); const changed = [], exports = []; let saves = 0;
  schema ||= JSON.parse(await readFile(new URL('./local/policy-schema.json', import.meta.url)));
  policy ||= JSON.parse(await readFile(new URL('./local/linux-example.json', import.meta.url)));
  globalThis.fetch = async path => { if (path === '/local/policy-schema.json') return { ok: true, json: async () => schema }; if (path === '/local/linux-example.json') return { ok: true, text: async () => JSON.stringify(policy) }; throw Error('Unexpected owned fixture request'); };
  const { configuration } = await import(`./local/configuration.js?owned-editor=${++sequence}`); const state = { policy: structuredClone(policy) }; const root = x.document.querySelector('#app');
  await configuration(root, state, { changed: value => changed.push(value), save: () => saves++, exportJSON: (value, filename) => exports.push({ value: structuredClone(value), filename }) });
  async function action(name, attributes = {}) { const button = [...root.querySelectorAll('[data-cfg]')].find(node => node.dataset.cfg === name && Object.entries(attributes).every(([key, value]) => node.dataset[key] === value)); assert.ok(button, 'actual rendered action ' + name + JSON.stringify(attributes)); await root.onclick({ target: button, preventDefault() {} }); }
  function input(selector, value) { const control = root.querySelector(selector); assert.ok(control, 'actual rendered control ' + selector); control.value = value; control.onchange({ target: control }); }
  const proposal = () => JSON.parse(root.querySelectorAll('.cfg-diff pre')[1].textContent);
  return { ...x, root, state, changed, exports, saves: () => saves, action, input, proposal };
}

test('complete editor actually creates supported fields/map/array values, changes types and retains unknown siblings', async t => {
  const schema = { type: 'object', properties: { sources: { type: 'object', additionalProperties: { type: 'object', properties: { type: { enum: ['json', 'sops', 'vault', 'gcp', 'custom'] }, token: { type: 'string' }, flag: { type: 'boolean' }, count: { type: 'integer', minimum: 2 }, fixed: { const: 'bound' }, list: { type: 'array', items: { type: 'number' } }, arbitrary: {} }, additionalProperties: false } }, grants: { type: 'array', items: { type: 'object', properties: { workload: { type: 'string' } } } }, repository: { type: 'object' } } };
  const x = await editor(t, { sources: { own: { type: 'json', token: 'OWNED_MASK', unsupported: 'preserved', list: [1, 2], arbitrary: null } }, 'unknown<root>': { retained: true } }, schema);
  assert.equal(x.root.querySelector('[data-pointer="/sources/own/token"]').type, 'password'); assert.match(x.root.textContent, /Unsupported field preserved/);
  x.input('[data-add-select="/sources/own"]', 'flag'); assert.equal(x.proposal().sources.own.flag, false); x.input('[data-pointer="/sources/own/flag"][data-type]', 'true'); assert.equal(x.proposal().sources.own.flag, true);
  x.input('[data-add-select="/sources/own"]', 'count'); assert.equal(x.proposal().sources.own.count, 2); x.input('[data-add-select="/sources/own"]', 'fixed'); assert.equal(x.proposal().sources.own.fixed, 'bound');
  await x.action('replace-null', { pointer: '/sources/own/arbitrary' }); assert.equal(x.proposal().sources.own.arbitrary, '');
  for (const [type, value] of [['object', {}], ['array', []], ['boolean', false], ['number', 0], ['null', null], ['string', '']]) { x.input('[data-change-type="/sources/own/arbitrary"]', type); assert.deepEqual(x.proposal().sources.own.arbitrary, value); }
  await x.action('append', { pointer: '/sources/own/list' }); assert.deepEqual(x.proposal().sources.own.list, [1, 2, 0]); await x.action('move-up', { pointer: '/sources/own/list/2' }); assert.deepEqual(x.proposal().sources.own.list, [1, 0, 2]); await x.action('move-down', { pointer: '/sources/own/list/0' }); assert.deepEqual(x.proposal().sources.own.list, [0, 1, 2]);
  await x.action('map-add', { pointer: '/sources' }); let form = x.root.querySelector('#cfg-dialog-form'); form.querySelector('[name="key"]').value = 'own'; await form.onsubmit({ target: form, preventDefault() {} }); assert.match(x.root.querySelector('#cfg-dialog-error').textContent, /already exists/);
  form.querySelector('[name="key"]').value = 'second'; await form.onsubmit({ target: form, preventDefault() {} }); assert.deepEqual(x.proposal().sources.second, {});
  await x.action('section', { section: 'repository' }); await x.action('add-root', { section: 'repository' }); assert.deepEqual(x.proposal().repository, {});
  await x.action('section', { section: 'unknown<root>' }); assert.equal(x.root.querySelector('script'), null); assert.match(x.root.textContent, /unknown<root>/);
});

test('editor controls preserve invalid numbers across move/remove/undo and block save/export without coercion', async t => {
  const x = await editor(t, { sources: { own: { values: [1, 2] } } }, { properties: { sources: { additionalProperties: { properties: { values: { type: 'array', items: { type: 'number' } } } } } } });
  x.input('[data-pointer="/sources/own/values/0"][data-type]', '-'); await x.action('move-down', { pointer: '/sources/own/values/0' }); assert.equal(x.root.querySelector('[data-pointer="/sources/own/values/1"][data-type]').value, '-');
  await x.action('remove', { pointer: '/sources/own/values/0' }); await x.action('undo-remove'); assert.equal(x.root.querySelector('[data-pointer="/sources/own/values/1"][data-type]').value, '-');
  await x.action('save'); await x.action('export'); assert.equal(x.saves(), 0); assert.equal(x.exports.length, 0);
  x.input('[data-pointer="/sources/own/values/1"][data-type]', '3'); await x.action('save'); assert.equal(x.saves(), 1); assert.deepEqual(x.state.policy.sources.own.values, [2, 3]);
  await x.action('export'); assert.equal(x.exports[0].filename, 'cq-local-configuration.json'); await x.action('coverage'); assert.equal(x.exports[1].value.runtimeQualification, false);
});

test('editor import validates roots/duplicates, refuses dirty replacement, and catches rejected file reads without revealing text', async t => {
  const x = await editor(t); const original = structuredClone(x.state.policy); await x.action('import'); let form = x.root.querySelector('#cfg-dialog-form');
  for (const text of ['null', '[]', '{"version":1,"version":2}', '{']) { form.querySelector('textarea').value = text; await form.onsubmit({ target: form, preventDefault() {} }); assert.ok(x.root.querySelector('#cfg-dialog-error').textContent); assert.deepEqual(x.state.policy, original); }
  let file = x.root.querySelector('#cfg-import-file'); await file.onchange({ target: { files: [] } }); await file.onchange({ target: { files: [{ size: 1048577 }] } }); assert.match(x.root.querySelector('#cfg-dialog-error').textContent, /Maximum/);
  form.querySelector('textarea').value = 'OWNED_DRAFT'; await file.onchange({ target: { files: [{ size: 1, text: async () => { throw Error('PRIVATE_FILE_DATA'); } }] } }); assert.equal(form.querySelector('textarea').value, 'OWNED_DRAFT'); assert.match(x.root.querySelector('#cfg-dialog-error').textContent, /could not be read/); assert.ok(!x.root.textContent.includes('PRIVATE_FILE_DATA'));
  let resolveOld; const older = file.onchange({ target: { files: [{ size: 1, text: () => new Promise(resolve => { resolveOld = resolve; }) }] } });
  await file.onchange({ target: { files: [{ size: 2, text: async () => '{"unsupported":null}' }] } }); resolveOld('STALE_JSON'); await older; assert.equal(form.querySelector('textarea').value, '{"unsupported":null}'); await form.onsubmit({ target: form, preventDefault() {} }); assert.deepEqual(x.state.nativeEditor.document, { unsupported: null }); assert.deepEqual(x.state.policy, original);
  await x.action('save'); assert.match(x.root.textContent, /INVALID draft/); await x.action('discard'); assert.deepEqual(x.state.nativeEditor.document, { unsupported: null });
  await x.action('load-policy'); assert.deepEqual(x.state.nativeEditor.document, original); await x.action('section', { section: 'observability' });
  const input = x.root.querySelector('[data-pointer="/observability/service_name"][data-type]'); input.value = 'owned-change'; input.onchange({ target: input });
  for (const action of ['load-policy', 'load-linux', 'import']) { await x.action(action); assert.match(x.root.querySelector('.cfg-message').textContent, /Save or discard/); }
  const event = { target: x.root.querySelector('a[href]'), preventDefault() { this.prevented = true; } }; await x.root.onclick(event); assert.equal(event.prevented, true);
  await x.action('discard'); await x.action('load-linux'); assert.match(x.root.textContent, /Imported locally/);
});

test('editor search/details/more actions keep focus and absent/imported states explicit', async t => {
  const x = await editor(t); const details = x.root.querySelector('details[data-path]'); details.open = false; details.ontoggle(); details.open = true; details.ontoggle();
  let search = x.root.querySelector('#cfg-search'); search.value = 'nonmatching-owned-path'; search.oninput({ target: search }); assert.equal(x.document.activeElement, x.root.querySelector('#cfg-search')); assert.equal(x.root.querySelectorAll('.cfg-field').length, 0);
  search = x.root.querySelector('#cfg-search'); search.value = ''; search.oninput({ target: search });
  const more = x.root.querySelector('.cfg-more'); more.open = true; more.ontoggle(); more.onkeydown({ key: 'Escape', preventDefault() {} }); assert.equal(more.open, false); assert.equal(x.document.activeElement, more.querySelector('summary'));
  await x.action('validate'); assert.match(x.root.textContent, /Checked locally/); assert.equal(x.sources.length, 0);
});

test('schema and bundled-example failures stay local, fixed and nonactivating', async t => {
  const x = browser(t, { hash: '#/configuration' }); globalThis.fetch = async () => ({ ok: false }); const module = await import(`./local/configuration.js?unavailable=${++sequence}`);
  await module.configuration(x.document.querySelector('#app'), { policy: {} }, {}); assert.match(x.document.querySelector('#app').textContent, /Schema unavailable/);
  await assert.rejects(module.fetchExampleDocument(async () => ({ ok: false })), /unavailable/); assert.equal(await module.fetchExampleDocument(async () => ({ ok: true, text: async () => 'owned' })), 'owned');
  const messages = []; await module.performConfigurationAction(async () => { throw Error('PRIVATE_FILE_DATA'); }, text => messages.push(text)); assert.equal(messages.length, 1); assert.ok(!messages[0].includes('PRIVATE')); assert.match(messages[0], /no native operation/);
});

test('replaced configuration import dialogs never receive stale read completions or raw errors', async t => {
  const x = await editor(t); const before = structuredClone(x.state.policy);
  for (const fail of [false, true]) {
    await x.action('import'); const file = x.root.querySelector('#cfg-import-file'); let resolve, reject; const old = file.onchange({ target: { files: [{ size: 1, text: () => new Promise((a, b) => { resolve = a; reject = b; }) }] } });
    await x.action('close'); await x.action('import'); x.root.querySelector('#cfg-import-text').value = 'NEW_OWNED_DRAFT'; fail ? reject(Error('PRIVATE_STALE_FILE')) : resolve('STALE_JSON'); await old;
    assert.equal(x.root.querySelector('#cfg-import-text').value, 'NEW_OWNED_DRAFT'); assert.equal(x.root.querySelector('#cfg-dialog-error').textContent, ''); await x.action('close');
  }
  await x.action('import'); const file = x.root.querySelector('#cfg-import-file'); let reject; const old = file.onchange({ target: { files: [{ size: 1, text: () => new Promise((_, b) => { reject = b; }) }] } }); await file.onchange({ target: { files: [] } }); reject(Error('PRIVATE_CANCELLED_FILE')); await old; assert.equal(x.root.querySelector('#cfg-dialog-error').textContent, ''); assert.deepEqual(x.state.policy, before);
});
