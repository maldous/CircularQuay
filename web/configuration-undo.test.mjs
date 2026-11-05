import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { configuration } from './local/configuration.js';

const decode = value => value.replace(/&(amp|lt|gt|quot|#39);/g, (_, name) => ({ amp: '&', lt: '<', gt: '>', quot: '"', '#39': "'" }[name]));

// Exercise the shipped renderer/event bindings without a DOM dependency. This
// adapter supplies only the native control operations used by these scenarios;
// actual browser layout and keyboard behavior remain separately checked.
class EditorRoot {
  isConnected = true;
  controls = [];
  message = { textContent: '' };
  more = { open: false, querySelector: () => ({ focus() {} }) };
  search = { focus() {}, setSelectionRange() {} };
  picker = {};
  set innerHTML(value) {
    this.markup = value;
    this.controls = [...value.matchAll(/<(?:input|select)\b[^>]*data-pointer="([^"]*)"[^>]*data-type="([^"]*)"[^>]*>/g)].map(match => ({
      dataset: { pointer: decode(match[1]), type: match[2] },
      value: decode(/\bvalue="([^"]*)"/.exec(match[0])?.[1] ?? ''),
      setCustomValidity(value) { this.validityMessage = value; },
    }));
    this.message.textContent = decode(/class="cfg-message" role="status">([^<]*)/.exec(value)?.[1] ?? '');
  }
  get innerHTML() { return this.markup; }
  querySelector(selector) {
    return ({ '.cfg-more': this.more, '.cfg-message': this.message, '#cfg-search': this.search, '#cfg-section-picker': this.picker })[selector] ?? null;
  }
  querySelectorAll(selector) { return selector === '[data-pointer][data-type]' ? this.controls : []; }
  async click(action, pointer) {
    const button = { dataset: { cfg: action, pointer, section: pointer } };
    await this.onclick({ target: { closest: selector => selector === '[data-cfg]' ? button : null } });
  }
  input(pointer, value) {
    const control = this.controls.find(item => item.dataset.pointer === pointer);
    assert.ok(control, `rendered control ${pointer}`);
    control.value = value;
    control.onchange({ target: control });
  }
  proposal() {
    return JSON.parse(decode([...this.markup.matchAll(/<pre>([\s\S]*?)<\/pre>/g)][1][1]));
  }
}

async function editor(t) {
  const previousLocation = Object.getOwnPropertyDescriptor(globalThis, 'location');
  Object.defineProperty(globalThis, 'location', { value: { hash: '#/configuration' }, configurable: true });
  t.after(() => { if (previousLocation) Object.defineProperty(globalThis, 'location', previousLocation); else delete globalThis.location; });
  const schema = JSON.parse(await readFile(new URL('./local/policy-schema.json', import.meta.url)));
  const policy = JSON.parse(await readFile(new URL('./local/linux-example.json', import.meta.url)));
  t.mock.method(globalThis, 'fetch', async path => {
    assert.equal(path, '/local/policy-schema.json');
    return { ok: true, json: async () => schema };
  });
  const root = new EditorRoot();
  const state = { policy: structuredClone(policy) };
  await configuration(root, state, { changed() {}, save() {}, exportJSON() {} });
  assert.equal(typeof root.onclick, 'function', root.innerHTML);
  await root.click('section', 'observability');
  return { root, state, policy };
}

test('a later retained invalid numeric edit invalidates removal undo and cannot be lost', async t => {
  const { root, state, policy } = await editor(t);
  await root.click('remove', '/observability/service_name');
  assert.equal(root.innerHTML.includes('data-cfg="undo-remove"'), true);
  root.input('/observability/timeout_seconds', '-');
  assert.equal(root.innerHTML.includes('data-cfg="undo-remove"'), false);
  await root.click('undo-remove'); // A stale dispatched action must also be harmless.
  await root.click('validate');
  assert.equal(Object.hasOwn(root.proposal().observability, 'service_name'), false);
  assert.equal(root.controls.find(item => item.dataset.pointer === '/observability/timeout_seconds').value, '-');
  await root.click('save');
  assert.match(root.message.textContent, /invalid numeric inputs before saving/);
  assert.deepEqual(state.policy, policy);
});

test('undo still restores invalid numeric input retained before the removal', async t => {
  const { root, state, policy } = await editor(t);
  root.input('/observability/timeout_seconds', '-');
  const before = root.proposal();
  await root.click('remove', '/observability/service_name');
  await root.click('undo-remove');
  assert.deepEqual(root.proposal(), before);
  assert.equal(root.controls.find(item => item.dataset.pointer === '/observability/timeout_seconds').value, '-');
  await root.click('save');
  assert.match(root.message.textContent, /invalid numeric inputs before saving/);
  assert.deepEqual(state.policy, policy);
});
