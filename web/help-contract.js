import { helpRelease } from './help-release.js';
const maximumBytes = 4 * 1024 * 1024;
const schemaHash = '72a83cfa922f63144142717852b01527eeb5657daf6bb2e6fe9ce98de869d842';
const topicPattern = /^[a-z][a-z0-9.-]{0,100}$/;
const hashPattern = /^[a-f0-9]{64}$/;
const versionPattern = /^sha256:[a-f0-9]{64}$/;
const assetPattern = /^\/help\/generated\/(?:catalog|search|article-[a-z][a-z0-9.-]{0,100})-[a-f0-9]{64}\.json$/;
const fail = () => { throw new Error('Help content is unavailable or incompatible. Your workspace has not been changed.'); };
const text = (value, limit = 262144) => typeof value === 'string' && value.length > 0 && value.length <= limit;

function exactObject(value, keys) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) fail();
  const actual = Object.keys(value);
  if (actual.length !== keys.length || actual.some(key => !keys.includes(key))) fail();
}
function stringArray(value, limit = 2048) {
  if (!Array.isArray(value) || value.length > limit || value.some(item => !text(item, 1000))) fail();
}
function version(value, expected) {
  if (!versionPattern.test(value) || (expected && value !== expected)) fail();
}

export function validateHelpAsset(value) {
  exactObject(value, ['path', 'sha256', 'bytes']);
  if (!assetPattern.test(value.path) || !hashPattern.test(value.sha256) || !value.path.endsWith(`-${value.sha256}.json`) || !Number.isSafeInteger(value.bytes) || value.bytes < 1 || value.bytes > maximumBytes) fail();
  return value;
}

export function validateHelpManifest(value) {
  exactObject(value, ['contractId', 'schemaVersion', 'contentVersion', 'authoringSchemaSha256', 'catalog', 'search']);
  if (value.contractId !== 'cq.help.distribution' || value.schemaVersion !== 1 || value.authoringSchemaSha256 !== schemaHash) fail();
  version(value.contentVersion, helpRelease.contentVersion);
  if (helpRelease.contentVersion !== `sha256:${helpRelease.sourceSha256}`) fail();
  validateHelpAsset(value.catalog); validateHelpAsset(value.search);
  if (!value.catalog.path.startsWith('/help/generated/catalog-') || !value.search.path.startsWith('/help/generated/search-')) fail();
  return value;
}

export function validateHelpCatalog(value, expectedVersion) {
  exactObject(value, ['contentVersion', 'documents']); version(value.contentVersion, expectedVersion);
  if (!Array.isArray(value.documents) || !value.documents.length || value.documents.length > 2048) fail();
  const ids = new Set();
  for (const item of value.documents) {
    exactObject(item, ['id', 'title', 'summary', 'audiences', 'article']);
    if (!topicPattern.test(item.id) || ids.has(item.id) || !text(item.title, 500) || !text(item.summary, 2000)) fail();
    ids.add(item.id); stringArray(item.audiences, 100); validateHelpAsset(item.article);
    if (!item.article.path.startsWith(`/help/generated/article-${item.id}-`)) fail();
  }
  return value;
}

export function validateHelpPostings(value, expectedVersion, documents) {
  exactObject(value, ['contentVersion', 'documentIds', 'postings']); version(value.contentVersion, expectedVersion);
  stringArray(value.documentIds);
  if (value.documentIds.length !== documents.length || value.documentIds.some((id, index) => id !== documents[index].id)) fail();
  if (!value.postings || typeof value.postings !== 'object' || Array.isArray(value.postings) || Object.keys(value.postings).length > 100000) fail();
  let count = 0;
  for (const [term, items] of Object.entries(value.postings)) {
    if (!/^[\p{L}\p{N}]{1,256}$/u.test(term) || !Array.isArray(items) || items.length > documents.length) fail();
    const ids = new Set();
    for (const pair of items) {
      if (!Array.isArray(pair) || pair.length !== 2) fail();
      const [id, weight] = pair;
      if (!Number.isInteger(id) || id < 0 || id >= documents.length || ids.has(id) || !Number.isFinite(weight) || weight <= 0 || weight > 1000) fail();
      ids.add(id); count += 1;
      if (count > 1000000) fail();
    }
  }
  return value;
}

export function validateHelpArticle(value, expectedVersion, expectedId) {
  exactObject(value, ['contentVersion', 'id', 'title', 'bodyMarkdown', 'capabilityStatus', 'canonicalRefs']); version(value.contentVersion, expectedVersion);
  if (value.id !== expectedId || !topicPattern.test(value.id) || !text(value.title, 500) || !text(value.bodyMarkdown) || !text(value.capabilityStatus, 2000) || !Array.isArray(value.canonicalRefs) || value.canonicalRefs.length > 100) fail();
  for (const reference of value.canonicalRefs) {
    exactObject(reference, ['repoPath', 'revision', 'sha256']);
    if (!text(reference.repoPath, 500) || reference.repoPath.startsWith('/') || reference.repoPath.split('/').some(part => part === '..' || part === '.') || !/^[A-Za-z0-9_./-]+$/.test(reference.repoPath) || !/^[a-f0-9]{40}$/.test(reference.revision) || !hashPattern.test(reference.sha256)) fail();
  }
  return value;
}

export async function readHelpJson(response, descriptor) {
  if (!response.ok || !response.body || !/^application\/json(?:;|$)/i.test(response.headers.get('content-type') || '')) fail();
  const bound = descriptor?.bytes || maximumBytes;
  const length = response.headers.get('content-length');
  if (length !== null && (!/^\d+$/.test(length) || Number(length) > bound)) fail();
  const reader = response.body.getReader();
  const chunks = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > bound) fail();
      chunks.push(value);
    }
  } catch (error) {
    await reader.cancel().catch(() => {});
    throw error;
  } finally { reader.releaseLock(); }
  if (descriptor && size !== descriptor.bytes) fail();
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
  if (descriptor) {
    const hash = [...new Uint8Array(await crypto.subtle.digest('SHA-256', bytes))].map(value => value.toString(16).padStart(2, '0')).join('');
    if (hash !== descriptor.sha256) fail();
  }
  try { return JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes)); }
  catch { fail(); }
}
