const root = document.documentElement;
const stage = document.querySelector('.journey');
const ecosystem = document.querySelector('.ecosystem');
const chapters = [...document.querySelectorAll('[data-chapter-button]')];
const pauseButton = document.querySelector('#pause');
const pauseLabel = document.querySelector('#pause-label');
const replayButton = document.querySelector('#replay');
const startMotionButton = document.querySelector('#start-motion');
const copy = document.querySelector('#story-copy');
const position = document.querySelector('#story-position');
const note = document.querySelector('#motion-note');
const status = document.querySelector('#interaction-status');
const theme = document.querySelector('#theme');
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)');
const coarsePointer = matchMedia('(pointer: coarse)');

const story = {
  manual: { number: '01', copy: 'Find the right integration. Check access. Prepare each app. Then do it all again.' },
  flow: { number: '02', copy: 'CircularQuay coordinates authenticated access and local policy before preparing an approved app.' },
  denied: { number: '03', copy: 'No matching grant? The request stops at local policy. Nothing is delivered to an app.' },
};
const groups = ['identity', 'sources', 'applications'];
const duration = 14000;
const tokens = [];
const selections = new Map();
const routes = [];
let chapter = 'flow';
let elapsed = 0;
let playing = false;
let inView = false;
let frame = 0;
let lastTime = null;
let lastPaint = 0;
let lastLabelChange = 0;
let geometryDirty = true;
let bounds = { width: 0, height: 0, protected: [] };

function hash(value) {
  let valueHash = 2166136261;
  for (const character of value) valueHash = Math.imul(valueHash ^ character.charCodeAt(0), 16777619);
  return (valueHash >>> 0) / 4294967296;
}

function announce(message) { status.textContent = message; }
function clamp(value, minimum, maximum) { return Math.min(maximum, Math.max(minimum, value)); }
function easeOut(value) { return 1 - Math.pow(1 - clamp(value, 0, 1), 4); }

function setChapter(value, announceChange = false) {
  if (!Object.hasOwn(story, value)) return;
  if (value === 'flow' && chapter === 'manual' && playing) {
    tokens.forEach(token => {
      token.flowX = token.x;
      token.flowY = token.y;
    });
  }
  chapter = value;
  root.dataset.chapter = value;
  ecosystem?.querySelector('.hub-disc')?.classList.toggle('is-denied', value === 'denied');
  copy.textContent = story[value].copy;
  position.textContent = `${story[value].number} / 03`;
  chapters.forEach(button => button.setAttribute('aria-pressed', String(button.dataset.chapterButton === value)));
  if (announceChange) announce(story[value].copy);
}

function initialiseTokens() {
  document.querySelectorAll('.integration-token').forEach((element, index) => {
    const group = element.dataset.group;
    if (!groups.includes(group)) return;
    const image = element.querySelector('img');
    const id = element.dataset.integrationId || `integration-${index}`;
    tokens.push({ element, group, id, index, name: element.dataset.name || image?.alt || id, seed: hash(id), radius: 20, x: NaN, y: NaN, vx: 0, vy: 0, homeX: 0, homeY: 0, manualX: 0, manualY: 0, flowX: 0, flowY: 0 });
  });
  groups.forEach(group => {
    selections.set(group, document.querySelector(`.scene-examples [data-selection="${group}"]`) || document.querySelector(`[data-selection="${group}"]`));
  });
}

function initialiseRoutes() {
  document.querySelectorAll('.journey-connections [data-route]').forEach(path => {
    const svg = path.ownerSVGElement;
    let packets = svg.querySelector('.packets');
    if (!packets) {
      packets = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      packets.classList.add('packets');
      svg.append(packets);
    }
    const packet = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
    packet.setAttribute('r', '4');
    packets.append(packet);
    routes.push({ name: path.dataset.route, path, packet, samples: [] });
  });
}

function isProtected(x, y, radius) {
  return bounds.protected.some(rect => x + radius > rect.left && x - radius < rect.right && y + radius > rect.top && y - radius < rect.bottom);
}

function constrainToken(token) {
  token.x = clamp(token.x, token.radius, bounds.width - token.radius);
  token.y = clamp(token.y, token.radius, bounds.height - token.radius);
  bounds.protected.forEach(rect => {
    const radius = token.radius + 1;
    if (!(token.x + radius > rect.left && token.x - radius < rect.right && token.y + radius > rect.top && token.y - radius < rect.bottom)) return;
    const exits = [
      { x: rect.left - token.radius - 1, y: token.y },
      { x: rect.right + token.radius + 1, y: token.y },
      { x: token.x, y: rect.top - token.radius - 1 },
      { x: token.x, y: rect.bottom + token.radius + 1 },
    ].map(point => ({ ...point, distance: Math.hypot(point.x - token.x, point.y - token.y) }))
      .filter(point => point.x >= token.radius && point.x <= bounds.width - token.radius && point.y >= token.radius && point.y <= bounds.height - token.radius)
      .sort((left, right) => left.distance - right.distance);
    if (exits[0]) Object.assign(token, exits[0]);
  });
}

