import { motionCatalog } from './motion-catalog.js';
import { normaliseStoryWord, createScenarioSession, createOptionalSchedule, createVariantRotation, createStoryRotation, sessionStep, staticScenario, storyReservationSessions } from './scenario-flow.js';

const root = document.documentElement;
const journey = document.querySelector('.journey');
const ecosystem = document.querySelector('.ecosystem');
const svg = document.querySelector('.journey-connections');
const hub = document.querySelector('.hub-disc');
const nodes = [...document.querySelectorAll('.flow-node')];
const story = document.querySelector('#session-story');
const storyPrefix = document.querySelector('[data-story-prefix]');
const storyClauses = document.querySelector('[data-story-clauses]');
const storyAlternative = document.querySelector('[data-story-alternative]');
const optionalSchedule = createOptionalSchedule();
const storyRotation = createStoryRotation();
let variantRotation;
const routes = [];
const iconStates = new Map();
let inView = false;
let pausedByFocus = false;
let pausedByEscape = false;
let geometryDirty = true;
let geometrySignature = '';
let frame = 0;
let lastTime = null;
let lastPaint = 0;
let sessionElapsed = 0;
let session = null;
let preparingSession = false;
let renderedStoryKey = '';
let storyWords = [];
let activeRouteId = '';
let deliveredSessionId = '';
let storyReservationKey = '';
let reservationFrame = 0;

function isReducedMotion() { return root.dataset.reducedMotion === 'true'; }
function motionEnabled() { return root.dataset.layoutReady === 'true' && !isReducedMotion() && inView && !document.hidden && !pausedByFocus && !pausedByEscape; }
function updateMotionState() { root.dataset.playing = String(motionEnabled() && !preparingSession); }
function parseVariants(node) { try { return JSON.parse(node.dataset.variants || '[]').filter(item => item?.name && item?.asset); } catch { return []; } }

function initialiseNodes() {
  nodes.forEach(node => {
    const current = node.querySelector('[data-icon-current]');
    const next = node.querySelector('[data-icon-next]');
    if (!current || !next) return;
    const name = node.querySelector('.node-name'), role = node.querySelector('.node-role');
    name?.removeAttribute('data-i18n'); role?.removeAttribute('data-i18n');
    node.lang = 'en'; node.dir = 'ltr';
    iconStates.set(node.dataset.node, { node, current, next, name, role, variants: motionCatalog[node.dataset.node] || parseVariants(node) });
  });
}

function initialiseRoutes() {
  svg?.querySelectorAll('[data-route], [data-optional-route]').forEach(path => {
    const marker = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    const ring = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
    const label = document.createElementNS('http://www.w3.org/2000/svg', 'text');
    marker.classList.add('flow-marker'); marker.setAttribute('aria-hidden', 'true');
    marker.style.display = 'none';
    ring.setAttribute('r', '9'); label.setAttribute('dominant-baseline', 'central'); label.setAttribute('text-anchor', 'middle');
    marker.append(ring, label); svg.querySelector('.packets')?.append(marker);
    routes.push({ id: path.dataset.route || `optional-${path.dataset.optionalRoute}`, path, packet: marker, markerLabel: label, kind: path.dataset.kind || 'optional', samples: [], hubProgress: .5 });
  });
}

function pointFor(rect, area) { return { left: rect.left - area.left, right: rect.right - area.left, top: rect.top - area.top, bottom: rect.bottom - area.top }; }
function cachePath(route, start, end, vertical = false) {
  const dx = end.x - start.x, dy = end.y - start.y;
  const bend = Math.min(90, (vertical ? Math.abs(dy) : Math.abs(dx)) * .42);
  const first = vertical ? { x: start.x, y: start.y + Math.sign(dy || 1) * bend } : { x: start.x + Math.sign(dx || 1) * bend, y: start.y };
  const second = vertical ? { x: end.x, y: end.y - Math.sign(dy || 1) * bend } : { x: end.x - Math.sign(dx || 1) * bend, y: end.y };
  route.path.setAttribute('d', `M${start.x} ${start.y} C${first.x} ${first.y} ${second.x} ${second.y} ${end.x} ${end.y}`);
  const length = route.path.getTotalLength();
  route.samples = Array.from({ length: 81 }, (_, index) => route.path.getPointAtLength(length * index / 80));
}

