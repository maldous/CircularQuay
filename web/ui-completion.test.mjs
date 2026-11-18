import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { browser, Element } from './coverage-dom-fixture.test.mjs';
import { start, brandMarkup } from './ui.js';
import { model as localModel } from './local/model.js';
import { model as remoteModel } from './remote/model.js';
import { parseDocument, replacePolicy } from './local/schema-tools.js';

function fixture(t, plane = 'local', options = {}) {
  const x = browser(t, { hash: plane === 'local' ? '#/configuration' : '#/account/overview', ...options });
  globalThis.fetch = async path => { assert.equal(path, '/local/policy-schema.json'); return { ok: true, json: async () => JSON.parse(readFileSync(new URL('./local/policy-schema.json', import.meta.url))) }; };
  const source = plane === 'local' ? localModel : remoteModel; const model = structuredClone(source); const observed = {};
  model.parseDocument = parseDocument; model.replacePolicy = replacePolicy;
  for (const name of ['configuration', 'workbench']) model[name] = (root, state, hooks) => { observed.state = state; observed.hooks = hooks; root.innerHTML = `<main id="main">Owned ${name} delegate</main>`; };
  start(model); const root = x.document.querySelector('#app');
  const node = selector => { const result = x.document.querySelector(selector); assert.ok(result, `actual rendered ${selector}`); return result; };
  const route = value => { x.location.hash = '#/' + value; x.window.dispatchEvent({ type: 'hashchange' }); };
  const action = (name, suffix = '') => x.event(node(`[data-action="${name}"]${suffix}`), 'click', { bubbles: true });
  const change = (selector, value, type = 'change') => { const control = node(selector); control.value = value; x.event(control, type); };
  const submit = selector => x.event(node(selector), 'submit');
  const saved = () => JSON.parse(x.data.get('cq.preview.v1.' + plane));
  x.beforeRestore(() => x.window.dispatchEvent({ type: 'pagehide' }));
  return { ...x, root, model, observed, node, route, action, change, submit, saved, notice: () => node('#notice').textContent };
}

test('all genuine local routes render the declared synthetic boundary and delegate only local tools', t => {
  const x = fixture(t); const initial = structuredClone(x.observed.state.policy);
  for (const route of ['landing', ...localModel.navigation.Client, 'overview', 'variables', 'not-an-allowed-route']) {
    x.route(route); assert.ok(x.root.textContent.length); assert.equal(x.root.querySelector('script'), null);
    if (['workbench', 'overview', 'variables'].includes(route)) assert.match(x.root.textContent, /Owned workbench/);
  }
  assert.deepEqual(x.observed.state.policy, initial); assert.equal(x.sources.length, 1); assert.equal(x.sources[0].path, '/__reload');
  assert.match(brandMarkup('<synthetic>'), /&lt;synthetic&gt;/);
});

test('all account/operator routes retain presentation-only role boundaries and no account credentials', t => {
  const x = fixture(t, 'remote');
  for (const scope of ['Account', 'Operator']) for (const key of remoteModel.navigation[scope]) {
    x.route(scope.toLowerCase() + '/' + key); assert.ok(x.root.textContent.length); assert.match(x.root.textContent, /synthetic|demo|preview/i); assert.equal(x.root.querySelector('input[type=password]'), null);
  }
  x.route('account/overview'); assert.match(x.root.textContent, /No changes yet/); x.action('event'); assert.equal(x.saved().events.length, 1); assert.equal(x.saved().events[0].target, 'customer-1');
  x.route('account/unavailable'); assert.match(x.root.textContent, /Page not found/);
});

