import {
  DEFAULT_SERVICE_CONFIG,
  DIAGNOSTIC_CODES,
  QUALIFIED_ORIGINS,
  SERVICE_CATALOG,
  SERVICE_STATES,
  SURFACES,
} from './service-config.js';

const serviceById = new Map(SERVICE_CATALOG.map(service => [service.id, service]));
const allowedTopLevelConfig = new Set(['mode', 'services']);
const allowedServiceConfig = new Set(['enabled', 'state', 'configured', 'qualifiedSurfaces', 'consentPurpose', 'config']);
const identifier = /^[A-Za-z0-9][A-Za-z0-9_.:-]{1,127}$/;

function copyKnown(value, allowed) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  return Object.fromEntries(Object.entries(value).filter(([key]) => allowed.has(key)));
}

function isQualifiedOrigin(origin) {
  return typeof origin === 'string' && QUALIFIED_ORIGINS.includes(origin);
}

function validPublicConfig(value) {
  const config = copyKnown(value, new Set(['origin', 'projectId', 'release']));
  if (!isQualifiedOrigin(config.origin)) return null;
  if (!identifier.test(config.projectId || '') || !identifier.test(config.release || '')) return null;
  return Object.freeze(config);
}

function normaliseService(base, candidate) {
  const supplied = copyKnown(candidate, allowedServiceConfig);
  const qualifiedSurfaces = Array.isArray(supplied.qualifiedSurfaces)
    ? supplied.qualifiedSurfaces.filter(surface => base.allowedSurfaces.includes(surface))
    : [];
  const state = SERVICE_STATES.includes(supplied.state) ? supplied.state : base.state;
  const config = validPublicConfig(supplied.config);
  return Object.freeze({
    ...base,
    state,
    enabled: supplied.enabled === true,
    configured: supplied.configured === true && config !== null,
    qualifiedSurfaces: Object.freeze(qualifiedSurfaces),
    consentPurpose: typeof supplied.consentPurpose === 'string' && identifier.test(supplied.consentPurpose)
      ? supplied.consentPurpose
      : null,
    config,
  });
}

export function normaliseBrowserServiceConfig(candidate = DEFAULT_SERVICE_CONFIG) {
  const top = copyKnown(candidate, allowedTopLevelConfig);
  const suppliedServices = top.services && typeof top.services === 'object' ? top.services : {};
  const services = Object.fromEntries(SERVICE_CATALOG.map(base => [base.id, normaliseService(base, suppliedServices[base.id])]));
  return Object.freeze({
    mode: top.mode === 'qualified' ? 'qualified' : 'preview',
    services: Object.freeze(services),
  });
}

function eventPayload(code) {
  return Object.freeze({ type: 'cq-browser-service', code });
}

/**
 * initialiseBrowserServices is a fail-closed, vendor-neutral hook. It never
 * loads a script or makes a network request. A later owner can register a
 * reviewed adapter only after actual public configuration and qualification.
 */
export function initialiseBrowserServices({ surface, config = DEFAULT_SERVICE_CONFIG } = {}) {
  const safeConfig = normaliseBrowserServiceConfig(config);
  const adapters = new Map();
  const consent = new Map();
  const active = new Set();
  let disposed = false;

  function canRun(service) {
    return !disposed
      && safeConfig.mode === 'qualified'
      && service.enabled
      && service.configured
      && service.config !== null
      && service.state === 'independently verified'
      && service.qualifiedSurfaces.includes(surface)
      && service.consentPurpose !== null
      && consent.get(service.consentPurpose) === true;
  }

  function safely(callback, payload) {
    try { callback(payload); } catch { /* Vendor errors never export raw text. */ }
  }

  function stop(serviceId, code) {
    if (!active.delete(serviceId)) return;
    const adapter = adapters.get(serviceId);
    if (adapter?.stop) safely(adapter.stop, eventPayload(code));
  }

  function reconcile(serviceId) {
    const service = safeConfig.services[serviceId];
    const adapter = adapters.get(serviceId);
    if (!service || !adapter) return false;
    if (!canRun(service)) {
      stop(serviceId, 'consent_withdrawn');
      return false;
    }
    if (!active.has(serviceId)) {
      active.add(serviceId);
      if (adapter.start) safely(adapter.start, eventPayload('service_initialised'));
    }
    return true;
  }

  const api = {
    registerAdapter(serviceId, adapter = {}) {
      if (disposed || !serviceById.has(serviceId) || !adapter || typeof adapter !== 'object') return false;
      const safeAdapter = Object.freeze({
        start: typeof adapter.start === 'function' ? adapter.start : null,
        stop: typeof adapter.stop === 'function' ? adapter.stop : null,
        event: typeof adapter.event === 'function' ? adapter.event : null,
      });
      adapters.set(serviceId, safeAdapter);
      return reconcile(serviceId);
    },
    setConsent({ purpose, granted } = {}) {
      if (disposed || !identifier.test(purpose || '') || typeof granted !== 'boolean') return false;
      consent.set(purpose, granted);
      for (const [serviceId, service] of Object.entries(safeConfig.services)) {
        if (service.consentPurpose === purpose) reconcile(serviceId);
      }
      return true;
    },
    emitDiagnostic({ serviceId, code } = {}) {
      if (disposed || !DIAGNOSTIC_CODES.includes(code)) return false;
      const service = safeConfig.services[serviceId];
      const adapter = adapters.get(serviceId);
      if (!service || !adapter?.event || !active.has(serviceId) || !canRun(service)) return false;
      safely(adapter.event, eventPayload(code));
      return true;
    },
    status() {
      return Object.freeze({
        surface: Object.values(SURFACES).includes(surface) ? surface : null,
        mode: safeConfig.mode,
        preview: safeConfig.mode !== 'qualified',
        active: Object.freeze([...active]),
        services: safeConfig.services,
      });
    },
    dispose() {
      if (disposed) return;
      for (const serviceId of [...active]) stop(serviceId, 'service_disposed');
      disposed = true;
      adapters.clear();
      consent.clear();
    },
  };

  return Object.freeze(api);
}