function refreshGeometry() {
  if (!ecosystem || !svg || !hub) return;
  const area = ecosystem.getBoundingClientRect();
  const signature = [area.width, area.height, innerWidth, innerHeight].join('|');
  if (signature === geometrySignature) { geometryDirty = false; return; }
  geometrySignature = signature;
  svg.setAttribute('viewBox', `0 0 ${area.width} ${area.height}`);
  const hubRect = pointFor(hub.getBoundingClientRect(), area);
  const hubCenter = { x: (hubRect.left + hubRect.right) / 2, y: (hubRect.top + hubRect.bottom) / 2 };
  const sidePorts = [.15, .38, .62, .85];
  const topPorts = ['age', 'kms', 'signed'];
  const bottomPorts = ['directory', 'clients', 'audit'];
  routes.forEach(route => {
    if (route.kind === 'incoming' || route.kind === 'outgoing') {
      const nodeKey = route.kind === 'incoming' ? route.path.dataset.from : route.path.dataset.to;
      const node = nodes.find(item => item.dataset.node === nodeKey);
      if (!node) return;
      const card = pointFor(node.getBoundingClientRect(), area);
      const primaryIndex = routes.filter(item => item.kind === route.kind).indexOf(route);
      const portY = hubRect.top + (hubRect.bottom - hubRect.top) * sidePorts[primaryIndex];
      const cardAnchor = primaryIndex === 0 ? .95 : primaryIndex === 3 ? .05 : .5;
      const cardY = card.top + (card.bottom - card.top) * cardAnchor;
      const start = route.kind === 'incoming' ? { x: card.right, y: cardY } : { x: hubRect.right, y: portY };
      const end = route.kind === 'incoming' ? { x: hubRect.left, y: portY } : { x: card.left, y: cardY };
      route.hubProgress = route.kind === 'incoming' ? 1 : 0;
      cachePath(route, start, end);
      return;
    }
    const optional = route.path.dataset.optionalRoute;
    const optionalNode = document.querySelector(`.optional-node[data-optional="${optional}"]`);
    const face = optionalNode?.querySelector('.optional-face');
    if (!optionalNode || !face) return;
    const nodeRect = pointFor(optionalNode.getBoundingClientRect(), area);
    const faceRect = pointFor(face.getBoundingClientRect(), area);
    const top = (nodeRect.top + nodeRect.bottom) / 2 < hubCenter.y;
    const optionIndex = (top ? topPorts : bottomPorts).indexOf(optional);
    const portX = hubRect.left + (hubRect.right - hubRect.left) * [.2, .5, .8][optionIndex];
    const gap = 6;
    const start = top ? { x: (nodeRect.left + nodeRect.right) / 2, y: nodeRect.bottom + gap } : { x: portX, y: hubRect.bottom };
    const end = top ? { x: portX, y: hubRect.top } : { x: (faceRect.left + faceRect.right) / 2, y: faceRect.top - gap };
    route.hubProgress = top ? 1 : 0;
    cachePath(route, start, end, true);
  });
  geometryDirty = false;
}

