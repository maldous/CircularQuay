const root = document.documentElement;
const themeToggle = document.querySelector('#theme-toggle');
const darkScheme = matchMedia('(prefers-color-scheme: dark)');
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)');
const themeStorageKey = 'circularquay-motion-theme';

function resolvedTheme() {
  const preference = root.dataset.theme || 'system';
  return preference === 'system' ? (darkScheme.matches ? 'dark' : 'light') : preference;
}

function themeToggleLabel(theme) {
  if (root.lang.toLowerCase().startsWith('ar')) {
    return theme === 'dark' ? 'التبديل إلى المظهر الفاتح' : 'التبديل إلى المظهر الداكن';
  }
  return theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme';
}

function updateThemeToggle() {
  const theme = resolvedTheme();
  themeToggle?.setAttribute('aria-pressed', String(theme === 'dark'));
  themeToggle?.setAttribute('aria-label', themeToggleLabel(theme));
}

function setTheme(value) {
  root.dataset.theme = value;
  updateThemeToggle();
  try { localStorage.setItem(themeStorageKey, value); } catch { /* Preference remains for this visit. */ }
}

function applyReducedMotion() {
  root.dataset.reducedMotion = String(reducedMotion.matches);
  window.dispatchEvent(new CustomEvent('appearance:reduced-motion', { detail: { reduced: reducedMotion.matches } }));
}

themeToggle?.addEventListener('click', () => setTheme(resolvedTheme() === 'dark' ? 'light' : 'dark'));
darkScheme.addEventListener('change', () => {
  if ((root.dataset.theme || 'system') === 'system') updateThemeToggle();
});
reducedMotion.addEventListener('change', applyReducedMotion);
window.addEventListener('localechange', updateThemeToggle);

if (themeToggle) themeToggle.disabled = false;
updateThemeToggle();
applyReducedMotion();
