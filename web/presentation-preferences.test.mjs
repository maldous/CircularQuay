import test from 'node:test';
import assert from 'node:assert/strict';
import { browserPresentation, createPresentationPreferences, presentationStorageKey, readPresentation, resolvePresentation } from './presentation-preferences.js';

const memory = initial => {
  const values = new Map(initial === undefined ? [] : [[presentationStorageKey, initial]]);
  const writes = [];
  return { writes, values, getItem: key => values.get(key) ?? null, setItem: (key, value) => { writes.push([key, value]); values.set(key, value); }, removeItem: key => { writes.push([key, null]); values.delete(key); } };
};

test('ordered supported browser languages and region hints remain separate', () => {
  assert.deepEqual(browserPresentation(['fr-CA', 'AR-eg', 'en-AU']), { language: 'ar', region: 'CA', languageSource: 'browser', regionSource: 'browser' });
  assert.equal(browserPresentation(['bad_tag', 'zh-Arab', 'de-DE']).language, 'en');
  assert.equal(browserPresentation(['en-ZZ']).region, null);
  assert.equal(browserPresentation(null).language, 'en');
  assert.equal(browserPresentation(['ar']).region, null);
});

test('presentation storage is closed, bounded and never accepts unrelated workspace fields', () => {
  for (const raw of ['{', 'null', '[]', '{"version":2}', '{"version":1,"theme":"sepia"}', '{"version":1,"language":"ar-EG"}', '{"version":1,"profile":"private"}', '{"version":1,"theme":null}', ' '.repeat(257)]) assert.throws(() => readPresentation(raw));
  assert.deepEqual(readPresentation(null), { version: 1 });
  assert.deepEqual(readPresentation('{"version":1,"language":"system"}'), { version: 1, language: 'system' });
});

test('explicit device fields outrank a profile, and explicit system bypasses profile values', () => {
  const profile = { version: 1, language: 'ar', theme: 'dark' };
  assert.equal(resolvePresentation({ profile, browserLanguages: ['en'] }).language, 'ar');
  const explicit = resolvePresentation({ device: { version: 1, language: 'system', theme: 'light' }, profile, browserLanguages: ['en'] });
  assert.equal(explicit.language, 'en'); assert.equal(explicit.languageSource, 'device'); assert.equal(explicit.theme, 'light');
  assert.equal(resolvePresentation({ device: { version: 1, theme: 'system' }, profile }).theme, 'system');
});

test('defaults and profile overlays are never persisted or copied across profile transitions', () => {
  const storage = memory(); const preferences = createPresentationPreferences({ storage, browserLanguages: ['ar'] });
  assert.equal(preferences.snapshot().language, 'ar'); assert.equal(storage.writes.length, 0);
  preferences.setProfile({ version: 1, language: 'en', theme: 'dark' });
  assert.equal(preferences.snapshot().language, 'en'); assert.equal(storage.writes.length, 0);
  preferences.setProfile({ version: 1, theme: 'light' });
  assert.equal(preferences.snapshot().language, 'ar'); assert.equal(preferences.snapshot().theme, 'light');
  preferences.setProfile(null); assert.equal(preferences.snapshot().theme, 'system'); assert.equal(storage.writes.length, 0);
});

test('clear follows defaults, while an explicit choice survives reload and remains device scoped', () => {
  const storage = memory(); const preferences = createPresentationPreferences({ storage, browserLanguages: ['en'] });
  preferences.set('language', 'ar'); preferences.set('theme', 'dark');
  const reloaded = createPresentationPreferences({ storage, browserLanguages: ['en'] });
  reloaded.setProfile({ version: 1, language: 'en', theme: 'light' }); assert.equal(reloaded.snapshot().language, 'ar');
  reloaded.clear('language'); assert.equal(reloaded.snapshot().language, 'en'); assert.equal(reloaded.snapshot().languageSource, 'profile');
  reloaded.clear('theme'); assert.deepEqual(JSON.parse(storage.values.get(presentationStorageKey)), { version: 1 });
});

test('storage failures retain explicit choices in memory and never affect unrelated local data', () => {
  const storage = { getItem() { throw Error('blocked'); }, setItem() { throw Error('quota'); }, removeItem() { throw Error('blocked'); } };
  const preferences = createPresentationPreferences({ storage, browserLanguages: ['en'] });
  preferences.set('language', 'ar'); assert.equal(preferences.snapshot().language, 'ar'); assert.equal(preferences.snapshot().persistence, 'memory');
  preferences.clear('language'); assert.equal(preferences.snapshot().language, 'en');
  assert.throws(() => preferences.set('workspace', 'private'));
});

test('system updates and subscriptions do not turn defaults into stored explicit choices', () => {
  const storage = memory(); const preferences = createPresentationPreferences({ storage, browserLanguages: ['en'] }); let seen;
  const unsubscribe = preferences.subscribe(value => { seen = value; }); preferences.setBrowserLanguages(['ar']);
  assert.equal(seen.language, 'ar'); assert.equal(seen.direction, 'rtl'); assert.equal(storage.writes.length, 0);
  unsubscribe(); preferences.setBrowserLanguages(['en']); assert.equal(seen.language, 'ar');
});


test('current V1 preferences do not read, copy or convert workspace fields', () => {
  const reads = []; const storage = memory();
  storage.values.set('cq.preview.v1.local', JSON.stringify({ settings: { theme: 'dark' }, policy: { private: 'untouched' } }));
  const originalRead = storage.getItem; storage.getItem = key => { reads.push(key); return originalRead(key); };
  const preferences = createPresentationPreferences({ storage });
  assert.equal(preferences.snapshot().theme, 'system'); assert.equal(storage.writes.length, 0);
  assert.deepEqual(reads, [presentationStorageKey]);
  preferences.set('theme', 'light'); preferences.clear('theme');
  const reloaded = createPresentationPreferences({ storage });
  assert.equal(reloaded.snapshot().theme, 'system');
  assert.deepEqual(JSON.parse(storage.values.get(presentationStorageKey)), { version: 1 });
  assert.equal(JSON.parse(storage.values.get('cq.preview.v1.local')).settings.theme, 'dark');
});

test('pre-CSS bootstrap agrees with the adapter without writing defaults or making requests', async () => {
  const { readFile } = await import('node:fs/promises');
  const { runInNewContext } = await import('node:vm');
  const source = await readFile(new URL('presentation-bootstrap.js', import.meta.url), 'utf8');
  for (const raw of [null, '{bad', '{"version":2}', '{"version":1}', '{"version":1,"theme":"dark"}', '{"version":1,"language":"system","theme":"light"}', '{"version":1,"language":"ar"}', ' '.repeat(257)]) {
    for (const languages of [['fr-CA', 'ar-EG'], ['en-AU'], ['bad_tag', 'zh'], []]) {
      const storage = memory(raw === null ? undefined : raw); const document = { documentElement: { dataset: {} } };
      runInNewContext(source, { localStorage: storage, navigator: { languages }, document, Intl });
      const expected = createPresentationPreferences({ storage, browserLanguages: languages }).snapshot();
      assert.equal(document.documentElement.lang, expected.language);
      assert.equal(document.documentElement.dir, expected.direction);
      assert.equal(document.documentElement.dataset.theme, expected.theme);
      assert.equal(storage.writes.length, 0);
    }
  }
  const document = { documentElement: { dataset: {} } };
  runInNewContext(source, { get localStorage() { throw Error('blocked'); }, navigator: { languages: ['ar'] }, document, Intl });
  assert.equal(document.documentElement.lang, 'ar'); assert.equal(document.documentElement.dataset.theme, 'system');
});
