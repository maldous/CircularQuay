import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { helpRelease } from './help-release.js';

const directory = new URL('.', import.meta.url);
const digest = bytes => createHash('sha256').update(bytes).digest('hex');
const localPath = descriptor => new URL(`generated/${descriptor.path.split('/').at(-1)}`, directory);

async function checkedJson(descriptor) {
  const bytes = await readFile(localPath(descriptor));
  assert.equal(bytes.byteLength, descriptor.bytes, `${descriptor.path} byte count`);
  assert.equal(digest(bytes), descriptor.sha256, `${descriptor.path} SHA-256`);
  return JSON.parse(bytes);
}

test('vendored help release retains all manifest-addressed static assets', async () => {
  const manifestBytes = await readFile(new URL('generated/manifest.json', directory));
  assert.equal(manifestBytes.byteLength, helpRelease.manifest.bytes);
  assert.equal(digest(manifestBytes), helpRelease.manifest.sha256);
  const manifest = JSON.parse(manifestBytes);
  assert.equal(manifest.contentVersion, helpRelease.contentVersion);
  const catalog = await checkedJson(manifest.catalog);
  await checkedJson(manifest.search);
  assert.ok(catalog.documents.length > 0);
  for (const document of catalog.documents) {
    const article = await checkedJson(document.article);
    assert.equal(article.id, document.id);
  }
});
