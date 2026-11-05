// Read only this prototype's preference; never read or change workbench storage.
try {
  const theme = localStorage.getItem('circularquay-motion-theme');
  if (theme === 'light' || theme === 'dark') document.documentElement.dataset.theme = theme;
} catch {
  // A browser that blocks storage still follows the system theme.
}
