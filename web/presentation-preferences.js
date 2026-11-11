// Current initial V1 presentation state: no authentication, network, workspace or telemetry inputs.
// Browser binding is separate so the precedence/storage boundary is testable.
export const presentationStorageKey = 'cq.presentation.v1';
const languages = new Set(['system', 'en', 'ar']);
const themes = new Set(['system', 'light', 'dark']);
const empty = () => ({ version: 1 });

export function validatePresentation(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value) || value.version !== 1 || Object.keys(value).some(key => !['version', 'language', 'theme'].includes(key))) throw new Error('Unsupported presentation preferences.');
  if (Object.hasOwn(value, 'language') && !languages.has(value.language) || Object.hasOwn(value, 'theme') && !themes.has(value.theme)) throw new Error('Unsupported presentation choice.');
  return { version: 1, ...Object.fromEntries(['language', 'theme'].filter(key => Object.hasOwn(value, key)).map(key => [key, value[key]])) };
}

export function readPresentation(raw) {
  if (raw === null || raw === undefined) return empty();
  if (typeof raw !== 'string' || raw.length > 256) throw new Error('Unsupported presentation storage.');
  return validatePresentation(JSON.parse(raw));
}

export function browserPresentation(browserLanguages) {
  let language;
  let region;
  for (const candidate of (Array.isArray(browserLanguages) ? browserLanguages : []).slice(0, 32)) {
    if (typeof candidate !== 'string' || candidate.length > 100) continue;
    try {
      const locale = new Intl.Locale(candidate);
      if (!region && locale.region && locale.region !== 'ZZ') region = locale.region;
      if (!language && ['en', 'ar'].includes(locale.language)) language = locale.language;
    } catch { /* Ignore malformed browser tags, never infer a language from country. */ }
  }
  return { language: language || 'en', region: region || null, languageSource: language ? 'browser' : 'fallback', regionSource: region ? 'browser' : 'unavailable' };
}

export function resolvePresentation({ device = empty(), profile = null, browserLanguages = [] } = {}) {
  device = validatePresentation(device);
  if (profile !== null) profile = validatePresentation(profile);
  const browser = browserPresentation(browserLanguages);
  const choose = field => Object.hasOwn(device, field) ? [device[field], 'device'] : profile && Object.hasOwn(profile, field) ? [profile[field], 'profile'] : ['system', 'browser'];
  const [languageChoice, languageSource] = choose('language');
  const [theme, themeSource] = choose('theme');
  const language = languageChoice === 'system' ? browser.language : languageChoice;
  return Object.freeze({ language, direction: language === 'ar' ? 'rtl' : 'ltr', languageChoice, theme, languageSource, themeSource, browserLanguageSource: browser.languageSource, region: browser.region, regionSource: browser.regionSource });
}

export function createPresentationPreferences({ storage, browserLanguages = [] } = {}) {
  let device = empty();
  let profile = null;
  let persistence = 'available';
  const listeners = new Set();
  try { const raw = storage?.getItem(presentationStorageKey); device = readPresentation(raw); if (!storage) persistence = 'memory'; }
  catch { persistence = 'memory'; }
  const snapshot = () => Object.freeze({ ...resolvePresentation({ device, profile, browserLanguages }), persistence, device: Object.freeze({ ...device }) });
  const emit = () => { const value = snapshot(); for (const listener of listeners) listener(value); return value; };
  const persist = () => {
    try {
      if (!storage) throw new Error('Storage unavailable.');
      // Persist the explicit clear as an empty current V1 record. No
      // resolved default or profile value is stored.
      storage.setItem(presentationStorageKey, JSON.stringify(device));
      persistence = 'available';
    } catch { persistence = 'memory'; }
    return emit();
  };
  return {
    snapshot,
    set(field, value) {
      if (!['language', 'theme'].includes(field)) throw new Error('Unsupported presentation field.');
      const next = validatePresentation({ ...device, [field]: value });
      device = next;
      return persist();
    },
    clear(field) {
      if (!['language', 'theme'].includes(field)) throw new Error('Unsupported presentation field.');
      const next = { ...device }; delete next[field]; device = next;
      return persist();
    },
    // Not connected to a real profile service. A verified identity
    // binding must replace/clear this overlay on every identity transition.
    setProfile(value = null) { profile = value === null ? null : validatePresentation(value); return emit(); },
    setBrowserLanguages(value) { browserLanguages = value; return emit(); },
    subscribe(listener) { listeners.add(listener); return () => listeners.delete(listener); },
  };
}
