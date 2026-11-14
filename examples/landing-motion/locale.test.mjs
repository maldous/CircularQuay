import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import vm from 'node:vm';

const source = await readFile(new URL('./locale.js', import.meta.url), 'utf8');
const landing = await readFile(new URL('./index.html', import.meta.url), 'utf8');
const messageKeys = [...new Set([...landing.matchAll(/data-i18n="([^"]+)"/g)].map(match => match[1]))];
const storageKey = 'circularquay-motion-language';

function fixture({ saved = null, blockedStorage = false, loading = false } = {}) {
  const messages = messageKeys.map(key => ({ dataset: { i18n: key }, textContent: `untranslated:${key}` }));
  const listeners = new Map(), ready = [], writes = [], events = [], timers = [];
  const select = {
    value: '', disabled: true, attributes: {},
    setAttribute(name, value) { this.attributes[name] = value; },
    addEventListener(name, callback) { listeners.set(name, callback); },
  };
  const flag = { src: '' }, status = { hidden: true, textContent: '' };
  const login = { disabled: true, addEventListener(name, callback) { listeners.set(`login:${name}`, callback); } };
  const nodes = new Map([['#language', select], ['.language-flag', flag], ['#login-status', status], ['#login', login]]);
  const document = {
    documentElement: {}, readyState: loading ? 'loading' : 'complete',
    querySelectorAll(selector) { assert.equal(selector, '[data-i18n]'); return messages; },
    querySelector(selector) { return nodes.get(selector); },
    addEventListener(name, callback, options) { ready.push({ name, callback, options }); },
  };
  vm.runInNewContext(source, {
    document,
    localStorage: {
      getItem(key) { assert.equal(key, storageKey); if (blockedStorage) throw new Error('unavailable'); return saved; },
      setItem(key, value) { if (blockedStorage) throw new Error('unavailable'); writes.push({ key, value }); },
    },
    window: { dispatchEvent(event) { events.push({ type: event.type, locale: event.detail.locale }); } },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } },
    setTimeout(callback, delay) { timers.push({ callback, delay }); return timers.length; },
    clearTimeout() {},
  });
  return { document, select, flag, status, login, messages, ready, writes, events, timers, listeners };
}

test('English startup translates landing messages without overwriting a saved preference', () => {
  const view = fixture();
  assert.equal(view.document.documentElement.lang, 'en');
  assert.equal(view.document.documentElement.dir, 'ltr');
  assert.equal(view.select.value, 'en');
  assert.equal(view.select.disabled, false);
  assert.equal(view.select.attributes['aria-label'], 'Language');
  assert.equal(view.flag.src, 'assets/flag-en.svg');
  assert.ok(view.messages.length > 0);
  assert.ok(view.messages.every(node => !node.textContent.startsWith('untranslated:')));
  assert.deepEqual(view.writes, []);
  assert.deepEqual(view.events, [{ type: 'localechange', locale: 'en' }]);
});

test('Arabic preference and subsequent changes preserve direction, controls and login feedback', () => {
  const view = fixture({ saved: 'ar' });
  assert.equal(view.document.documentElement.lang, 'ar');
  assert.equal(view.document.documentElement.dir, 'rtl');
  assert.equal(view.flag.src, 'assets/flag-ar.svg');
  assert.equal(view.select.attributes['aria-label'], 'اللغة');
  assert.ok(view.messages.every(node => !node.textContent.startsWith('untranslated:')));
  view.select.value = 'unsupported';
  view.listeners.get('change')({ currentTarget: view.select });
  assert.equal(view.document.documentElement.lang, 'en');
  assert.equal(view.document.documentElement.dir, 'ltr');
  assert.deepEqual(view.writes, [{ key: storageKey, value: 'en' }]);
  view.listeners.get('login:click')();
  assert.equal(view.status.hidden, false);
  assert.match(view.status.textContent, /isn’t connected/);
  assert.equal(view.timers.at(-1).delay, 8000);
  view.timers.at(-1).callback();
  assert.equal(view.status.hidden, true);
});

test('language switching remains usable when browser storage is unavailable', () => {
  const view = fixture({ blockedStorage: true });
  view.select.value = 'ar';
  view.listeners.get('change')({ currentTarget: view.select });
  assert.equal(view.document.documentElement.lang, 'ar');
  assert.equal(view.document.documentElement.dir, 'rtl');
  assert.equal(view.select.value, 'ar');
  assert.deepEqual(view.events.at(-1), { type: 'localechange', locale: 'ar' });
});

test('initialization waits for the DOM and registers a one-time ready listener', () => {
  const view = fixture({ saved: 'ar', loading: true });
  assert.equal(view.select.disabled, true);
  assert.equal(view.ready.length, 1);
  assert.equal(view.ready[0].name, 'DOMContentLoaded');
  assert.equal(view.ready[0].options.once, true);
  view.ready[0].callback();
  assert.equal(view.select.disabled, false);
  assert.equal(view.document.documentElement.lang, 'ar');
});