function createSlotPool(radius) {
  const step = radius * 2 + 5;
  const slots = [];
  for (let y = radius + 4; y <= bounds.height - radius - 4; y += step) {
    for (let x = radius + 4; x <= bounds.width - radius - 4; x += step) {
      if (!isProtected(x, y, radius + 1)) slots.push({ x, y });
    }
  }
  return slots;
}

function refreshGeometry() {
  if (!ecosystem || !tokens.length) return;
  // Transforms are the animation layer; clear them before measuring the stable grid.
  tokens.forEach(token => { token.element.style.transform = 'none'; });
  const area = ecosystem.getBoundingClientRect();
  const protectedElements = [
    ecosystem.querySelector('.local-hub'),
    ecosystem.querySelector('.device-label'),
  ].filter(Boolean);
  bounds = {
    width: area.width,
    height: area.height,
    protected: protectedElements.map(element => {
      const rect = element.getBoundingClientRect();
      const padding = element.matches('.local-hub') ? 7 : 2;
      return { left: rect.left - area.left - padding, right: rect.right - area.left + padding, top: rect.top - area.top - padding, bottom: rect.bottom - area.top + padding };
    }),
  };
  tokens.forEach(token => {
    const rect = token.element.getBoundingClientRect();
    token.homeX = rect.left - area.left + rect.width / 2;
    token.homeY = rect.top - area.top + rect.height / 2;
    token.radius = Math.max(rect.width, rect.height) / 2;
  });
  // A broad, unique slot pool lets every group roam the full ecosystem safely.
  const homes = [...tokens].sort((left, right) => left.id.localeCompare(right.id));
  const radius = Math.max(...tokens.map(token => token.radius));
  let slots = createSlotPool(radius).sort((left, right) => hash(`${left.x}:${left.y}`) - hash(`${right.x}:${right.y}`));
  if (slots.length < tokens.length) slots = homes.map(token => ({ x: token.homeX, y: token.homeY }));
  tokens.forEach((token, index) => {
    const slot = slots[index];
    token.manualX = slot.x;
    token.manualY = slot.y;
    if (!Number.isFinite(token.x)) {
      token.x = token.manualX;
      token.y = token.manualY;
    }
  });
  refreshRoutes(area);
  if (!playing && chapter !== 'manual') {
    tokens.forEach(token => {
      token.x = token.homeX;
      token.y = token.homeY;
      token.vx = 0;
      token.vy = 0;
    });
  }
  geometryDirty = false;
}

function applyPositions() {
  tokens.forEach(token => {
    token.element.style.transform = `translate3d(${(token.x - token.homeX).toFixed(2)}px, ${(token.y - token.homeY).toFixed(2)}px, 0)`;
  });
}

function setSettled(refresh = true) {
  if (refresh) refreshGeometry();
  tokens.forEach(token => {
    token.x = token.homeX;
    token.y = token.homeY;
    token.vx = 0;
    token.vy = 0;
  });
  applyPositions();
}

function cacheRoute(route, start, end) {
  const deltaX = end.x - start.x;
  const deltaY = end.y - start.y;
  const bend = Math.min(100, Math.hypot(deltaX, deltaY) * .4);
  const directionX = Math.abs(deltaX) > Math.abs(deltaY) ? Math.sign(deltaX) : 0;
  const directionY = directionX ? 0 : Math.sign(deltaY);
  const first = { x: start.x + directionX * bend, y: start.y + directionY * bend };
  const second = { x: end.x - directionX * bend, y: end.y - directionY * bend };
  route.path.setAttribute('d', `M${start.x} ${start.y} C${first.x} ${first.y} ${second.x} ${second.y} ${end.x} ${end.y}`);
  const length = route.path.getTotalLength();
  route.samples = Array.from({ length: 81 }, (_, index) => route.path.getPointAtLength(length * index / 80));
}