function setStory(activeIndex, complete, suppliedRefs, candidate = session, live = true) {
  if (!candidate) return;
  const refs = suppliedRefs ?? { story, prefix: storyPrefix, clauses: storyClauses, alternative: storyAlternative };
  const key = `${candidate.id}:${activeIndex}:${complete}`;
  if (live && key === renderedStoryKey) return;
  if (live) renderedStoryKey = key;
  if (live) { root.dataset.session = candidate.id; root.dataset.step = String(activeIndex); }
  if (refs.story) { refs.story.dataset.session = candidate.id; refs.story.dataset.step = String(activeIndex); }
  if (refs.prefix) refs.prefix.textContent = candidate.prefix;
  if (refs.alternative) { refs.alternative.textContent = candidate.suffix; refs.alternative.dataset.state = complete ? 'complete' : 'pending'; }
  if (!refs.clauses) return;
  const renderedWords = [];
  const clauseIndex = new Map(candidate.clauses.map((item, index) => [item, index]));
  refs.clauses.replaceChildren(...candidate.tasks.map((task, workloadIndex) => {
    const taskElement = document.createElement('span');
    taskElement.className = 'story-task'; taskElement.dataset.workloadIndex = String(workloadIndex);
    if (workloadIndex) taskElement.append(' ');
    task.forEach((clauseData, taskClauseIndex) => {
      const index = clauseIndex.get(clauseData);
      const clause = document.createElement('span');
      clause.className = 'story-clause';
      clause.dataset.state = complete || index < activeIndex ? 'complete' : index === activeIndex ? 'active' : 'pending';
      if (taskClauseIndex) clause.append(taskClauseIndex === task.length - 1 ? ' and ' : ', ');
      const stepNumber = document.createElement('span');
      stepNumber.className = 'story-step-number'; stepNumber.setAttribute('aria-label', `Step ${index + 1}`); stepNumber.textContent = String(index + 1);
      clause.append(stepNumber, ' ');
      const rawWords = clauseData.text.split(/\s+/);
      const references = clauseData.references.map(reference => reference.split(/\s+/).map(normalise)).sort((left, right) => right.length - left.length);
      let offset = 0;
      const records = [];
      for (let wordIndex = 0; wordIndex < rawWords.length;) {
        const match = references.find(reference => reference.every((part, partIndex) => normaliseStoryWord(rawWords[wordIndex + partIndex] || '') === part));
        const count = match?.length || 1;
        if (wordIndex) clause.append(' ');
        const container = match ? document.createElement('span') : clause;
        if (match) container.className = 'story-product';
        for (let part = 0; part < count; part++) {
          const word = rawWords[wordIndex + part];
          const wrapper = document.createElement('span');
          const base = document.createElement('span');
          const reveal = document.createElement('span');
          wrapper.className = 'story-word'; base.className = 'story-word-base'; reveal.className = 'story-word-reveal';
          base.textContent = word; reveal.textContent = word; reveal.setAttribute('aria-hidden', 'true');
          if (part) container.append(' ');
          wrapper.append(base, reveal); container.append(wrapper);
          records.push({ reveal, start: offset, length: word.length });
          offset += word.length + 1;
        }
        if (match) clause.append(container);
        wordIndex += count;
      }
      renderedWords[index] = { words: records, length: clauseData.text.length };
      taskElement.append(clause);
    });
    const lastWord = [...taskElement.querySelectorAll('.story-word')].at(-1);
    lastWord?.querySelectorAll('.story-word-base, .story-word-reveal').forEach(word => {
      word.textContent += workloadIndex === 0 ? ';' : ',';
    });
    return taskElement;
  }));
  if (live) storyWords = renderedWords;
}

function reserveStoryHeight() {
  if (!story || !session) return;
  const styles = [story, storyPrefix, storyClauses, storyAlternative].filter(Boolean).map(node => {
    const value = getComputedStyle(node);
    return [value.font, value.lineHeight, value.letterSpacing, value.maxWidth, value.paddingInline, value.marginBlock].join('/');
  });
  const key = [story.clientWidth, document.documentElement.lang, document.documentElement.dir, ...styles].join('|');
  if (key === storyReservationKey) return;
  storyReservationKey = key;
  const probe = story.cloneNode(true);
  probe.removeAttribute('id'); probe.setAttribute('aria-hidden', 'true');
  Object.assign(probe.style, { position: 'fixed', left: '-10000px', top: '0', visibility: 'hidden', pointerEvents: 'none', width: `${story.getBoundingClientRect().width}px`, height: 'auto', minHeight: '0', maxHeight: 'none', overflow: 'visible', zIndex: '-1' });
  document.body.append(probe);
  const refs = { story: probe, prefix: probe.querySelector('[data-story-prefix]'), clauses: probe.querySelector('[data-story-clauses]'), alternative: probe.querySelector('[data-story-alternative]') };
  const measure = item => {
    const token = document.createElement('span');
    token.className = item.name ? 'story-product' : 'story-word'; token.textContent = item.name || item.text; token.style.whiteSpace = 'nowrap'; token.style.position = 'absolute';
    if (item.text) token.style.fontWeight = '720';
    refs.clauses.append(token); const width = token.getBoundingClientRect().width; token.remove(); return width;
  };
  let maximum = 0;
  for (const candidate of storyReservationSessions(variantPools(), measure)) {
    setStory(candidate.clauses.length - 1, false, refs, candidate, false);
    maximum = Math.max(maximum, probe.getBoundingClientRect().height);
    refs.alternative.style.animation = 'none'; refs.alternative.style.fontWeight = '850';
    setStory(candidate.clauses.length - 1, true, refs, candidate, false);
    maximum = Math.max(maximum, probe.getBoundingClientRect().height);
    refs.alternative.style.fontWeight = '650';
  }
  probe.remove();
  const reserved = `${Math.ceil(maximum) + 3}px`;
  story.style.setProperty('--story-reserved-height', reserved);
  story.style.height = reserved; story.style.minHeight = reserved;
  geometryDirty = true;
  refreshGeometry();
}

