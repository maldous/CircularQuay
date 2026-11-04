const field = (key, label, type = 'text', options) => ({ key, label, type, options });
const collection = (title, description, fields, rows, extra = {}) => ({ title, description, fields, rows, ...extra });
export const model = {
  plane: 'local', name: 'Your workspace', intro: 'The right secrets, ready for your apps.',
  scopes: ['Client'],
  settings: { name: 'Alex Morgan', email: 'alex@example.invalid', organization: 'Personal workspace', theme: 'system', initials: 'AM', brand: 'CircularQuay', notifications: 'Security and usage', retention: '30 days' },
  navigation: { Client: ['workbench', 'configuration', 'environments', 'sources', 'workloads', 'inheritance', 'policy', 'windows', 'custody', 'migration', 'fleet', 'integrations', 'telemetry', 'identity', 'settings'] },
  policy: {
    version: 1,
    sources: { demo: { type: 'json', path: './synthetic-values.json' } },
    workloads: { studio: { mode: 'offline', environment: { APP_ENV: 'development' }, bindings: [{ name: 'DEMO_MESSAGE', source: 'demo', field: 'message' }] } },
    grants: [], updates: { check: 'never' },
  },
  collections: {
    environments: collection('Environments', 'Local context selection never establishes identity or permission. Switching context clears the simulated active generation.', [field('name', 'Environment'), field('profile', 'Local profile'), field('mode', 'Workload mode', 'select', ['offline', 'online', 'hybrid']), field('status', 'State', 'select', ['draft', 'validated', 'active'])], [
      { id: 'env-dev', name: 'development', profile: 'Personal workspace', mode: 'offline', status: 'active' },
      { id: 'env-stage', name: 'staging', profile: 'Release pipeline', mode: 'online', status: 'draft' },
    ]),
    variables: collection('Variables & bindings', 'Guided local editor. Values here are synthetic non-secret examples. Secret bindings use source references; real secrets must not be pasted into this preview.', [field('name', 'Variable name'), field('kind', 'Value kind', 'select', ['Non-secret literal', 'Source binding']), field('value', 'Demo literal / field selector'), field('source', 'Source ID (binding only)', 'optional'), field('environment', 'Environment'), field('origin', 'Provenance', 'select', ['Organization locked', 'Group default', 'Environment', 'Workload', 'User override']), field('expires', 'Override expiry', 'optional')], [
      { id: 'var-1', name: 'APP_ENV', kind: 'Non-secret literal', value: 'development', source: '', environment: 'development', origin: 'Organization locked', expires: '' },
      { id: 'var-2', name: 'DEMO_MESSAGE', kind: 'Source binding', value: 'message', source: 'demo', environment: 'development', origin: 'Workload', expires: '' },
      { id: 'var-3', name: 'LOG_LEVEL', kind: 'Non-secret literal', value: 'info', source: '', environment: 'development', origin: 'User override', expires: '2026-11-05T18:00' },
    ], { note: 'Declared → inherited → effective. Organization locks cannot be relaxed. A user override is finite and tied to its context/base revision; this preview simulates those checks locally.' }),
    sources: collection('Sources', 'Local source descriptors; no provider connection is made. Browser CORS/VPN/file access constraints are honored. There is no hosted fetch proxy.', [field('name', 'Source ID'), field('type', 'Provider', 'select', ['json', 'env', 'sops', 'openbao', 'vault', 'aws-sm', 'aws-ssm', 'azure', 'gcp', 'doppler', 'infisical', 'bitwarden', 'akeyless', 'onepassword']), field('location', 'Local path / provider resource'), field('domain', 'Custody domain', 'select', ['personal', 'corporate']), field('identity', 'Identity reference (not credentials)'), field('status', 'State', 'select', ['not connected', 'synthetic ready', 'unavailable'])], [
      { id: 'source-1', name: 'demo', type: 'json', location: './synthetic-values.json', domain: 'personal', identity: 'none', status: 'synthetic ready' },
      { id: 'source-2', name: 'corporate', type: 'openbao', location: 'https://vault.example.invalid', domain: 'corporate', identity: './protected-token-reference', status: 'not connected' },
    ], { note: 'For every provider-specific option, use the complete policy editor. Descriptor UI is a local design model, not an added native schema or working adapter.' }),
    workloads: collection('Workloads & delivery', 'Resolve every required binding before simulated activation. Declared, resolved, projected and active are distinct. No process, provider or publication target is executed.', [field('name', 'Workload ID'), field('environment', 'Environment'), field('mode', 'Mode', 'select', ['offline', 'online', 'hybrid']), field('delivery', 'Delivery design', 'select', ['Process environment', 'Protected file', 'systemd', 'Windows', 'Cloudflare', 'Netlify', 'Windmill']), field('status', 'State', 'select', ['declared', 'resolved', 'projected', 'active', 'denied'])], [
      { id: 'workload-1', name: 'studio', environment: 'development', mode: 'offline', delivery: 'Process environment', status: 'declared' },
    ]),
    inheritance: collection('Sources & inheritance', 'Trusted includes are pinned, namespaced and bounded. No last-writer-wins merge. Personal/corporate origins and credentials stay local.', [field('name', 'Include ID'), field('namespace', 'Namespace'), field('location', 'Local path / pinned repository'), field('sha256', 'SHA-256 pin'), field('depends', 'Earlier dependency IDs', 'optional'), field('visibility', 'Visibility', 'select', ['local', 'shared']), field('status', 'State', 'select', ['draft', 'pin verified', 'conflict', 'stale'])], [
      { id: 'include-1', name: 'personal', namespace: 'personal', location: './personal.json', sha256: '0'.repeat(64), depends: '', visibility: 'local', status: 'draft' },
    ], { note: 'Use the policy editor for exact composition descriptors, overrides and equality locks. Demo pins are placeholders, not proof of file identity or signatures.' }),
    windows: collection('Local policy & windows', 'Windows intersect; deny wins. Hybrid is an explicit choice and fallback is only for positively classified availability failures, never trust, denial, malformed data or expiry.', [field('name', 'Window'), field('environment', 'Environment'), field('action', 'Action', 'select', ['Workload launch', 'Snapshot use', 'Key retrieval']), field('starts', 'Starts', 'datetime-local'), field('ends', 'Ends', 'datetime-local'), field('status', 'Policy', 'select', ['allow within window', 'deny', 'approval required'])], [
      { id: 'window-1', name: 'Development hours', environment: 'development', action: 'Workload launch', starts: '2026-11-04T09:00', ends: '2026-11-04T18:00', status: 'allow within window' },
    ]),
    custody: collection('Keys & custody', 'Local age, hosted age retrieval and remote wrap/unwrap are different boundaries. Retained keys/plaintext are copyable. Revocation blocks future release, never recalls prior copies.', [field('name', 'Local profile'), field('mode', 'Selected custody', 'select', ['Local age', 'Hosted software', 'Independent managed KMS required', 'HSM required']), field('hybrid', 'Fallback', 'select', ['Disabled', 'Explicit availability-only snapshot']), field('kms', 'KMS association', 'select', ['not enrolled', 'synthetic ready', 'unavailable']), field('age', 'Age association', 'select', ['not enrolled', 'synthetic ready', 'unavailable']), field('status', 'State', 'select', ['draft', 'ready', 'denied'])], [
      { id: 'custody-1', name: 'Personal workspace', mode: 'Local age', hybrid: 'Disabled', kms: 'not enrolled', age: 'not enrolled', status: 'draft' },
    ], { note: 'Software service: shared runtime wrapping key outside DB and independent random profile DEKs. A compromised runtime/shared key exposes reachable DEKs. Required independent/HSM custody is unavailable and denies; no silent downgrade.' }),
    migration: collection('Migration & recovery', 'Plan locally, stage explicitly, verify and reconcile interruptions. This simulation never imports keys, provisions an account or rotates a real credential.', [field('name', 'Migration plan'), field('from', 'Current custody', 'select', ['Local age', 'OS protected', 'Hosted software']), field('to', 'Target', 'select', ['Hosted software', 'Independent managed KMS', 'HSM']), field('stage', 'Stage', 'select', ['planned', 'authorized', 'staged', 'verified', 'cutover', 'reconciled']), field('status', 'State', 'select', ['pending approval', 'simulating', 'blocked', 'complete'])], [
      { id: 'migration-1', name: 'Personal workspace migration', from: 'Local age', to: 'Hosted software', stage: 'planned', status: 'pending approval' },
    ]),
    fleet: collection('Selected devices', 'Your explicitly selected devices only. Direct agent connectivity, independent authorization and current policy must exist. No all-fleet wildcard or production dispatch.', [field('name', 'Device'), field('target', 'Immutable target ID'), field('environment', 'Environment'), field('profile', 'Authorizing profile'), field('status', 'Connectivity', 'select', ['synthetic online', 'offline', 'denied'])], [
      { id: 'device-1', name: 'Studio laptop', target: 'device-demo-a', environment: 'development', profile: 'Personal workspace', status: 'synthetic online' },
      { id: 'device-2', name: 'Build machine', target: 'device-demo-b', environment: 'staging', profile: 'Release pipeline', status: 'offline' },
    ]),
    integrations: collection('Integrations', 'Optional capabilities require qualified implementations and separate grants. These are local plans; enabling a demo row never connects a provider.', [field('name', 'Integration', 'select', ['CLI', 'Go SDK', 'TypeScript SDK', 'Python SDK', 'MCP', 'CI/CD', 'SCIM', 'GitHub exposure gate', 'GitGuardian gate', 'OIDC/SPIFFE', 'Device posture', 'External policy engine']), field('scope', 'Exact scope'), field('freshness', 'Evidence maximum age (minutes)', 'number'), field('required', 'Required gate', 'select', ['optional', 'required']), field('status', 'Capability', 'select', ['not configured', 'demo', 'unsupported', 'error'])], [
      { id: 'integration-1', name: 'MCP', scope: 'status only / studio', freshness: 15, required: 'optional', status: 'demo' },
      { id: 'integration-2', name: 'GitHub exposure gate', scope: 'synthetic repository / exact credential version', freshness: 15, required: 'required', status: 'not configured' },
      { id: 'integration-3', name: 'SCIM', scope: 'No provider connected', freshness: 15, required: 'optional', status: 'unsupported' },
    ], { note: 'Scanner workflow closure is not issuer revocation/verified rotation. Known exposure blocks survive closure, caches and restore. Unknown required evidence denies. MCP raw read is a separate disclosure grant.' }),
    telemetry: collection('Client telemetry', 'Local synthetic signals. Collector ingestion and query endpoints are separate capabilities. This demo makes no requests to either; real CORS/VPN/auth remain prerequisites.', [field('name', 'Signal'), field('value', 'Synthetic value'), field('status', 'State')], [
      { id: 'signal-1', name: 'Complete generations', value: '1', status: 'synthetic' },
      { id: 'signal-2', name: 'Required audit', value: 'No connected ledger', status: 'unavailable' },
    ], { readonly: true }),
  },
};