function refreshRoutes(area) {
  const hub = ecosystem.querySelector('.hub-disc')?.getBoundingClientRect();
  if (!hub) return;
  const hubRect = { left: hub.left - area.left, right: hub.right - area.left, top: hub.top - area.top, bottom: hub.bottom - area.top };
  const hubCenter = { x: (hubRect.left + hubRect.right) / 2, y: (hubRect.top + hubRect.bottom) / 2 };
  const edgeToward = (rect, target) => {
    const center = { x: (rect.left + rect.right) / 2, y: (rect.top + rect.bottom) / 2 };
    const deltaX = target.x - center.x;
    const deltaY = target.y - center.y;
    if (Math.abs(deltaX) > Math.abs(deltaY)) return { x: deltaX > 0 ? rect.right : rect.left, y: center.y };
    return { x: center.x, y: deltaY > 0 ? rect.bottom : rect.top };
  };
  const cloudEdge = group => {
    const cloud = tokens.filter(token => token.group === group);
    if (!cloud.length) return null;
    const rect = cloud.reduce((box, token) => ({
      left: Math.min(box.left, token.homeX - token.radius),
      right: Math.max(box.right, token.homeX + token.radius),
      top: Math.min(box.top, token.homeY - token.radius),
      bottom: Math.max(box.bottom, token.homeY + token.radius),
    }), { left: Infinity, right: -Infinity, top: Infinity, bottom: -Infinity });
    return edgeToward(rect, hubCenter);
  };
  const hubEdge = target => edgeToward(hubRect, target);
  const identity = cloudEdge('identity');
  const sources = cloudEdge('sources');
  const applications = cloudEdge('applications');
  routes.forEach(route => {
    route.path.ownerSVGElement.setAttribute('viewBox', `0 0 ${area.width} ${area.height}`);
    if (route.name === 'identity' && identity) cacheRoute(route, identity, hubEdge(identity));
    if (route.name === 'sources' && sources) cacheRoute(route, sources, hubEdge(sources));
    if (['apps', 'applications'].includes(route.name) && applications) cacheRoute(route, hubEdge(applications), applications);
    // Denial reaches the policy boundary from a source and never draws delivery to an app.
    if (route.name === 'denied' && sources) cacheRoute(route, sources, hubEdge(sources));
  });
}

function resetFlock() {
  refreshGeometry();
  tokens.forEach(token => {
    token.x = token.manualX;
    token.y = token.manualY;
    token.vx = (token.seed - .5) * 10;
    token.vy = (hash(`${token.id}-velocity`) - .5) * 10;
  });
  applyPositions();
}

function flock(delta) {
  const cellSize = 72;
  const grid = new Map();
  const key = (x, y) => `${Math.floor(x / cellSize)}:${Math.floor(y / cellSize)}`;
  tokens.forEach(token => {
    const cell = key(token.x, token.y);
    if (!grid.has(cell)) grid.set(cell, []);
    grid.get(cell).push(token);
  });
  tokens.forEach(token => {
    let separateX = 0;
    let separateY = 0;
    let alignX = 0;
    let alignY = 0;
    let cohesionX = 0;
    let cohesionY = 0;
    let neighbors = 0;
    const column = Math.floor(token.x / cellSize);
    const row = Math.floor(token.y / cellSize);
    for (let x = column - 1; x <= column + 1; x += 1) for (let y = row - 1; y <= row + 1; y += 1) {
      (grid.get(`${x}:${y}`) || []).forEach(other => {
        if (other === token) return;
        const dx = token.x - other.x;
        const dy = token.y - other.y;
        const distance = Math.hypot(dx, dy) || .01;
        const minimum = token.radius + other.radius + 4;
        if (distance < minimum) {
          separateX += dx / distance * (minimum - distance) / minimum;
          separateY += dy / distance * (minimum - distance) / minimum;
        }
        if (distance < 82 && other.group === token.group) {
          alignX += other.vx;
          alignY += other.vy;
          cohesionX += other.x;
          cohesionY += other.y;
          neighbors += 1;
        }
      });
    }
    const phase = elapsed / 1000 + token.seed * Math.PI * 2;
    token.vx += separateX * .28 + (neighbors ? (alignX / neighbors - token.vx) * .014 + (cohesionX / neighbors - token.x) * .0025 : 0) + Math.cos(phase) * .035;
    token.vy += separateY * .28 + (neighbors ? (alignY / neighbors - token.vy) * .014 + (cohesionY / neighbors - token.y) * .0025 : 0) + Math.sin(phase * .8) * .035;
  });
  tokens.forEach(token => {
    token.vx *= .91;
    token.vy *= .91;
    const nextX = clamp(token.x + token.vx * delta / 16, token.radius, bounds.width - token.radius);
    const nextY = clamp(token.y + token.vy * delta / 16, token.radius, bounds.height - token.radius);
    if (isProtected(nextX, nextY, token.radius + 1)) {
      token.vx *= -.45;
      token.vy *= -.45;
    } else {
      token.x = nextX;
      token.y = nextY;
    }
  });
  resolveCollisions();
}

