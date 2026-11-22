// A deterministic component boundary, not a layout/browser/security substitute.
// It parses only repository-authored fixture markup, executes no markup, and
// supplies explicit synthetic geometry/timers. Unexpected network fails closed.
import { webcrypto } from 'node:crypto';
import { readFileSync } from 'node:fs';

const decode = text => String(text).replace(/&(amp|lt|gt|quot|#39|nbsp);/g, (_, key) => ({ amp: '&', lt: '<', gt: '>', quot: '"', '#39': "'", nbsp: ' ' }[key]));
const kebab = key => key.replace(/[A-Z]/g, letter => '-' + letter.toLowerCase());

export class Element {
  constructor(tag = 'div', document) {
    this.tagName = tag.toUpperCase(); this.ownerDocument = document;
    this.children = []; this.attributes = new Map(); this.dataset = {};
    this.listeners = new Map(); this.style = { setProperty(key, value) { this[key] = value; }, removeProperty(key) { delete this[key]; } };
    this.rect = { left: 0, top: 0, right: 100, bottom: 50, width: 100, height: 50 };
    this.scrollTop = 0; this.selectionStart = 0; this.valid = true;
  }
  get classList() {
    const node = this;
    return { contains: name => node.className.split(/\s+/).includes(name), add(...names) { node.className = [...new Set([...node.className.split(/\s+/), ...names])].filter(Boolean).join(' '); }, remove(...names) { node.className = node.className.split(/\s+/).filter(name => !names.includes(name)).join(' '); }, toggle(name, force) { const has = this.contains(name); const next = force ?? !has; next ? this.add(name) : this.remove(name); return next; } };
  }
  get className() { return this.attributes.get('class') || ''; }
  set className(value) { this.attributes.set('class', String(value)); }
  get id() { return this.attributes.get('id') || ''; }
  set id(value) { this.attributes.set('id', String(value)); }
  get name() { return this.attributes.get('name') || ''; }
  set name(value) { this.attributes.set('name', String(value)); }
  set name(value) { this.attributes.set('name', String(value)); }
  get type() { return this.attributes.get('type') || ''; }
  set type(value) { this.attributes.set('type', String(value)); }
  get lang() { return this.attributes.get('lang') || ''; }
  set lang(value) { this.attributes.set('lang', String(value)); }
  get dir() { return this.attributes.get('dir') || ''; }
  set dir(value) { this.attributes.set('dir', String(value)); }
  get href() { return this.attributes.get('href') || ''; }
  set href(value) { this.attributes.set('href', String(value)); }
  get options() { return this.querySelectorAll('option'); }
  get elements() { return this.querySelectorAll('input,select,textarea,button'); }
  get value() {
    if (this._value !== undefined) return this._value;
    if (this.tagName === 'SELECT') { const option = this.options.find(item => item.hasAttribute('selected')) || this.options[0]; return option?.value ?? ''; }
    return this.attributes.get('value') ?? (['TEXTAREA', 'OPTION'].includes(this.tagName) ? this.textContent : '');
  }
  set value(value) { this._value = String(value); }
  get disabled() { return this.attributes.has('disabled'); }
  set disabled(value) { value ? this.attributes.set('disabled', '') : this.attributes.delete('disabled'); }
  get hidden() { return this.attributes.has('hidden'); }
  set hidden(value) { value ? this.attributes.set('hidden', '') : this.attributes.delete('hidden'); }
  get checked() { return this.attributes.has('checked'); }
  set checked(value) { value ? this.attributes.set('checked', '') : this.attributes.delete('checked'); }
  get open() { return this.attributes.has('open'); }
  set open(value) { value ? this.attributes.set('open', '') : this.attributes.delete('open'); }
  get parentElement() { return this.parentNode; }
  get clientWidth() { return this.rect.width; }
  get src() { const value = this._src ?? this.getAttribute('src'); return value ? new URL(value, this.ownerDocument.baseURI).href : ''; }
  set src(value) { this._src = String(value); const failed = this.ownerDocument.imageFailures?.has('*') || this.ownerDocument.imageFailures?.has(this._src); this.complete = true; this.naturalWidth = failed ? 0 : 100; queueMicrotask(() => failed ? this.onerror?.() : this.onload?.()); }
  get isConnected() { return Boolean(this.ownerDocument?.body === this || this.parentNode?.isConnected); }
  setAttribute(key, value) {
    this.attributes.set(key, String(value));
    if (key.startsWith('data-')) this.dataset[key.slice(5).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase())] = String(value);
    if (['checked', 'disabled', 'hidden', 'open'].includes(key)) this[key] = true;
  }
  getAttribute(key) { return key.startsWith('data-') ? this.dataset[key.slice(5).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase())] ?? null : this.attributes.get(key) ?? null; }
  hasAttribute(key) { return this.getAttribute(key) !== null; }
  removeAttribute(key) { this.attributes.delete(key); if (key.startsWith('data-')) delete this.dataset[key.slice(5).replace(/-([a-z])/g, (_, letter) => letter.toUpperCase())]; if (['checked', 'disabled', 'hidden', 'open'].includes(key)) this[key] = false; }
  append(...nodes) { for (let node of nodes) { if (typeof node !== 'object') node = this.ownerDocument.createTextNode(String(node)); node.remove(); node.parentNode = this; this.children.push(node); } }
  prepend(...nodes) { const old = [...this.children]; this.children = []; this.append(...nodes, ...old); }
  replaceChildren(...nodes) { for (const node of this.children) node.parentNode = null; this.children = []; this.append(...nodes); }
  remove() { if (this.parentNode) this.parentNode.children = this.parentNode.children.filter(node => node !== this); this.parentNode = null; }
  contains(node) { return this === node || this.children.some(child => child.contains(node)); }
  cloneNode(deep = false) { const clone = new Element(this.tagName, this.ownerDocument); for (const [key, value] of this.attributes) clone.setAttribute(key, value); clone.dataset = { ...this.dataset }; clone.rect = { ...this.rect }; clone._text = this._text; if (deep) clone.append(...this.children.map(child => child.cloneNode(true))); return clone; }
  insertAdjacentHTML(position, markup) { const fragment = new Element('div', this.ownerDocument); fragment.innerHTML = markup; position === 'afterbegin' ? this.prepend(...fragment.children) : this.append(...fragment.children); }
  get textContent() { return this._text || this.children.map(child => child.textContent).join(''); }
  set textContent(value) { this.replaceChildren(); this._text = String(value); }
  get innerHTML() { return this.markup || ''; }
  set innerHTML(value) {
    this.markup = String(value); this._text = ''; this.replaceChildren();
    const stack = [this];
    for (const match of this.markup.matchAll(/<!--[\s\S]*?-->|<\/?[A-Za-z][^>]*>|[^<]+/g)) {
      const text = match[0];
      if (text.startsWith('<!--')) continue;
      if (text.startsWith('</')) { if (stack.length > 1) stack.pop(); continue; }
      if (!text.startsWith('<')) { stack.at(-1).append(this.ownerDocument.createTextNode(decode(text))); continue; }
      const tag = /^<([\w-]+)/.exec(text)[1]; const node = new Element(tag, this.ownerDocument);
      for (const attr of text.slice(tag.length + 1, -1).matchAll(/([\w:-]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+)))?/g)) node.setAttribute(attr[1], decode(attr[2] ?? attr[3] ?? attr[4] ?? ''));
      stack.at(-1).append(node);
      if (!/^(input|img|br|hr|meta|link|source|wbr)$/i.test(tag) && !text.endsWith('/>')) stack.push(node);
    }
  }
  matches(selector) {
    if (this.tagName === '#TEXT') return false;
    if (selector.includes(',')) return selector.split(',').some(item => this.matches(item.trim()));
    const parts = selector.trim().split(/\s+(?![^\[]*\])/);
    if (parts.length > 1) { const last = parts.pop(); if (!this.matches(last)) return false; let parent = this.parentNode; while (parent) { if (parent.matches(parts.join(' '))) return true; parent = parent.parentNode; } return false; }
    if (selector.startsWith('#') && !selector.includes('[')) return this.id === selector.slice(1).replace(/\\(.)/g, '$1');
    const attrs = [...selector.matchAll(/\[([^\]=]+)(?:=["']?([^\]"']*)["']?)?\]/g)];
    if (attrs.some(([, key, value]) => !this.hasAttribute(key) || value !== undefined && this.getAttribute(key) !== value)) return false;
    const plain = selector.replace(/\[[^\]]*\]/g, '');
    const id = /#([\w-]+)/.exec(plain); if (id && this.id !== id[1]) return false;
    if ([...plain.matchAll(/\.([\w-]+)/g)].some(([, name]) => !this.classList.contains(name))) return false;
    const tag = /^[A-Za-z][\w-]*/.exec(plain); return !tag || this.tagName === tag[0].toUpperCase();
  }
  querySelectorAll(selector) { return this.children.flatMap(child => [...(child.matches(selector) ? [child] : []), ...child.querySelectorAll(selector)]); }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  closest(selector) { return this.matches(selector) ? this : this.parentNode?.closest(selector) || null; }
  addEventListener(type, fn, options) { if (!this.listeners.has(type)) this.listeners.set(type, []); this.listeners.get(type).push({ fn, once: options?.once }); }
  removeEventListener(type, fn) { this.listeners.set(type, (this.listeners.get(type) || []).filter(item => item.fn !== fn)); }
  dispatchEvent(event) {
    event.target ||= this; event.currentTarget = this; event.preventDefault ||= () => { event.defaultPrevented = true; }; event.stopPropagation ||= () => { event.stopped = true; };
    this['on' + event.type]?.(event);
    for (const item of [...(this.listeners.get(event.type) || [])]) { item.fn(event); if (item.once) this.removeEventListener(event.type, item.fn); }
    if (event.bubbles && !event.stopped) this.parentNode?.dispatchEvent(event);
    return !event.defaultPrevented;
  }
  focus() { this.ownerDocument.activeElement = this; }
  click() { if (this.disabled || this.closest('fieldset[disabled]')) return; return this.dispatchEvent({ type: 'click', bubbles: true }); }
  setSelectionRange(start, end) { this.selectionStart = start; this.selectionEnd = end; }
  setCustomValidity(value) { this.validationMessage = value; }
  checkValidity() { return this.valid && !this.validationMessage; }
  reportValidity() { this.validityReported = true; return this.checkValidity(); }
  showModal() { this.open = true; this.setAttribute('open', ''); }
  close() { this.open = false; this.removeAttribute('open'); this.dispatchEvent({ type: 'close' }); }
  getBoundingClientRect() { return this.rect; }
  getTotalLength() { return 100; }
  getPointAtLength(distance) { return { x: distance, y: distance / 2 }; }
  scrollIntoView() { this.scrolledIntoView = true; }
}

