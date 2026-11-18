const explorePath = '/explore';
const touchThreshold = 56;
const wheelThreshold = 72;
const interactiveSelector = 'a, header, input, textarea, select, button, [contenteditable="true"], [role="listbox"], .header-search, .site-footer';

export function isAtDocumentTop(scrollY = globalThis.scrollY || 0, documentTop = globalThis.document?.documentElement?.scrollTop || 0) {
  return scrollY <= 0 && documentTop <= 0;
}

export function isIgnoredGestureTarget(target) {
  return Boolean(target?.closest?.(interactiveSelector));
}

export function isUpwardSwipe({ startX, startY, endX, endY, threshold = touchThreshold }) {
  const vertical = startY - endY;
  return vertical >= threshold && vertical > Math.abs(endX - startX);
}

export function isUpwardWheel({ deltaX = 0, deltaY = 0 }) {
  return deltaY < 0 && Math.abs(deltaY) > Math.abs(deltaX);
}

export function createGestureController({
  navigate,
  atTop = () => isAtDocumentTop(),
  ignoredTarget = isIgnoredGestureTarget,
  clearTimer = clearTimeout,
  schedule = setTimeout,
} = {}) {
  let touchStart = null;
  let wheelDistance = 0;
  let wheelReset;
  let navigating = false;
  const resetWheel = () => {
    wheelDistance = 0;
    clearTimer(wheelReset);
    wheelReset = undefined;
  };
  const mayNavigate = target => atTop() && !ignoredTarget(target);
  const openExplore = () => {
    if (navigating) return;
    navigating = true;
    navigate?.(explorePath);
  };

  return {
    onTouchStart(event) {
      if (event.touches.length !== 1 || !mayNavigate(event.target)) {
        touchStart = null;
        return;
      }
      const point = event.touches[0];
      touchStart = { x: point.clientX, y: point.clientY, target: event.target };
    },
    onTouchEnd(event) {
      if (!touchStart || event.changedTouches.length !== 1) {
        touchStart = null;
        return;
      }
      const point = event.changedTouches[0];
      const start = touchStart;
      touchStart = null;
      if (mayNavigate(start.target) && isUpwardSwipe({ startX: start.x, startY: start.y, endX: point.clientX, endY: point.clientY })) openExplore();
    },
    onTouchCancel() { touchStart = null; },
    onWheel(event) {
      if (event.ctrlKey || !mayNavigate(event.target) || !isUpwardWheel(event)) {
        resetWheel();
        return;
      }
      wheelDistance += -event.deltaY;
      clearTimer(wheelReset);
      wheelReset = schedule(resetWheel, 240);
      if (wheelDistance >= wheelThreshold) openExplore();
    },
  };
}

if (typeof document !== 'undefined') {
  const gestures = createGestureController({ navigate: path => window.location.assign(path) });
  document.addEventListener('touchstart', gestures.onTouchStart, { passive: true });
  document.addEventListener('touchend', gestures.onTouchEnd, { passive: true });
  document.addEventListener('touchcancel', gestures.onTouchCancel, { passive: true });
  document.addEventListener('wheel', gestures.onWheel, { passive: true });
}