function resolveCollisionPass() {
  const cellSize = Math.max(...tokens.map(token => token.radius * 2 + 4));
  const grid = new Map();
  const key = (x, y) => `${Math.floor(x / cellSize)}:${Math.floor(y / cellSize)}`;
  tokens.forEach(token => {
    const cell = key(token.x, token.y);
    if (!grid.has(cell)) grid.set(cell, []);
    grid.get(cell).push(token);
  });
  tokens.forEach(token => {
    const column = Math.floor(token.x / cellSize);
    const row = Math.floor(token.y / cellSize);
    for (let x = column - 1; x <= column + 1; x += 1) for (let y = row - 1; y <= row + 1; y += 1) {
      (grid.get(`${x}:${y}`) || []).forEach(other => {
        if (other.index <= token.index) return;
        let dx = other.x - token.x;
        let dy = other.y - token.y;
        let distance = Math.hypot(dx, dy);
        if (distance < .01) {
          const angle = hash(`${token.id}:${other.id}`) * Math.PI * 2;
          dx = Math.cos(angle);
          dy = Math.sin(angle);
          distance = 1;
        }
        const minimum = token.radius + other.radius + 4;
        if (distance >= minimum) return;
        const move = (minimum - distance) / 2;
        const unitX = dx / distance;
        const unitY = dy / distance;
        const first = { x: clamp(token.x - unitX * move, token.radius, bounds.width - token.radius), y: clamp(token.y - unitY * move, token.radius, bounds.height - token.radius) };
        const second = { x: clamp(other.x + unitX * move, other.radius, bounds.width - other.radius), y: clamp(other.y + unitY * move, other.radius, bounds.height - other.radius) };
        Object.assign(token, first);
        Object.assign(other, second);
        constrainToken(token);
        constrainToken(other);
      });
    }
  });
}

function resolveCollisions() {
  for (let pass = 0; pass < 24; pass += 1) {
    resolveCollisionPass();
    tokens.forEach(constrainToken);
  }
}

function organise(progress) {
  const eased = easeOut(progress);
  tokens.forEach(token => {
    token.x = token.flowX + (token.homeX - token.flowX) * eased;
    token.y = token.flowY + (token.homeY - token.flowY) * eased;
    token.vx = 0;
    token.vy = 0;
    if (progress >= .99) {
      token.x = token.homeX;
      token.y = token.homeY;
      token.vx = 0;
      token.vy = 0;
    }
  });
  resolveCollisions();
}

function paintRoutes(time) {
  routes.forEach((route, index) => {
    const incoming = ['identity', 'sources'].includes(route.name);
    const outgoing = ['apps', 'applications'].includes(route.name);
    const denied = route.name === 'denied';
    const visible = !reducedMotion.matches && ((chapter === 'flow' && (incoming || outgoing)) || (chapter === 'denied' && denied));
    route.path.dataset.active = String(visible);
    route.packet.style.display = visible ? '' : 'none';
    if (!visible || !route.samples.length) return;
    const deniedStep = Math.min(1, Math.max(0, (time - 9000) / 900));
    const step = denied ? deniedStep * 80 : ((time / 2400 + index * .31) % 1) * 80;
    const lower = Math.floor(step);
    const upper = Math.min(80, lower + 1);
    const fraction = step - lower;
    const start = route.samples[lower];
    const end = route.samples[upper];
    route.packet.setAttribute('cx', String(start.x + (end.x - start.x) * fraction));
    route.packet.setAttribute('cy', String(start.y + (end.y - start.y) * fraction));
  });
}

function rotateSelections() {
  groups.forEach(group => {
    const element = selections.get(group);
    const options = tokens.filter(token => token.group === group);
    if (!element || !options.length) return;
    const selected = options[Math.floor(elapsed / 2500) % options.length];
    element.dataset.selectedId = selected.id;
    element.textContent = selected.name;
    tokens.filter(token => token.group === group).forEach(token => token.element.classList.toggle('is-highlighted', token === selected));
  });
}

function renderPlayback() {
  root.dataset.playing = String(playing);
  const label = playing ? 'Pause' : elapsed >= duration ? 'Play again' : 'Play';
  pauseLabel.textContent = label;
  pauseButton.setAttribute('aria-label', playing ? 'Pause animation' : 'Play animation');
  pauseButton.disabled = reducedMotion.matches;
  replayButton.disabled = reducedMotion.matches;
  note.textContent = reducedMotion.matches ? 'Reduced motion · complete grouped view.' : '14 seconds · plays once. Pause or choose a chapter.';
}

