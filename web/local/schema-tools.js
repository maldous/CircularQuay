export function parseDocument(text) {
  if (text.length > 1048576) throw new Error('Maximum document size is 1 MiB.');
  let position = 0;
  const skip = () => { while (/[\x20\t\r\n]/.test(text[position] || '') && position < text.length) position++; };
  function value(depth = 0) {
    if (depth > 64) throw new Error('Maximum JSON depth is 64.');
    skip();
    if (text[position] === '{') {
      position++; const entries = []; const keys = new Set(); skip();
      if (text[position] === '}') { position++; return {}; }
      while (true) {
        skip(); const key = string(); if (keys.has(key)) throw new Error(`Duplicate object key ${key}; source was not accepted.`); keys.add(key);
        skip(); if (text[position++] !== ':') throw new Error('Expected colon after object key.');
        entries.push([key, value(depth + 1)]); skip(); const next = text[position++];
        if (next === '}') return Object.fromEntries(entries);
        if (next !== ',') throw new Error('Expected comma or end of object.');
      }
    }
    if (text[position] === '[') {
      position++; const items = []; skip(); if (text[position] === ']') { position++; return items; }
      while (true) { items.push(value(depth + 1)); skip(); const next = text[position++]; if (next === ']') return items; if (next !== ',') throw new Error('Expected comma or end of array.'); }
    }
    if (text[position] === '"') return string();
    const match = text.slice(position).match(/^(true|false|null|-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?)/);
    if (!match) throw new Error(`Invalid JSON near offset ${position}.`);
    position += match[0].length; const result = JSON.parse(match[0]); if (typeof result === 'number' && !Number.isFinite(result)) throw new Error('Non-finite JSON number.'); return result;
  }
  function string() {
    const match = text.slice(position).match(/^"(?:[^"\\\x00-\x1f]|\\(?:["\\/bfnrt]|u[0-9a-fA-F]{4}))*"/);
    if (!match) throw new Error(`Invalid JSON string near offset ${position}.`);
    position += match[0].length; return JSON.parse(match[0]);
  }
  const result = value(); skip(); if (position !== text.length) throw new Error('Unexpected content after JSON document.'); return result;
}

export function dereference(schema, root) {
  if (schema?.$ref) return { ...dereference(schema.$ref.slice(2).split('/').reduce((s, key) => s?.[key], root), root), ...Object.fromEntries(Object.entries(schema).filter(([key]) => key !== '$ref')) };
  return schema || {};
}
export function shape(schema, root, seen = new Set()) {
  schema = dereference(schema, root);
  if (seen.has(schema)) return schema;
  seen = new Set(seen).add(schema);
  const output = { ...schema, properties: { ...(schema.properties || {}) }, required: [...(schema.required || [])] };
  for (const branch of [...(schema.allOf || []), ...(schema.anyOf || []), ...(schema.oneOf || []), ...(schema.then ? [schema.then] : []), ...(schema.else ? [schema.else] : [])]) {
    const item = shape(branch, root, seen);
    for (const [key, child] of Object.entries(item.properties || {})) {
      // Branches constrain the complete document, not the available control.
      // Intersecting alternative true/false constants here hides valid choices.
      // Keep the base field declaration; validate() enforces each active branch.
      if (!Object.hasOwn(output.properties, key)) output.properties[key] = child;
    }
    output.required.push(...(item.required || []));
    if (!output.type && item.type) output.type = item.type;
    if (!output.items && item.items) output.items = item.items;
    for (const key of ['enum', 'const', 'format', 'pattern', 'minimum', 'maximum', 'minLength', 'maxLength']) {
      if (!Object.hasOwn(output, key) && Object.hasOwn(item, key)) output[key] = item[key];
    }
    if (output.additionalProperties === undefined && item.additionalProperties !== undefined) output.additionalProperties = item.additionalProperties;
  }
  return output;
}
const equal = (a, b) => {
  if (a === b) return true;
  if (Array.isArray(a) && Array.isArray(b)) return a.length === b.length && a.every((v, i) => equal(v, b[i]));
  if (a && b && typeof a === 'object' && typeof b === 'object') return Object.keys(a).length === Object.keys(b).length && Object.keys(a).every(k => Object.hasOwn(b, k) && equal(a[k], b[k]));
  return false;
};
export { equal as documentsEqual };

// A separate saved editor draft must survive edits from another local surface.
// When the editor still matches the accepted policy, move both views together.
export function replacePolicy(state, next) {
  const retainedDraft = Boolean(state.nativeEditor && !equal(state.nativeEditor.document, state.policy));
  const copy = value => JSON.parse(JSON.stringify(value));
  state.policy = copy(next);
  if (!retainedDraft) state.nativeEditor = { original: copy(next), document: copy(next) };
  return retainedDraft;
}

export function reindexArrayInputs(inputs, pointer, nextIndex) {
  const prefix = pointer + '/';
  const entries = [];
  for (const [path, value] of inputs) {
    if (!path.startsWith(prefix)) { entries.push([path, value]); continue; }
    const [index, ...tail] = path.slice(prefix.length).split('/');
    if (!/^(0|[1-9]\d*)$/.test(index)) { entries.push([path, value]); continue; }
    const next = nextIndex(Number(index));
    if (next !== null) entries.push([prefix + [next, ...tail].join('/'), value]);
  }
  inputs.clear();
  for (const [path, value] of entries) inputs.set(path, value);
}
export function validate(value, schema, root = schema, path = '', depth = 0) {
  if (depth > 100) return [`${path}: schema depth exceeded`];
  if (schema === false) return [`${path}: unsupported field/value`];
  if (schema === true) return [];
  schema = dereference(schema, root);
  const errors = [];
  const add = text => errors.push(`${path || '/'}: ${text}`);
  const sub = (v, s, p = path) => validate(v, s, root, p, depth + 1);
  if (schema.type) {
    const types = Array.isArray(schema.type) ? schema.type : [schema.type];
    const actual = value === null ? 'null' : Array.isArray(value) ? 'array' : typeof value;
    if (!types.includes(actual) && !(types.includes('integer') && actual === 'number' && Number.isInteger(value))) { add(`expected ${types.join(' or ')}, found ${actual}`); return errors; }
  }
  if (Object.hasOwn(schema, 'const') && !equal(value, schema.const)) add(`must equal ${JSON.stringify(schema.const)}`);
  if (schema.enum && !schema.enum.some(v => equal(v, value))) add('must be a supported enum value');
  if (schema.allOf) for (const s of schema.allOf) errors.push(...sub(value, s));
  if (schema.anyOf && !schema.anyOf.some(s => !sub(value, s).length)) add('does not satisfy any permitted shape');
  if (schema.oneOf && schema.oneOf.filter(s => !sub(value, s).length).length !== 1) add('must satisfy exactly one permitted shape');
  if (schema.not && !sub(value, schema.not).length) add('forbidden field combination/value');
  if (schema.if) { const branch = !sub(value, schema.if).length ? schema.then : schema.else; if (branch) errors.push(...sub(value, branch)); }
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    const keys = Object.keys(value);
    if (schema.minProperties !== undefined && keys.length < schema.minProperties) add(`requires at least ${schema.minProperties} properties`);
    if (schema.maxProperties !== undefined && keys.length > schema.maxProperties) add(`allows at most ${schema.maxProperties} properties`);
    for (const key of schema.required || []) if (!Object.hasOwn(value, key)) add(`missing ${key}`);
    for (const key of keys) {
      const p = `${path}/${key.replace(/~/g, '~0').replace(/\//g, '~1')}`;
      if (schema.propertyNames) errors.push(...sub(key, schema.propertyNames, p + ' (name)'));
      if (Object.hasOwn(schema.properties || {}, key)) errors.push(...sub(value[key], schema.properties[key], p));
      else if (schema.additionalProperties === false) errors.push(`${p}: unsupported field preserved; remove or migrate explicitly`);
      else if (schema.additionalProperties && typeof schema.additionalProperties === 'object') errors.push(...sub(value[key], schema.additionalProperties, p));
    }
  }
  if (Array.isArray(value)) {
    if (schema.minItems !== undefined && value.length < schema.minItems) add(`requires at least ${schema.minItems} items`);
    if (schema.maxItems !== undefined && value.length > schema.maxItems) add(`allows at most ${schema.maxItems} items`);
    if (schema.uniqueItems && value.some((v, i) => value.slice(0, i).some(other => equal(v, other)))) add('duplicate array items');
    if (schema.items) value.forEach((v, i) => errors.push(...sub(v, schema.items, `${path}/${i}`)));
    if (schema.contains && !value.some(v => !sub(v, schema.contains).length)) add('missing required matching array item');
  }
  if (typeof value === 'string') {
    if (schema.minLength !== undefined && [...value].length < schema.minLength) add(`minimum length ${schema.minLength}`);
    if (schema.maxLength !== undefined && [...value].length > schema.maxLength) add(`maximum length ${schema.maxLength}`);
    if (schema.pattern && !new RegExp(schema.pattern, 'u').test(value)) add('does not match required format/pattern');
    if (schema.format === 'date-time' && (!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)$/.test(value) || !Number.isFinite(Date.parse(value)))) add('requires RFC3339 date-time with timezone');
  }
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) add('number must be finite');
    if (schema.minimum !== undefined && value < schema.minimum) add(`minimum ${schema.minimum}`);
    if (schema.maximum !== undefined && value > schema.maximum) add(`maximum ${schema.maximum}`);
  }
  return errors;
}

