import test from 'node:test';
import assert from 'node:assert/strict';
import { browser, publicHelpFetcher } from '../../web/coverage-dom-fixture.test.mjs';
import { normaliseBrowserServiceConfig, initialiseBrowserServices } from './browser-services.js';
import { SURFACES, QUALIFIED_ORIGINS } from './service-config.js';
let sequence = 0;
const load = name => import(new URL(`${name}?owned-browser=${++sequence}`, import.meta.url));

test('appearance and locale run actual event lifecycles without loading optional service providers', async t => {
  const x = browser(t, { html: '<button id="theme-toggle" disabled></button><select id="language"><option value="en"></option><option value="ar"></option></select><img class="language-flag"><span data-i18n="title1"></span><span data-i18n="unknown">Unchanged</span><button id="login"></button><p id="login-status" hidden></p>', stored: { 'circularquay-motion-language': 'ar', 'circularquay-motion-theme': 'system' } });
  x.media.matches = true; await load('preference-bootstrap.js'); assert.equal(x.document.documentElement.lang, 'ar'); assert.equal(x.document.documentElement.dir, 'rtl'); assert.equal(x.document.documentElement.dataset.layoutReady, 'false');
  await load('appearance.js'); const toggle = x.document.querySelector('#theme-toggle'); assert.equal(toggle.disabled, false); assert.match(toggle.getAttribute('aria-label'), /الفاتح/);
  toggle.click(); assert.equal(x.document.documentElement.dataset.theme, 'light'); assert.equal(x.data.get('circularquay-motion-theme'), 'light');
  x.setStorageBlocked(true); toggle.click(); assert.equal(x.document.documentElement.dataset.theme, 'dark'); x.setStorageBlocked(false);
  const reduced = x.mediaFor('(prefers-reduced-motion: reduce)'); reduced.matches = true; x.event(reduced, 'change'); assert.equal(x.document.documentElement.dataset.reducedMotion, 'true');
  x.document.documentElement.dataset.theme = 'system'; x.media.matches = false; x.event(x.media, 'change'); assert.match(toggle.getAttribute('aria-label'), /الداكن/);
  await load('locale.js'); const language = x.document.querySelector('#language'); assert.equal(language.value, 'ar');
  language.value = 'en'; x.event(language, 'change'); assert.equal(x.document.documentElement.lang, 'en'); assert.match(toggle.getAttribute('aria-label'), /dark/); assert.equal(x.document.querySelector('[data-i18n="unknown"]').textContent, 'Unchanged');
  const login = x.document.querySelector('#login'); login.click(); assert.equal(x.document.querySelector('#login-status').hidden, false); language.value = 'ar'; x.event(language, 'change'); assert.match(x.document.querySelector('#login-status').textContent, /تسجيل/); await x.advance(8000); assert.equal(x.document.querySelector('#login-status').hidden, true);
  x.setStorageBlocked(true); language.value = 'unsupported'; x.event(language, 'change'); assert.equal(x.document.documentElement.lang, 'en'); assert.equal(x.sources.length, 0);
});

test('blocked preferences and optional absent appearance controls remain safe while DOM-ready locale initializes once', async t => {
  const x = browser(t, { html: '<span data-i18n="title1"></span><button id="login"></button>', blockedStorage: true });
  await load('preference-bootstrap.js'); assert.equal(x.document.documentElement.dataset.theme, 'light'); assert.equal(x.document.documentElement.dir, 'ltr');
  await load('appearance.js'); x.event(x.media, 'change'); x.document.readyState = 'loading'; await load('locale.js'); x.event(x.document, 'DOMContentLoaded'); x.document.querySelector('#login').click(); assert.match(x.document.querySelector('[data-i18n]').textContent, /right/);
});