function stopFrame() { cancelAnimationFrame(frame); frame = 0; lastTime = null; }
function canAnimate() { return playing && inView && !document.hidden && !reducedMotion.matches; }

function renderScene(delta = 16) {
  if (geometryDirty) refreshGeometry();
  if (!playing || reducedMotion.matches) {
    if (chapter !== 'manual') setSettled(false);
    else applyPositions();
    return;
  }
  if (chapter === 'manual') flock(delta);
  else if (chapter === 'flow') organise((elapsed - 4000) / 5000, delta);
  else setSettled(false);
  applyPositions();
}

function tick(timestamp) {
  frame = 0;
  if (!canAnimate()) return;
  const delta = lastTime === null ? 0 : timestamp - lastTime;
  if (lastTime !== null) elapsed = Math.min(duration, elapsed + delta);
  lastTime = timestamp;
  const next = elapsed < 4000 ? 'manual' : elapsed < 9000 ? 'flow' : elapsed < 12000 ? 'denied' : 'flow';
  if (next !== chapter) setChapter(next);
  const maximumFps = coarsePointer.matches ? 20 : 30;
  if (timestamp - lastPaint >= 1000 / maximumFps) {
    const physicsDelta = Math.min(32, Math.max(0, timestamp - lastPaint));
    renderScene(physicsDelta);
    paintRoutes(elapsed);
    if (elapsed - lastLabelChange >= 2500) {
      rotateSelections();
      lastLabelChange = elapsed;
    }
    lastPaint = timestamp;
  }
  if (elapsed >= duration) {
    playing = false;
    setChapter('flow');
    setSettled();
    paintRoutes(elapsed);
    renderPlayback();
    return;
  }
  frame = requestAnimationFrame(tick);
}

function updateFrame() { stopFrame(); if (canAnimate()) frame = requestAnimationFrame(tick); }
function replay() {
  if (reducedMotion.matches) return;
  elapsed = 0;
  lastLabelChange = 0;
  resetFlock();
  rotateSelections();
  playing = true;
  setChapter('manual');
  renderPlayback();
  announce('Playing the journey from manual work to coordinated delivery.');
  updateFrame();
}
function chooseChapter(value) {
  playing = false;
  stopFrame();
  elapsed = { manual: 0, flow: 8000, denied: 10000 }[value];
  setChapter(value, true);
  if (value === 'manual') resetFlock(); else setSettled();
  paintRoutes(elapsed);
  renderPlayback();
}

initialiseTokens();
initialiseRoutes();
chapters.forEach(button => {
  button.disabled = false;
  button.addEventListener('click', () => chooseChapter(button.dataset.chapterButton));
});
pauseButton.addEventListener('click', () => {
  if (elapsed >= duration) return replay();
  playing = !playing;
  renderPlayback();
  paintRoutes(elapsed);
  announce(playing ? 'Animation playing.' : 'Animation paused.');
  updateFrame();
});
replayButton.addEventListener('click', replay);
if (startMotionButton) startMotionButton.disabled = false;
startMotionButton?.addEventListener('click', () => {
  if (reducedMotion.matches) {
    playing = false;
    setChapter('flow');
    setSettled();
    announce('Reduced motion is on. The grouped route is shown without animation.');
    return;
  }
  replay();
});
theme.disabled = false;
theme.value = root.dataset.theme || 'light';
theme.addEventListener('change', () => {
  const value = theme.value;
  root.dataset.theme = value;
  try {
    localStorage.setItem('circularquay-motion-theme', value);
  } catch { announce('Theme changed for this visit. This browser is blocking saved preferences.'); }
});
const resizeObserver = new ResizeObserver(() => { geometryDirty = true; if (!playing) renderScene(); });
if (ecosystem) resizeObserver.observe(ecosystem);
document.querySelectorAll('.integration-grid').forEach(grid => resizeObserver.observe(grid));
document.addEventListener('visibilitychange', updateFrame);
reducedMotion.addEventListener('change', () => {
  playing = false;
  stopFrame();
  setChapter('flow');
  setSettled();
  paintRoutes(elapsed);
  renderPlayback();
});
if ('IntersectionObserver' in window) {
  const observer = new IntersectionObserver(entries => { inView = entries[0].isIntersecting; updateFrame(); }, { threshold: .15 });
  observer.observe(stage);
} else inView = true;
document.querySelector('.playback').hidden = false;
if (reducedMotion.matches) {
  playing = false;
  setChapter('flow');
  setSettled();
} else {
  resetFlock();
  playing = true;
  setChapter('manual');
}
renderPlayback();
paintRoutes(elapsed);
updateFrame();