test('each declared persona limits local presentation controls without granting server authority', t => {
  const x = fixture(t, 'remote');
  const accountWriters = new Set(['Account owner', 'Enterprise administrator', 'Global administrator']);
  const operatorWriters = { 'Service operator': 'operations', 'Billing specialist': 'billing', 'Product owner': 'customers' };
  for (const persona of remoteModel.personas) {
    for (const [scope, key] of [['account', 'support'], ['operator', 'customers'], ['operator', 'operations'], ['operator', 'billing']]) {
      if (!remoteModel.navigation[scope === 'account' ? 'Account' : 'Operator'].includes(key)) continue;
      x.route(scope + '/' + key); x.change('#persona', persona);
      const permitted = scope === 'account' ? accountWriters.has(persona) || persona === 'Support specialist' : persona === 'Global administrator' || operatorWriters[persona] === key;
      const writable = permitted && !remoteModel.collections[key].readonly;
      assert.equal(Boolean(x.root.querySelector('[data-action=create]')), writable, `${scope}/${key}/${persona}`);
      if (!permitted) assert.match(x.root.textContent, /Read-only|cannot|No server authority/);
    }
  }
});

test('table state, filtering, search, inspection and local export preserve complete records', async t => {
  const x = fixture(t); x.route('environments'); const before = structuredClone(x.observed.state.collections.environments.rows);
  x.change('#view-state', 'loading'); assert.match(x.root.textContent, /Loading synthetic/);
  x.change('#view-state', 'error'); assert.match(x.root.textContent, /saved data is intact/); x.action('retry');
  x.change('#search', 'not-in-records', 'input'); assert.match(x.root.textContent, /No matching records/); x.action('clear');
  x.change('#view-state', 'empty'); assert.match(x.root.textContent, /No matching records/); x.action('clear');
  x.action('inspect'); assert.ok(x.node('#editor').open); assert.equal(x.node('#editor').querySelector('form'), null); x.action('close-dialog');
  x.action('export'); assert.deepEqual(JSON.parse(await x.exports[0].text()), before); await x.advance(1000); assert.equal(x.revoked.length, 1);
  assert.deepEqual(x.observed.state.collections.environments.rows, before);
});

test('create/edit duplicate refusal/delete confirmation change only browser collection and safe audit metadata', async t => {
  const x = fixture(t); x.route('environments'); const collection = x.observed.state.collections.environments; const count = collection.rows.length;
  x.action('create'); x.node('#record-form [name=name]').value = collection.rows[0].name; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /already exists/); assert.equal(collection.rows.length, count);
  x.node('#record-form [name=name]').value = 'owned-fixture'; x.submit('#record-form'); await x.drain(); assert.equal(collection.rows.length, count + 1);
  const created = collection.rows.at(-1); x.action('edit', `[data-id="${created.id}"]`); x.node('#record-form [name=name]').value = '<owned-edit>'; x.submit('#record-form'); await x.drain(); assert.equal(collection.rows.at(-1).name, '<owned-edit>'); assert.equal(x.root.querySelector('owned-edit'), null);
  x.action('edit', `[data-id="${created.id}"]`); x.action('delete', `[data-id="${created.id}"]`); x.submit('#confirmation'); await x.drain(); assert.equal(collection.rows.length, count);
  const audit = x.observed.state.events; assert.equal(audit.length, 3); assert.deepEqual(Object.keys(audit[0]).sort(), ['actor', 'id', 'name', 'outcome', 'target', 'time']); assert.ok(!JSON.stringify(audit).includes('owned-edit'));
});

test('local variable/source constraints refuse authority widening, bad names/references and nonfinite expiry', async t => {
  const x = fixture(t); x.route('variables'); // This alias deliberately routes to the workbench.
  assert.match(x.root.textContent, /Owned workbench/);
  x.route('sources'); const source = x.observed.state.collections.sources.rows[0]; x.action('edit', `[data-id="${source.id}"]`);
  const before = structuredClone(x.observed.state.policy); x.node('#record-form [name=name]').value = 'renamed-owned-source'; x.submit('#record-form'); await x.drain();
  assert.ok(x.node('#form-error').textContent); assert.deepEqual(x.observed.state.policy, before);
  x.action('close-dialog'); x.route('inheritance'); x.action('create'); x.node('#record-form [name=name]').value = 'owned-include'; x.node('#record-form [name=sha256]').value = 'NOT-A-PIN'; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /lowercase SHA-256/);
});