test('actual account preview enables owned controls but submits no identity or account request', async t => {
  const x = browser(t, { html: '<fieldset class="provider-grid" disabled><button class="provider-choice" data-provider="Fabricated">Provider</button></fieldset><button data-mode="create" data-i18n-login="createMode"></button><button data-mode="reset" data-i18n-login="resetMode"></button><form id="account-form"><fieldset disabled><input id="email"><button id="account-submit"></button></fieldset></form><p id="provider-status"></p><p id="account-status"></p>' });
  x.document.readyState = 'loading'; await load('login.js'); x.event(x.document, 'DOMContentLoaded'); assert.equal(x.document.querySelector('.provider-grid').disabled, false);
  x.document.querySelector('.provider-choice').click(); assert.match(x.document.querySelector('#provider-status').textContent, /Fabricated.*not connected/);
  const email = x.document.querySelector('#email'); email.valid = false; x.event(x.document.querySelector('#account-form'), 'submit'); assert.equal(email.validityReported, true); assert.equal(x.document.querySelector('#account-status').textContent, '');
  email.valid = true; x.event(x.document.querySelector('#account-form'), 'submit'); assert.match(x.document.querySelector('#account-status').textContent, /Account creation/);
  x.document.querySelector('[data-mode="reset"]').click(); x.event(x.document.querySelector('#account-form'), 'submit'); assert.match(x.document.querySelector('#account-status').textContent, /Password reset/);
  x.document.documentElement.lang = 'ar'; x.event(x.window, 'localechange'); assert.match(email.getAttribute('aria-label'), /البريد/); assert.equal(x.document.querySelector('#account-status').textContent, '');
  x.document.querySelector('[data-mode="create"]').click(); x.event(x.document.querySelector('#account-form'), 'submit'); assert.match(x.document.querySelector('#account-status').textContent, /إنشاء/); assert.equal(x.sources.length, 0); assert.equal(x.writes.length, 0);
});

test('service config closes malformed metadata and all consent lifecycle callbacks export only fixed codes', () => {
  for (const value of [null, [], false, { mode: 'qualified', services: null }, { services: [] }]) assert.equal(Object.isFrozen(normaliseBrowserServiceConfig(value)), true);
  const config = { mode: 'qualified', services: { sentry: { enabled: true, configured: true, state: 'independently verified', qualifiedSurfaces: [SURFACES.PUBLIC_LANDING, 'unknown'], consentPurpose: 'owned-purpose', config: { origin: QUALIFIED_ORIGINS[0], projectId: 'owned-project', release: 'owned-release', secret: 'discarded' } } } };
  const service = initialiseBrowserServices({ surface: SURFACES.PUBLIC_LANDING, config }); const events = [];
  assert.equal(service.registerAdapter('unknown'), false); assert.equal(service.registerAdapter('sentry', null), false); assert.equal(service.registerAdapter('sentry', { start: value => { events.push(value); throw Error('PRIVATE_ERROR'); }, event: value => { events.push(value); throw Error('PRIVATE_ERROR'); }, stop: value => { events.push(value); throw Error('PRIVATE_ERROR'); } }), false);
  assert.equal(service.setConsent({ purpose: 'invalid purpose', granted: true }), false); assert.equal(service.setConsent({ purpose: 'owned-purpose', granted: 'true' }), false);
  assert.equal(service.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), false); assert.equal(service.setConsent({ purpose: 'owned-purpose', granted: true }), true); assert.deepEqual(service.status().active, ['sentry']);
  assert.equal(service.emitDiagnostic({ serviceId: 'unknown', code: 'service_initialised' }), false); assert.equal(service.emitDiagnostic({ serviceId: 'sentry', code: 'PRIVATE_ERROR' }), false); assert.equal(service.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), true);
  service.setConsent({ purpose: 'owned-purpose', granted: false }); assert.deepEqual(service.status().active, []); service.setConsent({ purpose: 'owned-purpose', granted: true }); service.dispose(); service.dispose();
  assert.ok(events.every(value => Object.keys(value).join(',') === 'type,code')); assert.ok(events.every(value => !JSON.stringify(value).includes('PRIVATE'))); assert.equal(service.registerAdapter('sentry'), false); assert.equal(service.setConsent({ purpose: 'owned-purpose', granted: true }), false);
  assert.equal(initialiseBrowserServices({ surface: 'unknown' }).status().surface, null);
  const noCallbacks = initialiseBrowserServices({ surface: SURFACES.PUBLIC_LANDING, config }); noCallbacks.registerAdapter('sentry'); noCallbacks.setConsent({ purpose: 'owned-purpose', granted: true }); assert.equal(noCallbacks.emitDiagnostic({ serviceId: 'sentry', code: 'service_initialised' }), false); noCallbacks.dispose();
  for (const changed of [{ config: null }, { config: { origin: 'https://unqualified.invalid', projectId: 'id', release: 'v1' } }, { config: { origin: QUALIFIED_ORIGINS[0], projectId: '', release: '' } }, { state: 'bad', qualifiedSurfaces: null, consentPurpose: '' }]) { const copy = structuredClone(config); Object.assign(copy.services.sentry, changed); const api = initialiseBrowserServices({ surface: SURFACES.PUBLIC_LANDING, config: copy }); api.registerAdapter('sentry'); api.setConsent({ purpose: 'owned-purpose', granted: true }); assert.deepEqual(api.status().active, []); }
});