export function publicHelpFetcher(records, { transform = value => value, before = async () => {} } = {}) {
  return async (resource, options) => {
    const path = String(resource);
    if (!/^\/(?:help|client-help)\/generated\/[a-z0-9.-]+\.json$/.test(path)) throw Error('Unexpected asset path in owned fixture');
    records.push({ path, options }); await before(path);
    const filename = path.slice(path.lastIndexOf('/') + 1);
    const bytes = readFileSync(new URL('./help/generated/' + filename, import.meta.url));
    return new Response(transform(bytes, path), { headers: { 'Content-Type': 'application/json' } });
  };
}

export class Document extends Element {
  constructor(markup = '') {
    super('document'); this.ownerDocument = this; this.readyState = 'complete'; this.hidden = false; this.baseURI = 'https://preview.example.invalid/'; this.imageFailures = new Set();
    this.documentElement = new Element('html', this); this.body = new Element('body', this);
    this.append(this.documentElement); this.documentElement.append(this.body); this.body.innerHTML = markup;
    this.fonts = new Element('fonts', this); this.fonts.ready = Promise.resolve();
  }
  createElement(tag) { return new Element(tag, this); }
  createElementNS(_namespace, tag) { return this.createElement(tag); }
  createTextNode(text) { const node = new Element('#text', this); node.textContent = text; return node; }
  getElementById(id) { return this.querySelector('#' + id); }
}

