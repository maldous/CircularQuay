import test from 'node:test';
import assert from 'node:assert/strict';
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
