import { parseDocument } from './schema-tools.js';
const h = value => String(value ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const copy = value => JSON.parse(JSON.stringify(value));
const zones = ['UTC', ...(Intl.supportedValuesOf ? Intl.supportedValuesOf('timeZone') : ['Australia/Melbourne', 'America/Los_Angeles', 'Europe/London', 'Asia/Tokyo'])];
const providers = ['Local .env', 'Local JSON', 'SOPS / age', 'OpenBao', 'Vault', 'AWS Secrets Manager', 'AWS SSM', 'Azure Key Vault', 'Google Secret Manager', 'Doppler', 'Infisical', 'Bitwarden', 'Akeyless', '1Password'];
const profiles = ['personal-local / age-local-v1', 'studio-hosted / kms-demo-v1', 'release-hosted / age-demo-v2'];

// Browser-only window interpretation. Reject DST gaps instead of silently changing
// the declared wall time. Ambiguous fall-back times need explicit UTC in this demo.
function instant(wall, zone) {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(wall)) throw new Error('Use a complete date and time.');
  const [year, month, day] = wall.slice(0, 10).split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) throw new Error('Invalid calendar date.');
  const wanted = Date.parse(wall + ':00Z');
  const formatter = new Intl.DateTimeFormat('en-CA', { timeZone: zone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23' });
  const parts = value => Object.fromEntries(formatter.formatToParts(value).map(p => [p.type, p.value]));
  const formattedUTC = value => { const p = parts(value); return Date.parse(`${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}:${p.second}Z`); };
  let time = wanted;
  for (let i = 0; i < 3; i++) time += wanted - formattedUTC(time);
  if (formattedUTC(time) !== wanted) throw new Error('This wall time does not exist in the selected timezone (DST gap).');
  if (formattedUTC(time + 3600000) === wanted || formattedUTC(time - 3600000) === wanted) throw new Error('This wall time is ambiguous (DST). Use UTC explicitly.');
  return time;
}

function seed(state) {
  const day = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
  const rows = state.collections.variables.rows.map((v, index) => ({
    id: v.id, name: v.name, value: v.value, kind: v.kind === 'Source binding' ? 'Provider reference' : 'Literal', provider: v.source ? 'Local JSON' : 'Local .env', resource: v.source ? `${v.source}/${v.value}` : '', environment: v.environment, group: 'Personal', profile: profiles[index % profiles.length], mode: 'local', custody: 'Local age', start: `${day}T09:00`, end: `${day}T17:00`, zone: 'Australia/Melbourne', origin: v.origin, inherited: v.origin === 'Organization locked' ? v.value : v.name === 'LOG_LEVEL' ? 'warn' : '', override: v.origin !== 'Organization locked', required: true, resolution: 'none', revision: 'demo-r1',
  }));
  rows.push({ id: 'demo-conflict', name: 'SERVICE_REGION', value: '', kind: 'Literal', provider: 'Local .env', resource: '', environment: 'development', group: 'Engineering + Operations', profile: profiles[0], mode: 'local', custody: 'Local age', start: `${day}T09:00`, end: `${day}T17:00`, zone: 'UTC', origin: 'Conflicting group defaults', inherited: 'au-east / eu-west', override: false, required: true, resolution: 'none', revision: 'demo-r1' });
  return { rows, selected: rows[1]?.id || rows[0]?.id, context: state.context || 'development', group: 'All groups', log: ['Ready. Local synthetic adapters; no native/provider execution.'], generation: 0, savedAt: '', validated: false, fault: 'none', document: state.policy };
}

function effective(row) {
  if (row.origin === 'Conflicting group defaults') return row.resolution === 'none' ? null : row.resolution === 'approved-demo-au' ? 'au-east' : 'eu-west';
  return row.origin === 'Organization locked' || (!row.override && row.inherited) ? row.inherited : row.kind === 'Provider reference' ? `[${row.provider}: ${row.resource}]` : row.value;
}

function problems(rows) {
  const errors = [];
  const names = new Set();
  for (const row of rows) {
    const prefix = row.name || 'Unnamed variable';
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(row.name)) errors.push(`${prefix}: invalid variable name.`);
    const name = `${row.environment}:${row.name.toLowerCase()}`;
    if (names.has(name)) errors.push(`${prefix}: duplicate name in ${row.environment}.`);
    names.add(name);
    if (effective(row) === null) errors.push(`${prefix}: conflicting group defaults; no approved resolution.`);
    if (row.kind === 'Literal' && effective(row) === '' && row.required) errors.push(`${prefix}: required value is empty.`);
    if (row.kind === 'Provider reference' && !row.resource.trim()) errors.push(`${prefix}: exact provider resource/field is required.`);
    if (!profiles.includes(row.profile)) errors.push(`${prefix}: unknown key profile.`);
    if (!providers.includes(row.provider) || !['local', 'remote', 'hybrid'].includes(row.mode)) errors.push(`${prefix}: unsupported provider or resolution mode.`);
    if (!['Local age', 'Hosted software KMS + age', 'Independent KMS required', 'HSM required'].includes(row.custody)) errors.push(`${prefix}: unsupported custody profile.`);
    if (row.custody === 'Independent KMS required' || row.custody === 'HSM required') errors.push(`${prefix}: required stronger custody unavailable; software cannot substitute.`);
    if (row.mode === 'hybrid' && !row.fallback) errors.push(`${prefix}: hybrid requires an explicit availability-only snapshot choice.`);
    try { if (instant(row.end, row.zone) <= instant(row.start, row.zone)) errors.push(`${prefix}: availability end must follow its start.`); }
    catch (error) { errors.push(`${prefix}: ${error.message}`); }
  }
  return errors;
}

