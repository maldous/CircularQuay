import { readFile, writeFile } from 'node:fs/promises';

const directory = new URL('.', import.meta.url);
const catalog = JSON.parse(await readFile(new URL('integrations.json', directory), 'utf8'));
const expectedCounts = { identity: 30, sources: 35, applications: 48 };
const expectedTotal = 113;
const seen = new Set();

const formations = {
  desktop: {
    // Extra room lets the low-frequency warp retain a real 4px tile-edge gap.
    width: 760, height: 600, step: 47.2, tile: 34, padding: 23,
    // Includes the device label beneath the 112px hub disc.
    hub: { left: 330, top: 230, right: 470, bottom: 390, clearance: 4 },
    targets: { identity: [185, 150], sources: [185, 455], applications: [595, 300] },
  },
  mobile: {
    // This is intentionally denser than desktop but still validates every rendered tile gap.
    width: 350, height: 400, step: 31.9, tile: 22, padding: 16,
    // Includes the device label beneath the mobile hub disc.
    hub: { left: 132, top: 150, right: 218, bottom: 270, clearance: 4 },
    targets: { identity: [77, 90], sources: [77, 312], applications: [278, 200] },
  },
};

function escape(value) {
  return String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;').replaceAll('"', '&quot;');
}

function makeHtml(indexSource) {
  let html = indexSource;
  for (const group of catalog.groups) {
    if (group.items.length !== expectedCounts[group.id]) throw new Error(`Incomplete ${group.id} inventory`);
    const items = group.items.map(item => {
      if (seen.has(item.id)) throw new Error(`Duplicate integration ID: ${item.id}`);
      seen.add(item.id);
      if (!item.asset) throw new Error(`Missing local asset: ${item.id}`);
      const image = `<img src="${escape(item.asset)}" alt="${escape(item.name)}" title="${escape(item.name)}" width="28" height="28" decoding="async">`;
      return `                <li class="integration-token" data-integration-id="${escape(item.id)}" data-group="${group.id}" data-name="${escape(item.name)}">${image}</li>`;
    }).join('\n');
    const start = `<!-- integrations:${group.id}:start -->`;
    const end = `<!-- integrations:${group.id}:end -->`;
    const first = html.indexOf(start);
    const last = html.indexOf(end);
    if (first < 0 || last < first) throw new Error(`Missing ${group.id} inventory markers`);
    html = html.slice(0, first + start.length) + '\n' + items + '\n              ' + html.slice(last);
  }
  if (seen.size !== expectedTotal) throw new Error(`The original ${expectedTotal}-entry inventory must be complete`);
  return html;
}

function smoothWarp(x, y, config) {
  const mobile = config.width === 350;
  const amplitude = mobile ? 1.25 : 3.35;
  const scale = mobile ? .034 : .019;
  return {
    x: x + amplitude * (Math.sin(x * scale + y * scale * .61) * .63 + Math.cos(y * scale * .79) * .37),
    y: y + amplitude * (Math.cos(y * scale * .91 - x * scale * .42) * .66 + Math.sin(x * scale * .57) * .34),
  };
}

function hexCandidates({ width, height, step, padding, tile, hub, ...config }) {
  const row = step * Math.sqrt(3) / 2;
  const half = tile / 2;
  const candidates = [];
  for (let y = padding + half, index = 0; y <= height - padding - half; y += row, index++) {
    const start = padding + half + (index % 2 ? step / 2 : 0);
    for (let x = start; x <= width - padding - half; x += step) {
      const point = smoothWarp(x, y, { width, ...config });
      const clearance = hub.clearance ?? 0;
      const outsideHub = point.x < hub.left - half - clearance || point.x > hub.right + half + clearance || point.y < hub.top - half - clearance || point.y > hub.bottom + half + clearance;
      if (outsideHub) candidates.push(point);
    }
  }
  return candidates;
}

function deterministicOrder(candidate, target, group, config) {
  const [tx, ty] = target;
  const dx = candidate.x - tx;
  const dy = candidate.y - ty;
  const [sx, sy] = group === 'applications' ? [1.12, .93] : [1, 1.04];
  const distance = (dx / sx) ** 2 + (dy / sy) ** 2;
  const [hx, hy] = [config.width / 2, config.height / 2];
  const toHubX = hx - tx;
  const toHubY = hy - ty;
  const hubLength = Math.hypot(toHubX, toHubY);
  const towardHub = (dx * toHubX + dy * toHubY) / hubLength;
  const sideways = (dx * -toHubY + dy * toHubX) / hubLength;
  // A tapered bias curves each cloud toward the centre without turning it into a panel or disc.
  const tapered = -towardHub * (group === 'applications' ? .13 : .18) + Math.abs(sideways) * .035;
  const terrain = Math.sin(candidate.x * .071 + candidate.y * .049 + group.length) * 2.2
    + Math.cos(candidate.y * .057 - candidate.x * .031) * 1.4;
  return distance + tapered + terrain;
}

function tileRatio(id) {
  let hash = 2166136261;
  for (const character of id) hash = Math.imul(hash ^ character.charCodeAt(0), 16777619);
  return .96 + ((hash >>> 0) % 801) / 10000;
}

function tileRadius(item, config) {
  return config.tile * tileRatio(item.id) / 2;
}