test('native-policy JSON validation covers every refusal without replacing accepted policy or discarding draft', async t => {
  const x = fixture(t); x.route('policy'); const before = structuredClone(x.observed.state.policy);
  const failures = [
    ['x'.repeat(1048577), /1 MiB/], ['[]', /object with version 1/], ['{"version":2}', /version 1/],
    ['{"version":1,"unexpected":true}', /Unknown root field/], ['{"version":1,"sources":{},"workloads":{}}', /non-empty/],
    [JSON.stringify({ version: 1, sources: { a: {} }, workloads: { a: { mode: 'bad', bindings: [] } } }), /mode and non-empty/],
    [JSON.stringify({ version: 1, sources: { a: {} }, workloads: { a: { mode: 'offline', bindings: [{ name: 'bad-name', source: 'a' }] } } }), /invalid\/duplicate/],
    [JSON.stringify({ version: 1, sources: { a: {} }, workloads: { a: { mode: 'offline', bindings: [{ name: 'A', source: 'a' }, { name: 'a', source: 'a' }] } } }), /invalid\/duplicate/],
    [JSON.stringify({ version: 1, sources: { a: {} }, workloads: { a: { mode: 'offline', bindings: [{ name: 'A', source: 'missing' }] } } }), /missing source/],
    [JSON.stringify({ version: 1, sources: { a: {} }, workloads: { a: { mode: 'online', snapshot: { path: 'owned' }, bindings: [{ name: 'A', source: 'a' }] } } }), /offline snapshot/],
  ];
  for (const [text, expected] of failures) { x.change('#policy-json', text, 'input'); x.action('save-policy'); assert.match(x.node('#policy-result').textContent, expected); assert.deepEqual(x.observed.state.policy, before); }
  x.change('#policy-json', JSON.stringify({ version: 1, composition: { includes: [] } }), 'input'); x.action('validate-policy'); assert.match(x.node('#policy-result').textContent, /Preliminary/); assert.deepEqual(x.observed.state.policy, before);
  x.change('#policy-json', JSON.stringify(before), 'input'); x.action('save-policy'); assert.match(x.node('#policy-result').textContent, /Saved locally/);
  x.action('export-policy'); assert.deepEqual(JSON.parse(await x.exports[0].text()), before);
  x.change('#policy-json', '{bad', 'input'); const unload = x.event(x.window, 'beforeunload'); assert.equal(unload.defaultPrevented, true); x.sources[0].onmessage(); assert.equal(x.location.reloads, 0); assert.match(x.notice(), /export\/save or discard/);
  x.action('discard-policy'); x.sources[0].onmessage(); assert.equal(x.location.reloads, 1);
});