export function browser(t, { html = '<div id="app"></div>', hash = '#/landing', languages = ['en-AU'], stored = {}, blockedStorage = false } = {}) {
  const document = new Document(html); const window = new Element('window', document);
  const data = new Map(Object.entries(stored)); const writes = [];
  const storage = { getItem(key) { if (blockedStorage) throw Error('synthetic blocked storage'); return data.get(key) ?? null; }, setItem(key, value) { if (blockedStorage) throw Error('synthetic quota'); writes.push([key, value]); data.set(key, value); }, removeItem(key) { data.delete(key); } };
  window.localStorage = storage;
  const media = new Element('media', document); media.matches = false;
  const mediaQueries = new Map([['(prefers-color-scheme: dark)', media]]);
  window.matchMedia = query => { if (!mediaQueries.has(query)) { const item = new Element('media', document); item.matches = false; mediaQueries.set(query, item); } return mediaQueries.get(query); };
  const location = { hash, href: '', reloads: 0, reload() { this.reloads++; } };
  const timers = new Map(); const frames = new Map(); let now = 0; let next = 0;
  const timeout = (fn, delay = 0, interval = false) => { const id = ++next; timers.set(id, { fn, at: now + delay, delay, interval }); return id; };
  const sources = []; const exports = []; const revoked = []; const observers = { resize: [], intersection: [] };
  class SyntheticEvents { constructor(path) { this.path = path; this.closed = false; sources.push(this); } close() { this.closed = true; } }
  class SyntheticFormData { constructor(form) { this.rows = form.elements.filter(input => input.name && !input.disabled && (!['checkbox', 'radio'].includes(input.type) || input.checked)).map(input => [input.name, input.type === 'checkbox' ? input.value || 'on' : input.value]); } entries() { return this.rows[Symbol.iterator](); } [Symbol.iterator]() { return this.entries(); } get(key) { return this.rows.find(row => row[0] === key)?.[1] ?? null; } }
  class SyntheticFileReader { readAsDataURL(file) { this.result = file.syntheticDataURL || 'data:' + file.type + ';base64,QUJD'; queueMicrotask(() => this.onload?.()); } }
  class SyntheticImage extends Element { constructor() { super('img', document); } }
  const observer = kind => class { constructor(callback) { this.callback = callback; this.nodes = []; observers[kind].push(this); } observe(node) { this.nodes.push(node); } unobserve(node) { this.nodes = this.nodes.filter(item => item !== node); } disconnect() { this.nodes = []; } };
  const globals = { window, document, localStorage: storage, location, navigator: { languages }, crypto: webcrypto,
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options?.detail; } }, FormData: SyntheticFormData,
    FileReader: SyntheticFileReader, EventSource: SyntheticEvents, Image: SyntheticImage, ResizeObserver: observer('resize'), IntersectionObserver: observer('intersection'), innerWidth: 1200, innerHeight: 900,
    matchMedia: window.matchMedia, addEventListener: window.addEventListener.bind(window), removeEventListener: window.removeEventListener.bind(window),
    CSS: { escape: value => String(value).replace(/[^A-Za-z0-9_-]/g, character => '\\' + character) },
    getComputedStyle: () => ({ getPropertyValue: () => '0', display: 'block' }),
    setTimeout: (fn, delay) => timeout(fn, delay), clearTimeout: id => timers.delete(id),
    setInterval: (fn, delay) => timeout(fn, delay, true), clearInterval: id => timers.delete(id),
    requestAnimationFrame: fn => { const id = ++next; frames.set(id, fn); return id; }, cancelAnimationFrame: id => frames.delete(id),
    fetch: async () => { throw Error('Unexpected network request in synthetic fixture'); } };
  Object.assign(window, globals); document.defaultView = window;
  const cleanup = [];
  const descriptors = [];
  for (const [key, value] of Object.entries(globals)) {
    descriptors.push([key, Object.getOwnPropertyDescriptor(globalThis, key)]);
    Object.defineProperty(globalThis, key, { configurable: true, writable: true, value });
  }
  // Node after hooks are FIFO. Own one lifetime envelope instead of relying
  // on a later test hook to run before any earlier global restoration hook.
  t.after(async () => {
    let failure;
    try {
      for (const callback of cleanup) {
        try { await callback(); } catch (error) { failure ||= error; }
      }
    } finally {
      timers.clear(); frames.clear();
      for (const [key, before] of descriptors) {
        if (before) Object.defineProperty(globalThis, key, before);
        else delete globalThis[key];
      }
    }
    if (failure) throw failure;
  });
  t.mock.method(URL, 'createObjectURL', blob => { exports.push(blob); return 'blob:owned-' + exports.length; });
  t.mock.method(URL, 'revokeObjectURL', value => revoked.push(value));
  return { document, window, media, location, data, writes, timers, frames, sources, exports, revoked, observers,
    beforeRestore(callback) { cleanup.push(callback); },
    mediaFor: query => window.matchMedia(query), setStorageBlocked(value) { blockedStorage = value; },
    event(node, type, extra = {}) { const event = { type, ...extra }; node.dispatchEvent(event); return event; },
    async drain(count = 80) { for (let index = 0; index < count; index++) await Promise.resolve(); },
    async until(predicate) { const deadline = Date.now() + 5000; while (!predicate()) { if (Date.now() > deadline) throw Error('Synthetic asynchronous boundary did not settle'); await new Promise(resolve => setImmediate(resolve)); } await this.drain(); },
    async advance(milliseconds) { now += milliseconds; let count = 0; while ([...timers.values()].some(item => item.at <= now)) { if (++count > 1000) throw Error('Unbounded synthetic timer'); const [id, item] = [...timers].find(([, value]) => value.at <= now); item.interval ? item.at += item.delay : timers.delete(id); item.fn(); await this.drain(); } },
    async frame(time) { const callbacks = [...frames.values()]; frames.clear(); for (const callback of callbacks) callback(time); await this.drain(); },
  };
}
