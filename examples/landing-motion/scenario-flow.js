export const STEP_DURATION = 1200;
export const SESSION_SETTLE_DURATION = 2200;
const OPTIONAL_PROBABILITY = .25;

const PERSONAS = [
  { id: 'new-hire', prefix: 'As a new hire on day one,', targets: ['shell', 'ci'] },
  { id: 'frontend-developer', prefix: 'As a frontend developer opening one last tab,', targets: ['cloud', 'ci'] },
  { id: 'backend-developer', prefix: 'As a backend developer following one request,', targets: ['cloud', 'container'] },
  { id: 'mobile-developer', prefix: 'As a mobile developer with a build waiting,', targets: ['ci', 'cloud'] },
  { id: 'friday-developer', prefix: 'As a developer shipping on Friday,', targets: ['cloud', 'ci'] },
  { id: 'sre', prefix: 'As an SRE reading one more dashboard,', targets: ['shell', 'cloud'] },
  { id: 'on-call', prefix: 'As the on-call engineer at 2 am,', targets: ['shell', 'container'] },
  { id: 'release-engineer', prefix: 'As a release engineer with one last pipeline,', targets: ['ci', 'cloud'] },
  { id: 'platform-engineer', prefix: 'As a platform engineer adding “just one service”,', targets: ['container', 'cloud'] },
  { id: 'qa-engineer', prefix: 'As a QA engineer with a helpful reproduction,', targets: ['ci', 'shell'] },
  { id: 'data-engineer', prefix: 'As a data engineer rerunning a familiar job,', targets: ['cloud', 'ci'] },
  { id: 'researcher', prefix: 'As a researcher rerunning yesterday’s job,', targets: ['shell', 'cloud'] },
  { id: 'contractor', prefix: 'As a contractor here for a week,', targets: ['shell', 'cloud'] },
  { id: 'maintainer', prefix: 'As a maintainer finding an old config,', targets: ['ci', 'container'] },
  { id: 'consultant', prefix: 'As a consultant joining a new project,', targets: ['cloud', 'shell'] },
  { id: 'support-engineer', prefix: 'As a support engineer tracing a customer report,', targets: ['shell', 'cloud'] },
  { id: 'automation-maintainer', prefix: 'As an automation maintainer fixing a scheduled run,', targets: ['ci', 'container'] },
  { id: 'homelab-owner', prefix: 'As a homelab owner adding one more container,', targets: ['container', 'shell'] },
  { id: 'security-reviewer', prefix: 'As a security reviewer opening another request,', targets: ['cloud', 'ci'] },
  { id: 'administrator', prefix: 'As an administrator rotating access again,', targets: ['cloud', 'shell'] },
  { id: 'devops-engineer', prefix: 'As a DevOps engineer tidying a deployment,', targets: ['container', 'ci'] },
  { id: 'incident-coordinator', prefix: 'As an incident coordinator collecting the next clue,', targets: ['shell', 'cloud'] },
  { id: 'product-engineer', prefix: 'As a product engineer trying a small change,', targets: ['cloud', 'ci'] },
  { id: 'ai-model', prefix: 'As an AI model,', targets: ['cloud', 'container'], leadIn: 'all my human operator has to do is' },
];
const PERSONA_ROLES = { 'new-hire': 'a new hire', 'frontend-developer': 'a frontend developer', 'backend-developer': 'a backend developer', 'mobile-developer': 'a mobile developer', 'friday-developer': 'a developer', sre: 'an SRE', 'on-call': 'the on-call engineer', 'release-engineer': 'a release engineer', 'platform-engineer': 'a platform engineer', 'qa-engineer': 'a QA engineer', 'data-engineer': 'a data engineer', researcher: 'a researcher', contractor: 'a contractor', maintainer: 'a maintainer', consultant: 'a consultant', 'support-engineer': 'a support engineer', 'automation-maintainer': 'an automation maintainer', 'homelab-owner': 'a homelab owner', 'security-reviewer': 'a security reviewer', administrator: 'an administrator', 'devops-engineer': 'a DevOps engineer', 'incident-coordinator': 'an incident coordinator', 'product-engineer': 'a product engineer', 'ai-model': 'an AI model' };
const STORY_CONTEXTS = ['on day one', 'during a Friday release', 'after another handoff', 'with one last pipeline', 'while tracing an incident', 'before the next deploy', 'on a short project', 'with a reproducible job', 'after a small config change', 'while adding one more service', 'between support requests', 'with a waiting build'];
const TARGET_ACTIONS = {
  cloud: name => `deploy to ${name}`,
  container: name => `run on ${name}`,
  shell: name => `run ${name}`,
  ci: name => `run a job in ${name}`,
};
const TARGET_INTENTS = {
  cloud: 'deploy an app',
  container: 'run a container',
  shell: 'open a terminal',
  ci: 'run a pipeline',
};
const OPTIONALS = [
  { route: 'optional-age', request: 'reverse', response: 'forward', text: 'use an age key', references: ['age'] },
  { route: 'optional-kms', request: 'reverse', response: 'forward', text: 'request KMS access', references: ['KMS'] },
  { route: 'optional-signed', request: 'reverse', response: 'forward', text: 'check signing', references: [] },
  { route: 'optional-directory', request: 'forward', response: 'reverse', text: 'read SCIM groups', references: ['SCIM'] },
  { route: 'optional-clients', request: 'forward', response: 'reverse', text: 'limit MCP access', references: ['MCP'] },
  { route: 'optional-audit', request: 'forward', response: 'reverse', text: 'log an audit event', references: [] },
];

