import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { browser } from '../../web/coverage-dom-fixture.test.mjs';

let sequence = 0;
const rect = (left, top, width, height) => ({ left, top, width, height, right: left + width, bottom: top + height });
async function fixture(t, { reduced = false, failImages = false, sparse = false } = {}) {
  const html = sparse ? '<section class="journey"><div class="ecosystem"></div></section>' : await readFile(new URL('./index.html', import.meta.url), 'utf8');
  const x = browser(t, { html }); const root = x.document.documentElement; root.lang = 'en'; root.dataset.reducedMotion = String(reduced);
  if (failImages) x.document.imageFailures.add('*');
  const ecosystem = x.document.querySelector('.ecosystem'); ecosystem.rect = rect(0, 0, 1000, 600);
  const hub = x.document.querySelector('.hub-disc'); if (hub) hub.rect = rect(450, 250, 100, 100);
  for (const [index, node] of x.document.querySelectorAll('.flow-node').entries()) node.rect = rect(['identity', 'identity-2', 'local', 'remote'].includes(node.dataset.node) ? 20 : 820, 50 + index % 4 * 120, 150, 75);
  for (const [index, node] of x.document.querySelectorAll('.optional-node').entries()) { const top = ['age', 'kms', 'signed'].includes(node.dataset.optional); node.rect = rect(180 + index % 3 * 300, top ? 70 : 470, 100, 50); const face = node.querySelector('.optional-face'); if (face) face.rect = { ...node.rect }; }
  t.mock.method(Math, 'random', () => 0.23);
  await import(new URL(`./motion.js?owned-fixture=${++sequence}`, import.meta.url));
  await x.drain(); await x.frame(0); await x.frame(16); await x.drain();
  const visibility = value => { for (const observer of x.observers.intersection) observer.callback([{ isIntersecting: value }]); };
  x.beforeRestore(() => { x.document.hidden = true; x.event(x.document, 'visibilitychange'); });
  return { ...x, root, ecosystem, visibility };
}

test('the real referenced-story first scene completes normalization, layout readiness and bounded geometry', async t => {
  const x = await fixture(t);
  // Regression: the old .map(normalise) used a nonexistent binding and the
  // outer catch silently left this actual first scene unready.
  assert.equal(x.root.dataset.layoutReady, 'true');
  assert.equal(x.document.querySelector('.journey').getAttribute('aria-busy'), 'false');
  assert.ok(x.document.querySelectorAll('.story-word-base').length > 5);
  assert.ok(x.document.querySelectorAll('.story-product').length > 0);
  assert.match(x.document.querySelector('#session-story').style.height, /^\d+px$/);
  for (const path of x.document.querySelectorAll('[data-route], [data-optional-route]')) { assert.match(path.getAttribute('d'), /^M/); assert.doesNotMatch(path.getAttribute('d'), /NaN|undefined|Infinity/); }
  assert.equal(x.root.dataset.playing, 'false'); x.visibility(true); assert.equal(x.root.dataset.playing, 'true');
  await x.frame(32); await x.frame(96); assert.ok(x.document.querySelectorAll('.flow-marker').length > 0);
});

test('focus, Escape, intersection and visibility gates pause and resume the same synthetic scene', async t => {
  const x = await fixture(t); x.visibility(true); await x.frame(32); await x.frame(1000);
  const session = x.root.dataset.session; const journey = x.document.querySelector('.journey');
  x.event(journey, 'focusin'); assert.equal(x.root.dataset.playing, 'false'); assert.equal(x.frames.size, 0);
  x.event(journey, 'focusout'); assert.equal(x.root.dataset.playing, 'true');
  x.event(x.document, 'keydown', { key: 'Escape', defaultPrevented: true }); assert.equal(x.root.dataset.playing, 'true');
  x.event(x.document, 'keydown', { key: 'Escape' }); assert.equal(x.root.dataset.playing, 'false');
  x.event(x.document, 'keydown', { key: 'Escape' }); assert.equal(x.root.dataset.playing, 'true');
  x.document.hidden = true; x.event(x.document, 'visibilitychange'); assert.equal(x.root.dataset.playing, 'false');
  x.document.hidden = false; x.event(x.document, 'visibilitychange'); assert.equal(x.root.dataset.playing, 'true');
  x.visibility(false); assert.equal(x.root.dataset.playing, 'false'); x.visibility(true); assert.equal(x.root.dataset.session, session);
});