function rectangleGap(first, second, firstRadius, secondRadius) {
  const x = Math.max(0, Math.abs(first.x - second.x) - firstRadius - secondRadius);
  const y = Math.max(0, Math.abs(first.y - second.y) - firstRadius - secondRadius);
  return Math.hypot(x, y);
}

function legalPoint(point, item, selected, config) {
  const radius = tileRadius(item, config);
  if (point.x - radius < 4 || config.width - point.x - radius < 4 || point.y - radius < 4 || config.height - point.y - radius < 4) return false;
  const hub = config.hub;
  const gapX = point.x < hub.left ? hub.left - point.x - radius : point.x > hub.right ? point.x - hub.right - radius : -radius;
  const gapY = point.y < hub.top ? hub.top - point.y - radius : point.y > hub.bottom ? point.y - hub.bottom - radius : -radius;
  if (Math.max(gapX, gapY) < (hub.clearance ?? 0)) return false;
  return selected.every(entry => rectangleGap(point, entry.point, radius, entry.radius) >= 4);
}

function generateFormation(config) {
  const candidates = hexCandidates(config);
  const selected = [];
  const byId = new Map();
  for (const group of catalog.groups) {
    const target = config.targets[group.id];
    const ranked = [...candidates].sort((a, b) => deterministicOrder(a, target, group.id, config) - deterministicOrder(b, target, group.id, config));
    for (const item of group.items) {
      const point = ranked.find(candidate => !selected.some(entry => entry.point === candidate) && legalPoint(candidate, item, selected, config));
      if (!point) throw new Error(`Not enough legal ${group.id} formation points`);
      selected.push({ point, radius: tileRadius(item, config), item });
      byId.set(item.id, point);
    }
  }
  return { selected, byId };
}

function nearestDistance(points) {
  let nearest = Infinity;
  for (let first = 0; first < points.length; first++) for (let second = first + 1; second < points.length; second++) {
    nearest = Math.min(nearest, Math.hypot(points[first].x - points[second].x, points[first].y - points[second].y));
  }
  return nearest;
}

function proof(config, points) {
  const half = config.tile / 2;
  const minEdge = Math.min(...points.flatMap(point => [point.x - half, config.width - point.x - half, point.y - half, config.height - point.y - half]));
  const minHub = Math.min(...points.map(point => {
    const dx = point.x < config.hub.left ? config.hub.left - point.x - half : point.x > config.hub.right ? point.x - config.hub.right - half : -half;
    const dy = point.y < config.hub.top ? config.hub.top - point.y - half : point.y > config.hub.bottom ? point.y - config.hub.bottom - half : -half;
    return Math.max(dx, dy);
  }));
  const minimum = nearestDistance(points);
  if (minimum + .001 < config.step) throw new Error(`Formation separation ${minimum} is below ${config.step}`);
  if (minEdge < 0) throw new Error(`Formation escapes canvas by ${minEdge}`);
  if (minHub + .001 < (config.hub.clearance ?? 0)) throw new Error(`Formation is too close to the protected hub: ${minHub}`);
  return { count: points.length, minCenterSeparation: minimum, minCanvasEdgeClearance: minEdge, minHubClearance: minHub };
}

function declaration(id, desktop, mobile) {
  const d = desktop.byId.get(id);
  const m = mobile.byId.get(id);
  return `[data-integration-id="${id}"]{--home-x:${(d.x / formations.desktop.width * 100).toFixed(4)}%;--home-y:${(d.y / formations.desktop.height * 100).toFixed(4)}%;--mobile-home-x:${(m.x / formations.mobile.width * 100).toFixed(4)}%;--mobile-home-y:${(m.y / formations.mobile.height * 100).toFixed(4)}%;}`;
}

function makeFormationCss(desktop, mobile, desktopProof, mobileProof) {
  const declarations = catalog.groups.flatMap(group => group.items).map(item => declaration(item.id, desktop, mobile)).join('\n');
  return `/* Generated by render-integrations.mjs. Do not hand-position individual IDs.\n * Desktop proof: ${JSON.stringify(desktopProof)}\n * Mobile proof: ${JSON.stringify(mobileProof)}\n */\n\n${declarations}\n\n@media (max-width:500px){.integration-token{--home-x:var(--mobile-home-x);--home-y:var(--mobile-home-y)}}\n`;
}

const index = new URL('index.html', directory);
const formationFile = new URL('formations.css', directory);
const indexSource = await readFile(index, 'utf8');
const renderedHtml = makeHtml(indexSource);
const desktop = generateFormation(formations.desktop);
const mobile = generateFormation(formations.mobile);
const desktopProof = proof(formations.desktop, desktop.selected);
const mobileProof = proof(formations.mobile, mobile.selected);
const renderedCss = makeFormationCss(desktop, mobile, desktopProof, mobileProof);

if (process.argv.includes('--check')) {
  if (indexSource !== renderedHtml) throw new Error('index.html does not match integrations.json');
  if (await readFile(formationFile, 'utf8') !== renderedCss) throw new Error('formations.css does not match integrations.json');
  process.stdout.write(`Static HTML preserves all ${expectedTotal} catalog entries; formation proofs pass.\n`);
} else {
  await writeFile(formationFile, renderedCss);
  if (process.argv.includes('--write-html')) await writeFile(index, renderedHtml);
  process.stdout.write(`Rendered formation CSS for all ${expectedTotal} integrations; index.html was not changed.\n`);
}