export function createOptionalSchedule({ random = Math.random } = {}) {
  const masks = [[], [0], [1], [0, 1]];
  let first = true;
  let bag = [];
  const shuffle = values => {
    const copy = values.map(mask => [...mask]);
    for (let index = copy.length - 1; index > 0; index--) {
      const swap = Math.floor(random() * (index + 1));
      [copy[index], copy[swap]] = [copy[swap], copy[index]];
    }
    return copy;
  };
  return function nextMask() {
    if (!bag.length) {
      if (first) {
        first = false;
        const initial = random() < .5 ? [0] : [1];
        bag = [initial, ...shuffle(masks.filter(mask => mask.join() !== initial.join()))];
      } else bag = shuffle(masks);
    }
    return [...bag.shift()];
  };
}

export function createStoryRotation({ random = Math.random } = {}) {
  let actors = [], contexts = [], lastActor = '', lastContext = '';
  const refill = (values, last) => shuffled(values, random, last).map(value => typeof value === 'string' ? value : value.id);
  return function nextChoice() {
    if (!actors.length) actors = refill(PERSONAS, lastActor);
    if (!contexts.length) contexts = refill(STORY_CONTEXTS, lastContext);
    const personaId = actors.shift(); const context = contexts.shift();
    lastActor = personaId; lastContext = context;
    return Object.freeze({ personaId, context });
  };
}

function choose(list, random) { return list[Math.floor(random() * list.length)]; }
function shuffled(list, random, previousId = '') {
  const copy = [...list];
  for (let index = copy.length - 1; index > 0; index--) { const swap = Math.floor(random() * (index + 1)); [copy[index], copy[swap]] = [copy[swap], copy[index]]; }
  if (copy.length > 1 && (copy[0]?.id || copy[0]) === previousId) [copy[0], copy[1]] = [copy[1], copy[0]];
  return copy;
}
function validVariant(item, pool) { return pool?.some(candidate => candidate?.asset === item?.asset && candidate?.id === item?.id); }

