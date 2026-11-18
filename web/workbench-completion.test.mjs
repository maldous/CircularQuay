import test from 'node:test';
import assert from 'node:assert/strict';
import { browser, Element } from './coverage-dom-fixture.test.mjs';
import { workbench } from './local/workbench.js';
import { model } from './local/model.js';

const copy = value => structuredClone(value);
const row = (extra = {}) => ({ id: 'owned-1', name: 'OWNED_VALUE', value: 'synthetic-only', kind: 'Literal', provider: 'Local .env', resource: '', environment: 'development', group: 'Personal', profile: 'personal-local / age-local-v1', mode: 'local', custody: 'Local age', start: '2026-01-01T09:00', end: '2026-01-01T17:00', zone: 'UTC', origin: 'User override', inherited: '', override: true, required: true, resolution: 'none', revision: 'fixture-r1', ...extra });

function fixture(t, rows = [row()], extra = {}) {
  const env = browser(t, { hash: '#/workbench' }); const root = env.document.querySelector('#app');
  const state = { settings: copy(model.settings), collections: copy(model.collections), policy: copy(model.policy), context: 'development', workbench: { rows: copy(rows), selected: rows[0]?.id, context: 'development', group: 'All groups', log: [], generation: 3, savedAt: '', validated: false, fault: 'none', evaluation: '2026-01-01T10:00', ...extra } };
  const saves = []; const changes = []; const exports = [];
  const hooks = { save() { saves.push(copy(state)); }, changed(value) { changes.push(value); }, exportJSON(value, name) { exports.push({ value: copy(value), name }); } };
  workbench(root, state, hooks);
  const select = selector => { const node = root.querySelector(selector); assert.ok(node, `actual rendered ${selector}`); return node; };
  const action = (name, more = '') => name === 'row-save' ? env.event(select('#wb-row-form'), 'submit') : env.event(select(`[data-wb="${name}"]${more}`), 'click', { bubbles: true });
  const input = (name, value, type = 'input') => { const node = select(`[name="${name}"]`); node.type === 'checkbox' ? node.checked = value : node.value = value; env.event(node, type, { bubbles: true }); };
  return { ...env, blobs: env.exports, root, state, saves, changes, exports, select, action, input, message: () => select('#wb-message').textContent };
}

test('workbench seeds only synthetic rows and renders empty, locked and conflict provenance honestly', t => {
  const x = fixture(t); delete x.state.workbench; workbench(x.root, x.state, { save() {}, changed() {}, exportJSON() {} });
  assert.equal(x.state.workbench.rows.length, model.collections.variables.rows.length + 1);
  assert.ok(x.state.workbench.rows.some(value => value.origin === 'Conflicting group defaults'));
  assert.match(x.root.textContent, /no native\/provider execution|synthetic adapters/i);
  x.window.dispatchEvent({ type: 'pagehide' });
});

