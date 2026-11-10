const field = (key, label, type = 'text', options) => ({ key, label, type, options });
const collection = (title, description, fields, rows, extra = {}) => ({ title, description, fields, rows, ...extra });
const state = ['active', 'suspended', 'revoked', 'pending'];
export const model = {
  plane: 'remote',
  name: 'Managed service',
  intro: 'Your account, clear access, predictable usage.',
  scopes: ['Account', 'Operator'],
  personas: ['Account owner', 'Enterprise administrator', 'Support specialist', 'Security auditor', 'Service operator', 'Billing specialist', 'Product owner', 'AI agent', 'Global administrator'],
  settings: { name: 'Alex Morgan', email: 'alex@example.invalid', organization: 'Harbour Studio', initials: 'AM', brand: 'CQ', notifications: 'Security and usage', retention: '30 days' },
  navigation: {
    Account: ['overview', 'profiles', 'usage', 'billing', 'support', 'members', 'groups', 'identity', 'settings'],
    Operator: ['overview', 'customers', 'associations', 'grants', 'roles', 'billing', 'support', 'operations', 'audit', 'telemetry', 'identity', 'settings'],
  },
  collections: {
    profiles: collection('Profiles', 'Hosted associations only. Your client configuration stays on your device. Enrollment reconciles both associations without fetching a key.', [field('name', 'Profile name'), field('environment', 'Environment'), field('custody', 'Required custody', 'select', ['Software', 'Independent managed KMS', 'HSM']), field('mode', 'Access mode', 'select', ['online', 'offline', 'hybrid']), field('status', 'Status', 'select', state)], [
      { id: 'profile-studio', name: 'Studio development', environment: 'development', custody: 'Software', mode: 'online', status: 'active', kms: 'ready', age: 'ready' },
      { id: 'profile-release', name: 'Release pipeline', environment: 'staging', custody: 'Software', mode: 'hybrid', status: 'pending', kms: 'ready', age: 'pending' },
    ], { action: 'Enroll profile', note: 'Software custody uses a runtime wrapping key outside the database and random profile DEKs. A runtime/shared-key compromise can expose all reachable profile keys. Independent custody and HSM requirements are unavailable in this demo and deny enrollment.' }),
    usage: collection('Usage & history', 'Age retrieval, KMS crypto and association enrollment are separate event types. Denied/precommit attempts and local reuse do not count as successful age fetches. Retries share one logical event.', [field('name', 'Logical event'), field('profile', 'Profile'), field('type', 'Event type'), field('status', 'Outcome'), field('units', 'Age fetch units', 'number')], [
      { id: 'evt-3', name: 'Access 0003', profile: 'Studio development', type: 'age retrieval', status: 'committed', units: 1 },
      { id: 'evt-2', name: 'Access 0002', profile: 'Studio development', type: 'KMS unwrap', status: 'committed', units: 0 },
      { id: 'evt-1', name: 'Access 0001', profile: 'Release pipeline', type: 'age retrieval', status: 'denied', units: 0 },
    ], { readonly: true }),
    billing: collection('Subscription & invoices', 'Illustrative demo entitlements only; no product tariff is approved. Financial actions change synthetic state and never charge. No payment-card fields are collected.', [field('name', 'Invoice'), field('period', 'Period'), field('status', 'State'), field('amount', 'Demo amount')], [
      { id: 'inv-1', name: 'DEMO-1001', period: 'November 2026', status: 'paid (synthetic)', amount: 'USD 0.00' },
      { id: 'inv-2', name: 'DEMO-1000', period: 'October 2026', status: 'paid (synthetic)', amount: 'USD 0.00' },
    ], { readonly: true }),
    support: collection('Support', 'Structured synthetic diagnostics only. Never paste configuration, environment dumps, keys or source credentials. No attachments or external delivery.', [field('name', 'Subject'), field('category', 'Category', 'select', ['Enrollment', 'Usage', 'Access', 'Billing', 'Other']), field('priority', 'Priority', 'select', ['Normal', 'High']), field('status', 'Status', 'select', ['open', 'in review', 'resolved']), field('summary', 'Safe diagnostic summary', 'textarea')], [
      { id: 'ticket-1', name: 'Partial enrollment', category: 'Enrollment', priority: 'Normal', status: 'open', summary: 'Synthetic profile has one association pending. No key bytes included.' },
    ]),
    members: collection('People & roles', 'Each role is bound to its own scope and conditions. Group membership does not itself grant key access.', [field('name', 'Principal'), field('kind', 'Identity kind', 'select', ['Human', 'Service principal', 'AI agent']), field('role', 'Role', 'select', ['Account owner', 'Enterprise administrator', 'Member', 'Auditor']), field('scope', 'Assigned scope'), field('status', 'Status', 'select', state)], [
      { id: 'member-1', name: 'Alex Morgan', kind: 'Human', role: 'Account owner', scope: 'Harbour Studio / development', status: 'active' },
      { id: 'member-2', name: 'Build agent', kind: 'AI agent', role: 'Member', scope: 'Harbour Studio / staging / status-only', status: 'active' },
    ]),
    groups: collection('Groups', 'Account association metadata only. Configuration and direct provider membership editing belong on the local end-user client.', [field('name', 'Group'), field('scope', 'Organization / environment'), field('members', 'Principal IDs'), field('status', 'Status', 'select', state)], [
      { id: 'group-1', name: 'Developers', scope: 'Harbour Studio / development', members: 'member-1', status: 'active' },
    ]),
    customers: collection('Customers', 'Assigned synthetic customers only. Suspension affects future service access; offboarding retains required financial/audit records.', [field('name', 'Customer'), field('plan', 'Demo entitlement', 'select', ['Free demo', 'Team demo', 'Enterprise demo']), field('region', 'Approximate service region', 'select', ['Australia', 'Europe', 'North America']), field('status', 'Status', 'select', state)], [
      { id: 'customer-1', name: 'Harbour Studio', plan: 'Team demo', region: 'Australia', status: 'active' },
      { id: 'customer-2', name: 'Northstar Labs', plan: 'Free demo', region: 'Europe', status: 'active' },
    ]),
    associations: collection('Key associations', 'Public metadata, version and provisioning status only. No key values can be displayed, uploaded or exported here.', [field('name', 'Association'), field('profile', 'Profile ID'), field('tenant', 'Customer ID'), field('kind', 'Service', 'select', ['KMS', 'age']), field('version', 'Key version'), field('status', 'Status', 'select', ['ready', 'pending', 'revoked'])], [
      { id: 'assoc-1', name: 'Studio wrapping', profile: 'profile-studio', tenant: 'customer-1', kind: 'KMS', version: 'v1', status: 'ready' },
      { id: 'assoc-2', name: 'Studio age association', profile: 'profile-studio', tenant: 'customer-1', kind: 'age', version: 'v1', status: 'ready' },
    ]),
    grants: collection('Access grants & windows', 'Exact tenant/profile/action/version windows. Future release can be revoked; copied plaintext or private keys cannot be recalled.', [field('name', 'Grant'), field('principal', 'Principal ID'), field('tenant', 'Customer ID'), field('profile', 'Profile ID'), field('action', 'Allowed operation', 'select', ['age retrieve', 'KMS wrap', 'KMS unwrap', 'status read']), field('version', 'Key version'), field('starts', 'Starts', 'datetime-local'), field('ends', 'Ends', 'datetime-local'), field('reason', 'Approval reason'), field('status', 'Status', 'select', state)], [
      { id: 'grant-1', name: 'Development access', principal: 'member-1', tenant: 'customer-1', profile: 'profile-studio', action: 'status read', version: 'v1', starts: '2026-11-04T09:00', ends: '2026-11-04T18:00', reason: 'Synthetic approved working window', status: 'active' },
    ]),
    roles: collection('Scoped role assignments', 'Synthetic role + scope tuples, never permission/scope cross-products. Exactly one protected global application administrator; no routine key read or financial bypass.', [field('name', 'Assignment'), field('principal', 'Principal ID'), field('role', 'Role', 'select', ['Support specialist', 'Security auditor', 'Service operator', 'Billing specialist', 'Product owner', 'AI agent', 'Recovery custodian']), field('scope', 'Exact customer / resource scope'), field('ends', 'Valid until', 'datetime-local'), field('reason', 'Approval reason'), field('status', 'Status', 'select', state)], [
      { id: 'role-1', name: 'Assigned support', principal: 'support-avery', role: 'Support specialist', scope: 'customer-1 / ticket-1 / read', ends: '2026-11-05T18:00', reason: 'Customer support request', status: 'active' },
    ]),
    operations: collection('Service operations', 'Synthetic maintenance intents and reconciliation, separate from production or custody recovery. No arbitrary SQL or command execution.', [field('name', 'Operation'), field('kind', 'Safe operation', 'select', ['Reconcile enrollment', 'Inspect health', 'Review retention', 'Propose key rotation', 'Plan stronger-custody migration']), field('target', 'Exact target'), field('reason', 'Reason'), field('status', 'State', 'select', ['proposed', 'awaiting approval', 'simulated complete', 'uncertain'])], [
      { id: 'op-1', name: 'Reconcile release profile', kind: 'Reconcile enrollment', target: 'profile-release', reason: 'One association pending', status: 'proposed' },
    ]),
    audit: collection('Audit trail', 'Bounded synthetic metadata. Demo telemetry is not a durable immutable audit service. No configuration or secret-bearing payloads.', [field('name', 'Event'), field('actor', 'Actor'), field('target', 'Target'), field('outcome', 'Outcome'), field('time', 'Time')], [
      { id: 'audit-1', name: 'Profile enrolled', actor: 'Account owner', target: 'profile-studio', outcome: 'synthetic success', time: '2026-11-04 09:00' },
    ], { readonly: true }),
    telemetry: collection('Service health', 'Synthetic service signals only. Approximate region indicates processing location, not a person’s physical location.', [field('name', 'Service'), field('region', 'Processing region'), field('latency', 'Latency ms', 'number'), field('status', 'Health')], [
      { id: 'service-1', name: 'Age release', region: 'Australia', latency: 42, status: 'healthy' },
      { id: 'service-2', name: 'Software crypto', region: 'Australia', latency: 18, status: 'healthy' },
      { id: 'service-3', name: 'Required audit', region: 'Australia', latency: 29, status: 'healthy' },
    ], { readonly: true }),
  },
};