export function referenceErrors(document) {
  const errors = [];
  const hasOwnEntry = (value, key) => value !== null && typeof value === 'object' && Object.hasOwn(value, key);
  for (const [id, workload] of Object.entries(document.workloads || {})) {
    const seen = new Set();
    for (const binding of workload.bindings || []) {
      if (binding && typeof binding.name === 'string') { const name = binding.name.toLowerCase(); if (seen.has(name)) errors.push(`/workloads/${id}/bindings: duplicate output name`); seen.add(name); }
      if (binding && !hasOwnEntry(document.sources, binding.source) && !document.composition) errors.push(`/workloads/${id}/bindings: unknown source ${binding.source}`);
    }
    if (workload.mode === 'online' && workload.snapshot && Object.keys(workload.snapshot).length) errors.push(`/workloads/${id}: online cannot use a snapshot`);
  }
  for (const [index, grant] of (document.grants || []).entries()) {
    if (grant && !hasOwnEntry(document.workloads, grant.workload) && !document.composition) errors.push(`/grants/${index}/workload: unknown workload`);
    if (grant?.authority && !hasOwnEntry(document.mcp?.authorities, grant.authority)) errors.push(`/grants/${index}/authority: unknown authority`);
  }
  return errors;
}

const ownEntry = (entries, name) => entries && Object.hasOwn(entries, name) ? entries[name] : undefined;