function scheduleStoryReservation() {
  cancelAnimationFrame(reservationFrame);
  reservationFrame = requestAnimationFrame(() => { reservationFrame = 0; reserveStoryHeight(); });
}

function updateWordReveals(activeIndex, progress, complete) {
  storyWords.forEach((group, index) => group.words.forEach(word => {
    const characters = complete || index < activeIndex ? group.length : index === activeIndex ? progress * group.length : 0;
    const reveal = Math.max(0, Math.min(100, (characters - word.start) / word.length * 100));
    word.reveal.style.setProperty('--reveal', `${reveal}%`);
  }));
}

function setActiveRoute(route) {
  const nextId = route?.id || '';
  if (nextId === activeRouteId) return;
  document.querySelectorAll('.flow-node.is-session-active, .optional-node.is-session-active').forEach(node => node.classList.remove('is-session-active'));
  if (route) {
    const nodeKey = route.kind === 'incoming' ? route.path.dataset.from : route.kind === 'outgoing' ? route.path.dataset.to : null;
    const node = nodeKey ? nodes.find(item => item.dataset.node === nodeKey) : document.querySelector(`.optional-node[data-optional="${route.path.dataset.optionalRoute}"]`);
    node?.classList.add('is-session-active');
  }
  activeRouteId = nextId;
}

function nodeForRoute(route) {
  const nodeKey = route.kind === 'incoming' ? route.path.dataset.from : route.kind === 'outgoing' ? route.path.dataset.to : null;
  return nodeKey ? nodes.find(item => item.dataset.node === nodeKey) : document.querySelector(`.optional-node[data-optional="${route.path.dataset.optionalRoute}"]`);
}

function clearDelivered() {
  deliveredSessionId = '';
  journey?.classList.remove('is-delivered');
  routes.forEach(route => { delete route.path.dataset.delivered; });
  document.querySelectorAll('.flow-node.is-session-delivered, .optional-node.is-session-delivered').forEach(node => node.classList.remove('is-session-delivered'));
}

function showDelivered() {
  if (!session || deliveredSessionId === session.id) return;
  deliveredSessionId = session.id;
  journey?.classList.add('is-delivered');
  const used = new Set(session.legs.map(leg => leg.route));
  routes.filter(route => used.has(route.id)).forEach(route => {
    route.path.dataset.delivered = 'true';
    nodeForRoute(route)?.classList.add('is-session-delivered');
  });
}

function preload(selection) {
  return Promise.all(Object.values(selection).filter(item => item.asset).map(item => new Promise(resolve => {
    const image = new Image(); image.onload = () => resolve(true); image.onerror = () => resolve(false); image.src = item.asset;
  }))).then(results => results.every(Boolean));
}