/** Rotates real integration variants without exhausting a group invisibly. */
export function createVariantRotation({ variantPools = {}, random = Math.random } = {}) {
  const bags = new Map();
  const last = new Map();
  const draw = (group, pool, unavailable = new Set()) => {
    const candidates = (pool || []).filter(item => item?.id && item?.name && item?.asset && !unavailable.has(item.id));
    if (!candidates.length) return Object.freeze({ id: 'illustrative-example', name: 'illustrative example', asset: '' });
    let bag = bags.get(group) || [];
    if (!bag.length || !bag.some(item => candidates.some(candidate => candidate.id === item.id))) bag = shuffled(candidates, random, last.get(group));
    let index = bag.findIndex(item => !unavailable.has(item.id));
    if (index < 0) { bag = shuffled(candidates, random, last.get(group)); index = 0; }
    const [item] = bag.splice(index, 1); bags.set(group, bag); last.set(group, item.id); return Object.freeze(item);
  };
  return function next() {
    const identity = draw('identity', variantPools.identity);
    const identity2 = draw('identity', variantPools['identity-2'] || variantPools.identity, new Set([identity.id]));
    const local = draw('sources', variantPools.local);
    const remote = draw('sources', variantPools.remote || variantPools.local, new Set([local.id]));
    const selection = { identity, 'identity-2': identity2, local, remote };
    for (const node of ['shell', 'container', 'cloud', 'ci']) selection[node] = draw(`application:${node}`, variantPools[node]);
    return Object.freeze(selection);
  };
}
function pickVariant(list, previousAsset, random, unavailable = new Set()) {
  const all = Array.isArray(list) ? list.filter(item => item?.name && item?.asset) : [];
  const available = all.filter(item => !unavailable.has(item.asset));
  const candidates = available.length ? available : all;
  if (!candidates.length) return Object.freeze({ name: 'illustrative example', asset: '' });
  const rotated = candidates.filter(item => item.asset !== previousAsset);
  return Object.freeze(choose(rotated.length ? rotated : candidates, random));
}
function longestVariant(list, unavailable = new Set(), measure = item => item.name.length) {
  const available = (Array.isArray(list) ? list : []).filter(item => item?.name && item?.asset && !unavailable.has(item.asset));
  return Object.freeze([...available].sort((left, right) => measure(right) - measure(left))[0] || { name: 'illustrative example', asset: '' });
}
function reservationVariant(list, unavailable = new Set(), measure = item => item.name?.length || item.text?.length || 0) {
  const candidates = (Array.isArray(list) ? list : []).filter(item => item?.name && item?.asset && !unavailable.has(item.asset));
  const nameVariant = longestVariant(candidates, new Set(), measure);
  const intent = candidates.map(item => item.intent).filter(Boolean).sort((left, right) => measure({ text: right }) - measure({ text: left }))[0];
  const action = candidates.map(item => item.action?.template || item.action).filter(template => typeof template === 'string').sort((left, right) => measure({ text: right.replaceAll('{name}', nameVariant.name) }) - measure({ text: left.replaceAll('{name}', nameVariant.name) }))[0];
  return Object.freeze({ ...nameVariant, ...(intent ? { intent } : {}), ...(action ? { action: { template: action } } : {}) });
}
function clause(text, references, legs, workloadIndex) { return Object.freeze({ text, references: Object.freeze(references), legs: Object.freeze(legs.map(leg => Object.freeze(leg))), workloadIndex }); }
function personaPrefix(persona, selection = {}, storyChoice = null) {
  const context = persona.prefix.replace(/,\s*$/, '');
  const needs = persona.targets.map(target => selection[target]?.intent || TARGET_INTENTS[target]).join(' and ');
  if (storyChoice?.context) {
    const role = PERSONA_ROLES[persona.id] || context.replace(/^As /, '');
    return persona.id === 'ai-model'
      ? `As an AI model ${storyChoice.context}, my human operator needs to ${needs},`
      : `As ${role} ${storyChoice.context}, I need to ${needs},`;
  }
  return persona.id === 'ai-model'
    ? `${context}, my human operator needs to ${needs},`
    : `${context}, I need to ${needs},`;
}
function targetAction(target, variant) {
  const template = variant?.action?.template || variant?.action || TARGET_ACTIONS[target];
  return name => typeof template === 'function' ? template(name) : String(template).replaceAll('{name}', name);
}
function task({ identity, source, target, optional, lead, selection, workloadIndex, action }) {
  const steps = [
    clause(`${lead} sign in to ${selection[identity].name}`, [selection[identity].name], [{ route: identity, direction: 'reverse' }, { route: identity, direction: 'forward' }], workloadIndex),
    clause(`open the ${selection[source].name} vault`, [selection[source].name], [{ route: source, direction: 'reverse' }, { route: source, direction: 'forward' }], workloadIndex),
  ];
  if (optional) steps.push(clause(optional.text, optional.references, [{ route: optional.route, direction: optional.request }, { route: optional.route, direction: optional.response }], workloadIndex));
  steps.push(clause(action(selection[target].name), [selection[target].name], [{ route: `out-${target}`, direction: 'forward' }, { route: `out-${target}`, direction: 'reverse' }], workloadIndex));
  return Object.freeze(steps);
}