test('all actual story legs reveal, settle and rotate with explicit image/timer boundaries', async t => {
  const x = await fixture(t); x.visibility(true); const sessions = new Set([x.root.dataset.session]);
  for (let time = 32; time <= 90000; time += 500) { await x.frame(time); await x.advance(500); sessions.add(x.root.dataset.session); }
  assert.ok(sessions.size > 1, 'completed real scenario advances through queued icon transition');
  assert.equal(x.document.querySelector('#session-story').dataset.storyReady, 'true');
  assert.ok(x.document.querySelectorAll('.story-word-reveal').some(word => /%$/.test(word.style['--reveal'])));
  assert.ok(x.document.querySelectorAll('.flow-node').some(node => Boolean(node.dataset.integrationId)));
  assert.ok(x.frames.size <= 2); assert.ok(x.timers.size <= 16);
});

test('reduced motion renders a complete static story and switching back preserves manual pause rules', async t => {
  const x = await fixture(t, { reduced: true }); x.visibility(true);
  assert.equal(x.root.dataset.layoutReady, 'true'); assert.equal(x.root.dataset.playing, 'false'); assert.equal(x.frames.size, 0);
  assert.equal(x.document.querySelector('.journey').classList.contains('is-delivered'), true);
  assert.ok(x.document.querySelectorAll('.story-clause').every(node => node.dataset.state === 'complete'));
  x.root.dataset.reducedMotion = 'false'; x.event(x.window, 'appearance:reduced-motion'); assert.equal(x.root.dataset.playing, 'true');
  await x.frame(100); await x.frame(1500); x.root.dataset.reducedMotion = 'true'; x.event(x.window, 'appearance:reduced-motion'); await x.drain(); assert.equal(x.root.dataset.playing, 'false');
});

test('resizing, language and font events reserve only owned measurement nodes and refresh finite paths', async t => {
  const x = await fixture(t); const count = x.document.querySelectorAll('#session-story').length;
  x.ecosystem.rect = rect(0, 0, 800, 900); x.event(x.window, 'resize'); await x.frame(100);
  x.root.lang = 'ar'; x.event(x.window, 'localechange'); await x.frame(120);
  x.event(x.document.fonts, 'loadingdone'); await x.frame(140);
  for (const observer of x.observers.resize) observer.callback([]);
  assert.equal(x.document.querySelectorAll('#session-story').length, count);
  assert.ok(!x.document.body.children.some(node => node.style.left === '-10000px'));
  assert.equal(x.document.querySelector('.journey-connections').getAttribute('viewBox'), '0 0 800 900');
});

test('image failure takes the real static fallback and later failed transitions release pending state', async t => {
  const x = await fixture(t, { failImages: true }); assert.equal(x.root.dataset.layoutReady, 'true');
  assert.equal(x.document.querySelector('.journey').getAttribute('aria-busy'), 'false'); assert.ok(x.document.querySelectorAll('.story-clause').length);
  x.document.imageFailures.clear(); x.visibility(true);
  for (let time = 32; time <= 40000; time += 1000) { await x.frame(time); await x.advance(1000); }
  x.document.imageFailures.add('*'); for (let time = 41000; time <= 80000; time += 1000) { await x.frame(time); await x.advance(1000); }
  assert.equal(x.document.querySelector('#session-story').classList.contains('is-transitioning'), false); assert.ok(x.frames.size <= 2);
});

test('optional absent presentation nodes do not invent a provider action or require an SVG implementation', async t => {
  const x = await fixture(t, { sparse: true, reduced: true });
  assert.equal(x.root.dataset.layoutReady, 'true'); assert.equal(x.root.dataset.playing, 'false');
  assert.equal(x.document.querySelector('.flow-marker'), null); assert.equal(x.sources.length, 0);
});
