import test from 'node:test';
import assert from 'node:assert/strict';
import { initialiseBrowserServices, normaliseBrowserServiceConfig } from './browser-services.js';

const qualifiedSentry = {
  mode: 'qualified',
  services: {
    sentry: {
      enabled: true,
      configured: true,
      state: 'independently verified',
      qualifiedSurfaces: ['public-landing'],
      consentPurpose: 'technical-diagnostics',
      config: {
        origin: 'https://circularquay.aldous.info',
        projectId: 'cq-browser',
        release: 'candidate-1',
        rawError: 'must be dropped',
      },
    },
  },
};

test('preview emits no events before consent', () => {
  const events = [];
  const api = initialiseBrowserServices({ surface: 'public-landing' });
  assert.equal(api.registerAdapter('sentry', { start: event => events.push(event) }), false);
  assert.equal(api.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), false);
  assert.deepEqual(events, []);
});

test('unqualified or unsuitable surfaces fail closed', () => {
  const events = [];
  const api = initialiseBrowserServices({ surface: 'app-account', config: qualifiedSentry });
  assert.equal(api.registerAdapter('sentry', { start: event => events.push(event) }), false);
  assert.equal(api.setConsent({ purpose: 'technical-diagnostics', granted: true }), true);
  assert.equal(api.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), false);
  assert.deepEqual(events, []);
});

test('configured state alone is not a verified service capability', () => {
  const events = [];
  const config = structuredClone(qualifiedSentry);
  config.services.sentry.state = 'configured';
  const api = initialiseBrowserServices({ surface: 'public-landing', config });
  api.setConsent({ purpose: 'technical-diagnostics', granted: true });
  assert.equal(api.registerAdapter('sentry', { start: event => events.push(event) }), false);
  assert.deepEqual(events, []);
});

test('unknown properties and error text are dropped from adapter payloads', () => {
  const config = normaliseBrowserServiceConfig(qualifiedSentry);
  assert.deepEqual(Object.keys(config.services.sentry.config).sort((left, right) => left.localeCompare(right, 'en')), ['origin', 'projectId', 'release']);
  const events = [];
  const api = initialiseBrowserServices({ surface: 'public-landing', config: qualifiedSentry });
  assert.equal(api.registerAdapter('sentry', { start: event => events.push(event), event: event => events.push(event) }), false);
  assert.equal(api.setConsent({ purpose: 'technical-diagnostics', granted: true }), true);
  assert.equal(api.registerAdapter('sentry', { start: event => events.push(event), event: event => events.push(event) }), true);
  assert.equal(api.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised', rawError: 'secret', url: 'https://private.example' }), true);
  assert.deepEqual(events, [
    { type: 'cq-browser-service', code: 'service_initialised' },
    { type: 'cq-browser-service', code: 'service_initialised' },
  ]);
});

test('consent withdrawal stops future events and disposal is idempotent', () => {
  const events = [];
  const api = initialiseBrowserServices({ surface: 'public-landing', config: qualifiedSentry });
  api.setConsent({ purpose: 'technical-diagnostics', granted: true });
  api.registerAdapter('sentry', {
    start: event => events.push(event.code),
    stop: event => events.push(event.code),
    event: event => events.push(event.code),
  });
  assert.equal(api.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), true);
  api.setConsent({ purpose: 'technical-diagnostics', granted: false });
  assert.equal(api.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), false);
  api.dispose(); api.dispose();
  assert.deepEqual(events, ['service_initialised', 'service_initialised', 'consent_withdrawn']);
});