function applySelection(selection, animate = false) {
  const changes = Object.entries(selection).map(([nodeKey, variant]) => {
    const state = iconStates.get(nodeKey);
    if (!state || !variant.asset) return Promise.resolve(true);
    const commit = () => {
      state.current.src = variant.asset; state.current.alt = variant.fullName || variant.name;
      state.next.src = variant.asset; state.next.alt = '';
      if (state.name) { state.name.textContent = variant.name; state.name.title = variant.fullName || variant.name; }
      if (state.role && variant.role) state.role.textContent = variant.role;
      state.node.dataset.integrationId = variant.id || '';
      state.node.dataset.integrationGroup = variant.group || '';
      state.node.classList.remove('is-changing', 'is-committing');
    };
    if (!animate || state.current.src === new URL(variant.asset, document.baseURI).href) {
      commit();
      return Promise.resolve(true);
    }
    return new Promise(resolve => {
      let revealed = false;
      const revealIncoming = () => {
        if (revealed) return;
        revealed = true; state.next.onload = null; state.next.onerror = null;
        state.node.classList.add('is-changing');
        window.setTimeout(() => {
          commit(); state.node.classList.add('is-committing');
          window.setTimeout(() => state.node.classList.remove('is-committing'), 0);
          resolve(true);
        }, 460);
      };
      state.next.onload = revealIncoming;
      state.next.onerror = () => { state.next.onload = null; state.next.onerror = null; resolve(false); };
      state.next.src = variant.asset; state.next.alt = '';
      if (state.next.complete && state.next.naturalWidth > 0) revealIncoming();
    });
  });
  return Promise.all(changes).then(results => results.every(Boolean));
}

function variantPools() { return Object.fromEntries([...iconStates].map(([key, state]) => [key, state.variants])); }
function nextScenario(previous = null) {
  return createScenarioSession({ variantPools: variantPools(), previous, optionalSlots: optionalSchedule(), variantSelection: variantRotation(), storyChoice: storyRotation() });
}
function revealStory() {
  if (!story || story.dataset.storyReady === 'true') return;
  story.dataset.storyReady = 'true';
  story.removeAttribute('aria-hidden');
}
function beginSession(next, complete = false, animate = false, applyIcons = true) {
  clearDelivered();
  session = next; sessionElapsed = 0; renderedStoryKey = '';
  // Session setup does not wait for icon readiness; queued transitions await
  // applySelection before entering this function with applyIcons=false.
  if (applyIcons) void applySelection(session.selection, animate);
  setStory(complete ? session.clauses.length - 1 : 0, complete);
  if (complete) { updateWordReveals(session.clauses.length - 1, 1, true); showDelivered(); revealStory(); }
}
async function queueNextSession() {
  if (preparingSession) return;
  preparingSession = true;
  clearDelivered();
  const next = nextScenario(session);
  story?.classList.add('is-transitioning');
  updateFrame();
  if (!(await preload(next.selection))) {
    story?.classList.remove('is-transitioning'); preparingSession = false; updateFrame(); return;
  }
  if (!(await applySelection(next.selection, true))) {
    story?.classList.remove('is-transitioning'); preparingSession = false; updateFrame(); return;
  }
  beginSession(next, false, false, false);
  story?.classList.remove('is-transitioning');
  await new Promise(resolve => window.setTimeout(resolve, 220));
  preparingSession = false;
  updateFrame();
}

function pointAt(route, progress) {
  const position = Math.min(80, Math.max(0, progress * 80));
  const lower = Math.floor(position), upper = Math.min(80, lower + 1), fraction = position - lower;
  const start = route.samples[lower], end = route.samples[upper];
  return { x: start.x + (end.x - start.x) * fraction, y: start.y + (end.y - start.y) * fraction };
}

function easeInOut(progress) {
  const clamped = Math.min(1, Math.max(0, progress));
  return clamped * clamped * (3 - 2 * clamped);
}

