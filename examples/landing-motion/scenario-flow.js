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

function choose(list, random) { return list[Math.floor(random() * list.length)]; }
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
function clause(text, references, legs, workloadIndex) { return Object.freeze({ text, references: Object.freeze(references), legs: Object.freeze(legs.map(leg => Object.freeze(leg))), workloadIndex }); }
function personaPrefix(persona) {
  const context = persona.prefix.replace(/,\s*$/, '');
  const needs = `${TARGET_INTENTS[persona.targets[0]]} and ${TARGET_INTENTS[persona.targets[1]]}`;
  return persona.id === 'ai-model'
    ? `${context}, my human operator needs to ${needs},`
    : `${context}, I need to ${needs},`;
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
export function createScenarioSession({ variantPools = {}, previous = null, random = Math.random, optionalProbability = OPTIONAL_PROBABILITY } = {}) {
  const persona = choose(PERSONAS.filter(item => item.id !== previous?.id), random);
  const selection = {};
  selection.identity = pickVariant(variantPools.identity, previous?.selection?.identity?.asset, random);
  selection['identity-2'] = pickVariant(variantPools['identity-2'], previous?.selection?.['identity-2']?.asset, random, new Set([selection.identity.asset]));
  Object.entries(variantPools).filter(([node]) => node !== 'identity' && node !== 'identity-2').forEach(([node, variants]) => { selection[node] = pickVariant(variants, previous?.selection?.[node]?.asset, random); });
  const [targetOne, targetTwo] = persona.targets;
  const firstOptional = random() < optionalProbability ? choose(OPTIONALS, random) : null;
  const secondOptional = random() < optionalProbability ? choose(OPTIONALS.filter(optional => optional.route !== firstOptional?.route), random) : null;
  const firstIdentity = choose(['identity', 'identity-2'], random);
  const firstSource = choose(['local', 'remote'], random);
  const secondIdentity = firstIdentity === 'identity' ? 'identity-2' : 'identity';
  const secondSource = firstSource === 'local' ? 'remote' : 'local';
  const tasks = [
    task({ identity: firstIdentity, source: firstSource, target: targetOne, optional: firstOptional, lead: `${persona.leadIn || 'all I have to do is'}`, selection, workloadIndex: 0, action: TARGET_ACTIONS[targetOne] }),
    task({ identity: secondIdentity, source: secondSource, target: targetTwo, optional: secondOptional, lead: 'then', selection, workloadIndex: 1, action: TARGET_ACTIONS[targetTwo] }),
  ];
  const clauses = tasks.flat();
  const legs = clauses.flatMap((item, clauseIndex) => item.legs.map((leg, clauseLegIndex) => Object.freeze({ ...leg, clauseIndex, clauseLegIndex, clauseLegCount: item.legs.length, workloadIndex: item.workloadIndex })));
  return Object.freeze({ id: persona.id, prefix: personaPrefix(persona), suffix: 'or I can use CircularQuay.', selection: Object.freeze(selection), tasks: Object.freeze(tasks), clauses: Object.freeze(clauses), legs: Object.freeze(legs), duration: legs.length * STEP_DURATION + SESSION_SETTLE_DURATION });
}

export function staticScenario(variantPools = {}) { return createScenarioSession({ variantPools, previous: { id: 'ai-model', selection: {} }, random: () => 0, optionalProbability: 0 }); }
/** A bounded, conservative set used only to reserve story height before playback. */
export function storyReservationSessions(variantPools = {}, measure = item => item.name?.length || item.text?.length || 0) {
  const selection = Object.fromEntries(Object.entries(variantPools).map(([node, variants]) => [node, longestVariant(variants, new Set(), measure)]));
  selection.identity = longestVariant(variantPools.identity, new Set(), measure);
  selection['identity-2'] = longestVariant(variantPools['identity-2'], new Set([selection.identity.asset]), measure);
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
      task({ identity: firstIdentity, source: firstSource, target: targetOne, optional: firstOptional, lead: persona.leadIn || 'all I have to do is', selection, workloadIndex: 0, action: TARGET_ACTIONS[targetOne] }),
      task({ identity: secondIdentity, source: secondSource, target: targetTwo, optional: secondOptional, lead: 'then', selection, workloadIndex: 1, action: TARGET_ACTIONS[targetTwo] }),
    ];
    const clauses = tasks.flat();
    return Object.freeze({ id: `reserve-${persona.id}-${order}-${firstOptional?.route || 'none'}-${secondOptional?.route || 'none'}`, prefix: personaPrefix(persona), suffix: 'or I can use CircularQuay.', selection: Object.freeze(selection), tasks: Object.freeze(tasks), clauses: Object.freeze(clauses) });
  }))));
}
export function sessionStep(session, elapsed) {
  const index = Math.min(session.legs.length - 1, Math.max(0, Math.floor(elapsed / STEP_DURATION)));
  const leg = session.legs[index];
  const progress = Math.min(1, Math.max(0, (elapsed - index * STEP_DURATION) / STEP_DURATION));
  return Object.freeze({ index, leg, progress, clauseIndex: leg.clauseIndex, clauseProgress: Math.min(1, (leg.clauseLegIndex + progress) / leg.clauseLegCount), settled: elapsed >= session.legs.length * STEP_DURATION });
}
export const PERSONA_COUNT = PERSONAS.length;
