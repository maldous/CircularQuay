import assert from 'node:assert/strict';
import test from 'node:test';
import { createGestureController, isAtDocumentTop, isIgnoredGestureTarget, isUpwardSwipe, isUpwardWheel } from './landing-navigation.js';

const point = (x, y) => ({ clientX: x, clientY: y });
const plainTarget = { closest: () => null };
const linkTarget = { closest: selector => selector.includes('a') ? {} : null };
const controller = (options = {}) => {
  const paths = [];
  return { paths, gestures: createGestureController({ navigate: path => paths.push(path), atTop: () => true, schedule: () => 1, clearTimer: () => {}, ...options }) };
};

test('qualifies deliberate upward touch swipes only', () => {
  assert.equal(isUpwardSwipe({ startX: 100, startY: 240, endX: 106, endY: 160 }), true);
  assert.equal(isUpwardSwipe({ startX: 100, startY: 240, endX: 160, endY: 200 }), false);
  assert.equal(isUpwardSwipe({ startX: 100, startY: 160, endX: 100, endY: 240 }), false);
});

test('qualifies vertical upward wheel movement and requires the document top', () => {
  assert.equal(isUpwardWheel({ deltaX: 0, deltaY: -48 }), true);
  assert.equal(isUpwardWheel({ deltaX: 52, deltaY: -20 }), false);
  assert.equal(isUpwardWheel({ deltaX: 0, deltaY: 24 }), false);
  assert.equal(isAtDocumentTop(0, 0), true);
  assert.equal(isAtDocumentTop(1, 0), false);
});

test('resets wheel accumulation for reverse, ignored, and off-top events', () => {
  const { paths, gestures } = controller();
  gestures.onWheel({ target: plainTarget, deltaX: 0, deltaY: -50, ctrlKey: false });
  gestures.onWheel({ target: plainTarget, deltaX: 0, deltaY: 100, ctrlKey: false });
  gestures.onWheel({ target: plainTarget, deltaX: 0, deltaY: -22, ctrlKey: false });
  assert.deepEqual(paths, []);
  const second = controller();
  second.gestures.onWheel({ target: plainTarget, deltaX: 0, deltaY: -50, ctrlKey: false });
  second.gestures.onWheel({ target: linkTarget, deltaX: 0, deltaY: -50, ctrlKey: false });
  second.gestures.onWheel({ target: plainTarget, deltaX: 0, deltaY: -22, ctrlKey: false });
  assert.deepEqual(second.paths, []);
});

test('clears a pending single touch when a multi-touch gesture begins', () => {
  const { paths, gestures } = controller();
  gestures.onTouchStart({ target: plainTarget, touches: [point(100, 240)] });
  gestures.onTouchStart({ target: plainTarget, touches: [point(100, 240), point(120, 220)] });
  gestures.onTouchEnd({ changedTouches: [point(100, 150)] });
  assert.deepEqual(paths, []);
});

test('protects anchors and header controls from gesture navigation', () => {
  assert.equal(isIgnoredGestureTarget(linkTarget), true);
  const { paths, gestures } = controller();
  gestures.onTouchStart({ target: linkTarget, touches: [point(100, 240)] });
  gestures.onTouchEnd({ changedTouches: [point(100, 150)] });
  assert.deepEqual(paths, []);
});