function paintSession() {
  routes.forEach(route => { route.packet.style.display = 'none'; route.path.dataset.active = 'false'; });
  if (!session || isReducedMotion()) { setActiveRoute(null); hub?.style.setProperty('--activity', '0'); return; }
  if (motionEnabled() && !preparingSession) revealStory();
  const state = sessionStep(session, sessionElapsed);
  setStory(state.clauseIndex, state.settled);
  const route = routes.find(item => item.id === state.leg.route);
  if (!route?.samples.length || state.settled) {
    updateWordReveals(state.clauseIndex, state.clauseProgress, state.settled);
    setActiveRoute(null);
    if (state.settled) showDelivered();
    hub?.style.setProperty('--activity', '.1');
    return;
  }
  setActiveRoute(route);
  route.path.dataset.active = 'true'; route.packet.style.display = '';
  route.packet.dataset.step = String(state.clauseIndex + 1); route.markerLabel.textContent = String(state.clauseIndex + 1);
  const phase = easeInOut(state.progress);
  const clauseProgress = Math.min(1, (state.leg.clauseLegIndex + phase) / state.leg.clauseLegCount);
  updateWordReveals(state.clauseIndex, clauseProgress, false);
  const progress = state.leg.direction === 'reverse' ? 1 - phase : phase;
  const point = pointAt(route, progress);
  route.packet.setAttribute('transform', `translate(${point.x} ${point.y})`);
  hub?.style.setProperty('--activity', String(.1 + Math.max(0, 1 - Math.abs(progress - route.hubProgress) * 5) * .7));
}

function tick(timestamp) {
  frame = 0;
  if (!motionEnabled()) return;
  sessionElapsed += lastTime === null ? 0 : timestamp - lastTime; lastTime = timestamp;
  if (geometryDirty) refreshGeometry();
  if (timestamp - lastPaint >= 1000 / 30) { paintSession(); lastPaint = timestamp; }
  if (session && sessionElapsed >= session.duration) { void queueNextSession().catch(() => { story?.classList.remove('is-transitioning'); preparingSession = false; updateFrame(); }); return; }
  if (motionEnabled()) frame = requestAnimationFrame(tick);
}

function updateFrame() {
  cancelAnimationFrame(frame); frame = 0; lastTime = null; updateMotionState();
  if (motionEnabled() && !preparingSession) frame = requestAnimationFrame(tick);
}

initialiseNodes();
variantRotation = createVariantRotation({ variantPools: variantPools() });
initialiseRoutes();
preparingSession = true;
async function prepareFirstScene() {
  journey?.setAttribute('aria-busy', 'true');
  await document.fonts?.ready;
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  const firstSession = isReducedMotion() ? staticScenario(variantPools()) : nextScenario();
  session = firstSession;
  reserveStoryHeight();
  const ready = await preload(firstSession.selection);
  if (ready) {
    beginSession(firstSession, isReducedMotion(), false);
  } else {
    beginSession(staticScenario(variantPools()), true, false);
  }
  root.dataset.layoutReady = 'true';
  journey?.setAttribute('aria-busy', 'false');
  preparingSession = false;
  updateFrame();
}
void prepareFirstScene().catch(() => { journey?.setAttribute('aria-busy', 'false'); preparingSession = false; updateFrame(); });
journey?.addEventListener('focusin', () => { pausedByFocus = true; updateFrame(); });
journey?.addEventListener('focusout', () => { pausedByFocus = false; updateFrame(); });
document.addEventListener('keydown', event => { if (event.key === 'Escape' && !event.defaultPrevented) { pausedByEscape = !pausedByEscape; updateFrame(); } });
document.addEventListener('visibilitychange', updateFrame);
window.addEventListener('appearance:reduced-motion', () => { if (root.dataset.layoutReady !== 'true') return; if (isReducedMotion()) { const staticSession = staticScenario(variantPools()); void preload(staticSession.selection).then(ready => { if (ready) { beginSession(staticSession, true); paintSession(); } updateFrame(); }).catch(() => updateFrame()); } else updateFrame(); });
window.addEventListener('resize', scheduleStoryReservation, { passive: true });
window.addEventListener('localechange', () => { storyReservationKey = ''; scheduleStoryReservation(); });
document.fonts?.addEventListener('loadingdone', () => { storyReservationKey = ''; scheduleStoryReservation(); });
const resizeObserver = new ResizeObserver(() => { geometryDirty = true; if (!motionEnabled()) refreshGeometry(); });
if (ecosystem) resizeObserver.observe(ecosystem);
if ('IntersectionObserver' in window) new IntersectionObserver(entries => { inView = entries[0].isIntersecting; updateFrame(); }, { threshold: .15 }).observe(journey); else inView = true;
updateFrame();