/** Creates two connected workloads from only page-supplied variants. */
export function createScenarioSession({ variantPools = {}, previous = null, random = Math.random, optionalProbability = OPTIONAL_PROBABILITY, optionalSlots, variantSelection, storyChoice } = {}) {
  const persona = PERSONAS.find(item => item.id === storyChoice?.personaId) || choose(PERSONAS.filter(item => item.id !== previous?.id), random);
  const selection = {};
  selection.identity = validVariant(variantSelection?.identity, variantPools.identity) ? variantSelection.identity : pickVariant(variantPools.identity, previous?.selection?.identity?.asset, random);
  selection['identity-2'] = validVariant(variantSelection?.['identity-2'], variantPools['identity-2']) && variantSelection['identity-2'].asset !== selection.identity.asset ? variantSelection['identity-2'] : pickVariant(variantPools['identity-2'], previous?.selection?.['identity-2']?.asset, random, new Set([selection.identity.asset]));
  Object.entries(variantPools).filter(([node]) => node !== 'identity' && node !== 'identity-2').forEach(([node, variants]) => { selection[node] = validVariant(variantSelection?.[node], variants) && (node !== 'remote' || variantSelection[node].asset !== selection.local?.asset) ? variantSelection[node] : pickVariant(variants, previous?.selection?.[node]?.asset, random, node === 'remote' ? new Set([selection.local?.asset]) : new Set()); });
  const [targetOne, targetTwo] = persona.targets;
  const slots = Array.isArray(optionalSlots) ? new Set(optionalSlots.filter(slot => slot === 0 || slot === 1)) : null;
  const firstOptional = slots ? (slots.has(0) ? choose(OPTIONALS, random) : null) : (random() < optionalProbability ? choose(OPTIONALS, random) : null);
  const secondOptional = slots ? (slots.has(1) ? choose(OPTIONALS.filter(optional => optional.route !== firstOptional?.route), random) : null) : (random() < optionalProbability ? choose(OPTIONALS.filter(optional => optional.route !== firstOptional?.route), random) : null);
  const firstIdentity = choose(['identity', 'identity-2'], random);
  const firstSource = choose(['local', 'remote'], random);
  const secondIdentity = firstIdentity === 'identity' ? 'identity-2' : 'identity';
  const secondSource = firstSource === 'local' ? 'remote' : 'local';
  const tasks = [
    task({ identity: firstIdentity, source: firstSource, target: targetOne, optional: firstOptional, lead: `${persona.leadIn || 'all I have to do is'}`, selection, workloadIndex: 0, action: targetAction(targetOne, selection[targetOne]) }),
    task({ identity: secondIdentity, source: secondSource, target: targetTwo, optional: secondOptional, lead: 'then', selection, workloadIndex: 1, action: targetAction(targetTwo, selection[targetTwo]) }),
  ];
  const clauses = tasks.flat();
  const legs = clauses.flatMap((item, clauseIndex) => item.legs.map((leg, clauseLegIndex) => Object.freeze({ ...leg, clauseIndex, clauseLegIndex, clauseLegCount: item.legs.length, workloadIndex: item.workloadIndex })));
  return Object.freeze({ id: persona.id, prefix: personaPrefix(persona, selection, storyChoice), suffix: 'or I can use CircularQuay.', selection: Object.freeze(selection), tasks: Object.freeze(tasks), clauses: Object.freeze(clauses), legs: Object.freeze(legs), duration: legs.length * STEP_DURATION + SESSION_SETTLE_DURATION });
}

