const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
const html = escapeHTML;
const titleCase = value => value[0].toUpperCase() + value.slice(1);
const clone = value => JSON.parse(JSON.stringify(value));

export function brandMarkup(subtitle, name = 'CircularQuay') {
  return `<span class="brand-symbol" aria-hidden="true"><img class="brand-mark brand-on-light" src="/logo-light.svg" width="44" height="40" alt=""><img class="brand-mark brand-on-dark" src="/logo-dark.svg" width="44" height="40" alt=""></span><span class="brand-copy">${html(name)}<span class="pronunciation">${html(subtitle)}</span></span>`;
}

let policySchemaPromise;
let guidedPolicyToolsPromise;
function loadPolicySchema() {
  policySchemaPromise ||= fetch('/local/policy-schema.json').then(response => {
    if (!response.ok) throw new Error('Bundled policy schema is unavailable; guided policy changes were not saved.');
    return response.json();
  });
  return policySchemaPromise;
}
const loadGuidedPolicyTools = () => guidedPolicyToolsPromise ||= import('/local/schema-tools.js');

export function start(model) {
  const root = document.querySelector('#app');
  const local = model.plane === 'local';
  const storageKey = `cquay.preview.v1.${model.plane}`;
  let state = { settings: clone(model.settings), collections: clone(model.collections), policy: clone(model.policy ?? {}), context: 'development', generation: 'none', plan: 'Free demo', identity: 'not configured', events: [] };
  let storageWarning = '';
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved) state = { ...state, ...JSON.parse(saved) };
  } catch { storageWarning = 'Browser storage is unavailable or invalid. Changes remain in memory.'; }
  let scope = local ? 'Client' : (location.hash.startsWith('#/operator') ? 'Operator' : 'Account');
  let persona = 'Account owner';
  let route = '';
  let search = '';
  let filter = '';
  let viewState = 'ready';
  let selected = new Set();
  let simulation = null;
  let connectionTimer = null;
  let draft = '';
  let dirty = false;
  const applyTheme = () => document.documentElement.dataset.theme = state.settings.theme;
  const save = () => {
    try { localStorage.setItem(storageKey, JSON.stringify(state)); }
    catch { storageWarning = 'Browser storage is full/unavailable. Export local work before closing.'; }
  };
  const notify = (message, error = false) => {
    const target = document.querySelector('#notice');
    if (target) { target.textContent = message; target.className = `notice ${error ? 'error' : 'success'}`; target.focus(); }
  };
  const audit = (action, target) => {
    const row = { id: crypto.randomUUID(), name: action, actor: local ? 'Local end user' : persona, target, outcome: 'synthetic success', time: new Date().toISOString() };
    // Only typed operation metadata: never configuration, values, form bodies or source provenance.
    state.events.unshift(row);
    state.events = state.events.slice(0, 100);
    if (state.collections.audit) state.collections.audit.rows = [...state.events];
    save();
  };
  const canWrite = key => {
    if (local) return true;
    if (persona === 'Global administrator') return true;
    if (persona === 'Security auditor' || persona === 'AI agent') return false;
    if (scope === 'Account') {
      if (['Account owner', 'Enterprise administrator'].includes(persona)) return true;
      return persona === 'Support specialist' && key === 'support';
    }
    return ({ 'Support specialist': [], 'Service operator': ['operations'], 'Billing specialist': ['billing'], 'Product owner': ['customers'] }[persona] ?? []).includes(key);
  };
  // Guided edits patch the same local native document. UI-only plans (custody,
  // identity, fleet, hierarchy designs) never become invented native fields.
  async function syncGuided(key, next, previous) {
    if (!local) return;
    if (!['variables', 'workloads', 'sources'].includes(key)) return;
    const schema = await loadPolicySchema();
    const { prepareGuidedPolicyChange } = await loadGuidedPolicyTools();
    const candidate = prepareGuidedPolicyChange(state.policy, key, next, previous, schema, state.collections.workloads.rows);
    model.replacePolicy(state, candidate);
  }
  function refreshGuidedFromPolicy() {
    const context = state.context;
    const oldVariables = state.collections.variables.rows;
    state.collections.sources.rows = Object.entries(state.policy.sources || {}).map(([name, source]) => ({ id: `native-source-${name}`, name, type: source.type, location: source.path || source.endpoint || source.project || source.account || '', domain: 'personal', identity: source.identity || 'none', status: 'not connected' }));
    state.collections.workloads.rows = Object.entries(state.policy.workloads || {}).map(([name, workload]) => ({ id: `native-workload-${name}`, name, environment: context, mode: workload.mode, delivery: 'Process environment', status: 'declared' }));
    const variables = new Map();
    for (const workload of Object.values(state.policy.workloads || {})) {
      for (const [name, value] of Object.entries(workload.environment || {})) variables.set(name, { id: crypto.randomUUID(), name, kind: 'Non-secret literal', value, source: '', environment: context, origin: 'Workload', expires: '' });
      for (const binding of workload.bindings || []) variables.set(binding.name, { id: crypto.randomUUID(), name: binding.name, kind: 'Source binding', value: binding.field || binding.resource || '', source: binding.source, environment: context, origin: 'Workload', expires: '' });
    }
    for (const [name, variable] of variables) {
      const old = oldVariables.find(row => row.name === name && row.value === variable.value && row.environment === context);
      if (old) variables.set(name, { ...variable, origin: old.origin, expires: old.expires });
    }
    state.collections.variables.rows = [...variables.values()];
  }
  const getRoute = () => {
    const bits = location.hash.replace(/^#\//, '').split('/');
    if (!local && ['account', 'operator'].includes(bits[0])) scope = titleCase(bits.shift());
    const candidate = local && ['overview', 'variables'].includes(bits[0]) ? 'workbench' : bits[0] || (local ? 'landing' : 'overview');
    return candidate === 'landing' && local ? candidate : model.navigation[scope].includes(candidate) ? candidate : 'not-found';
  };
  const link = key => `#/${local ? '' : `${scope.toLowerCase()}/`}${key}`;
  const badge = value => `<span class="badge ${/denied|error|revoked|blocked|unavailable|conflict/.test(value) ? 'bad' : /pending|draft|not |offline|uncertain/.test(value) ? 'warn' : ''}">${html(value)}</span>`;
  const button = (label, action, extra = '', kind = '') => `<button type="button" class="${kind}" data-action="${action}" ${extra}>${label}</button>`;
  const createLabel = (key, fallback = 'Create') => model.collections[key]?.action || fallback;
  const stopSimulation = () => { clearInterval(simulation); simulation = null; };
  const exportJSON = (value, name) => {
    const objectUrl = URL.createObjectURL(new Blob([JSON.stringify(value, null, 2)], { type: 'application/json' }));
    const anchor = document.createElement('a'); anchor.href = objectUrl; anchor.download = name; anchor.click();
    setTimeout(() => URL.revokeObjectURL(objectUrl), 1000);
  };

  function landing() {
    if (local) return `<div class="demo-entry"><a class="wordmark" href="#/landing">${brandMarkup('LOCAL CLIENT')}</a><h1>Environment workbench</h1><p>Local synthetic workspace. Add or import .env variables, inspect inheritance, assign providers/keys and availability windows, then validate and simulate a complete run.</p><a class="primary" href="#/workbench">DEMO · Open workbench</a><p class="hint">No login. No configuration upload. No real provider or native execution.</p><a href="#/settings">Profile & theme settings</a></div>`;
    return `<div class="landing"><header class="landing-header"><a class="wordmark" href="#/landing">${brandMarkup('/ c-key /')}</a><span class="badge">Local development · synthetic</span><label>Theme<select id="theme"><option>system</option><option>light</option><option>dark</option></select></label></header><main><h1>The right secrets,<br><em>ready for your apps.</em></h1><p class="lead">Connect the providers you already trust to the apps you approve. Shape each environment, understand every override, and keep your configuration close.</p><a class="primary cta" href="#/overview">Open the demo <span aria-hidden="true">↗</span></a><span class="caption">No login. No credentials. Reset whenever you like.</span><div class="flow" aria-label="Providers flow through local policy into approved apps"><div><span class="step">01</span><h2>Your providers</h2><p>Files · SOPS/age · Cloud stores</p></div><span aria-hidden="true">→</span><div><span class="step">02</span><h2>Your local policy</h2><p>Resolve · inspect · validate</p></div><span aria-hidden="true">→</span><div><span class="step">03</span><h2>Your approved apps</h2><p>Complete generations · explicit grants</p></div></div><div class="landing-bottom"><div><h3>Built for your workflow</h3><p>CLI, Go, Python and TypeScript SDKs. Optional MCP, enterprise identity and SCIM.</p></div><div><h3>Clear boundaries</h3><p>Configuration stays in this browser. All connections and financial actions in this preview are synthetic.</p></div><div><h3>Choose custody honestly</h3><p>Local or hosted software, with a stronger-custody migration path. Copied keys and plaintext cannot be recalled.</p></div></div></main><footer>Interactive development preview · No backend connected <a href="#/settings">Workspace settings</a></footer></div>`;
  }

  function shell(content) {
    const collection = state.collections[route];
    const avatar = state.settings.avatar ? `<img class="avatar" src="${html(state.settings.avatar)}" alt="Your selected avatar">` : `<span class="avatar">${html(state.settings.initials)}</span>`;
    return `<a class="skip" href="#main">Skip to content</a><div class="shell"><aside><a class="wordmark" href="${local ? '#/landing' : '#/account/overview'}">${brandMarkup('/ c-key /', state.settings.brand)}</a><div class="workspace-label">${local ? 'LOCAL CLIENT' : scope === 'Operator' ? 'OPERATOR CONSOLE' : 'YOUR ACCOUNT'}</div>${!local ? `<label class="scope-select">Surface<select id="scope">${model.scopes.map(item => `<option ${scope === item ? 'selected' : ''}>${item}</option>`).join('')}</select></label>` : ''}<nav aria-label="Main navigation">${model.navigation[scope].map((key, index) => `<a href="${link(key)}" ${route === key ? 'aria-current="page"' : ''}>${html(state.collections[key]?.title || ({ workbench: 'Variable workbench', configuration: 'Complete configuration', overview: 'Overview', policy: 'Complete policy', identity: 'Identity & access', settings: 'Settings' }[key]))}</a>`).join('')}</nav><div class="sidebar-foot"><span class="online-dot"></span> Synthetic adapters only<p>${local ? 'End-user workspace · on this device' : 'No real authentication or authority'}</p></div></aside><div class="workspace"><header class="topbar"><div><span class="breadcrumb">${html(model.name)} / ${html(scope)}</span>${local ? `<label class="context">Context<select id="context">${state.collections.environments.rows.map(row => `<option ${state.context === row.name ? 'selected' : ''}>${html(row.name)}</option>`).join('')}</select></label>` : `<label class="context">Demo presentation<select id="persona">${model.personas.map(item => `<option ${persona === item ? 'selected' : ''}>${html(item)}</option>`).join('')}</select></label>`}</div><a class="user" href="${link('settings')}">${avatar}<span>${html(state.settings.name)}<small>${local ? 'End user' : html(persona)}</small></span></a></header><div class="demo-bar"><strong>DEMO</strong> ${local ? 'Configuration stays in your browser. No native/provider connection.' : 'Synthetic presentation and permissions only. Role selection grants no server authority.'}<span>No production data</span></div><main id="main" tabindex="-1"><div class="page-head"><div><h1>${html(collection?.title || ({ overview: local ? 'A clearer view of your environment.' : scope === 'Operator' ? 'Service at a glance.' : 'Welcome back, Alex.', policy: 'Your complete local policy', identity: 'Identity & access', settings: 'Make it yours', 'not-found': 'Page not found' }[route]))}</h1><p class="description">${html(collection?.description || (route === 'overview' ? model.intro : ''))}</p></div><span class="date">04 NOV 2026<br>DEVELOPMENT PREVIEW</span></div><div id="notice" class="notice ${storageWarning ? 'error' : ''}" tabindex="-1" role="status" aria-live="polite">${html(storageWarning)}</div>${content}</main><footer><span>CircularQuay · ${local ? 'Local configuration boundary' : 'Managed service metadata boundary'}</span>${button('Reset demo', 'reset', '', 'quiet')}</footer></div></div><dialog id="editor" aria-labelledby="dialog-title"></dialog>`;
  }

  function overview() {
    const stats = local ? [
      ['Environments', state.collections.environments.rows.length, 'Explicit contexts'], ['Variables', state.collections.variables.rows.length, 'Values & source bindings'], ['Sources', state.collections.sources.rows.length, 'Local descriptors'], ['Generation', state.generation, 'Simulated activation'],
    ] : scope === 'Operator' ? [
      ['Customers', state.collections.customers.rows.length, 'Assigned demo perimeter'], ['Associations', state.collections.associations.rows.length, 'Public metadata only'], ['Active grants', state.collections.grants.rows.filter(row => row.status === 'active').length, 'Version & time bound'], ['Service', 'Demo', 'No backend connected'],
    ] : [['Profiles', state.collections.profiles.rows.length, 'KMS + age association'], ['Age fetches', state.collections.usage.rows.reduce((sum, row) => sum + Number(row.units), 0), 'Committed logical uses'], ['Subscription', state.plan, 'Illustrative entitlement'], ['Support', state.collections.support.rows.length, 'Synthetic tickets']];
    const quick = local ? ['variables', 'sources', 'policy'] : scope === 'Operator' ? ['customers', 'grants', 'operations'] : ['profiles', 'usage', 'billing'];
    return `<div class="stats">${stats.map(([label, value, detail]) => `<article><span>${label}</span><strong>${html(value)}</strong><small>${detail}</small></article>`).join('')}</div><div class="overview-grid"><article class="panel feature"><h2>${local ? 'Local workspace' : scope === 'Operator' ? 'Service workspace' : 'Account workspace'}</h2><p>${local ? 'Follow a value from its source through inheritance to its workload. Edit, validate and save locally, then inspect a complete candidate before activation.' : 'Inspect profile associations, exact access windows and logical usage. Every action here is a resettable simulation with no external effects.'}</p><div class="quick-links">${quick.map(key => `<a href="${link(key)}">${html(state.collections[key]?.title || 'Complete policy')} <span>↗</span></a>`).join('')}</div></article><article class="panel"><h2>${local ? 'A complete local workflow' : 'Your next steps'}</h2><ol class="steps"><li><strong>${local ? 'Choose a context' : 'Review your profile'}</strong><p>${local ? 'Keep development and staging separate.' : 'Enrollment reconciles KMS and age associations separately.'}</p></li><li><strong>${local ? 'Trace a variable' : 'Inspect effective access'}</strong><p>${local ? 'Locks, source bindings and finite overrides remain visible.' : 'Current identity, scope, version and window must all agree.'}</p></li><li><strong>${local ? 'Validate before saving' : 'Understand the limits'}</strong><p>${local ? 'UI checks are preliminary. Native validation remains required.' : 'Software custody is not independent non-exportable custody. Revocation cannot recall copies.'}</p></li></ol></article></div><article class="panel"><div class="panel-head"><div><h2>Small signals. Clear outcomes.</h2></div>${button('Add demo event', 'event', '', 'quiet')}</div>${state.events.length ? `<ul class="activity">${state.events.slice(0, 5).map(row => `<li><strong>${html(row.name)}</strong><span>${html(row.target)}</span><small>${html(row.time)}</small></li>`).join('')}</ul>` : '<p class="empty-inline">No changes yet. Your actions will appear here as safe metadata.</p>'}</article>`;
  }

  function special(key) {
    if (key === 'usage') return `<article class="panel compact"><h2>Exercise the meter</h2><p>Successful logical age fetches count once. Retry the same logical event to reconcile it without a second unit. No key bytes exist or are released.</p><form id="meter-form" class="form-grid"><label>Profile<select name="profile">${state.collections.profiles.rows.map(p => `<option value="${html(p.id)}">${html(p.name)}</option>`).join('')}</select></label><label>Operation<select name="type"><option>age retrieval</option><option>KMS wrap</option><option>KMS unwrap</option><option>association enrollment</option><option>local reuse</option></select></label><label>Simulated result<select name="result"><option>committed</option><option>denied</option><option>audit unavailable</option><option>precommit failure</option><option>uncertain</option></select></label><div class="actions"><button>Mock event</button>${button('Retry last event', 'retry-meter', '', 'quiet')}</div></form></article>`;
    if (key === 'billing') return `<article class="panel compact"><h2>${html(state.plan)}</h2><p>No real subscription or pricing provider is connected. This only previews a confirmation flow.</p><div class="actions">${button('Change plan', 'plan')}${button('Cancel plan', 'cancel-plan', '', 'quiet')}${button('Portal preview', 'portal', '', 'quiet')}</div></article>`;
    if (key === 'profiles') return `<div class="callout">Enrollment creates two associations idempotently. It does not fetch an age key, charge a usage unit, download an offline key or enable hybrid.</div>`;
    if (key === 'fleet') return `<div class="callout">${selected.size} explicitly selected devices. ${button('Inspect selected', 'fleet-inspect')}${button('Mock refresh', 'fleet-refresh', '', 'quiet')}<p id="fleet-result">Offline/denied targets report individually. No cross-fleet atomicity or real dispatch.</p></div>`;
    if (key === 'workloads') return `<div class="callout">${button('Mock activate', 'activate')} ${button('Mock failure', 'activation-failure', '', 'quiet')}<p>Active generation: ${html(state.generation)}. Failed candidates preserve the preceding complete generation.</p></div>`;
    if (key === 'migration') return `<div class="callout">${button('Next stage', 'migrate')}${button('Mock interruption', 'migration-failure', '', 'quiet')}<p>Progress is resumable synthetic state; old copied keys remain usable. No hardware import or recovery proof.</p></div>`;
    if (key === 'telemetry') return `<article class="panel compact"><div class="panel-head"><h2>Live simulation</h2><div>${button(simulation ? 'Pause simulation' : 'Start simulation', 'simulation')}${button('Reset signals', 'reset-signals', '', 'quiet')}</div></div><p>One bounded 3-second timer, last 100 events, paused in background. No collector or query requests.</p>${local ? `<form id="telemetry-settings" class="form-grid"><label>Collector ingestion endpoint<input name="collector" type="url" value="${html(state.collector || 'https://collector.example.invalid/v1/traces')}"></label><label>Separate query endpoint<input name="query" type="url" value="${html(state.query || 'https://query.example.invalid')}"></label><label>Signals<select name="signals"><option>Disabled</option><option>Demo only</option></select></label><button>Save locally</button></form>` : ''}</article>`;
    return '';
  }

  function table(key) {
    const c = state.collections[key];
    const rows = viewState === 'empty' ? [] : c.rows.filter(row => (!(!local && persona === 'Support specialist' && key === 'customers') || row.id === 'customer-1') && JSON.stringify(row).toLowerCase().includes(search.toLowerCase()) && (!filter || (row.status || row.outcome) === filter));
    const statuses = [...new Set(c.rows.map(row => row.status || row.outcome).filter(Boolean))];
    const readOnly = c.readonly || !canWrite(key);
    return `${special(key)}${c.note ? `<div class="callout">${html(c.note)}</div>` : ''}${!local && !canWrite(key) ? '<div class="callout">Read-only for this synthetic presentation. Real servers must independently verify identity and capabilities.</div>' : ''}<article class="panel table-panel"><div class="table-tools"><label class="search">Search ${html(c.title)}<input type="search" id="search" value="${html(search)}" placeholder="Search this view…"></label><label>Filter<select id="filter"><option value="">All states</option>${statuses.map(status => `<option ${filter === status ? 'selected' : ''}>${html(status)}</option>`).join('')}</select></label><label>Preview state<select id="view-state">${['ready', 'loading', 'error', 'empty'].map(item => `<option ${viewState === item ? 'selected' : ''}>${item}</option>`).join('')}</select></label><div class="table-actions">${button('Export', 'export', '', 'quiet')}${!readOnly ? button(createLabel(key), 'create') : ''}</div></div><div id="table-content">${viewState === 'loading' ? '<div class="empty" role="status">Loading synthetic records… <p>Choose Ready to restore this view.</p></div>' : viewState === 'error' ? `<div class="empty error" role="alert">The demo adapter is unavailable.<p>Your saved data is intact.</p>${button('Retry', 'retry')}</div>` : !rows.length ? `<div class="empty"><h2>No matching records</h2><p>Adjust your search/filter or create a synthetic record.</p>${button('Clear filters', 'clear', '', 'quiet')}</div>` : `<div class="table-scroll" role="region" aria-label="${html(c.title)} table" tabindex="0"><table><caption class="sr-only">${html(c.title)}</caption><thead><tr>${key === 'fleet' ? '<th>Select</th>' : ''}${c.fields.slice(0, 5).map(field => `<th scope="col">${html(field.label)}</th>`).join('')}<th scope="col">${c.fields.slice(0, 5).some(f => f.key === 'status') ? 'Actions' : 'State / actions'}</th></tr></thead><tbody>${rows.map(row => `<tr>${key === 'fleet' ? `<td><input type="checkbox" aria-label="Select ${html(row.name)}" data-select="${html(row.id)}" ${selected.has(row.id) ? 'checked' : ''}></td>` : ''}${c.fields.slice(0, 5).map((field, index) => `<td>${field.key === 'status' ? badge(row[field.key] || '') : index === 0 ? `<button type="button" class="row-title" data-action="inspect" data-id="${html(row.id)}">${html(row[field.key])}</button>` : html(row[field.key])}</td>`).join('')}<td><div class="row-actions">${!c.fields.slice(0, 5).some(f => f.key === 'status') && row.status ? badge(row.status) : ''}${button('Details', 'inspect', `data-id="${html(row.id)}"`, 'quiet')}${!readOnly ? button('Edit', 'edit', `data-id="${html(row.id)}"`, 'quiet') : ''}</div></td></tr>`).join('')}</tbody></table></div>`}</div><div class="table-foot"><span>${rows.length} of ${c.rows.length} records · browser demo store</span><span>${readOnly ? 'Read-only view' : 'Changes are local and resettable'}</span></div></article>`;
  }

  function identity() {
    return `<div class="overview-grid"><article class="panel"><h2>Choose an integration intentionally.</h2><p>${local ? 'Demo entry needs no login. An end user’s source/provider credentials remain on their client.' : 'Demo presentation is not authentication. Real integrations must verify credentials, issuer/audience, tenant and server-issued capabilities.'}</p><form id="identity-form" class="stack"><label>Provider<select name="provider">${['Demo OIDC', 'Netlify Identity (entitlement unverified)', 'Cloudflare verified issuer (not configured)', 'Unsupported provider'].map(provider => `<option ${state.identityProvider === provider ? 'selected' : ''}>${html(provider)}</option>`).join('')}</select></label><label>Issuer metadata URL<input type="url" name="issuer" placeholder="https://issuer.example.invalid/.well-known/openid-configuration" value="${html(state.issuer || '')}"></label><label>Client/application identifier<input name="client" value="${html(state.identityClient || '')}" placeholder="synthetic-public-client"></label><p>No credential fields. These endpoints are stored as demo metadata and are never fetched.</p><div class="actions"><button>Save setup</button>${button('Mock connect', 'connect-identity', '', 'quiet')}${button('Disconnect', 'disconnect-identity', '', 'quiet')}</div></form></article><article class="panel"><h2 id="identity-status">${html(state.identity)}</h2><p>Supported states are not configured, connecting, demo, unsupported and error. Platform logos or paid hosting do not prove an entitled identity capability.</p><label>Exercise a result<select id="identity-result"><option>demo</option><option>error</option><option>unsupported</option><option>not configured</option></select></label><div class="callout">${local ? 'Always an end-user workspace. There is no client administrator role selector.' : 'Selected presentation: ' + html(persona) + '. No role change here confers actual access.'}</div><h3>Real integration prerequisites</h3><ul><li>Verified issuer, audience and client enrollment</li><li>Current tenant-bound server capabilities</li><li>Qualified MFA/possession and revocation</li><li>No shared human login for AI/service principals</li></ul></article></div>`;
  }

  function settings() {
    const s = state.settings;
    return `<form id="settings-form" class="settings-grid"><article class="panel"><h2>A familiar workspace.</h2><div class="form-grid"><label>Display name<input name="name" required maxlength="80" value="${html(s.name)}"></label><label>Email<input name="email" type="email" required value="${html(s.email)}"></label><label>Organization<input name="organization" required value="${html(s.organization)}"></label><label>Avatar initials<input name="initials" required maxlength="3" value="${html(s.initials)}"></label><label>Avatar image<input id="avatar-file" type="file" accept="image/png,image/jpeg,image/webp"></label><label>Brand image<input id="brand-file" type="file" accept="image/png,image/jpeg,image/webp"></label></div><p class="hint">Images stay in this browser’s demo store. PNG/JPEG/WebP only, maximum 256 KB. No remote image URL or upload.</p>${s.avatar ? `<img class="image-preview" src="${html(s.avatar)}" alt="Avatar preview">${button('Remove avatar', 'remove-avatar', '', 'quiet')}` : ''}${s.brandImage ? `<img class="brand-preview" src="${html(s.brandImage)}" alt="Brand image preview">${button('Remove brand image', 'remove-brand', '', 'quiet')}` : ''}</article><article class="panel"><h2>Set your rhythm.</h2><div class="stack"><label>Theme<select name="theme">${['system', 'light', 'dark'].map(t => `<option ${s.theme === t ? 'selected' : ''}>${t}</option>`).join('')}</select></label><label>Workspace brand<input name="brand" required maxlength="40" value="${html(s.brand)}"></label><label>Notifications<select name="notifications">${['Security and usage', 'Security only', 'None'].map(t => `<option ${s.notifications === t ? 'selected' : ''}>${t}</option>`).join('')}</select></label><label>Demo retention preference<select name="retention">${['7 days', '30 days', '90 days'].map(t => `<option ${s.retention === t ? 'selected' : ''}>${t}</option>`).join('')}</select></label><button>Save settings</button></div><p>Notification/retention preferences are synthetic. No emails, server retention policy or financial-record deletion.</p>${button('Reset all demo data', 'reset', '', 'quiet')}</article></form>`;
  }

  function policy() {
    if (!dirty) draft = JSON.stringify(state.policy, null, 2);
    return `<div class="callout">Complete native policy authoring stays local. Edit every source/workload/delivery, composition, grant, MCP, repository, observability and update field here. Upcoming UI-only designs are kept separate from native configuration fields. Browser checks are preliminary; authenticate the selected native binary and run its complete validation before use.</div><article class="panel"><div class="panel-head"><div><h2>Policy document</h2><p>JSON · local browser draft · ${dirty ? 'unsaved edits' : 'saved'}</p></div><div class="actions">${button('Import local JSON', 'import-policy', '', 'quiet')}${button('Export saved JSON', 'export-policy', '', 'quiet')}</div></div><label>Configuration JSON<textarea id="policy-json" class="code" spellcheck="false">${html(draft)}</textarea></label><div class="actions">${button('Check candidate', 'validate-policy')}${button('Save locally', 'save-policy')}${button('Discard draft', 'discard-policy', '', 'quiet')}</div><pre id="policy-result" class="result" aria-live="polite">No native command or source retrieval will run.</pre><input type="file" id="policy-import" accept="application/json,.json" hidden></article><article class="panel"><h2>Native contract reference</h2><p>Schema-backed fields: version, sources, workloads, grants, mcp, repository, observability, updates and composition. Secret material belongs outside policy; protected references are distinct from the credentials they point to.</p><div class="flow small"><div>Local file / draft</div><span>→</span><div>Complete candidate</div><span>→</span><div>Local download</div></div><p>No upload, URL fetch, telemetry body, hosted validation or server persistence exists. Files are read only after your explicit browser selection.</p></article>`;
  }

  function validateRow(key, row, old) {
    const c = state.collections[key];
    if (c.rows.some(item => item.id !== old?.id && item.name?.toLowerCase() === row.name?.toLowerCase() && (!local || item.environment === row.environment))) return 'A record with that name already exists in this context.';
    if (row.starts && row.ends && new Date(row.ends) <= new Date(row.starts)) return 'The window end must be after its start.';
    if (row.ends && !Number.isFinite(Date.parse(row.ends))) return 'A finite valid deadline is required.';
    if (key === 'variables') {
      if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(row.name)) return 'Use a valid environment variable name.';
      if (old?.origin === 'Organization locked') return 'This organization value is locked. A separately authorized central revision is required.';
      if (row.origin === 'Organization locked') return 'End-user edits cannot create an organization lock.';
      if (!state.collections.environments.rows.some(item => item.name === row.environment)) return 'Choose an existing local environment.';
      if (row.kind === 'Source binding' && !state.collections.sources.rows.some(item => item.name === row.source)) return 'Select an existing local source ID.';
      if (row.origin === 'User override' && (!row.expires || !Number.isFinite(Date.parse(row.expires)) || Date.parse(row.expires) <= Date.now())) return 'User overrides require a future finite expiry.';
    }
    if (key === 'sources' && old && old.name !== row.name && state.collections.variables.rows.some(v => v.source === old.name)) return 'This source is referenced. Update its bindings before renaming it.';
    if (key === 'inheritance' && !/^[0-9a-f]{64}$/.test(row.sha256)) return 'A 64-character lowercase SHA-256 pin is required.';
    if (key === 'profiles' && row.custody !== 'Software') return 'Required independent/HSM custody is unavailable. No downgrade or enrollment occurred.';
    if (key === 'custody' && /required/.test(row.mode)) return 'Required stronger custody is unavailable; this profile cannot become ready.';
    if (key === 'grants' && !state.collections.profiles.rows.some(item => item.id === row.profile)) return 'Unknown profile. The selected profile must belong to the exact assigned tenant.';
    if (key === 'grants' && row.tenant !== 'customer-1') return 'This demo grant scope is customer-1 only. Cross-tenant grants are denied.';
    if (key === 'roles' && /\*/.test(row.scope)) return 'An explicit bounded scope is required; no wildcard delegation.';
    return '';
  }

  function editDialog(key, id, inspect = false) {
    const c = state.collections[key];
    const row = c.rows.find(item => item.id === id);
    const dialog = document.querySelector('#editor');
    const readonly = inspect || c.readonly || !canWrite(key);
    dialog.innerHTML = `<div class="dialog-head"><div><h2 id="dialog-title">${row ? html(row.name) : 'Create a synthetic record'}</h2></div>${button('Close', 'close-dialog', 'aria-label="Close dialog"', 'quiet')}</div>${readonly ? `<dl class="details">${Object.entries(row || {}).map(([key, value]) => `<dt>${html(titleCase(key))}</dt><dd>${html(value)}</dd>`).join('')}</dl><p>${html(c.note || c.description)}</p>${!c.readonly && canWrite(key) ? button('Edit record', 'edit', `data-id="${html(id)}"`) : ''}` : `<form id="record-form"><div class="form-grid">${c.fields.map(f => `<label>${html(f.label)}${f.type === 'select' ? `<select name="${f.key}">${f.options.map(v => `<option ${row?.[f.key] === v ? 'selected' : ''}>${html(v)}</option>`).join('')}</select>` : f.type === 'textarea' ? `<textarea name="${f.key}" required maxlength="1000">${html(row?.[f.key])}</textarea>` : `<input name="${f.key}" type="${['number', 'datetime-local'].includes(f.type) ? f.type : 'text'}" ${f.type === 'optional' ? '' : 'required'} ${f.type === 'number' ? 'min="0" max="1000000"' : 'maxlength="512"'} value="${html(row?.[f.key])}">`}</label>`).join('')}</div><p id="form-error" class="error" role="alert"></p><div class="dialog-actions"><button>${row ? 'Save changes' : createLabel(key, 'Create record')}</button>${row ? button(['grants', 'roles', 'associations'].includes(key) ? 'Revoke' : 'Delete', 'delete', `data-id="${html(row.id)}"`, 'danger') : ''}${button('Cancel', 'close-dialog', '', 'quiet')}</div><p class="hint">Synthetic changes only. No remote authority, provider call or charge.</p></form>`}`;
    if (!dialog.open) dialog.showModal();
    const form = dialog.querySelector('form');
    if (form) form.onsubmit = async event => {
      event.preventDefault();
      if (!canWrite(key)) return;
      const values = Object.fromEntries(new FormData(form));
      for (const f of c.fields) if (f.type === 'number') values[f.key] = Number(values[f.key]);
      const error = validateRow(key, values, row);
      if (error) { dialog.querySelector('#form-error').textContent = error; return; }
      const next = { ...row, ...values, id: row?.id || crypto.randomUUID() };
      if (key === 'profiles' && !row) {
        next.kms = 'ready'; next.age = 'ready'; next.status = 'active';
        for (const kind of ['KMS', 'age']) state.collections.associations.rows.push({ id: crypto.randomUUID(), name: `${next.name} ${kind}`, profile: next.id, tenant: 'customer-1', kind, version: 'v1', status: 'ready' });
      }
      try { await syncGuided(key, next, row); }
      catch (error) { dialog.querySelector('#form-error').textContent = error.message; return; }
      if (row) c.rows[c.rows.indexOf(row)] = next; else c.rows.push(next);
      audit(row ? 'Record updated' : 'Record created', next.id);
      dialog.close(); render(); notify('Synthetic record saved locally after browser schema/reference checks. Native validation and activation have not run.');
    };
  }

  function checkPolicy(text) {
    if (text.length > 1024 * 1024) throw new Error('Document exceeds the 1 MiB limit.');
    const value = model.parseDocument(text);
    if (!value || Array.isArray(value) || typeof value !== 'object' || value.version !== 1) throw new Error('Policy must be an object with version 1.');
    const roots = ['version', 'sources', 'workloads', 'grants', 'mcp', 'repository', 'observability', 'updates', 'composition'];
    for (const key of Object.keys(value)) if (!roots.includes(key)) throw new Error(`Unknown root field: ${key}`);
    if (!value.composition && (!Object.keys(value.sources || {}).length || !Object.keys(value.workloads || {}).length)) throw new Error('Sources and workloads must be non-empty without composition.');
    for (const [id, w] of Object.entries(value.workloads || {})) {
      if (!['offline', 'online', 'hybrid'].includes(w.mode) || !Array.isArray(w.bindings) || !w.bindings.length) throw new Error(`Workload ${id} needs a mode and non-empty bindings.`);
      const names = new Set();
      for (const binding of w.bindings) {
        if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(binding.name) || names.has(binding.name.toLowerCase())) throw new Error(`Workload ${id}: invalid/duplicate binding name.`);
        names.add(binding.name.toLowerCase());
        if (!Object.hasOwn(value.sources || {}, binding.source) && !value.composition) throw new Error(`Workload ${id}: missing source ${binding.source}.`);
      }
      if (w.mode === 'online' && w.snapshot && Object.keys(w.snapshot).length) throw new Error('Online mode cannot have an offline snapshot.');
    }
    return value;
  }

  function render() {
    route = getRoute();
    applyTheme();
    if (route === 'configuration' && local) { model.configuration(root, state, { save: () => { refreshGuidedFromPolicy(); save(); }, exportJSON, changed: value => { dirty = value; } }); return; }
    if (route === 'workbench' && local) { model.workbench(root, state, { save, exportJSON, reset: () => { location.hash = '#/settings'; }, changed: value => { dirty = value; } }); return; }
    if (route === 'landing') root.innerHTML = landing();
    else root.innerHTML = shell(route === 'overview' ? overview() : route === 'identity' ? identity() : route === 'settings' ? settings() : route === 'policy' && local ? policy() : state.collections[route] ? table(route) : `<article class="panel"><p>This route is unavailable.</p><a href="${link('overview')}">Return to overview</a></article>`);
    bind();
  }

  function bind() {
    const on = (id, event, handler) => { const target = document.getElementById(id); if (target) target.addEventListener(event, handler); };
    on('theme', 'change', e => { state.settings.theme = e.target.value; save(); applyTheme(); });
    if (document.querySelector('#theme')) document.querySelector('#theme').value = state.settings.theme;
    on('scope', 'change', e => { scope = e.target.value; location.hash = `#/${scope.toLowerCase()}/overview`; });
    on('persona', 'change', e => { persona = e.target.value; render(); notify('Synthetic presentation changed. No server authority was granted.'); });
    on('context', 'change', e => { state.context = e.target.value; state.generation = 'none'; save(); render(); notify('Context changed. Previous simulated activation cleared.'); });
    on('search', 'input', e => { const cursor = e.target.selectionStart; search = e.target.value; render(); const input = document.querySelector('#search'); input.focus(); input.setSelectionRange(cursor, cursor); });
    on('filter', 'change', e => { filter = e.target.value; render(); });
    on('view-state', 'change', e => { viewState = e.target.value; render(); });
    on('policy-json', 'input', e => { draft = e.target.value; dirty = true; });
    on('settings-form', 'submit', e => {
      e.preventDefault(); state.settings = { ...state.settings, ...Object.fromEntries(new FormData(e.target)) }; save(); render(); notify('Profile and preferences saved in this browser.');
    });
    for (const kind of ['avatar', 'brand']) on(`${kind}-file`, 'change', async e => {
      const file = e.target.files[0];
      if (!file) return;
      if (file.size > 262144 || !['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) { notify('Choose PNG, JPEG or WebP under 256 KB.', true); return; }
      const reader = new FileReader();
      reader.onload = () => { state.settings[kind === 'avatar' ? 'avatar' : 'brandImage'] = reader.result; save(); render(); notify('Image saved locally; no upload occurred.'); };
      reader.readAsDataURL(file);
    });
    on('identity-form', 'submit', e => {
      e.preventDefault(); const form = new FormData(e.target); state.issuer = form.get('issuer'); state.identityClient = form.get('client'); state.identityProvider = form.get('provider');
      state.identity = state.identityProvider === 'Unsupported provider' ? 'unsupported' : 'not configured'; save(); render(); notify('Demo identity metadata saved. No issuer or account was contacted.');
    });
    on('telemetry-settings', 'submit', e => { e.preventDefault(); const fields = Object.fromEntries(new FormData(e.target)); state.collector = fields.collector; state.query = fields.query; save(); notify('Endpoints saved locally; no network call or exporter enabled.'); });
    on('meter-form', 'submit', e => {
      e.preventDefault(); const fields = Object.fromEntries(new FormData(e.target));
      const p = state.collections.profiles.rows.find(p => p.id === fields.profile);
      const result = p?.status === 'active' ? fields.result : 'denied';
      const event = { id: crypto.randomUUID(), name: `Access ${state.collections.usage.rows.length + 1}`, profile: p?.name || 'unknown', type: fields.type, status: result, units: result === 'committed' && fields.type === 'age retrieval' ? 1 : 0 };
      state.collections.usage.rows.unshift(event); state.collections.usage.rows = state.collections.usage.rows.slice(0, 100); state.lastMeter = event.id; audit('Synthetic logical event', event.id); render(); notify(`Synthetic ${result}: ${event.units} age fetch unit. No key was disclosed.`);
    });
    on('policy-import', 'change', async e => {
      const file = e.target.files[0]; if (!file) return;
      try { if (file.size > 1024 * 1024) throw new Error('Maximum file size is 1 MiB.'); const text = await file.text(); checkPolicy(text); draft = text; dirty = true; render(); notify('Imported into local draft only. Check and save when ready.'); }
      catch (error) { notify(error.message, true); }
    });
    for (const checkbox of document.querySelectorAll('[data-select]')) checkbox.onchange = e => { e.target.checked ? selected.add(e.target.dataset.select) : selected.delete(e.target.dataset.select); render(); };
    root.onclick = e => {
      if (e.target.closest('.skip')) { e.preventDefault(); document.querySelector('#main')?.focus(); return; }
      const trigger = e.target.closest('[data-action]');
      if (trigger) action(trigger.dataset.action, trigger.dataset.id);
    };
  }

  function confirmation(title, description, perform) {
    const dialog = document.querySelector('#editor');
    dialog.innerHTML = `<div class="dialog-head"><h2 id="dialog-title">${html(title)}</h2>${button('Close', 'close-dialog', '', 'quiet')}</div><p>${html(description)}</p><form id="confirmation"><label class="check-label"><input name="approved" type="checkbox" required> I confirm this synthetic demo action.</label><div class="actions"><button>Confirm</button>${button('Cancel', 'close-dialog', '', 'quiet')}</div></form>`;
    if (!dialog.open) dialog.showModal();
    dialog.querySelector('form').onsubmit = async e => {
      e.preventDefault();
      try {
        const completed = await perform();
        if (completed !== false) dialog.close();
      } catch (error) {
        notify(error.message, true);
      }
    };
  }

  function action(name, id) {
    const c = state.collections[route];
    if (['create', 'edit', 'delete'].includes(name) && (!canWrite(route) || c?.readonly)) { notify('Read-only synthetic capability.', true); return; }
    if (name === 'close-dialog') document.querySelector('#editor').close();
    else if (name === 'create' || name === 'edit' || name === 'inspect') editDialog(route, id, name === 'inspect');
    else if (name === 'delete') {
      const row = c.rows.find(item => item.id === id);
      if (route === 'sources' && state.collections.variables.rows.some(v => v.source === row.name)) { document.querySelector('#form-error').textContent = 'This source is referenced. Remove or rebind its variables first.'; return; }
      if (row.origin === 'Organization locked') { document.querySelector('#form-error').textContent = 'An organization-locked value cannot be deleted locally.'; return; }
      confirmation('Confirm demo change', 'Revocation affects future release only. Existing copied keys/plaintext cannot be recalled. Reset can restore these synthetic records.', async () => {
        if (['grants', 'roles', 'associations'].includes(route)) row.status = 'revoked'; else { await syncGuided(route, null, row); c.rows = c.rows.filter(item => item.id !== id); }
        selected.delete(id); audit('Record removed or revoked', id); render(); notify('Synthetic record changed. No external effect.');
      });
    } else if (name === 'reset') confirmation('Reset this demo?', 'This resets only this surface’s synthetic browser data, settings and images. Export any local draft first.', () => {
      stopSimulation(); clearTimeout(connectionTimer); state = { settings: clone(model.settings), collections: clone(model.collections), policy: clone(model.policy ?? {}), context: 'development', generation: 'none', plan: 'Free demo', identity: 'not configured', events: [] };
      dirty = false; search = ''; filter = ''; selected.clear(); save(); render(); notify('Demo restored to seeded state.');
    });
    else if (name === 'export') exportJSON(c.rows, `cquay-${model.plane}-${route}-demo.json`);
    else if (name === 'retry' || name === 'clear') { viewState = 'ready'; search = ''; filter = ''; render(); }
    else if (name === 'retry-meter') notify(state.lastMeter ? 'Last logical event reconciled. No additional meter unit or duplicate release.' : 'Simulate a logical event first.', !state.lastMeter);
    else if (name === 'event') { audit('Demo event', local ? 'local-workspace' : 'customer-1'); render(); }
    else if (name === 'validate-policy' || name === 'save-policy') {
      const result = document.querySelector('#policy-result');
      try { const value = checkPolicy(draft); if (name === 'save-policy') { const retainedDraft = model.replacePolicy(state, value); refreshGuidedFromPolicy(); dirty = false; save(); if (retainedDraft) notify('Saved policy changed. Your separate complete-configuration draft is retained; use Load saved policy there to replace it explicitly.'); } result.textContent = `${name === 'save-policy' ? 'Saved locally. Guided sources, workloads and bindings refreshed. ' : ''}Preliminary JSON/root/binding checks passed. Full native schema, composition, provider, permissions and trust validation have NOT run.`; result.className = 'result success'; }
      catch (error) { result.textContent = error.message; result.className = 'result error'; }
    } else if (name === 'discard-policy') { dirty = false; render(); }
    else if (name === 'import-policy') document.querySelector('#policy-import').click();
    else if (name === 'export-policy') exportJSON(state.policy, 'cquay-local-policy.json');
    else if (name === 'plan' || name === 'cancel-plan' || name === 'portal') {
      if (!canWrite('billing')) { notify('This presentation cannot change billing.', true); return; }
      confirmation('Synthetic financial preview', 'No payment service is connected, no card data is collected and no money will move. This previews an explicit user confirmation.', () => {
        state.plan = name === 'plan' ? (state.plan === 'Team demo' ? 'Free demo' : 'Team demo') : name === 'cancel-plan' ? 'Cancelled demo' : state.plan;
        audit('Synthetic billing confirmation', 'customer-1'); render(); notify(name === 'portal' ? 'Demo portal preview complete. A real portal requires a verified customer-bound session.' : 'Demo subscription state changed. No charge or real subscription.');
      });
    } else if (name === 'connect-identity') {
      clearTimeout(connectionTimer); const result = document.querySelector('#identity-result').value;
      state.identity = 'connecting'; document.querySelector('#identity-status').textContent = 'connecting';
      connectionTimer = setTimeout(() => { state.identity = state.identityProvider === 'Unsupported provider' ? 'unsupported' : result; save(); if (route === 'identity') render(); }, 500);
    } else if (name === 'disconnect-identity') { clearTimeout(connectionTimer); state.identity = 'not configured'; save(); render(); }
    else if (name === 'remove-avatar' || name === 'remove-brand') { delete state.settings[name === 'remove-avatar' ? 'avatar' : 'brandImage']; save(); render(); }
    else if (name === 'activate' || name === 'activation-failure') {
      if (name === 'activate') { state.generation = `demo-${state.events.length + 1}`; state.collections.workloads.rows.forEach(row => { row.status = 'active'; }); audit('Simulated activation', 'complete-local-candidate'); render(); notify('Complete synthetic candidate activated. No native command ran.'); }
      else notify(`Required-source failure simulated. Active generation ${state.generation} preserved; no partial activation or fallback after denial.`, true);
    } else if (name === 'fleet-inspect' || name === 'fleet-refresh') {
      if (!selected.size) { notify('Select at least one explicit device.', true); return; }
      document.querySelector('#fleet-result').textContent = state.collections.fleet.rows.filter(row => selected.has(row.id)).map(row => `${row.name}: ${row.status === 'synthetic online' ? 'simulated result ready' : 'unavailable — no dispatch'}`).join(' · ');
      audit('Selected fleet simulation', `${selected.size} explicit synthetic targets`);
    } else if (name === 'migrate' || name === 'migration-failure') {
      const row = c.rows[0]; if (!row) { notify('Create a local migration plan first.', true); return; }
      if (name === 'migration-failure') { row.status = 'blocked'; save(); render(); notify('Interruption recorded. Resume from the retained stage; no keys moved.', true); return; }
      const stages = ['planned', 'authorized', 'staged', 'verified', 'cutover', 'reconciled'];
      confirmation('Advance synthetic migration?', 'Separate authorization, verification and cutover are required in a real migration. This advances only a local mock stage.', () => { row.stage = stages[Math.min(stages.indexOf(row.stage) + 1, 5)]; row.status = row.stage === 'reconciled' ? 'complete' : 'simulating'; audit('Migration stage simulated', row.id); render(); });
    } else if (name === 'simulation') {
      if (simulation) stopSimulation();
      else simulation = setInterval(() => {
        if (document.hidden) return;
        const rows = state.collections.telemetry.rows;
        rows.forEach((row, i) => { if (local) row.value = String(1 + (state.events.length + i) % 50); else row.latency = 15 + (state.events.length + i * 17) % 65; });
        audit('Synthetic health sample', local ? 'client-signals' : 'service-signals'); if (route === 'telemetry') render();
      }, 3000);
      render();
    } else if (name === 'reset-signals') { stopSimulation(); state.collections.telemetry = clone(model.collections.telemetry); save(); render(); }
  }

  window.addEventListener('hashchange', () => { search = ''; filter = ''; viewState = 'ready'; render(); document.querySelector('#main')?.focus(); });
  window.addEventListener('beforeunload', event => { if (dirty) { event.preventDefault(); event.returnValue = ''; } });
  const reload = new EventSource('/__reload');
  reload.onmessage = () => { if (!dirty) location.reload(); else notify('Source changed; export/save or discard your draft before reloading.'); };
  window.addEventListener('pagehide', () => { stopSimulation(); clearTimeout(connectionTimer); reload.close(); });
  render();
}
