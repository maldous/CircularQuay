import test from 'node:test';
import assert from 'node:assert/strict';
import { browser } from '../../web/coverage-dom-fixture.test.mjs';
import { documentation } from './explore-manifest.js';
import { navigationGroups } from './explore-navigation.js';
import { createGestureController, isAtDocumentTop, isIgnoredGestureTarget, isUpwardWheel } from './landing-navigation.js';
import { createScenarioSession, storyReservationSessions, createOptionalSchedule, createVariantRotation, sessionStep, STEP_DURATION } from './scenario-flow.js';

let sequence = 0;
const markup = '<main id="main"><div id="article"><div id="document-content"></div><p id="document-status"></p></div><div id="document-list"></div><div id="contents-list"></div><a id="topics-back"></a><a id="mobile-topic-link"></a><p id="rail-revision"></p><p id="revision-stamp"></p><a id="repository-link"></a></main>';
test('actual documentation reader traverses complete snapshot, recovery, groups, cache and anchors with no remote fetch', async t => {
  const x = browser(t, { html: markup, hash: '' }); await import(`./explore.js?owned-reader=${++sequence}`);
  const root = x.document.querySelector('#main'), content = x.document.querySelector('#document-content'); assert.equal(root.dataset.view, 'home'); assert.match(content.textContent, /Explore CircularQuay/);
  for (const group of navigationGroups) { x.location.hash = '#group=' + encodeURIComponent(group.id); x.event(x.window, 'hashchange'); assert.equal(root.dataset.view, 'group'); assert.ok(x.document.querySelector('#document-list').children.length); }
  for (const item of documentation.documents) for (const part of item.parts) {
    x.location.hash = '#doc=' + encodeURIComponent(item.id) + '&part=' + encodeURIComponent(part.id) + '&anchor=' + encodeURIComponent(part.headings[0]?.id || 'not-a-heading'); x.event(x.window, 'hashchange');
    await x.until(() => x.document.querySelector('#document-status').hidden); assert.equal(root.dataset.view, 'document'); assert.ok(content.querySelector('h1')); assert.match(content.querySelector('.document-provenance a').href, /github\.com/); await x.frame(1);
  }
  x.location.hash = '#doc=not-a-document&part=not-a-part'; x.event(x.window, 'hashchange'); await x.until(() => !!content.querySelector('.document-recovery')); assert.match(content.textContent, /nearest available/);
  x.location.hash = '#doc=' + documentation.documents[0].id; x.event(x.window, 'hashchange'); x.location.hash = '#group=unknown'; x.event(x.window, 'hashchange'); await x.drain(); assert.equal(root.dataset.view, 'home'); assert.equal(x.sources.length, 0);
});

test('actual gesture module wires document events, keeps one navigation and clears interrupted/expired gestures', async t => {
  const x = browser(t); const paths = []; x.window.location.assign = path => paths.push(path); await import(`./landing-navigation.js?owned-events=${++sequence}`); const target = x.document.querySelector('#app');
  x.event(x.document, 'touchstart', { target, touches: [{ clientX: 10, clientY: 100 }] }); x.event(x.document, 'touchcancel'); x.event(x.document, 'touchend', { changedTouches: [{ clientX: 10, clientY: 0 }] }); assert.deepEqual(paths, []);
  x.event(x.document, 'wheel', { target, deltaY: -40, deltaX: 0 }); await x.advance(240); x.event(x.document, 'wheel', { target, deltaY: -40, deltaX: 0 }); assert.deepEqual(paths, []); x.event(x.document, 'wheel', { target, deltaY: -40, deltaX: 0 }); x.event(x.document, 'wheel', { target, deltaY: -100, deltaX: 0 }); assert.deepEqual(paths, ['/explore']);
  assert.equal(isIgnoredGestureTarget(null), false); assert.equal(isIgnoredGestureTarget({}), false); assert.equal(isAtDocumentTop(), true); x.document.documentElement.scrollTop = 1; assert.equal(isAtDocumentTop(), false); assert.equal(isUpwardWheel({}), false);
  const ignored = createGestureController(); ignored.onTouchEnd({ changedTouches: [] }); ignored.onTouchStart({ touches: [], target }); ignored.onWheel({ ctrlKey: true, target });
});

test('scenario public inputs retain finite fallback variants and every real optional/story/rotation branch', () => {
  const item = (name, extra = {}) => ({ name, asset: name + '.svg', ...extra });
  const pools = { identity: [item('same')], 'identity-2': [item('same')], local: [item('local')], remote: [item('local')], shell: [item('shell', { action: 'invoke {name}', intent: 'owned command' })], ci: [item('ci', { action: name => 'queue ' + name })], container: [item('container')], cloud: [item('cloud')] };
  const selected = Object.fromEntries(Object.entries(pools).map(([key, list]) => [key, list[0]]));
  const full = createScenarioSession({ variantPools: pools, variantSelection: selected, storyChoice: { personaId: 'new-hire', context: 'with owned fixtures' }, random: () => 0, optionalSlots: [0, 1, 3] }); assert.ok(full.clauses.some(row => row.text === 'queue ci')); assert.ok(full.clauses.some(row => row.text === 'invoke shell')); assert.equal(full.tasks[0].length, 4);
  const fallback = createScenarioSession({ variantPools: { identity: null, 'identity-2': [], local: [{ name: 'missing asset' }], remote: [], shell: [], ci: [], cloud: [], container: [] }, previous: full, random: () => 0.999, storyChoice: { personaId: 'ai-model', context: 'with no configured provider' }, optionalSlots: [] }); assert.match(fallback.prefix, /human operator/); assert.ok(Object.values(fallback.selection).every(value => value.name === 'illustrative example'));
  const reservations = storyReservationSessions(pools); assert.ok(reservations.length > 0); assert.ok(reservations.every(row => row.clauses.length === 6 || row.clauses.length === 8));
  const next = createOptionalSchedule({ random: () => 0.9 }); assert.deepEqual(next(), [1]); for (let i = 0; i < 8; i++) assert.ok(Array.isArray(next()));
  const rotate = createVariantRotation({ variantPools: { ...pools, cloud: null }, random: () => 0 }); assert.ok(rotate().identity); assert.ok(rotate().identity);
  assert.equal(sessionStep(full, -1).progress, 0); assert.equal(sessionStep(full, full.legs.length * STEP_DURATION + 100).progress, 1);
});
