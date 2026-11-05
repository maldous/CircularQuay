const root = document.documentElement;
const stage = document.querySelector('.journey');
const chapters = [...document.querySelectorAll('[data-chapter-button]')];
const pauseButton = document.querySelector('#pause');
const pauseLabel = document.querySelector('#pause-label');
const replayButton = document.querySelector('#replay');
const copy = document.querySelector('#story-copy');
const position = document.querySelector('#story-position');
const note = document.querySelector('#motion-note');
const status = document.querySelector('#interaction-status');
const theme = document.querySelector('#theme');
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)');
const narrowScreen = matchMedia('(max-width: 900px)');
const segmentedScreen = matchMedia('(horizontal-viewport-segments: 2)');

const story = {
  manual: { number: '01', copy: 'Find the file. Check access. Prepare each app. Then do it all again.' },
  flow: { number: '02', copy: 'CircularQuay checks local policy before preparing the environment for an approved app.' },
  denied: { number: '03', copy: 'No matching grant? The request stops here. Nothing is delivered to the app.' },
};
const duration = 14000;
let chapter = 'flow';
let elapsed = 0;
let playing = false;
let inView = false;
let frame = 0;
let lastTime = null;
let lastPaint = 0;

// Sample each fixed SVG path once. Frames only interpolate cached coordinates;
// they never measure layout or re-read path geometry.
const routes = [...document.querySelectorAll('[data-route]')].map(path => {
  const length = path.getTotalLength();
  const samples = Array.from({ length: 101 }, (_, index) => {
    const point = path.getPointAtLength(length * index / 100);
    return { x: point.x, y: point.y };
  });
  const packet = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
  packet.setAttribute('r', '5');
  const name = path.dataset.route;
  const routeChapter = name.includes('manual') ? 'manual' : name.includes('denied') ? 'denied' : 'flow';
  packet.classList.add(`packet-${routeChapter}`);
  path.ownerSVGElement.querySelector('.packets').append(packet);
  return { samples, packet, routeChapter, compact: name.startsWith('compact'), outgoing: name.startsWith('out-') || name === 'compact-out' };
});

function announce(message) {
  status.textContent = message;
}

function setChapter(value, announceChange = false) {
  if (!Object.hasOwn(story, value)) return;
  chapter = value;
  root.dataset.chapter = value;
  copy.textContent = story[value].copy;
  position.textContent = `${story[value].number} / 03`;
  chapters.forEach(button => button.setAttribute('aria-pressed', String(button.dataset.chapterButton === value)));
  if (announceChange) announce(story[value].copy);
}

function paintPackets(time) {
  const compact = narrowScreen.matches || segmentedScreen.matches;
  for (const [index, route] of routes.entries()) {
    const show = !reducedMotion.matches && route.routeChapter === chapter && route.compact === compact;
    route.packet.style.display = show ? '' : 'none';
    if (!show) continue;
    const cycle = chapter === 'manual' ? 3600 : 2400;
    // Outgoing packets begin after the incoming leg reaches local policy.
    const offset = route.outgoing ? 1200 : index * 240;
    let progress = ((time - offset + cycle * 10) % cycle) / cycle;
    if (chapter === 'denied') progress = Math.min(progress * 1.4, 1);
    const sample = progress * 100;
    const lower = Math.floor(sample);
    const upper = Math.min(100, lower + 1);
    const fraction = sample - lower;
    const start = route.samples[lower];
    const end = route.samples[upper];
    route.packet.setAttribute('cx', String(start.x + (end.x - start.x) * fraction));
    route.packet.setAttribute('cy', String(start.y + (end.y - start.y) * fraction));
  }
}

function renderPlayback() {
  root.dataset.playing = String(playing);
  const label = playing ? 'Pause' : elapsed >= duration ? 'Play again' : 'Play';
  pauseLabel.textContent = label;
  pauseButton.setAttribute('aria-label', playing ? 'Pause animation' : 'Play animation');
  pauseButton.disabled = reducedMotion.matches;
  replayButton.disabled = reducedMotion.matches;
  note.textContent = reducedMotion.matches
    ? 'Reduced motion is on. Choose a chapter to explore each static scene.'
    : 'A 14-second story, played once. Pause or choose a chapter at any time.';
}

function stopFrame() {
  cancelAnimationFrame(frame);
  frame = 0;
  lastTime = null;
}

function canAnimate() {
  return playing && inView && !document.hidden && !reducedMotion.matches;
}

function tick(timestamp) {
  frame = 0;
  if (!canAnimate()) return;
  // Use active elapsed time even when frames are dropped. Visibility and
  // intersection changes reset lastTime, so background time is never added.
  if (lastTime !== null) elapsed = Math.min(duration, elapsed + timestamp - lastTime);
  lastTime = timestamp;
  const next = elapsed < 4000 ? 'manual' : elapsed < 9000 ? 'flow' : elapsed < 12000 ? 'denied' : 'flow';
  if (next !== chapter) setChapter(next);
  if (timestamp - lastPaint >= 1000 / 30) {
    paintPackets(elapsed);
    lastPaint = timestamp;
  }
  if (elapsed >= duration) {
    playing = false;
    renderPlayback();
    paintPackets(elapsed);
    return;
  }
  frame = requestAnimationFrame(tick);
}

function updateFrame() {
  stopFrame();
  if (canAnimate()) frame = requestAnimationFrame(tick);
}

function replay() {
  if (reducedMotion.matches) return;
  elapsed = 0;
  playing = true;
  setChapter('manual');
  renderPlayback();
  announce('Playing the journey from manual work to policy-controlled delivery.');
  updateFrame();
}

chapters.forEach(button => {
  button.disabled = false;
  button.addEventListener('click', () => {
    playing = false;
    stopFrame();
    elapsed = { manual: 0, flow: 4500, denied: 9500 }[button.dataset.chapterButton];
    setChapter(button.dataset.chapterButton, true);
    renderPlayback();
    paintPackets(elapsed);
  });
});

pauseButton.addEventListener('click', () => {
  if (elapsed >= duration) return replay();
  playing = !playing;
  renderPlayback();
  paintPackets(elapsed);
  announce(playing ? 'Animation playing.' : 'Animation paused.');
  updateFrame();
});
replayButton.addEventListener('click', replay);

// Persist one appearance preference in the prototype's own origin only.
theme.value = root.dataset.theme || 'system';
theme.addEventListener('change', () => {
  const value = theme.value;
  if (value === 'light' || value === 'dark') root.dataset.theme = value;
  else delete root.dataset.theme;
  try {
    if (value === 'system') localStorage.removeItem('circularquay-motion-theme');
    else localStorage.setItem('circularquay-motion-theme', value);
  } catch {
    announce('Theme changed for this visit. This browser is blocking saved preferences.');
  }
});

document.addEventListener('visibilitychange', updateFrame);
narrowScreen.addEventListener('change', () => paintPackets(elapsed));
segmentedScreen.addEventListener('change', () => paintPackets(elapsed));
reducedMotion.addEventListener('change', () => {
  playing = false;
  stopFrame();
  setChapter('flow');
  paintPackets(elapsed);
  renderPlayback();
});

if ('IntersectionObserver' in window) {
  const observer = new IntersectionObserver(entries => {
    inView = entries[0].isIntersecting;
    updateFrame();
  }, { threshold: 0.15 });
  observer.observe(stage);
} else {
  inView = true;
}

document.querySelector('.playback').hidden = false;
playing = !reducedMotion.matches;
setChapter(playing ? 'manual' : 'flow');
renderPlayback();
paintPackets(elapsed);
updateFrame();
