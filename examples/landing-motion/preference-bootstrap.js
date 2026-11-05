try {
  const saved = localStorage.getItem('circularquay-motion-theme');
  document.documentElement.dataset.theme = ['light', 'dark', 'system'].includes(saved) ? saved : 'light';
} catch {
  document.documentElement.dataset.theme = 'light';
}