function rewriteNamedEntry(entries, previousName, nextName, value) {
  const rewritten = Object.entries(entries || []).filter(([name]) => name !== previousName && name !== nextName);
  if (nextName !== null) rewritten.push([nextName, value]);
  return Object.fromEntries(rewritten);
}

// This produces a browser-validated local candidate only. It does not replace
// native validation, protected-file checks, provider authorization, or activation.
export function prepareGuidedPolicyChange(policy, key, next, previous, schema, workloadRows = []) {
  const candidate = JSON.parse(JSON.stringify(policy));
  if (key === 'variables') {
    for (const workload of workloadRows) {
      const document = ownEntry(candidate.workloads, workload.name);
      if (!document) continue;
      if (previous && workload.environment === previous.environment) {
        if (document.environment) delete document.environment[previous.name];
        document.bindings = (document.bindings || []).filter(binding => binding.name !== previous.name);
      }
      if (!next || workload.environment !== next.environment) continue;
      if (next.kind === 'Non-secret literal') document.environment = { ...document.environment, [next.name]: next.value };
      else document.bindings = [...(document.bindings || []).filter(binding => binding.name !== next.name), { name: next.name, source: next.source, field: next.value }];
    }
  }
  if (key === 'workloads') {
    const workloads = candidate.workloads || {};
    const original = previous ? ownEntry(workloads, previous.name) : null;
    if (next) {
      if (previous?.name !== next.name && Object.hasOwn(workloads, next.name)) throw new Error('A policy workload with that identifier already exists. No change was made.');
      candidate.workloads = rewriteNamedEntry(workloads, previous?.name, next.name, { ...(original || { bindings: [], environment: {} }), mode: next.mode });
    } else if (previous) {
      candidate.workloads = rewriteNamedEntry(workloads, previous.name, null, null);
    }
  }
  if (key === 'sources') {
    const sources = candidate.sources || {};
    const original = previous ? ownEntry(sources, previous.name) : null;
    if (next) {
      if (previous?.name !== next.name && Object.hasOwn(sources, next.name)) throw new Error('A policy source with that identifier already exists. No change was made.');
      const source = { ...(original || {}), type: next.type, ...(['json', 'sops', 'env'].includes(next.type) ? { path: next.location } : { endpoint: next.location }) };
      if (next.type === 'json') delete source.identity;
      else source.identity = next.type === 'env' ? 'none' : (original?.identity || 'token_file');
      candidate.sources = rewriteNamedEntry(sources, previous?.name, next.name, source);
    } else if (previous) {
      candidate.sources = rewriteNamedEntry(sources, previous.name, null, null);
    }
  }
  if (!['variables', 'workloads', 'sources'].includes(key)) return candidate;
  const errors = validate(candidate, schema);
  if (!errors.length) errors.push(...referenceErrors(candidate));
  if (errors.length) throw new Error(`Guided change was not saved: ${errors[0]}`);
  return candidate;
}

export function schemaInventory(schema) {
  const paths = [];
  function visit(node, path, ancestors = new Set()) {
    const s = shape(node, schema);
    if (ancestors.has(node)) return;
    ancestors = new Set(ancestors).add(node);
    paths.push({ path: path || '/', type: s.enum ? 'enum select' : s.type || (Object.keys(s.properties || {}).length ? 'object/map' : 'unconstrained value'), control: s.type === 'array' ? 'list + typed item controls' : s.type === 'object' || Object.keys(s.properties || {}).length ? 'group + optional field/map controls' : s.enum || Object.hasOwn(s, 'const') ? 'select' : s.type === 'boolean' ? 'boolean select' : s.type === 'integer' || s.type === 'number' ? 'numeric input' : !s.type ? 'explicit type selector + typed value control' : 'text/path/reference input', route: '#/configuration' });
    for (const [key, child] of Object.entries(s.properties || {})) visit(child, `${path}/${key}`, ancestors);
    if (s.items) visit(s.items, `${path}/*`, ancestors);
    if (s.additionalProperties && typeof s.additionalProperties === 'object') visit(s.additionalProperties, `${path}/{id}`, ancestors);
  }
  visit(schema, ''); return paths;
}