export function workbench(root, state, hooks) {
  const data = state.workbench ||= seed(state);
  let selected = data.rows.find(row => row.id === data.selected) || data.rows[0];
  let draft = selected ? copy(selected) : null;
  let edited = false;
  let reveal = false;
  let query = '';
  let statusFilter = 'All';
  data.evaluation ||= new Date(Date.now() + 86400000).toISOString().slice(0, 10) + 'T02:00';
  let message = '';
  let messageError = false;
  let tab = 'value';
  const log = text => { data.log.unshift(text); data.log = data.log.slice(0, 30); };
  const persist = () => { hooks.save(); };
  const dirty = flag => { edited = flag; hooks.changed(flag); };
  const currentRows = () => data.rows.filter(row => row.environment === data.context && (data.group === 'All groups' || row.group === data.group));
  const option = (values, current) => (values.includes(current) ? values : [current, ...values]).map(value => `<option ${value === current ? 'selected' : ''}>${h(value)}</option>`).join('');
  const input = (label, key, type = 'text', extra = '') => `<label>${label}<input name="${key}" type="${type}" value="${h(draft?.[key])}" ${extra}></label>`;
  const select = (label, key, values) => `<label>${label}<select name="${key}">${option(values, draft?.[key])}</select></label>`;
  const pulse = text => { message = text; messageError = false; render(); };
  const fail = text => { message = text; messageError = true; render(); };
  function markChanged(event) {
    const target = event.target;
    if (!draft || !target.name) return;
    draft[target.name] = target.type === 'checkbox' ? target.checked : target.value;
    dirty(true);
  }

  function inspector() {
    if (!draft) return '<div class="wb-inspector"><h2>No variable selected</h2><p>Add a variable or import a .env file.</p></div>';
    const locked = draft.origin === 'Organization locked';
    const value = effective(draft);
    const issues = problems([draft]);
    let fields;
    if (tab === 'value') fields = `<fieldset ${locked ? 'disabled' : ''}>${input('Variable name', 'name', 'text', 'required pattern="[A-Za-z_][A-Za-z0-9_]*"')}${select('Value kind', 'kind', ['Literal', 'Provider reference'])}<label>${draft.kind === 'Literal' ? 'Local value (masked)' : 'Synthetic local placeholder (masked)'}<div class="wb-value"><input name="value" type="${reveal ? 'text' : 'password'}" value="${h(draft.value)}" autocomplete="off" spellcheck="false"><button type="button" data-wb="reveal">${reveal ? 'Mask' : 'Reveal'}</button></div></label>${input('Exact resource / field reference', 'resource')}${select('Provider', 'provider', providers)}${select('Key profile / key ID', 'profile', profiles)}<div class="wb-two">${select('Resolution mode', 'mode', ['local', 'remote', 'hybrid'])}${select('Key custody', 'custody', ['Local age', 'Hosted software KMS + age', 'Independent KMS required', 'HSM required'])}</div><label class="check-label"><input name="required" type="checkbox" ${draft.required ? 'checked' : ''}> Required before complete activation</label><label class="check-label"><input name="fallback" type="checkbox" ${draft.fallback ? 'checked' : ''}> Hybrid: explicitly permit availability-only snapshot</label></fieldset><p class="wb-help">All provider results are synthetic. Local age / hosted age / KMS are distinct. Retained keys or plaintext cannot be recalled.</p>`;
    else if (tab === 'window') fields = `<fieldset ${locked ? 'disabled' : ''}><h3>Availability window</h3>${input('Available from', 'start', 'datetime-local', 'required')}${input('Available until', 'end', 'datetime-local', 'required')}${select('Time zone', 'zone', zones)}<p class="wb-help">Wall times use the selected IANA zone, including DST. Parent windows intersect; a local setting cannot widen authority. This is a local design simulation.</p>${windowPreview(draft)}<div class="wb-two">${select('Environment', 'environment', state.collections.environments.rows.map(e => e.name))}${select('Group association', 'group', ['Personal', 'Engineering', 'Operations', 'Engineering + Operations'])}</div></fieldset>`;
    else fields = `<h3>Effective value & provenance</h3><div class="wb-effective"><code>${value === null ? 'CONFLICT — no effective value' : reveal ? h(value) : '••••••••'}</code><button type="button" data-wb="reveal">${reveal ? 'Mask' : 'Reveal'}</button></div><dl class="wb-provenance"><dt>Organization</dt><dd>${draft.origin === 'Organization locked' ? 'Locked · demo-r1' : 'Security floors · no authority widening'}</dd><dt>Group</dt><dd>${h(draft.group)} ${draft.origin === 'Conflicting group defaults' ? '· au-east ≠ eu-west' : ''}</dd><dt>Profile / key</dt><dd>${h(draft.profile)}</dd><dt>Source</dt><dd>${h(draft.provider)} ${h(draft.resource)}</dd><dt>Effective origin</dt><dd>${h(draft.origin)}</dd><dt>Revision</dt><dd>${h(draft.revision)} · local simulation</dd></dl>${draft.origin === 'Conflicting group defaults' ? select('Approved conflict resolution (simulation)', 'resolution', ['none', 'approved-demo-au', 'approved-demo-eu']) : `<fieldset ${locked ? 'disabled' : ''}><label class="check-label"><input name="override" type="checkbox" ${draft.override ? 'checked' : ''}> Use local override (where allowed)</label><p>Inherited value: ${draft.inherited ? reveal ? h(draft.inherited) : '••••••••' : 'No scalar inherited'}. Parent locks remain in force.</p></fieldset>`}<p class="wb-help">Approved demo resolution represents a separately authorized central revision, not end-user authority to override a conflict.</p>`;
    return `<section class="wb-inspector" aria-label="Guided variable inspector"><div class="wb-inspector-head"><div><small>GUIDED INSPECTOR</small><h2>${h(draft.name || 'New variable')}</h2></div><span class="badge ${locked ? 'warn' : ''}">${locked ? 'Inherited · locked' : edited ? 'Unsaved draft' : 'Saved row'}</span></div><div class="wb-tabs" role="tablist" aria-label="Variable details">${[['value', 'Value & source'], ['window', 'Availability'], ['inherit', 'Inheritance']].map(([key, name]) => `<button type="button" role="tab" aria-selected="${key === tab}" data-wb="tab" data-tab="${key}">${name}</button>`).join('')}</div><form id="wb-row-form">${fields}${issues.length ? `<div class="wb-issues" role="status">${issues.map(i => `<p>${h(i)}</p>`).join('')}</div>` : '<p class="wb-valid">Row checks pass · full native checks remain separate</p>'}<div class="wb-inspector-actions"><button ${locked ? 'disabled' : ''}>Save row</button><button type="button" class="quiet" data-wb="discard">Discard</button><button type="button" class="danger" data-wb="delete" ${locked ? 'disabled' : ''}>Delete</button></div></form></section>`;
  }
  function windowPreview(row) {
    try { return `<p class="wb-window-utc">UTC: ${new Date(instant(row.start, row.zone)).toISOString().slice(0, 16)} → ${new Date(instant(row.end, row.zone)).toISOString().slice(0, 16)}</p>`; }
    catch (error) { return `<p class="error">${h(error.message)}</p>`; }
  }
  function render() {
    const rows = currentRows().filter(row => row.name.toLowerCase().includes(query.toLowerCase()) && (statusFilter === 'All' || (statusFilter === 'Conflicts' ? problems([row]).length : row.origin === 'Organization locked')));
    root.innerHTML = `<a class="skip" href="#wb-grid">Skip to variables</a><div class="workbench"><header class="wb-header"><a class="wordmark" href="#/landing">CircularQuay<span class="pronunciation">LOCAL CLIENT / END USER</span></a><label>Environment<select id="wb-context">${option(state.collections.environments.rows.map(e => e.name), data.context)}</select></label><label>Group<select id="wb-group">${option(['All groups', 'Personal', 'Engineering', 'Operations', 'Engineering + Operations'], data.group)}</select></label><div class="wb-header-right"><span class="badge">SYNTHETIC · LOCAL ONLY</span><a href="#/settings">${h(state.settings.name)} · Settings</a></div></header><nav class="wb-nav" aria-label="Configuration workflows"><a href="#/workbench" aria-current="page">Variables</a><a href="#/sources">Sources</a><a href="#/inheritance">Includes & locks</a><a href="#/workloads">Workloads & delivery</a><a href="#/configuration">Complete configuration</a><a href="#/policy">JSON escape hatch</a><a href="#/custody">Custody</a><a href="#/migration">Migration</a><a href="#/fleet">Devices</a><a href="#/integrations">Integrations</a><a href="#/identity">Identity</a><a href="#/telemetry">Telemetry</a></nav><main class="wb-main"><div class="wb-title"><div><h1>Environment variables <span>${h(data.context)}</span></h1><p>Masked by default · ${data.rows.length} local declarations · inherited values stay traceable</p></div><div class="wb-actions"><button data-wb="add">+ Add variable</button><button class="quiet" data-wb="import">Import .env</button><button class="quiet" data-wb="export">Export</button><button class="quiet" data-wb="save">Save workspace</button></div></div><div id="wb-message" class="wb-message ${messageError ? 'error' : ''}" role="status" aria-live="polite">${h(message || (edited ? 'Unsaved row draft. Save or discard before changing the selection.' : data.savedAt ? `Saved locally ${data.savedAt}` : 'Local workspace ready. Select a variable to inspect source, custody and availability.'))}</div><div class="wb-layout"><section class="wb-grid-section"><div class="wb-grid-tools"><label>Find variable<input id="wb-search" type="search" placeholder="Name…" value="${h(query)}"></label><label>Show<select id="wb-filter">${option(['All', 'Conflicts', 'Inherited locks'], statusFilter)}</select></label><span>${rows.length} visible</span></div><div class="wb-grid-scroll"><table id="wb-grid" tabindex="-1"><caption class="sr-only">Local environment variable workbench</caption><thead><tr><th>Variable / effective value</th><th>Provider · profile / key</th><th>Availability · time zone</th><th>Provenance / validation</th></tr></thead><tbody>${rows.length ? rows.map(row => { const errors = problems([row]); return `<tr class="${row.id === draft?.id ? 'selected' : ''}"><td><button class="wb-variable" data-wb="select" data-id="${h(row.id)}">${h(row.name)}</button><code class="wb-masked">${effective(row) === null ? '— conflict —' : '••••••••'}</code></td><td><span>${h(row.provider)}</span><small>${h(row.profile)}</small></td><td><span>${h(row.start.slice(11))}–${h(row.end.slice(11))}</span><small>${h(row.zone)}</small><small>${h(row.start.slice(0, 10))}</small></td><td><span class="badge ${errors.length ? 'bad' : row.origin === 'Organization locked' ? 'warn' : ''}">${errors.length ? 'Needs attention' : row.origin === 'Organization locked' ? 'Locked' : 'Valid row'}</span><small>${h(row.origin)}</small></td></tr>`; }).join('') : '<tr><td colspan="4" class="empty">No variables in this view. Add/import a variable or adjust filters.</td></tr>'}</tbody></table></div><div class="wb-grid-footer">Local declarations only. Effective outputs are simulated; no provider secrets are fetched.</div><section class="wb-run"><div><h2>Validate & apply</h2><p>Complete candidate required. Failed/denied inputs retain the previous generation.</p></div><label>Preview time (UTC)<input id="wb-evaluation" type="datetime-local" value="${h(data.evaluation)}"></label><label>Adapter scenario<select id="wb-fault">${option(['none', 'source unavailable', 'access denied', 'audit unavailable', 'malformed source'], data.fault)}</select></label><div class="actions"><button data-wb="validate" class="quiet">Validate</button><button data-wb="apply">Mock apply</button><button data-wb="run" class="quiet">Mock run</button></div></section><section class="wb-pulse" aria-label="Local execution feedback"><header><h2>Execution feedback</h2><span>generation ${data.generation} · no process execution</span></header><ol>${data.log.slice(0, 6).map(text => `<li>${h(text)}</li>`).join('')}</ol></section></section>${inspector()}</div></main><footer><span>Configuration, source references and values stay on this client. No upload path. Software custody is not HSM custody.</span><button class="quiet" data-wb="reset">Reset demo…</button></footer><dialog id="wb-dialog" aria-labelledby="wb-dialog-title"></dialog></div>`;
    bind();
  }
  function bind() {
    root.onclick = event => {
      if (event.target.closest('.skip')) { event.preventDefault(); document.querySelector('#wb-grid').focus(); return; }
      if (edited && event.target.closest('a[href]')) { event.preventDefault(); fail('Save or discard the row draft before leaving the workbench.'); return; }
      const button = event.target.closest('[data-wb]');
      if (button) act(button.dataset.wb, button.dataset);
    };
    const on = (id, event, fn) => document.getElementById(id)?.addEventListener(event, fn);
    root.querySelector('.wb-tabs')?.addEventListener('keydown', event => { if (!['ArrowRight', 'ArrowLeft', 'Home', 'End'].includes(event.key)) return; event.preventDefault(); const tabs = ['value', 'window', 'inherit']; tab = event.key === 'Home' ? tabs[0] : event.key === 'End' ? tabs[2] : tabs[(tabs.indexOf(event.target.dataset.tab || tab) + (event.key === 'ArrowRight' ? 1 : 2)) % 3]; render(); root.querySelector('[role=tab][aria-selected=true]').focus(); });
    on('wb-row-form', 'input', markChanged);
    on('wb-row-form', 'change', event => { markChanged(event); if (event.target.tagName === 'SELECT') render(); });
    on('wb-row-form', 'submit', event => { event.preventDefault(); saveRow(); });
    on('wb-search', 'input', event => { query = event.target.value; render(); const el = document.querySelector('#wb-search'); el.focus(); el.setSelectionRange(query.length, query.length); });
    on('wb-filter', 'change', event => { statusFilter = event.target.value; render(); });
    on('wb-context', 'change', event => {
      if (edited) { event.target.value = data.context; fail('Save or discard the row draft before switching context.'); return; }
      data.context = event.target.value; state.context = data.context; data.generation = 0; data.validated = false; selected = currentRows()[0]; draft = selected ? copy(selected) : null; data.selected = draft?.id; persist(); render();
    });
    on('wb-group', 'change', event => { if (edited) { fail('Save or discard the draft before filtering groups.'); return; } data.group = event.target.value; render(); });
    on('wb-evaluation', 'change', event => { data.evaluation = event.target.value; persist(); });
    on('wb-fault', 'change', event => { data.fault = event.target.value; persist(); });
  }
  function saveRow() {
    if (!draft || draft.origin === 'Organization locked') return;
    const candidates = data.rows.filter(row => row.id !== draft.id).concat(draft);
    const own = problems(candidates).filter(error => error.startsWith(`${draft.name}:`) || error.startsWith('Unnamed variable:'));
    if (own.length) { fail(own.join(' ')); return; }
    const index = data.rows.findIndex(row => row.id === draft.id);
    index < 0 ? data.rows.push(copy(draft)) : data.rows[index] = copy(draft);
    selected = data.rows.find(row => row.id === draft.id); data.selected = selected.id; data.validated = false; dirty(false); persist(); pulse('Row saved locally. Validate the complete candidate before mock apply.');
  }
  function act(action, attributes = {}) {
    if (action === 'select') {
      if (edited) { fail('Unsaved row draft. Save row or Discard before selecting another variable.'); return; }
      selected = data.rows.find(row => row.id === attributes.id); draft = copy(selected); data.selected = selected.id; reveal = false; tab = 'value'; render();
    } else if (action === 'tab') { tab = attributes.tab; render(); }
    else if (action === 'reveal') { reveal = !reveal; render(); }
    else if (action === 'discard') { draft = selected ? copy(selected) : null; dirty(false); reveal = false; pulse('Row draft discarded. Saved values are unchanged.'); }
    else if (action === 'add') {
      if (edited) { fail('Save or discard the current row first.'); return; }
      const day = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
      selected = null; draft = { id: crypto.randomUUID(), name: '', value: '', kind: 'Literal', provider: 'Local .env', resource: '', profile: profiles[0], mode: 'local', custody: 'Local age', start: `${day}T09:00`, end: `${day}T17:00`, zone: 'UTC', environment: data.context, group: 'Personal', origin: 'User override', inherited: '', override: true, required: true, resolution: 'none', revision: 'local-draft' }; dirty(true); tab = 'value'; render(); document.querySelector('[name=name]').focus();
    } else if (action === 'delete') {
      if (!selected || selected.origin === 'Organization locked') return;
      openConfirm('Delete local variable?', 'This changes only the local demo declaration. It does not revoke an issuer credential or recall consumer copies.', () => { data.rows = data.rows.filter(row => row.id !== selected.id); selected = data.rows[0]; draft = selected ? copy(selected) : null; dirty(false); data.validated = false; persist(); pulse('Variable removed from local workspace.'); });
    } else if (action === 'import') { if (edited) fail('Save or discard the draft before importing.'); else importDialog(); }
    else if (action === 'save') {
      if (edited) { fail('Save or discard the row draft first; it is separate from the workspace.'); return; }
      data.savedAt = new Date().toISOString(); state.workbench = data; persist(); pulse('Workspace saved to this browser. Reload restores rows, windows, profiles and context.');
    } else if (action === 'export') {
      openExport();
    } else if (action === 'validate' || action === 'apply' || action === 'run') {
      if (edited) { fail('Unsaved row draft. Save or discard before validation or apply.'); return; }
      const errors = problems(currentRows());
      if (!currentRows().length) errors.push('Select at least one configured variable.');
      if (errors.length) { data.validated = false; log(`Candidate denied: ${errors.length} validation issue(s); generation ${data.generation} preserved.`); persist(); fail(errors.join(' ')); return; }
      if (action === 'validate') { data.validated = true; log('Local model checks passed: names, references, windows, conflicts and custody requirements. Native/schema/provider checks remain separate.'); persist(); pulse('Local candidate checks passed. No source read, native validation or network request.'); return; }
      const now = Date.parse(data.evaluation + ':00Z');
      const outside = currentRows().filter(row => row.required && (now < instant(row.start, row.zone) || now >= instant(row.end, row.zone)));
      if (!Number.isFinite(now) || outside.length) { log('Window denial: no complete required candidate at selected preview time; previous generation preserved.'); persist(); fail(!Number.isFinite(now) ? 'Select a valid preview time in UTC.' : 'Outside availability window at preview time: ' + outside.map(row => row.name).join(', ')); return; }
      if (data.fault !== 'none') {
        const fallback = data.fault === 'source unavailable' && currentRows().every(row => row.mode === 'hybrid' && row.fallback);
        if (!fallback) { log(`Mock ${action} denied (${data.fault}); complete generation ${data.generation} preserved. No fallback after denial/audit/trust failure.`); persist(); fail(`Mock ${action} failed: ${data.fault}. Previous complete generation preserved.`); return; }
        log('Synthetic availability-only snapshot selected. No real snapshot or credential validation.');
      }
      data.generation++; data.validated = true; log(`Mock ${action}: complete candidate generation ${data.generation}; ${currentRows().length} variables; no process launched or key disclosed. Availability checked at selected UTC preview time. No live authority was established.`); persist(); pulse(`Mock ${action} complete · generation ${data.generation}. No actual workload or external effect.`);
    } else if (action === 'reset') openConfirm('Reset workbench demo?', 'Existing native policy and account/operator data are preserved. This resets only workbench rows and synthetic execution feedback.', () => { state.workbench = seed(state); dirty(false); hooks.save(); workbench(root, state, hooks); });
    else if (action === 'close') document.querySelector('#wb-dialog').close();
  }
  function openConfirm(title, text, fn) {
    const dialog = document.querySelector('#wb-dialog');
    dialog.innerHTML = `<h2 id="wb-dialog-title">${h(title)}</h2><p>${h(text)}</p><form id="wb-confirm"><label class="check-label"><input type="checkbox" required> Confirm this local synthetic action</label><div class="actions"><button>Confirm</button><button type="button" class="quiet" data-wb="close">Cancel</button></div></form>`;
    dialog.showModal(); document.querySelector('#wb-confirm').onsubmit = e => { e.preventDefault(); dialog.close(); fn(); };
  }
  function importDialog() {
    const dialog = document.querySelector('#wb-dialog');
    dialog.innerHTML = `<h2 id="wb-dialog-title">Import local variables / workbench JSON</h2><p>Paste synthetic key/value lines, restore a workbench JSON export, or choose a local file. No interpolation, execution, URL request or upload. Existing duplicate names cause a complete rejection.</p><form id="wb-import"><label>.env content<textarea id="wb-env-text" spellcheck="false" placeholder="API_URL=https://api.example.invalid&#10;DEMO_TOKEN=synthetic-only" required></textarea></label><label>Choose local .env / workspace JSON file<input id="wb-env-file" type="file" accept=".env,.txt,.json,text/plain,application/json"></label><p id="wb-import-error" class="error" role="alert"></p><div class="actions"><button>Import variables</button><button type="button" class="quiet" data-wb="close">Cancel</button></div><p class="hint">Imported rows start with UTC windows. Select each row’s Availability tab to set independent dates/timezones, and Value & source for provider/key profiles.</p></form>`;
    dialog.showModal();
    document.querySelector('#wb-env-file').onchange = async e => { const file = e.target.files[0]; if (!file) return; if (file.size > 65536) { document.querySelector('#wb-import-error').textContent = 'Maximum .env size is 64 KiB.'; return; } document.querySelector('#wb-env-text').value = await file.text(); };
    document.querySelector('#wb-import').onsubmit = event => {
      event.preventDefault(); const text = document.querySelector('#wb-env-text').value;
      try {
        if (text.length > 65536) throw new Error('Maximum .env size is 64 KiB.');
        if (text.trim().startsWith('{')) {
          const backup = parseDocument(text);
          const candidate = backup.configuration;
          if (backup.format !== 'cquay-workbench-demo-v1' || !candidate || !Array.isArray(candidate.rows) || candidate.rows.length > 500) throw new Error('Choose a supported workbench JSON export. Native policies use Complete configuration.');
          const ids = new Set();
          for (const row of candidate.rows) {
            if (!row || ['id', 'name', 'value', 'kind', 'provider', 'resource', 'environment', 'group', 'profile', 'mode', 'custody', 'start', 'end', 'zone', 'origin'].some(key => typeof row[key] !== 'string') || typeof row.required !== 'boolean') throw new Error('Invalid workbench row types; import rejected without replacing state.');
            if (ids.has(row.id)) throw new Error('Duplicate row identity; workspace import rejected.');
            ids.add(row.id);
          }
          if (typeof candidate.context !== 'string' || typeof candidate.group !== 'string' || !Array.isArray(candidate.log) || candidate.log.some(line => typeof line !== 'string') || !Number.isFinite(candidate.generation)) throw new Error('Invalid workspace metadata; import rejected.');
          for (const [key, value] of Object.entries(candidate)) Object.defineProperty(data, key, { value, writable: true, configurable: true, enumerable: true });
          selected = data.rows.find(row => row.id === data.selected) || data.rows[0]; draft = selected ? copy(selected) : null;
          dirty(false); persist(); dialog.close(); pulse('Workbench JSON restored locally with windows, timezones, profiles and provenance intact. Native policy is preserved separately.'); return;
        }
        const imported = []; const names = new Set(data.rows.filter(row => row.environment === data.context).map(row => row.name.toLowerCase()));
        const day = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
        for (const [index, raw] of text.split(/\r?\n/).entries()) {
          if (!raw.trim() || raw.trim().startsWith('#')) continue;
          const match = raw.match(/^(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
          if (!match) throw new Error(`Line ${index + 1}: expected NAME=value.`);
          const [, name, content] = match;
          if (names.has(name.toLowerCase())) throw new Error(`Line ${index + 1}: duplicate ${name}; import rejected atomically.`);
          names.add(name.toLowerCase());
          let value = content.trim();
          if (value.startsWith('"') || value.startsWith("'")) { if (value.at(-1) !== value[0] || value.length < 2) throw new Error(`Line ${index + 1}: unmatched quote.`); value = value.slice(1, -1); }
          if (value.includes('\0') || value.length > 4096) throw new Error(`Line ${index + 1}: invalid or oversized value.`);
          imported.push({ id: crypto.randomUUID(), name, value, kind: 'Literal', provider: 'Local .env', resource: '', profile: profiles[0], mode: 'local', custody: 'Local age', environment: data.context, group: 'Personal', start: `${day}T09:00`, end: `${day}T17:00`, zone: 'UTC', origin: 'Local .env import', override: true, inherited: '', required: true, resolution: 'none', revision: 'local-import' });
        }
        if (!imported.length) throw new Error('No variables found.');
        data.rows.push(...imported); selected = imported[0]; draft = copy(selected); data.selected = selected.id; data.validated = false; dirty(false); persist(); dialog.close(); pulse(`Imported ${imported.length} variables locally. Assign provider/key profile and individual availability windows in the inspector.`);
      } catch (error) { document.querySelector('#wb-import-error').textContent = error.message; }
    };
  }
  function openExport() {
    const dialog = document.querySelector('#wb-dialog');
    dialog.innerHTML = `<h2 id="wb-dialog-title">Export local workspace</h2><p>Workspace JSON preserves per-variable sources, windows, zones and provenance. A plain .env file cannot encode those controls. Only synthetic demo values belong in this preview.</p><form id="wb-export"><label>Format<select name="format"><option value="workspace">Workspace JSON (full local model)</option><option value="env">.env (literal values only; no policies)</option></select></label><label class="check-label"><input type="checkbox" required> I understand export contains local values and cannot be recalled.</label><div class="actions"><button>Download local file</button><button type="button" class="quiet" data-wb="close">Cancel</button></div></form>`;
    dialog.showModal(); document.querySelector('#wb-export').onsubmit = event => {
      event.preventDefault(); const format = new FormData(event.target).get('format');
      if (format === 'workspace') hooks.exportJSON({ format: 'cquay-workbench-demo-v1', configuration: data, nativePolicy: state.policy }, 'cquay-workbench-local.json');
      else {
        const text = currentRows().filter(row => row.kind === 'Literal' && effective(row) !== null).map(row => `${row.name}=${JSON.stringify(effective(row))}`).join('\n') + '\n';
        const url = URL.createObjectURL(new Blob([text], { type: 'text/plain' })); const a = document.createElement('a'); a.href = url; a.download = 'cquay-demo.env'; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
      }
      dialog.close(); pulse('Local file exported. Nothing was uploaded. .env does not enforce windows or custody policy.');
    };
  }
  render();
}