test('file import boundaries, image choices, settings, identity and telemetry never make provider requests', async t => {
  const x = fixture(t); x.route('policy'); const picker = x.node('#policy-import');
  x.event(picker, 'change', { target: { files: [] } }); x.event(picker, 'change', { target: { files: [{ size: 1048577 }] } }); await x.drain(); assert.match(x.notice(), /1 MiB/);
  x.event(picker, 'change', { target: { files: [{ size: 4, text: async () => '{bad' }] } }); await x.drain(); assert.ok(x.node('#notice').className.includes('error'));
  x.route('settings');
  for (const type of ['avatar', 'brand']) {
    x.event(x.node('#' + type + '-file'), 'change', { target: { files: [] } });
    x.event(x.node('#' + type + '-file'), 'change', { target: { files: [{ size: 1, type: 'text/html' }] } }); await x.drain(); assert.match(x.notice(), /PNG, JPEG or WebP/);
    x.event(x.node('#' + type + '-file'), 'change', { target: { files: [{ size: 10, type: 'image/png' }] } }); await x.drain(); assert.match(x.notice(), /no upload/); x.action('remove-' + type);
  }
  x.node('[name=name]').value = 'Owned local profile'; x.node('[name=theme]').value = 'dark'; x.node('[name=language]').value = 'ar'; x.submit('#settings-form'); assert.equal(x.observed.state.settings.name, 'Owned local profile'); assert.equal(x.document.documentElement.dir, 'rtl'); assert.equal(x.observed.state.settings.theme, undefined);
  x.route('identity'); x.node('[name=provider]').value = 'Unsupported provider'; x.node('[name=issuer]').value = 'https://issuer.example.invalid/public-metadata'; x.submit('#identity-form'); assert.equal(x.observed.state.identity, 'unsupported'); x.action('connect-identity'); assert.equal(x.observed.state.identity, 'connecting'); await x.advance(500); assert.equal(x.observed.state.identity, 'unsupported'); x.action('disconnect-identity'); assert.equal(x.observed.state.identity, 'not configured');
  x.route('telemetry'); x.node('[name=collector]').value = 'https://collector.example.invalid'; x.submit('#telemetry-settings'); x.action('simulation'); await x.advance(3000); assert.equal(x.observed.state.events.length, 1); x.document.hidden = true; await x.advance(3000); assert.equal(x.observed.state.events.length, 1); x.document.hidden = false; x.action('simulation'); x.action('reset-signals'); assert.equal(x.timers.size, 0);
});

test('metering, billing confirmation, revocation and role changes remain bounded synthetic operations', async t => {
  const x = fixture(t, 'remote'); x.route('account/usage'); x.action('retry-meter'); assert.match(x.notice(), /first/);
  const form = x.node('#meter-form'); form.querySelector('[name=profile]').value = remoteModel.collections.profiles.rows[0].id; form.querySelector('[name=type]').value = 'age retrieval'; form.querySelector('[name=result]').value = 'committed'; x.submit('#meter-form'); assert.equal(x.saved().collections.usage.rows[0].units, 1); const count = x.saved().collections.usage.rows.length; x.action('retry-meter'); assert.equal(x.saved().collections.usage.rows.length, count);
  x.route('account/billing'); x.action('plan'); x.submit('#confirmation'); await x.drain(); assert.equal(x.saved().plan, 'Team demo'); x.action('plan'); x.submit('#confirmation'); await x.drain(); assert.equal(x.saved().plan, 'Free demo'); x.action('cancel-plan'); x.submit('#confirmation'); await x.drain(); assert.equal(x.saved().plan, 'Cancelled demo'); x.action('portal'); x.submit('#confirmation'); await x.drain(); assert.match(x.notice(), /real portal requires/);
  x.change('#persona', 'AI agent'); const stale = new Element('button', x.document); stale.dataset.action = 'plan'; x.root.onclick({ target: stale }); assert.match(x.notice(), /cannot change billing/);
  x.route('operator/grants'); x.change('#persona', 'Global administrator'); x.action('edit'); const id = x.node('[data-action=delete]').dataset.id; x.action('delete'); x.submit('#confirmation'); await x.drain(); assert.equal(x.saved().collections.grants.rows.find(row => row.id === id).status, 'revoked');
});

