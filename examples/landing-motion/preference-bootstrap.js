document.documentElement.dataset.reducedMotion = String(matchMedia('(prefers-reduced-motion: reduce)').matches);

try {
  const saved = localStorage.getItem('circularquay-motion-theme');
  document.documentElement.dataset.theme = ['light', 'dark', 'system'].includes(saved) ? saved : 'light';
} catch {
  document.documentElement.dataset.theme = 'light';
}

try {
  const language = localStorage.getItem('circularquay-motion-language');
  const locale = language === 'ar' ? 'ar' : 'en';
  document.documentElement.lang = locale;
  document.documentElement.dir = locale === 'ar' ? 'rtl' : 'ltr';
} catch {
  document.documentElement.lang = 'en';
  document.documentElement.dir = 'ltr';
}