test('actual site bootstrap and compact explore entry are page-local and dispose on pagehide', async t => {
  const x = browser(t, { html: '<div data-explore-menu></div>' }); await load('landing-explore.js'); assert.match(x.document.querySelector('[data-explore-menu]').innerHTML, /explore/);
  const publicSite = await load('site-services.js'); assert.equal(publicSite.browserServices.status().surface, SURFACES.PUBLIC_LANDING); x.event(x.window, 'pagehide'); assert.equal(publicSite.browserServices.setConsent({ purpose: 'owned-purpose', granted: true }), false);
  x.document.body.classList.add('login-page'); const authSite = await load('site-services.js'); assert.equal(authSite.browserServices.status().surface, SURFACES.AUTH_PAGE); x.document.querySelector('[data-explore-menu]').remove(); await load('landing-explore.js'); assert.equal(x.sources.length, 0);
});

const searchMarkup = '<form class="header-search"><input id="site-search"><button type="submit" disabled></button><div id="site-search-popup" hidden><ul id="site-search-results"></ul><p id="site-search-status"></p></div></form>';
async function suggestions(x) { await x.until(() => /suggestions available|No matching/.test(x.document.querySelector('#site-search-status').textContent)); }
test('real header search handles keyboard completion, navigation, debounce and outside dismissal without query transport', async t => {
  const x = browser(t, { html: searchMarkup }); const requests = []; globalThis.fetch = publicHelpFetcher(requests); await load('header-search.js'); const input = x.document.querySelector('#site-search'); const form = x.document.querySelector('form'); const popup = x.document.querySelector('#site-search-popup'); const list = x.document.querySelector('#site-search-results');
  input.value = 'TypeScript'; x.event(input, 'input'); await x.advance(100); await suggestions(x); assert.ok(list.children.length > 0); assert.equal(input.getAttribute('aria-expanded'), 'true');
  x.event(input, 'keydown', { key: 'ArrowUp' }); x.event(input, 'keydown', { key: 'ArrowDown' }); x.event(input, 'keydown', { key: 'Tab' }); assert.equal(popup.hidden, true); assert.ok(input.value); x.event(input, 'keydown', { key: 'Enter' }); assert.match(x.location.href, /^\/(help|explore)#/);
  input.value = 'policy'; x.event(input, 'input'); await x.advance(100); await suggestions(x); x.event(list.children[0], 'mousemove', { bubbles: true }); const down = x.event(list.children[0], 'pointerdown', { bubbles: true }); assert.equal(down.defaultPrevented, true); list.children[0].click(); assert.ok(x.location.href);
  x.event(input, 'keydown', { key: 'Escape' }); assert.equal(popup.hidden, true); x.event(input, 'keydown', { key: 'ArrowDown' }); await suggestions(x); x.event(form, 'submit'); assert.ok(x.location.href);
  x.document.body.focus(); x.event(form, 'focusout'); await x.advance(0); assert.equal(popup.hidden, true); input.value = ''; x.event(input, 'input'); assert.equal(x.document.querySelector('#site-search-status').textContent, '');
  input.value = 'OWNED_QUERY_NEVER_SENT'; x.event(form, 'submit'); await suggestions(x); assert.equal(list.children.length, 0); x.event(x.document.body, 'pointerdown', { bubbles: true }); assert.equal(popup.hidden, true); assert.ok(requests.every(row => !row.path.includes('OWNED_QUERY')));
});

test('header errors retry and settled searches close without changing unrelated controls', async t => {
  const x = browser(t, { html: searchMarkup }); let fail = true; const requests = [];
  globalThis.fetch = publicHelpFetcher(requests, { transform: bytes => fail ? Buffer.from('BAD') : bytes });
  await load('header-search.js'); const input = x.document.querySelector('#site-search'); input.value = 'policy'; x.event(input, 'keydown', { key: 'ArrowDown' }); await x.until(() => /unavailable/.test(x.document.querySelector('#site-search-status').textContent));
  fail = false; x.event(input, 'input'); await x.advance(100); await suggestions(x); x.event(input, 'keydown', { key: 'Escape' }); assert.equal(x.document.querySelector('#site-search-popup').hidden, true);
  x.document.body.innerHTML = ''; await load('header-search.js'); assert.equal(x.document.querySelector('#site-search'), null);
});

test('cold held header success and failure completions cannot reopen an escaped search', async t => {
  const x = browser(t, { html: searchMarkup });
  for (const fails of [false, true]) {
    x.document.body.innerHTML = searchMarkup;
    const requests = []; let release; let fetchCompleted = false; let digestsCompleted = 0;
    const gate = new Promise(resolve => { release = resolve; });
    const actualDigest = crypto.subtle.digest.bind(crypto.subtle);
    const digest = t.mock.method(crypto.subtle, 'digest', async (...args) => {
      const value = await actualDigest(...args); digestsCompleted++; return value;
    });
    globalThis.fetch = publicHelpFetcher(requests, { before: async () => {
      await gate; fetchCompleted = true; if (fails) throw Error('OWNED_FETCH_FAILURE');
    } });
    // A fresh module creates a fresh store; no warm manifest/index can bypass
    // this actual fetch gate. The old query is cancelled while still pending.
    await load('header-search.js'); const input = x.document.querySelector('#site-search'); input.value = 'policy'; x.event(input, 'keydown', { key: 'ArrowDown' });
    await x.until(() => requests.length === 1); assert.equal(fetchCompleted, false);
    x.event(input, 'keydown', { key: 'Escape' }); release();
    await x.until(() => fails ? fetchCompleted : digestsCompleted >= 3);
    assert.equal(x.document.querySelector('#site-search-popup').hidden, true);
    assert.equal(x.document.querySelector('#site-search-status').textContent, '');
    digest.mock.restore();
  }
});

test('actual landing help page replaces its old viewer on valid and invalid hash navigation', async t => {
  const x = browser(t, { html: '<main id="help-page"></main>', hash: '#policy' }); const requests = []; globalThis.fetch = publicHelpFetcher(requests);
  await import(`./help-page.js?owned-page=${++sequence}`); const root = x.document.querySelector('#help-page'); await x.until(() => !!root.querySelector('.help-body')); assert.equal(root.querySelector('.help-page-link'), null);
  x.location.hash = '#not/valid'; x.event(x.window, 'hashchange'); await x.until(() => /Install and first run/.test(root.querySelector('.help-article').textContent)); assert.equal(x.document.querySelectorAll('#help-query').length, 1); assert.ok(requests.every(row => !row.path.includes('not/valid')));
});

test('public suggestion helpers retain empty-part fallback and deduplicate bounded guide/document identities', async () => {
  const { documentationSuggestions, mergeSuggestions } = await import('./header-search.js');
  const suggestions = documentationSuggestions('Python', [{ id: 'sdk-python-readme', title: 'Python', category: '', path: 'sdk/python/README.md', parts: [] }]);
  assert.equal(suggestions[0].title, 'Python · Python'); assert.equal(suggestions[0].summary, 'Documentation'); assert.match(suggestions[0].href, /part=part-1/);
  assert.deepEqual(documentationSuggestions('anything', []), []);
  const guide = { id: 'owned', title: 'Owned', summary: '' }; const document = { id: 'owned-doc', title: 'Doc', label: 'Documentation', href: '/explore#doc=owned' };
  const merged = mergeSuggestions([guide, guide], [document, document]); assert.equal(merged.length, 2); assert.equal(merged[0].label, 'Guide'); assert.match(merged[0].href, /^\/help#/);
});