test('fleet selection, migration interruption, generation preservation, audit cap and cleanup are explicit', async t => {
  const x = fixture(t); x.route('fleet'); x.action('fleet-inspect'); assert.match(x.notice(), /Select at least one/); const checkbox = x.node('[data-select]'); checkbox.checked = true; x.event(checkbox, 'change'); x.action('fleet-inspect'); assert.match(x.node('#fleet-result').textContent, /simulated result|unavailable/); x.action('fleet-refresh');
  x.route('workloads'); x.action('activate'); const generation = x.observed.state.generation; x.action('activation-failure'); assert.equal(x.observed.state.generation, generation);
  x.route('migration'); x.action('migration-failure'); assert.equal(x.observed.state.collections.migration.rows[0].status, 'blocked'); for (let index = 0; index < 8; index++) { x.action('migrate'); x.submit('#confirmation'); await x.drain(); } assert.equal(x.observed.state.collections.migration.rows[0].stage, 'reconciled');
  x.observed.state.collections.migration.rows = []; x.route('migration'); x.action('migrate'); assert.match(x.notice(), /Create a local migration plan/);
  x.route('telemetry'); for (let index = 0; index < 105; index++) { x.action('simulation'); await x.advance(3000); x.action('simulation'); } assert.equal(x.observed.state.events.length, 100);
  x.window.dispatchEvent({ type: 'pagehide' }); assert.equal(x.sources[0].closed, true); assert.equal(x.timers.size, 0);
  x.action('reset'); x.submit('#confirmation'); await x.drain(); x.route('configuration'); assert.equal(x.observed.state.events.length, 0); assert.deepEqual(x.observed.state.policy, localModel.policy);
});

test('actual guided source/create/edit/delete and workload edit preserve a full browser-valid policy atomically', async t => {
  const x = fixture(t); x.route('sources'); x.action('create');
  for (const [key, value] of Object.entries({ name: 'owned', type: 'json', location: './owned.json', domain: 'personal', identity: 'none', status: 'not connected' })) x.node('#record-form [name=' + key + ']').value = value;
  x.submit('#record-form'); await x.until(() => !!x.observed.state.policy.sources.owned); assert.deepEqual(x.observed.state.policy.sources.owned, { type: 'json', path: './owned.json' });
  const record = x.observed.state.collections.sources.rows.at(-1); x.action('edit', `[data-id="${record.id}"]`); x.node('#record-form [name=location]').value = './revised.json'; x.submit('#record-form'); await x.until(() => x.observed.state.policy.sources.owned.path === './revised.json');
  x.action('edit', `[data-id="${record.id}"]`); x.action('delete', `[data-id="${record.id}"]`); x.submit('#confirmation'); await x.until(() => !Object.hasOwn(x.observed.state.policy.sources, 'owned'));
  x.route('workloads'); x.action('edit'); x.node('#record-form [name=mode]').value = 'online'; x.submit('#record-form'); await x.until(() => x.observed.state.policy.workloads.studio.mode === 'online'); assert.ok(x.observed.state.policy.workloads.studio.bindings.length > 0);
  x.action('create'); x.node('#record-form [name=name]').value = 'empty-not-admitted'; x.submit('#record-form'); await x.until(() => !!x.node('#form-error').textContent); assert.equal(Object.hasOwn(x.observed.state.policy.workloads, 'empty-not-admitted'), false);
});

test('stored invalid/blocked state remains memory-local while grant/profile/scope refusals preserve account records', async t => {
  const x = fixture(t, 'remote', { stored: { 'cq.preview.v1.remote': '{bad' }, blockedStorage: true }); assert.match(x.root.textContent, /storage is unavailable/); x.action('event');
  x.setStorageBlocked(false); x.route('account/profiles'); x.action('edit'); const profile = structuredClone(remoteModel.collections.profiles.rows[0]); x.node('#record-form [name=custody]').value = 'HSM'; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /unavailable/); x.action('close-dialog');
  x.route('operator/grants'); x.change('#persona', 'Global administrator'); x.action('edit'); const current = x.node('#record-form [name=profile]').value; x.node('#record-form [name=profile]').value = 'unknown'; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /Unknown profile/); x.node('#record-form [name=profile]').value = current; x.node('#record-form [name=tenant]').value = 'different-tenant'; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /Cross-tenant/); x.action('close-dialog');
  x.route('operator/roles'); x.action('edit'); x.node('#record-form [name=scope]').value = '*'; x.submit('#record-form'); await x.drain(); assert.match(x.node('#form-error').textContent, /explicit bounded scope/); assert.equal(profile.custody, remoteModel.collections.profiles.rows[0].custody);
});