test('every reachable validation denial preserves the prior complete generation and native policy', t => {
  const cases = [
    [{ name: 'bad-name' }, /invalid variable name/], [{ name: '' }, /invalid variable name/],
    [{ value: '' }, /required value is empty/], [{ kind: 'Provider reference', resource: ' ' }, /exact provider resource/],
    [{ profile: 'unknown' }, /unknown key profile/], [{ provider: 'unknown' }, /unsupported provider/],
    [{ mode: 'unknown' }, /unsupported provider/], [{ custody: 'unknown' }, /unsupported custody/],
    [{ custody: 'HSM required' }, /stronger custody unavailable/], [{ custody: 'Independent KMS required' }, /stronger custody unavailable/],
    [{ mode: 'hybrid', fallback: false }, /explicit availability-only/], [{ start: 'bad' }, /complete date and time/],
    [{ start: '2026-02-30T09:00' }, /Invalid calendar date/], [{ end: '2026-01-01T08:00' }, /end must follow/],
    [{ start: '2026-03-08T02:30', end: '2026-03-08T04:00', zone: 'America/New_York' }, /does not exist/],
    [{ start: '2026-11-01T01:30', end: '2026-11-01T04:00', zone: 'America/New_York' }, /ambiguous/],
    [{ zone: 'not-a-time-zone' }, /time zone/i], [{ origin: 'Conflicting group defaults' }, /no approved resolution/],
  ];
  const x = fixture(t); const policy = copy(x.state.policy);
  for (const [change, expected] of cases) {
    x.state.workbench.rows = [row(change)]; workbench(x.root, x.state, { save() {}, changed() {}, exportJSON() {} });
    x.action('apply'); assert.match(x.message(), expected); assert.equal(x.state.workbench.generation, 3); assert.deepEqual(x.state.policy, policy);
  }
  x.state.workbench.rows = [row(), row({ id: 'owned-2', name: 'owned_value' })]; workbench(x.root, x.state, { save() {}, changed() {}, exportJSON() {} });
  x.action('validate'); assert.match(x.message(), /duplicate name/);
  x.state.workbench.rows = []; workbench(x.root, x.state, { save() {}, changed() {}, exportJSON() {} }); x.action('apply'); assert.match(x.message(), /at least one/); assert.match(x.root.textContent, /No variable selected/);
});

test('masking, inheritance, conflict resolution and keyboard tabs preserve values without disclosure effects', t => {
  const x = fixture(t, [row({ origin: 'User override', inherited: '<owned-inherited>', override: false }), row({ id: 'conflict', name: 'REGION', origin: 'Conflicting group defaults', resolution: 'approved-demo-au' }), row({ id: 'locked', name: 'LOCKED', origin: 'Organization locked', inherited: 'retained' })]);
  assert.equal(x.select('[name="value"]').type, 'password'); x.action('reveal'); assert.equal(x.select('[name="value"]').type, 'text');
  x.action('tab', '[data-tab="inherit"]'); assert.match(x.root.textContent, /<owned-inherited>/); assert.equal(x.root.querySelector('script'), null);
  x.event(x.select('.wb-tabs'), 'keydown', { key: 'End' }); assert.equal(x.select('[role=tab][aria-selected=true]').dataset.tab, 'inherit');
  for (const key of ['Home', 'ArrowRight', 'ArrowLeft', 'ignored']) x.event(x.select('.wb-tabs'), 'keydown', { key });
  x.action('select', '[data-id="conflict"]'); x.action('tab', '[data-tab="inherit"]'); assert.match(x.root.textContent, /au-east/);
  x.input('resolution', 'approved-demo-eu', 'change'); x.action('row-save'); assert.equal(x.state.workbench.rows[1].resolution, 'approved-demo-eu'); x.action('select', '[data-id="locked"]'); assert.equal(x.root.querySelector('fieldset').disabled, true);
});

test('row draft guards cover selection, navigation, context, group, import, save and apply until discard', t => {
  const x = fixture(t, [row(), row({ id: 'second', name: 'SECOND' })]); x.input('value', 'changed');
  for (const name of ['add', 'select', 'import', 'save', 'apply']) { x.action(name, name === 'select' ? '[data-id="second"]' : ''); assert.match(x.message(), /Save|save|Unsaved/); }
  const context = x.select('#wb-context'); context.value = 'staging'; x.event(context, 'change'); assert.equal(x.state.workbench.context, 'development');
  const group = x.select('#wb-group'); group.value = 'Engineering'; x.event(group, 'change'); assert.equal(x.state.workbench.group, 'All groups');
  const link = x.select('a[href="#/sources"]'); assert.equal(x.event(link, 'click', { bubbles: true }).defaultPrevented, true);
  x.action('discard'); assert.equal(x.state.workbench.rows[0].value, 'synthetic-only');
  const cleanContext = x.select('#wb-context'); cleanContext.value = 'staging'; x.event(cleanContext, 'change'); assert.equal(x.state.workbench.generation, 0); assert.match(x.root.textContent, /No variables in this view/);
});