export function staticScenario(variantPools = {}) { return createScenarioSession({ variantPools, previous: { id: 'ai-model', selection: {} }, random: () => 0, optionalProbability: 0 }); }
/** A bounded, conservative set used only to reserve story height before playback. */
export function storyReservationSessions(variantPools = {}, measure = item => item.name?.length || item.text?.length || 0) {
  const selection = Object.fromEntries(Object.entries(variantPools).map(([node, variants]) => [node, reservationVariant(variants, new Set(), measure)]));
  selection.identity = reservationVariant(variantPools.identity, new Set(), measure);
  selection['identity-2'] = reservationVariant(variantPools['identity-2'], new Set([selection.identity.asset]), measure);
  const optionalByLength = [...OPTIONALS].sort((left, right) => measure(right) - measure(left));
  const [firstOption, secondOption] = optionalByLength;
  const modes = [[null, null], [firstOption, secondOption]];
  return Object.freeze(PERSONAS.flatMap(persona => modes.flatMap(([firstOptional, secondOptional]) => ['forward', 'reverse'].map(order => {
    const [targetOne, targetTwo] = persona.targets;
    const firstIdentity = order === 'forward' ? 'identity' : 'identity-2';
    const secondIdentity = order === 'forward' ? 'identity-2' : 'identity';
    const firstSource = order === 'forward' ? 'local' : 'remote';
    const secondSource = order === 'forward' ? 'remote' : 'local';
    const tasks = [
      task({ identity: firstIdentity, source: firstSource, target: targetOne, optional: firstOptional, lead: persona.leadIn || 'all I have to do is', selection, workloadIndex: 0, action: targetAction(targetOne, selection[targetOne]) }),
      task({ identity: secondIdentity, source: secondSource, target: targetTwo, optional: secondOptional, lead: 'then', selection, workloadIndex: 1, action: targetAction(targetTwo, selection[targetTwo]) }),
    ];
    const clauses = tasks.flat();
    const longestContext = [...STORY_CONTEXTS].sort((left, right) => measure({ text: right }) - measure({ text: left }))[0];
    return Object.freeze({ id: `reserve-${persona.id}-${order}-${firstOptional?.route || 'none'}-${secondOptional?.route || 'none'}`, prefix: personaPrefix(persona, selection, { context: longestContext }), suffix: 'or I can use CircularQuay.', selection: Object.freeze(selection), tasks: Object.freeze(tasks), clauses: Object.freeze(clauses) });
  }))));
}
export function sessionStep(session, elapsed) {
  const index = Math.min(session.legs.length - 1, Math.max(0, Math.floor(elapsed / STEP_DURATION)));
  const leg = session.legs[index];
  const progress = Math.min(1, Math.max(0, (elapsed - index * STEP_DURATION) / STEP_DURATION));
  return Object.freeze({ index, leg, progress, clauseIndex: leg.clauseIndex, clauseProgress: Math.min(1, (leg.clauseLegIndex + progress) / leg.clauseLegCount), settled: elapsed >= session.legs.length * STEP_DURATION });
}
export const PERSONA_COUNT = PERSONAS.length;
