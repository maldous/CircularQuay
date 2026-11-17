import test from 'node:test';
import assert from 'node:assert/strict';
import { readSitePage } from './render-site-header.mjs';
import { documentationSuggestions, maximumSuggestions, mergeSuggestions } from './header-search.js';
import { documentation } from './explore-manifest.js';

test('documentation suggestions use the shared help-search ranking and Explore anchors', () => {
  const results = documentationSuggestions('getting started', [{ id: 'start', title: 'Getting started', category: 'Guides', parts: [{ id: 'part-1', title: 'Getting started', headings: [] }] }]);
  assert.deepEqual(results, [{ id: 'start', title: 'Getting started', summary: 'Guides', label: 'Documentation', href: '/explore#doc=start&part=part-1' }]);
});

test('suggestions retain guide destinations, deduplicate, and cap the result set', () => {
  const help = Array.from({ length: 9 }, (_, index) => ({ id: `guide-${index}`, title: `Guide ${index}`, summary: 'Help' }));
  const results = mergeSuggestions(help, [{ id: 'guide-0', title: 'Duplicate', summary: '', label: 'Documentation', href: '/explore#doc=guide-0&part=part-1' }]);
  assert.equal(results.length, maximumSuggestions);
  assert.equal(results[0].href, '/help#guide-0');
});

test('documentation fills unused reserved slots without skipping ranked guides', () => {
  const documentation = Array.from({ length: 9 }, (_, index) => ({ id: `doc-${index}`, title: `Doc ${index}`, summary: '', label: 'Documentation', href: `/explore#doc=${index}` }));
  assert.equal(mergeSuggestions([], documentation).length, maximumSuggestions);
  const sparse = mergeSuggestions([{ id: 'guide-1', title: 'First guide', summary: '' }], documentation);
  assert.equal(sparse.length, maximumSuggestions);
  assert.equal(sparse[0].href, '/help#guide-1');
  assert.equal(sparse.filter(item => item.label === 'Documentation').length, 7);
});

test('real manifest paths make the Python SDK discoverable with an unambiguous title', () => {
  const result = documentationSuggestions('Python', documentation.documents).find(item => item.id === 'sdk-python-readme');
  assert.equal(result?.title, 'cq-sdk · Python');
});

test('shared headers retain a permitted text combobox and unique popup references', async () => {
  const pages = ['index', 'about', 'accessibility', 'contact', 'cookies', 'privacy', 'security', 'status', 'terms', 'explore', 'help', 'login'];
  for (const page of pages) {
    const html = (await readSitePage(`${page}.html`)).toString('utf8');
    const header = html.match(/<header\b[^>]*>[\s\S]*?<\/header>/)?.[0];
    assert.ok(header, `${page} has a static header`);
    const inputs = [...header.matchAll(/<input\b[^>]*\sid="site-search"[^>]*>/g)];
    assert.equal(inputs.length, 1, `${page} has one search control`);
    const input = inputs[0][0];
    // ARIA in HTML permits combobox on text inputs without a list attribute.
    assert.match(input, /\stype="text"/, page);
    assert.match(input, /\srole="combobox"/, page);
    assert.doesNotMatch(input, /\slist\s*=/, page);
    assert.match(input, /\saria-autocomplete="list"/, page);
    assert.match(input, /\saria-controls="site-search-results"/, page);
    assert.match(input, /\saria-expanded="false"/, page);
    assert.match(input, /\saria-describedby="site-search-hint"/, page);
    assert.match(input, /\smaxlength="256"/, page);
    assert.match(input, /\sdisabled(?:\s|>)/, page);
    assert.match(header, /<label\b[^>]*\sfor="site-search"[^>]*>[^<]+<\/label>/, page);
    assert.match(header, /<form\b[^>]*\srole="search"[^>]*>/, page);
    assert.match(header, /<div\b[^>]*\sid="site-search-popup"[^>]*\shidden(?:\s|>)/, page);
    assert.match(header, /<ul\b[^>]*\sid="site-search-results"[^>]*\srole="listbox"[^>]*>/, page);
    for (const id of ['site-search', 'site-search-popup', 'site-search-results', 'site-search-status', 'site-search-hint']) {
      assert.equal([...html.matchAll(new RegExp(`\\sid="${id}"`, 'g'))].length, 1, `${page} has one ${id}`);
    }
  }
});
