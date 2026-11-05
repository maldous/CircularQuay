import test from 'node:test';
import assert from 'node:assert/strict';
import { sidebarNavigationMarkup, sidebarSections, sidebarSelectorMarkup, performConfigurationAction, fetchExampleDocument } from './local/configuration.js';
import { parseDocument, referenceErrors } from './local/schema-tools.js';

const documentFrom = value => parseDocument(JSON.stringify(value));

test('unexpected example fetch and response-read failures reach the generic action handler', async () => {
  const sentinel = 'SYNTHETIC_PRIVATE_EXAMPLE_DIAGNOSTIC';
  for (const fetcher of [
    async () => { throw new Error(sentinel); },
    async () => ({ ok: true, text: async () => { throw new Error(sentinel); } }),
    async () => ({ ok: false }),
  ]) {
    const reports = [];
    await performConfigurationAction(() => fetchExampleDocument(fetcher), message => reports.push(message));
    assert.equal(reports.length, 1);
    assert.match(reports[0], /local configuration action could not complete/);
    assert.equal(reports[0].includes(sentinel), false);
  }
});

test('configuration action dispatcher awaits completion and reports asynchronous rejection without diagnostic payloads', async () => {
  const reports = [];
  let completed = false;
  await performConfigurationAction(async () => {
    await Promise.resolve();
    completed = true;
  }, message => reports.push(message));
  assert.equal(completed, true);
  assert.deepEqual(reports, []);

  const protectedDiagnostic = 'SYNTHETIC_PRIVATE_FILE_DETAIL';
  await assert.doesNotReject(performConfigurationAction(async () => {
    await Promise.resolve();
    throw new Error(protectedDiagnostic);
  }, message => reports.push(message)));
  assert.equal(reports.length, 1);
  assert.match(reports[0], /local configuration action could not complete/);
  assert.equal(reports[0].includes(protectedDiagnostic), false);
});

test('unknown imported roots remain literal drafts while sidebar markup escapes their keys and labels', () => {
  const importedKey = '\"><img src=x data-untrusted=preview>';
  const document = documentFrom({ version: 1, sources: {}, workloads: {}, [importedKey]: { retained: false } });
  const sections = sidebarSections(document);
  const markup = sidebarNavigationMarkup(sections, document, importedKey);
  const selector = sidebarSelectorMarkup(sections, importedKey);

  assert.equal(Object.hasOwn(document, importedKey), true);
  assert.deepEqual(document[importedKey], { retained: false });
  assert.match(markup, /data-section="&quot;&gt;&lt;img src=x data-untrusted=preview&gt;"/);
  assert.match(markup, /Unsupported: &quot;&gt;&lt;img src=x data-untrusted=preview&gt;/);
  assert.equal(markup.includes('<img'), false);
  assert.match(selector, /value="&quot;&gt;&lt;img src=x data-untrusted=preview&gt;" selected/);
  assert.equal(selector.includes('<img'), false);
  assert.equal(markup.includes('data-untrusted=preview"'), false);

  for (const name of ['__proto__', 'constructor', 'prototype']) {
    const special = documentFrom({ version: 1, sources: {}, workloads: {}, [name]: { retained: name } });
    const specialMarkup = sidebarNavigationMarkup(sidebarSections(special), special, name);
    assert.equal(Object.hasOwn(special, name), true);
    assert.match(specialMarkup, new RegExp(`Unsupported: ${name}`));
  }
});

test('reference checks reject inherited names and accept only declared own special-name entries', () => {
  const inherited = documentFrom({
    version: 1,
    sources: {},
    workloads: { job: { bindings: [{ name: 'VALUE', source: 'toString' }] } },
    grants: [{ workload: 'constructor', authority: 'prototype' }],
    mcp: { authorities: {} },
  });
  const inheritedErrors = referenceErrors(inherited);
  assert.ok(inheritedErrors.some(error => error.includes('unknown source toString')));
  assert.ok(inheritedErrors.some(error => error.includes('unknown workload')));
  assert.ok(inheritedErrors.some(error => error.includes('unknown authority')));

  for (const name of ['__proto__', 'constructor', 'prototype']) {
    const declared = documentFrom({
      version: 1,
      sources: Object.fromEntries([[name, {}]]),
      workloads: Object.fromEntries([[name, { bindings: [{ name: 'VALUE', source: name }] }]]),
      grants: [{ workload: name, authority: name }],
      mcp: { authorities: Object.fromEntries([[name, {}]]) },
    });
    assert.equal(Object.hasOwn(declared.sources, name), true);
    assert.equal(Object.hasOwn(declared.workloads, name), true);
    assert.equal(Object.hasOwn(declared.mcp.authorities, name), true);
    assert.deepEqual(referenceErrors(declared), []);
  }
});
