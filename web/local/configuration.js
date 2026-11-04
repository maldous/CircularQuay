import { parseDocument, shape, validate, referenceErrors, schemaInventory, documentsEqual, reindexArrayInputs } from './schema-tools.js';
const h = value => String(value ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const clone = value => JSON.parse(JSON.stringify(value));
const encode = key => key.replace(/~/g, '~0').replace(/\//g, '~1');
const keys = pointer => pointer ? pointer.slice(1).split('/').map(k => k.replace(/~1/g, '/').replace(/~0/g, '~')) : [];
const hasOwn = (value, key) => value !== null && typeof value === 'object' && Object.hasOwn(value, key);
const get = (document, pointer) => keys(pointer).reduce((value, key) => hasOwn(value, key) ? value[key] : undefined, document);
const label = key => key.replace(/_/g, ' ');
const groups = { version: 'Version', sources: 'Providers & sources', workloads: 'Workloads, bindings & delivery', grants: 'Authorization grants', mcp: 'MCP identity & access', repository: 'Signed repository & startup', composition: 'Includes, overrides & locks', observability: 'Telemetry & diagnostics', updates: 'Release awareness' };
let schemaPromise;
async function loadSchema() { schemaPromise ||= fetch('/local/policy-schema.json').then(r => { if (!r.ok) throw new Error('Bundled schema unavailable'); return r.json(); }); return schemaPromise; }

// Unknown roots are intentionally retained in drafts. This boundary produces the
// complete sidebar markup, so both the key and derived label must be escaped.
export function sidebarSections(document) {
  return { ...groups, ...Object.fromEntries(Object.keys(document).filter(key => !Object.hasOwn(groups, key)).map(key => [key, `Unsupported: ${key}`])) };
}

export function sidebarNavigationMarkup(sections, document, section) {
  return Object.entries(sections).map(([key, name]) => `<button data-cfg="section" data-section="${h(key)}" ${section === key ? 'aria-current="page"' : ''}>${h(name)}<small>${Object.hasOwn(document, key) ? 'configured' : 'absent'}</small></button>`).join('');
}

export async function configuration(root, state, hooks) {
  root.innerHTML = '<main><h1>Complete configuration</h1><p role="status">Loading the bundled local schema…</p></main>';
  let schema;
  try { schema = await loadSchema(); } catch (error) { root.innerHTML = `<main><h1>Schema unavailable</h1><p>${h(error.message)}</p><a href="#/workbench">Back to workbench</a></main>`; return; }
  // The untouched source and editable proposal remain separate. Absent/false/
  // empty/null values are never materialized or removed merely by rendering.
  const saved = state.nativeEditor;
  let original = clone(saved?.original || state.policy);
  let document = clone(saved?.document || state.policy);
  let section = 'sources';
  let search = '';
  let errors = [];
  let message = documentsEqual(document, state.policy) ? 'No source/provider requests. All supported fields are available through typed controls.' : 'Editing a retained document that differs from the saved policy. Load saved policy explicitly to replace this editor document; no draft is silently discarded.';
  let dirty = false;
  const invalidControls = new Map();
  let showAbsent = false;
  let openPaths = new Set(['/sources', '/workloads', '/mcp', '/observability', '/repository', '/composition', '/grants', '/updates']);
  const notify = text => { message = text; render(); };
  const check = () => {
    errors = validate(document, schema);
    if (!errors.length) errors.push(...referenceErrors(document));
    for (const pointer of invalidControls.keys()) errors.push(`${pointer}: enter a finite JSON number; unsaved input retained`);
    return errors;
  };
  const change = () => { dirty = true; hooks.changed(true); check(); };
  const update = (pointer, value) => {
    const parts = keys(pointer); const key = parts.pop(); const target = parts.reduce((current, part) => hasOwn(current, part) ? current[part] : undefined, document);
    if (!target || typeof target !== 'object') throw new Error('The selected configuration path is no longer an own object path.');
    for (const path of invalidControls.keys()) if (path === pointer || path.startsWith(pointer + '/')) invalidControls.delete(path);
    Object.defineProperty(target, key, { value, writable: true, configurable: true, enumerable: true }); change();
  };
  const remove = pointer => {
    const parts = keys(pointer); const key = parts.pop(); const target = parts.reduce((current, part) => hasOwn(current, part) ? current[part] : undefined, document);
    if (!target || typeof target !== 'object') throw new Error('The selected configuration path is no longer an own object path.');
    for (const path of invalidControls.keys()) if (path === pointer || path.startsWith(pointer + '/')) invalidControls.delete(path);
    if (Array.isArray(target)) {
      target.splice(Number(key), 1);
      const parent = '/' + parts.map(encode).join('/');
      reindexArrayInputs(invalidControls, parent, index => index === Number(key) ? null : index > Number(key) ? index - 1 : index);
    } else delete target[key];
    change();
  };
  const findSchema = pointer => {
    let s = schema; let node = document;
    for (const key of keys(pointer)) {
      const f = shape(s, schema);
      s = Array.isArray(node) ? f.items || {} : hasOwn(f.properties, key) ? f.properties[key] : Object.hasOwn(f, 'additionalProperties') ? f.additionalProperties : {};
      node = hasOwn(node, key) ? node[key] : undefined;
    }
    return s;
  };
  function defaultValue(s) {
    const f = shape(s, schema);
    if (Object.hasOwn(f, 'const')) return clone(f.const);
    if (f.enum) return clone(f.enum[0]);
    if (f.type === 'object' || Object.keys(f.properties || {}).length) return {};
    if (f.type === 'array') return [];
    if (f.type === 'boolean') return false;
    if (f.type === 'integer' || f.type === 'number') return f.minimum ?? 0;
    return '';
  }
  function field(value, sourceSchema, pointer, name) {
    const s = shape(sourceSchema, schema);
    const actual = value === null ? 'null' : Array.isArray(value) ? 'array' : typeof value;
    const ownErrors = errors.filter(error => error.startsWith(`${pointer}:`));
    const attrs = `data-pointer="${h(pointer)}" aria-label="${h(pointer)}"`;
    const removeButton = `<button type="button" class="quiet small-button" data-cfg="remove" data-pointer="${h(pointer)}" aria-label="Remove ${h(pointer)}">Remove</button>`;
    const hint = `<span class="cfg-path">${h(pointer)}${s.format ? ` · ${h(s.format)}` : ''}${s.minimum !== undefined ? ` · min ${s.minimum}` : ''}${s.maximum !== undefined ? ` · max ${s.maximum}` : ''}</span>`;
    const issue = ownErrors.length ? `<div class="cfg-field-error">${ownErrors.map(h).join('<br>')}</div>` : '';
    const typeChoice = !s.type && !s.enum && !Object.hasOwn(s, 'const') ? `<label class="cfg-type-label">Explicit value type<select data-change-type="${h(pointer)}">${['string', 'number', 'boolean', 'object', 'array', 'null'].map(type => `<option ${actual === type ? 'selected' : ''}>${type}</option>`).join('')}</select></label>` : s.type && s.type !== actual && !(s.type === 'integer' && actual === 'number') ? `<button type="button" class="quiet small-button" data-cfg="replace-null" data-pointer="${h(pointer)}">Replace invalid type with schema type</button>` : '';
    if (actual === 'object' || actual === 'array') {
      const entries = Object.entries(value);
      const properties = s.properties || {};
      const missing = Object.keys(properties).filter(key => !Object.hasOwn(value, key));
      const matching = !search || pointer.toLowerCase().includes(search.toLowerCase()) || JSON.stringify(value).toLowerCase().includes(search.toLowerCase()) || missing.some(key => `${pointer}/${key}`.toLowerCase().includes(search.toLowerCase()));
      if (!matching) return '';
      return `<details class="cfg-group" data-path="${h(pointer)}" ${openPaths.has(pointer) || search ? 'open' : ''}><summary><strong>${h(label(name))}</strong><span>${actual === 'array' ? `${entries.length} items` : `${entries.length} set fields`}</span></summary><div class="cfg-group-body">${hint}${issue}${typeChoice}<div class="cfg-group-actions">${removeButton}${actual === 'array' ? `<button type="button" class="quiet small-button" data-cfg="append" data-pointer="${h(pointer)}">+ Add item to ${h(name)}</button>` : missing.length ? `<label>Add supported field<select data-add-select="${h(pointer)}"><option value="">Choose field…</option>${missing.map(key => `<option value="${h(key)}">${h(label(key))}</option>`).join('')}</select></label>` : ''}${actual === 'object' && (typeof s.additionalProperties === 'object' || s.additionalProperties === true || !Object.keys(properties).length) ? `<button type="button" class="quiet small-button" data-cfg="map-add" data-pointer="${h(pointer)}">+ Add named entry</button>` : ''}</div>${entries.map(([key, v]) => {
        const childSchema = actual === 'array' ? s.items || {} : hasOwn(properties, key) ? properties[key] : (typeof s.additionalProperties === 'object' ? s.additionalProperties : {});
        const unsupported = actual !== 'array' && !Object.hasOwn(properties, key) && s.additionalProperties === false;
        return `${unsupported ? `<div class="cfg-unsupported">Unsupported field preserved: ${h(key)}. Native schema rejects it.</div>` : ''}${actual === 'array' ? `<div class="cfg-order"><button type="button" class="quiet small-button" data-cfg="move-up" data-pointer="${h(pointer)}/${key}" ${Number(key) === 0 ? 'disabled' : ''}>Move item ${key} up</button><button type="button" class="quiet small-button" data-cfg="move-down" data-pointer="${h(pointer)}/${key}" ${Number(key) === entries.length - 1 ? 'disabled' : ''}>Move item ${key} down</button></div>` : ''}${field(v, childSchema, `${pointer}/${encode(key)}`, key)}`;
      }).join('')}${!entries.length ? '<p class="hint">Empty collection preserved. Add an entry explicitly; no defaults are silently inserted.</p>' : ''}</div></details>`;
    }
    if (search && !pointer.toLowerCase().includes(search.toLowerCase())) return '';
    const sensitive = /(?:token|password|secret|private|value)(?:$|\/)/i.test(pointer) && !/(?:token_file|identity_file|allow_|\/bindings\/)/.test(pointer);
    const referenceOptions = /\/bindings\/\d+\/source$/.test(pointer) ? Object.keys(document.sources || {}) : /^\/grants\/\d+\/workload$/.test(pointer) ? Object.keys(document.workloads || {}) : /^\/grants\/\d+\/authority$/.test(pointer) ? Object.keys(document.mcp?.authorities || {}) : null;
    let control;
    if (actual === 'null') control = `<p>Explicit null preserved; invalid where the schema disallows it.</p><button class="quiet" data-cfg="replace-null" data-pointer="${h(pointer)}">Replace null explicitly</button>`;
    else if (s.enum || Object.hasOwn(s, 'const') || referenceOptions) { const values = referenceOptions || s.enum || [s.const]; control = `<select ${attrs} data-type="${typeof value}">${!values.some(v => v === value) ? `<option selected value="${h(value)}">Unsupported/unknown: ${h(value)}</option>` : ''}${values.map(v => `<option value="${h(v)}" ${v === value ? 'selected' : ''}>${h(v)}</option>`).join('')}</select>`; }
    else if (actual === 'boolean') control = `<select ${attrs} data-type="boolean"><option ${value ? 'selected' : ''}>true</option><option ${value ? '' : 'selected'}>false</option></select>`;
    else control = `<input ${attrs} data-type="${actual}" type="${sensitive ? 'password' : 'text'}" ${actual === 'number' ? 'inputmode="decimal"' : ''} value="${h(invalidControls.get(pointer) ?? value)}" spellcheck="false" autocomplete="off">`;
    return `<div class="cfg-field ${ownErrors.length ? 'invalid' : ''}"><div class="cfg-field-label"><label>${h(label(name))}</label>${removeButton}</div>${hint}${typeChoice}${control}${sensitive ? `<small>Masked local value. No transport. This field remains editable.</small>` : ''}${issue}</div>`;
  }
  function render() {
    if (!root.isConnected || !location.hash.includes('configuration')) return;
    const selectedSchema = hasOwn(schema.properties, section) ? schema.properties[section] : {};
    const sections = sidebarSections(document);
    root.innerHTML = `<div class="configuration"><header class="cfg-header"><a class="wordmark" href="#/workbench">CircularQuay <span class="pronunciation">LOCAL CONFIGURATION</span></a><div><h1>Complete configuration</h1><p>Typed controls · proposed changes separate from untouched source · ${dirty ? 'unsaved' : 'saved / unchanged'}</p></div><span class="badge">NO NATIVE EXECUTION</span></header><div class="cfg-toolbar"><button data-cfg="load-linux">Load complete Linux example</button><button data-cfg="load-policy" class="quiet">Load saved policy</button><button data-cfg="import" class="quiet">Import local JSON</button><button data-cfg="validate" class="quiet">Validate</button><button data-cfg="save">Save locally</button><button data-cfg="export" class="quiet">Export JSON</button><button data-cfg="discard" class="quiet">Discard proposal</button><button data-cfg="coverage" class="quiet">Schema coverage</button><a href="#/workbench">← Variable workbench</a></div><div class="cfg-message" role="status">${h(message)}</div><div class="cfg-layout"><aside class="cfg-sidebar"><label>Find field<input type="search" id="cfg-search" value="${h(search)}" placeholder="e.g. activation, timeout, DPoP"></label><nav aria-label="Configuration sections">${sidebarNavigationMarkup(sections, document, section)}</nav><p>All platform target types are editable. Only local synthetic execution is available. Upcoming window/custody designs are in the workbench; they are not invented native schema fields.</p></aside><main class="cfg-content"><div class="cfg-section-head"><h2>${h(sections[section])}</h2><span>${Object.hasOwn(document, section) ? 'Present in proposal' : 'Absent · unchanged until explicitly added'}</span></div>${search ? `<p class="hint">Search applies to field paths in the selected section, including fields available through “Add supported field”.</p>` : ''}${Object.hasOwn(document, section) ? field(document[section], selectedSchema, `/${section}`, section) : `<div class="cfg-absent"><p>This optional section is absent. Rendering does not insert it.</p><button data-cfg="add-root" data-section="${h(section)}">Add ${h(sections[section])}</button></div>`}<section class="cfg-errors" aria-label="Validation findings"><h2>${errors.length ? `${errors.length} validation finding(s)` : 'Validation'}</h2>${errors.length ? `<ul>${errors.slice(0, 80).map(e => `<li>${h(e)}</li>`).join('')}</ul>` : '<p>No browser schema findings in the last check. Full native protected-file, composition, provider and authorization checks remain required.</p>'}</section><details class="cfg-diff"><summary>Untouched source / proposed changes</summary><p>Untouched source remains in browser storage. No parser/renderer defaults are applied.</p><div class="cfg-two"><div><h3>Untouched source</h3><pre>${h(JSON.stringify(original, null, 2))}</pre></div><div><h3>Current proposal</h3><pre>${h(JSON.stringify(document, null, 2))}</pre></div></div></details></main></div><dialog id="cfg-dialog" aria-labelledby="cfg-dialog-title"></dialog></div>`;
    bind();
  }
  function bind() {
    root.onclick = event => {
      const button = event.target.closest('[data-cfg]'); if (button) action(button.dataset.cfg, button.dataset);
      if (dirty && event.target.closest('a[href]')) { event.preventDefault(); message = 'Save locally or discard the proposal before leaving.'; render(); }
    };
    for (const details of root.querySelectorAll('details[data-path]')) details.ontoggle = () => details.open ? openPaths.add(details.dataset.path) : openPaths.delete(details.dataset.path);
    root.querySelector('#cfg-search').oninput = event => { search = event.target.value; render(); const input = root.querySelector('#cfg-search'); input.focus(); input.setSelectionRange(search.length, search.length); };
    for (const control of root.querySelectorAll('[data-pointer][data-type]')) {
      const applyControl = event => {
      const input = event.target; let value = input.value;
      if (input.dataset.type === 'boolean') value = value === 'true';
      else if (input.dataset.type === 'number') { if (!/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$/.test(value) || !Number.isFinite(Number(value))) { invalidControls.set(input.dataset.pointer, value); dirty = true; hooks.changed(true); check(); input.setCustomValidity('Enter a finite JSON number.'); root.querySelector('.cfg-message').textContent = 'Invalid numeric input retained. Save and export are blocked until corrected or discarded.'; return; } value = Number(value); }
      input.setCustomValidity(''); update(input.dataset.pointer, value); message = 'Field updated in the local proposal. Siblings and unknown fields are preserved.';
      root.querySelector('.cfg-message').textContent = message;
      };
      control.oninput = applyControl;
      control.onchange = event => { applyControl(event); render(); };
    }
    for (const select of root.querySelectorAll('[data-add-select]')) select.onchange = event => {
      const parent = event.target.dataset.addSelect; const key = event.target.value; if (!key) return;
      const s = shape(findSchema(parent), schema).properties[key]; update(`${parent}/${encode(key)}`, defaultValue(s)); openPaths.add(`${parent}/${encode(key)}`); render();
    };
    for (const select of root.querySelectorAll('[data-change-type]')) select.onchange = event => {
      const type = event.target.value; const values = { string: '', number: 0, boolean: false, object: {}, array: [], null: null };
      update(event.target.dataset.changeType, values[type]); render();
    };
  }
  function dialogForm(title, content, submit, handler) {
    const dialog = root.querySelector('#cfg-dialog');
    dialog.innerHTML = `<h2 id="cfg-dialog-title">${h(title)}</h2><form id="cfg-dialog-form">${content}<p id="cfg-dialog-error" class="error" role="alert"></p><div class="actions"><button>${h(submit)}</button><button type="button" class="quiet" data-cfg="close">Cancel</button></div></form>`;
    dialog.showModal(); dialog.querySelector('form').onsubmit = async event => { event.preventDefault(); try { await handler(new FormData(event.target)); } catch (error) { root.querySelector('#cfg-dialog-error').textContent = error.message; } };
  }
  function importText(text) {
    const parsed = parseDocument(text);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('Policy root must be an object.');
    original = clone(parsed); document = clone(parsed); section = 'sources'; dirty = false; hooks.changed(false); check();
    state.nativeEditor = { original: clone(original), document: clone(document) }; hooks.save();
    notify(`Imported locally without coercion. ${errors.length} schema finding(s); unknown fields preserved. No native/provider check.`);
  }
  async function action(name, attributes) {
    if (name === 'section') { section = attributes.section; search = ''; render(); }
    else if (name === 'load-policy') {
      if (dirty) { notify('Save or discard the current proposal before loading the saved policy.'); return; }
      importText(JSON.stringify(state.policy));
    } else if (name === 'load-linux') {
      if (dirty) { notify('Save or discard the current proposal before loading a different source.'); return; }
      try { const response = await fetch('/local/linux-example.json'); importText(await response.text()); }
      catch (error) { notify(error.message); }
    } else if (name === 'add-root') {
      const rootSchema = hasOwn(schema.properties, attributes.section) ? schema.properties[attributes.section] : {};
      update(`/${attributes.section}`, defaultValue(rootSchema));
      openPaths.add(`/${attributes.section}`);
      render();
    }
    else if (name === 'remove') { remove(attributes.pointer); render(); }
    else if (name === 'move-up' || name === 'move-down') { const parts = keys(attributes.pointer); const index = Number(parts.pop()); const parent = '/' + parts.map(encode).join('/'); const array = get(document, parent); const next = index + (name === 'move-up' ? -1 : 1); if (next >= 0 && next < array.length) { [array[index], array[next]] = [array[next], array[index]]; reindexArrayInputs(invalidControls, parent, current => current === index ? next : current === next ? index : current); change(); render(); } }
    else if (name === 'append') { const array = get(document, attributes.pointer); const s = shape(findSchema(attributes.pointer), schema); array.push(defaultValue(s.items || {})); change(); openPaths.add(`${attributes.pointer}/${array.length - 1}`); render(); }
    else if (name === 'replace-null') { update(attributes.pointer, defaultValue(findSchema(attributes.pointer))); render(); }
    else if (name === 'map-add') {
      const parentSchema = shape(findSchema(attributes.pointer), schema);
      const nameSchema = shape(parentSchema.propertyNames || {}, schema);
      const choices = nameSchema.enum;
      dialogForm('Add named configuration entry', `<label>Entry identifier${choices ? `<select name="key">${choices.map(key => `<option>${h(key)}</option>`).join('')}</select>` : `<input name="key" required maxlength="100" autocomplete="off" ${nameSchema.pattern ? `pattern="${h(nameSchema.pattern)}"` : ''}>`}</label><p>Identifiers are case-sensitive in JSON. Existing names are rejected; validation checks policy identifier rules.</p>`, 'Add entry', form => {
      const key = form.get('key'); const object = get(document, attributes.pointer);
      if (Object.hasOwn(object, key)) throw new Error('This entry already exists. No sibling was overwritten.');
      const s = shape(findSchema(attributes.pointer), schema); update(`${attributes.pointer}/${encode(key)}`, defaultValue(s.additionalProperties || {})); openPaths.add(`${attributes.pointer}/${encode(key)}`); render();
      });
    }
    else if (name === 'validate') { check(); notify(`${errors.length ? 'Validation rejected this proposal' : 'Browser schema/reference checks passed'}. Native filesystem, signature, include resolution, lifecycle and provider validation have not run.`); }
    else if (name === 'save') {
      if (invalidControls.size) { notify('Correct or discard invalid numeric inputs before saving. No value was coerced or lost.'); return; }
      check(); state.nativeEditor = { original: clone(original), document: clone(document) };
      // Invalid/forward documents can be retained as drafts without falsely accepting them as policy.
      if (!errors.length) state.policy = clone(document);
      dirty = false; hooks.changed(false); hooks.save(); notify(errors.length ? 'Saved locally as an INVALID draft with all fields intact. Active native policy was not replaced.' : 'Saved locally. No native validation or activation occurred.');
    } else if (name === 'discard') {
      invalidControls.clear(); document = clone(original); check();
      // Persist the restored document so navigation/reload cannot resurrect the
      // discarded proposal. Invalid imported sources remain drafts only.
      state.nativeEditor = { original: clone(original), document: clone(document) };
      if (!errors.length) state.policy = clone(document);
      dirty = false; hooks.changed(false); hooks.save();
      notify('Proposal discarded; untouched imported source restored exactly and saved locally.');
    }
    else if (name === 'export') { if (invalidControls.size) { notify('Correct or discard invalid numeric input before exporting.'); return; } hooks.exportJSON(document, 'cquay-local-configuration.json'); notify('Current proposal exported locally, including unsupported fields. No upload.'); }
    else if (name === 'coverage') { hooks.exportJSON({ schema: 'local/policy-schema.json', paths: schemaInventory(schema), runtimeQualification: false }, 'cquay-schema-control-coverage.json'); notify('Schema control inventory exported. This maps controls, not native runtime qualification.'); }
    else if (name === 'close') root.querySelector('#cfg-dialog').close();
    else if (name === 'import') {
      if (dirty) { notify('Save or discard the current proposal before importing.'); return; }
      dialogForm('Import local configuration JSON', '<label>JSON document<textarea name="json" id="cfg-import-text" required spellcheck="false"></textarea></label><label>Or select a local JSON file<input id="cfg-import-file" type="file" accept="application/json,.json"></label><p>Duplicate keys and malformed JSON reject the import. Unknown/unsupported fields remain visible with validation errors.</p>', 'Import locally', form => importText(form.get('json')));
      root.querySelector('#cfg-import-file').onchange = async event => { const file = event.target.files[0]; if (!file) return; if (file.size > 1048576) { root.querySelector('#cfg-dialog-error').textContent = 'Maximum 1 MiB'; return; } root.querySelector('#cfg-import-text').value = await file.text(); };
    }
  }
  check(); render();
}
