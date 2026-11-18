import test from 'node:test';
import assert from 'node:assert/strict';
import { browser } from './coverage-dom-fixture.test.mjs';
import { createPresentationPreferences, browserPresentation, resolvePresentation, validatePresentation } from './presentation-preferences.js';

let sequence = 0;
const load = name => import(new URL(`${name}?owned-fixture=${++sequence}`, import.meta.url));

test('the actual bootstrap URL, not anonymous evaluated code, preserves every closed preference/fallback path', async t => {
  const x = browser(t);
  for (const language of [undefined, 'en', 'ar', 'system', 'bad']) for (const theme of [undefined, 'light', 'dark', 'system', 'bad']) {
    x.data.delete('cq.presentation.v1');
    if (language !== undefined || theme !== undefined) x.data.set('cq.presentation.v1', JSON.stringify({ version: 1, ...(language !== undefined ? { language } : {}), ...(theme !== undefined ? { theme } : {}) }));
    await load('presentation-bootstrap.js');
    const valid = ['en', 'ar', 'system', undefined].includes(language) && ['light', 'dark', 'system', undefined].includes(theme);
    assert.equal(x.document.documentElement.lang, valid && language === 'ar' ? 'ar' : 'en');
    assert.equal(x.document.documentElement.dataset.theme, valid && theme !== undefined ? theme : 'system');
  }
  for (const raw of ['{bad', 'null', '[]', '{"version":2}', '{"version":1,"unexpected":true}', ' '.repeat(257)]) { x.data.set('cq.presentation.v1', raw); await load('presentation-bootstrap.js'); assert.equal(x.document.documentElement.dataset.theme, 'system'); }
  x.setStorageBlocked(true); await load('presentation-bootstrap.js'); assert.equal(x.document.documentElement.lang, 'en'); assert.equal(x.writes.length, 0);
});

test('all closed preference API branches reject invalid fields while optional storage stays page-local', () => {
  assert.equal(browserPresentation([null, 7, 'x'.repeat(101), 'ar-EG']).language, 'ar');
  const p = createPresentationPreferences(); assert.equal(p.snapshot().persistence, 'memory');
  p.set('theme', 'dark'); assert.equal(p.snapshot().theme, 'dark'); p.clear('theme');
  assert.throws(() => p.clear('private-profile')); assert.throws(() => p.setProfile({ version: 1, language: null }));
  p.setProfile({ version: 1, theme: 'light' }); p.setProfile(); assert.equal(p.snapshot().theme, 'system');
  for (const value of [null, [], { version: 1, language: 'bad' }, { version: 1, theme: 'bad' }]) assert.throws(() => validatePresentation(value));
  assert.equal(resolvePresentation({ profile: { version: 1, theme: 'dark' }, device: { version: 1, language: 'en' } }).themeSource, 'profile');
});

test('real presentation DOM updates every language/theme/source label while retaining ongoing form values', async t => {
  const x = browser(t, { html: '<form><input id="draft" value="KEEP_OWNED_DRAFT"><select data-presentation-setting="language"><option value=""></option><option value="en"></option><option value="ar"></option><option value="system"></option></select><select data-presentation-setting="theme"><option value=""></option><option value="dark"></option><option value="light"></option><option value="system"></option></select></form>' });
  const module = await load('presentation.js'); const controller = module.installPresentation(); assert.equal(module.installPresentation(), controller);
  const panel = x.document.querySelector('.presentation-panel'); assert.equal(panel.open, true);
  for (const [field, choices] of [['language', ['ar', 'system', 'en', '']], ['theme', ['dark', 'light', 'system', '']]]) for (const value of choices) {
    const input = x.document.querySelector(`[data-presentation-field="${field}"]`); input.value = value; x.event(input, 'change', { bubbles: true });
    assert.equal(x.document.querySelector('#draft').value, 'KEEP_OWNED_DRAFT'); assert.ok(module.presentationOptions(field).includes('selected'));
    assert.equal(x.document.querySelector(`[data-presentation-setting="${field}"]`).value, value);
  }
  controller.setProfile({ version: 1, language: 'ar', theme: 'dark' }); assert.equal(x.document.documentElement.dir, 'rtl');
  controller.set('language', 'system'); controller.setBrowserLanguages([]); assert.match(x.document.querySelector('[data-summary-language]').textContent, /fallback/);
  x.media.matches = true; x.event(x.media, 'change'); assert.ok(x.document.querySelector('[data-summary-theme]').textContent);
  x.location.hash = '#/settings'; x.event(x.window, 'hashchange'); assert.equal(panel.open, false);
  x.location.hash = '#/landing'; x.event(x.window, 'hashchange'); assert.equal(panel.open, true);
  x.window.navigator.languages = ['ar']; x.event(x.window, 'languagechange'); assert.equal(x.document.documentElement.lang, 'ar');
  x.setStorageBlocked(true); controller.set('theme', 'dark'); assert.match(x.document.querySelector('.presentation-message').textContent, /الحفظ|saving/);
  x.setStorageBlocked(false); controller.clear('theme'); assert.equal(controller.snapshot().persistence, 'available');
  x.event(x.document.querySelector('.presentation-controls'), 'change', { target: { dataset: {}, value: '' } });
});

test('blocked browser storage access and absent optional media API still apply the presentation without transport', async t => {
  const x = browser(t); Object.defineProperty(x.window, 'localStorage', { configurable: true, get() { throw Error('synthetic blocked access'); } }); delete x.window.matchMedia;
  const module = await load('presentation.js'); module.installPresentation(); assert.equal(module.presentationPreferences().snapshot().persistence, 'memory'); assert.equal(module.applyPresentation().language, 'en');
  assert.equal(x.sources.length, 0); assert.equal(x.document.querySelector('.presentation-panel').lang, 'en');
});

test('actual presentation entry installs the independent preference boundary without app or help startup', async t => {
  const x = browser(t); await load('presentation-entry.js'); assert.ok(x.document.querySelector('.presentation-panel')); assert.equal(x.document.querySelector('#help-launcher'), null); assert.equal(x.sources.length, 0);
});