test('add/save/delete/reset and filters operate only on owned browser rows, retaining native policy', t => {
  const x = fixture(t); const policy = copy(x.state.policy); x.action('add');
  x.action('row-save'); assert.match(x.message(), /invalid variable name/); x.input('name', 'NEW_VALUE'); x.input('value', 'synthetic-new'); x.action('row-save');
  assert.equal(x.state.workbench.rows.length, 2); assert.equal(x.state.workbench.rows[1].name, 'NEW_VALUE'); assert.ok(x.saves.length);
  const search = x.select('#wb-search'); search.value = 'NEW_'; x.event(search, 'input'); assert.equal(x.root.querySelectorAll('tr.selected').length, 1);
  x.action('delete'); x.event(x.select('#wb-confirm'), 'submit'); assert.equal(x.state.workbench.rows.length, 1); assert.deepEqual(x.state.policy, policy);
  x.action('save'); assert.ok(x.state.workbench.savedAt); x.action('reset'); x.event(x.select('#wb-confirm'), 'submit'); assert.deepEqual(x.state.policy, policy); assert.equal(x.state.workbench.generation, 0);
});

test('required windows, denied adapters and availability-only fallback never partially advance generation', t => {
  const x = fixture(t, [row({ mode: 'hybrid', fallback: true })]); x.action('validate'); assert.equal(x.state.workbench.validated, true);
  for (const evaluation of ['invalid', '2026-01-01T08:59', '2026-01-01T17:00']) { x.state.workbench.evaluation = evaluation; x.action('run'); assert.equal(x.state.workbench.generation, 3); assert.match(x.message(), /preview time|availability window/); }
  x.state.workbench.evaluation = '2026-01-01T10:00';
  for (const fault of ['access denied', 'audit unavailable', 'malformed source']) { x.state.workbench.fault = fault; x.action('apply'); assert.equal(x.state.workbench.generation, 3); assert.match(x.message(), /Previous complete generation preserved/); }
  x.state.workbench.fault = 'source unavailable'; x.action('apply'); assert.equal(x.state.workbench.generation, 4); assert.match(x.state.workbench.log.join(' '), /availability-only snapshot/);
  x.state.workbench.fault = 'none'; x.action('run'); assert.equal(x.state.workbench.generation, 5); assert.match(x.message(), /No actual workload/);
});

test('all malformed .env and backup imports reject atomically; valid imports preserve literal and policy fidelity', t => {
  const x = fixture(t); const before = copy(x.state.workbench.rows); const policy = copy(x.state.policy);
  const invalid = ['# only comment', 'not a declaration', 'OWNED_VALUE=duplicate', 'GOOD=x\nGOOD=y', 'QUOTED="broken', 'ZERO=\0', 'BIG=' + 'x'.repeat(4097), ' '.repeat(65537), '{"format":"wrong"}', '{"format":"cq-workbench-demo-v1","configuration":{"rows":[null]}}'];
  for (const text of invalid) { x.action('import'); x.select('#wb-env-text').value = text; x.event(x.select('#wb-import'), 'submit'); assert.ok(x.select('#wb-import-error').textContent); assert.deepEqual(x.state.workbench.rows, before); x.action('close'); }
  x.action('import'); x.select('#wb-env-text').value = '# public fixture\nexport FIRST="$(not-executed)"\nSECOND=\'raw<text>\'\n'; x.event(x.select('#wb-import'), 'submit');
  assert.equal(x.state.workbench.rows.length, 3); assert.equal(x.state.workbench.rows[1].value, '$(not-executed)'); assert.equal(x.state.workbench.rows[2].value, 'raw<text>'); assert.deepEqual(x.state.policy, policy);
  const backup = { format: 'cq-workbench-demo-v1', configuration: copy(x.state.workbench), nativePolicy: { ignored: true } };
  for (const change of [value => { value.configuration.rows[0].required = 'yes'; }, value => { value.configuration.rows.push(copy(value.configuration.rows[0])); }, value => { value.configuration.context = 1; }, value => { value.configuration.log = [null]; }, value => { value.configuration.generation = null; }]) {
    const invalidBackup = copy(backup); change(invalidBackup); const rows = copy(x.state.workbench.rows); x.action('import'); x.select('#wb-env-text').value = JSON.stringify(invalidBackup); x.event(x.select('#wb-import'), 'submit'); assert.deepEqual(x.state.workbench.rows, rows); assert.ok(x.select('#wb-import-error').textContent); x.action('close');
  }
  backup.configuration.rows = [row({ name: 'RESTORED', zone: 'UTC', origin: 'Retained provenance' })]; backup.configuration.selected = 'missing'; x.action('import'); x.select('#wb-env-text').value = JSON.stringify(backup); x.event(x.select('#wb-import'), 'submit'); assert.equal(x.state.workbench.rows[0].name, 'RESTORED'); assert.deepEqual(x.state.policy, policy);
});

