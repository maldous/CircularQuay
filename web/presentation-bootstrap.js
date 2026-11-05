// Runs before CSS. Only a bounded presentation record is read; no defaults,
// profile values, workspace contents or identity are written or requested.
(() => {
  let preferences = {};
  try {
    const raw = localStorage.getItem('cq.presentation.v1');
    if (raw !== null) {
      if (raw.length > 256) throw Error();
      const value = JSON.parse(raw);
      if (!value || Array.isArray(value) || value.version !== 1 || Object.keys(value).some(key => !['version', 'language', 'theme'].includes(key)) || Object.hasOwn(value, 'language') && !['system', 'en', 'ar'].includes(value.language) || Object.hasOwn(value, 'theme') && !['system', 'light', 'dark'].includes(value.theme)) throw Error();
      preferences = value;
    }
  } catch { /* Browser defaults remain usable when storage is blocked/invalid. */ }
  let browserLanguage = 'en';
  for (const candidate of (navigator.languages || []).slice(0, 32)) {
    if (typeof candidate !== 'string' || candidate.length > 100) continue;
    try { const language = new Intl.Locale(candidate).language; if (['en', 'ar'].includes(language)) { browserLanguage = language; break; } } catch {}
  }
  const language = ['en', 'ar'].includes(preferences.language) ? preferences.language : browserLanguage;
  document.documentElement.lang = language;
  document.documentElement.dir = language === 'ar' ? 'rtl' : 'ltr';
  document.documentElement.dataset.theme = preferences.theme || 'system';
})();
