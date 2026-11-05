/**
 * Browser-service inventory for the landing prototype. This is deliberately a
 * public, credential-free manifest: an entry being present never means that a
 * vendor is selected, configured, connected, deployed, or verified.
 */
export const SERVICE_STATES = Object.freeze([
  'pending applicability',
  'not applicable with rationale',
  'blocked with prerequisite',
  'configured',
  'deployed',
  'independently verified',
]);

export const SURFACES = Object.freeze({
  PUBLIC_LANDING: 'public-landing',
  AUTH_PAGE: 'auth-page',
  APP_ACCOUNT: 'app-account',
});

export const QUALIFIED_ORIGINS = Object.freeze([
  'https://circularquay.aldous.info',
  'https://cq.aldous.info',
]);

const pending = (id, label, allowedSurfaces, purpose, missing) => Object.freeze({
  id,
  label,
  allowedSurfaces: Object.freeze(allowedSurfaces),
  purpose,
  state: 'pending applicability',
  enabled: false,
  configured: false,
  qualifiedSurfaces: Object.freeze([]),
  consentPurpose: null,
  missing: Object.freeze(missing),
});

export const SERVICE_CATALOG = Object.freeze([
  pending('sentry', 'Sentry technical diagnostics', [SURFACES.PUBLIC_LANDING], 'coarse code-only diagnostics', ['CQ project', 'public routing reference', 'release identity', 'sampling and retention review', 'surface qualification', 'consent decision']),
  pending('iubenda', 'iubenda privacy and consent', [SURFACES.PUBLIC_LANDING, SURFACES.AUTH_PAGE], 'consent and withdrawal bridge', ['CQ site registration', 'licensed policy languages', 'purpose/vendor configuration', 'privacy-owner approval']),
  pending('adsense', 'Google AdSense advertising', [SURFACES.PUBLIC_LANDING], 'advertising', ['surface applicability', 'CQ site approval', 'slot configuration', 'consent/TCF qualification']),
  pending('google-ads', 'Google Ads / AdWords', [SURFACES.PUBLIC_LANDING], 'advertising or conversion measurement', ['campaign/conversion applicability', 'CQ account and site approval', 'consent qualification', 'public conversion configuration']),
  pending('zaraz', 'Cloudflare Zaraz optional analytics', [SURFACES.PUBLIC_LANDING], 'coarse optional analytics', ['CQ zone/tool selection', 'purpose allowlist', 'consent configuration', 'surface qualification']),
  pending('support', 'Support route', [SURFACES.PUBLIC_LANDING, SURFACES.AUTH_PAGE], 'user support', ['approved native support route', 'owner', 'privacy review']),
  pending('transactional-email-brevo', 'Brevo transactional email (conditional, not selected)', [], 'transactional email', ['approved support route', 'explicit provider selection', 'verified sender/domain', 'synthetic-recipient qualification']),
  pending('feedback', 'Feedback provider', [SURFACES.PUBLIC_LANDING, SURFACES.AUTH_PAGE], 'user feedback', ['explicit provider selection', 'consent/privacy review', 'surface qualification']),
  pending('cloudflare-hosting', 'Cloudflare hosting', [], 'hosting', ['verified CQ project', 'environment bindings', 'capacity and restore qualification']),
  pending('netlify-hosting', 'Netlify hosting', [], 'hosting', ['verified CQ project', 'environment bindings', 'capacity and restore qualification']),
  pending('cloudflare-identity', 'Cloudflare application identity', [], 'application identity', ['verified issuer/app registration', 'redirect origins', 'role qualification']),
  pending('netlify-identity', 'Netlify Identity', [], 'application identity', ['verified site', 'registration policy', 'provider/callback origins']),
  pending('database-storage', 'Database and storage custody', [], 'application data custody', ['selected platform', 'bindings', 'tenant isolation', 'restore qualification']),
  pending('status-monitoring', 'Status and monitoring', [SURFACES.PUBLIC_LANDING], 'public health projection', ['selected provider', 'ownership verification', 'safe alert/recovery exercise']),
  pending('discovery', 'Search/discovery property verification', [SURFACES.PUBLIC_LANDING], 'site discovery', ['selected property', 'ownership verification', 'public metadata review']),
  pending('engineering', 'Engineering services', [], 'CI and quality gates', ['selected service owner', 'protected CI identity', 'allowance and qualification']),
  pending('pwa', 'Browser/PWA delivery', [SURFACES.PUBLIC_LANDING], 'platform delivery', ['shipped-platform decision', 'offline/cache/accessibility qualification']),
  pending('android', 'Android delivery', [], 'native platform delivery', ['shipped-package decision', 'permission/policy/accessibility qualification']),
]);

export const DEFAULT_SERVICE_CONFIG = Object.freeze({
  mode: 'preview',
  services: Object.freeze(Object.fromEntries(SERVICE_CATALOG.map(service => [service.id, service]))),
});

export const DIAGNOSTIC_CODES = Object.freeze([
  'service_initialised',
  'service_disposed',
  'consent_withdrawn',
]);
