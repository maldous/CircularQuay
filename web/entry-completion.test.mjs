import test from 'node:test';
import assert from 'node:assert/strict';
import { browser } from './coverage-dom-fixture.test.mjs';

test('actual local and remote entry URLs bind their declared model/tool closures without native or provider startup', async t => {
  const x = browser(t, { hash: '#/workbench' }); await import('./local/entry.js?owned-entry=local'); assert.ok(x.document.querySelector('#wb-grid')); assert.equal(x.sources.length, 1); x.event(x.window, 'pagehide'); assert.equal(x.sources[0].closed, true);
  x.location.hash = '#/account/overview'; await import('./remote/entry.js?owned-entry=remote'); assert.match(x.document.querySelector('#app').textContent, /Account|overview/i); assert.equal(x.sources.length, 2); assert.equal(x.sources[1].path, '/__reload'); x.event(x.window, 'pagehide'); assert.equal(x.sources[1].closed, true);
});