test('explicit local file/export controls have byte limits and preserve full-vs-env policy distinctions', async t => {
  const x = fixture(t); x.action('import'); const file = x.select('#wb-env-file');
  await file.onchange({ target: { files: [] } }); await file.onchange({ target: { files: [{ size: 65537 }] } }); assert.match(x.select('#wb-import-error').textContent, /64 KiB/);
  const before = copy(x.state.workbench); x.select('#wb-env-text').value = 'RETAIN_DRAFT'; await file.onchange({ target: { files: [{ size: 1, text: async () => { throw Error('PRIVATE_FILE_DATA'); } }] } }); assert.match(x.select('#wb-import-error').textContent, /could not be read/); assert.ok(!x.root.textContent.includes('PRIVATE_FILE_DATA')); assert.equal(x.select('#wb-env-text').value, 'RETAIN_DRAFT'); assert.deepEqual(x.state.workbench, before);
  let resolveOld; const old = file.onchange({ target: { files: [{ size: 1, text: () => new Promise(resolve => { resolveOld = resolve; }) }] } });
  await file.onchange({ target: { files: [{ size: 12, text: async () => 'FILE=literal' }] } }); resolveOld('STALE=never'); await old; assert.equal(x.select('#wb-env-text').value, 'FILE=literal'); x.action('close');
  x.action('export'); x.event(x.select('#wb-export'), 'submit'); assert.equal(x.exports[0].name, 'cq-workbench-local.json'); assert.deepEqual(x.exports[0].value.nativePolicy, x.state.policy);
  x.action('export'); x.select('[name=format]').value = 'env'; x.event(x.select('#wb-export'), 'submit'); assert.equal(await x.blobs[0].text(), 'OWNED_VALUE="synthetic-only"\n'); assert.equal(x.exports.length, 1);
  assert.equal(x.exports.length, 1); assert.equal(x.revoked.length, 0); await x.advance(1000); assert.equal(x.revoked.length, 1);
});

test('new file selection, cancellation and replaced import dialogs invalidate old successes and failures', async t => {
  const x = fixture(t); const before = copy(x.state.workbench);
  for (const fail of [false, true]) {
    x.action('import'); const input = x.select('#wb-env-file'); let resolve, reject;
    const old = input.onchange({ target: { files: [{ size: 1, text: () => new Promise((a, b) => { resolve = a; reject = b; }) }] } });
    x.action('close'); x.action('import'); x.select('#wb-env-text').value = 'NEW_OWNED_DRAFT'; fail ? reject(Error('PRIVATE_STALE_FILE')) : resolve('STALE=do-not-copy'); await old;
    assert.equal(x.select('#wb-env-text').value, 'NEW_OWNED_DRAFT'); assert.equal(x.select('#wb-import-error').textContent, ''); x.action('close');
  }
  x.action('import'); const file = x.select('#wb-env-file'); let reject; const old = file.onchange({ target: { files: [{ size: 1, text: () => new Promise((_, b) => { reject = b; }) }] } }); await file.onchange({ target: { files: [] } }); reject(Error('PRIVATE_CANCELLED_FILE')); await old; assert.equal(x.select('#wb-import-error').textContent, ''); assert.deepEqual(x.state.workbench, before);
});
