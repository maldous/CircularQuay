import { createPresentationPreferences } from './presentation-preferences.js';
let preferences;
let bar;
let panel;
let installed = false;
const copy = {
  en: { group: 'Language & theme', language: 'Language / اللغة', theme: 'Theme', default: 'Automatic', systemLanguage: 'Browser only', system: 'Device system', light: 'Light', dark: 'Dark', device: 'Device choice', profile: 'Profile default', browser: 'Browser preference', updated: 'Presentation updated. Current work is unchanged.', memory: 'Preferences apply for this page; browser saving is unavailable.', incomplete: 'Arabic is a preview; product pages are currently in English.', arabic: 'العربية (معاينة)', rule: 'Automatic uses profile defaults where available, otherwise browser preferences. Browser only and Device system always follow this device.' },
  ar: { group: 'اللغة والمظهر', language: 'اللغة / Language', theme: 'المظهر', default: 'تلقائي', systemLanguage: 'المتصفح فقط', system: 'نظام الجهاز', light: 'فاتح', dark: 'داكن', device: 'اختيار هذا الجهاز', profile: 'الإعداد الافتراضي للملف الشخصي', browser: 'تفضيل المتصفح', updated: 'تم تحديث المظهر واللغة مع الاحتفاظ بالعمل الحالي.', memory: 'تُطبَّق التفضيلات على هذه الصفحة؛ الحفظ في المتصفح غير متاح.', incomplete: 'العربية متاحة للمعاينة؛ صفحات المنتج بالإنجليزية حاليًا.', arabic: 'العربية (معاينة)', rule: 'يستخدم الإعداد التلقائي تفضيلات الملف الشخصي إن توفرت، وإلا فتفضيلات المتصفح. يلتزم خيارا المتصفح فقط ونظام الجهاز بتفضيلات هذا الجهاز دائمًا.' },
};
export function presentationPreferences() {
  if (!preferences) {
    let storage;
    try { storage = window.localStorage; } catch {}
    preferences = createPresentationPreferences({ storage, browserLanguages: [...(navigator.languages || [])] });
  }
  return preferences;
}
export function applyPresentation() {
  const value = presentationPreferences().snapshot();
  document.documentElement.lang = value.language;
  document.documentElement.dir = value.direction;
  document.documentElement.dataset.theme = value.theme;
  return value;
}
export function installPresentation() {
  if (installed) return presentationPreferences();
  installed = true;
  const controller = presentationPreferences();
  panel = document.createElement('details');
  panel.className = 'presentation-panel';
  const summary = document.createElement('summary');
  summary.innerHTML = '<span data-summary-group></span> · <bdi data-summary-language></bdi> · <bdi data-summary-theme></bdi>';
  panel.append(summary);
  bar = document.createElement('section');
  bar.className = 'presentation-controls';
  bar.innerHTML = '<div class="presentation-field"><label for="presentation-language"></label><select id="presentation-language" data-presentation-field="language" aria-describedby="presentation-language-source presentation-language-boundary presentation-rule"><option value="" data-copy="default"></option><option value="system" data-copy="systemLanguage"></option><option value="en" lang="en">English</option><option value="ar" lang="ar">العربية</option></select><small id="presentation-language-source" data-source="language"></small><small id="presentation-language-boundary"></small></div><div class="presentation-field"><label for="presentation-theme"></label><select id="presentation-theme" data-presentation-field="theme" aria-describedby="presentation-theme-source presentation-rule"><option value="" data-copy="default"></option><option value="system" data-copy="system"></option><option value="light" data-copy="light"></option><option value="dark" data-copy="dark"></option></select><small id="presentation-theme-source" data-source="theme"></small></div><p id="presentation-rule" class="presentation-rule"></p><p class="presentation-message" role="status" aria-live="polite" aria-atomic="true"></p>';
  panel.append(bar);
  document.body.prepend(panel);
  const publicRoute = () => { panel.open = !location.hash || /^#\/(?:landing)?$/.test(location.hash); };
  publicRoute(); window.addEventListener('hashchange', publicRoute);
  const systemTheme = window.matchMedia?.('(prefers-color-scheme: dark)');
  function update(value, announce = false) {
    applyPresentation();
    const strings = copy[value.language];
    panel.lang = value.language;
    bar.setAttribute('aria-label', strings.group);
    const languageName = value.language === 'ar' ? strings.arabic : value.languageChoice === 'system' && value.browserLanguageSource === 'fallback' ? 'English (fallback)' : 'English';
    const themeName = strings[value.theme === 'system' ? systemTheme?.matches ? 'dark' : 'light' : value.theme];
    summary.querySelector('[data-summary-group]').textContent = strings.group;
    summary.querySelector('[data-summary-language]').textContent = languageName;
    summary.querySelector('[data-summary-language]').lang = value.language;
    summary.querySelector('[data-summary-theme]').textContent = themeName;
    bar.querySelector('#presentation-rule').textContent = strings.rule;
    bar.querySelector('#presentation-language-boundary').textContent = value.language === 'ar' ? strings.incomplete : '';
    bar.querySelector('option[value=ar]').textContent = strings.arabic;
    bar.querySelector('[for="presentation-language"]').textContent = strings.language;
    bar.querySelector('[for="presentation-theme"]').textContent = strings.theme;
    for (const option of bar.querySelectorAll('[data-copy]')) option.textContent = strings[option.dataset.copy];
    for (const field of ['language', 'theme']) {
      bar.querySelector(`[data-presentation-field="${field}"]`).value = value.device[field] || '';
      const outcome = document.createElement('bdi'); outcome.textContent = field === 'language' ? languageName : themeName;
      outcome.lang = value.language;
      bar.querySelector(`[data-source="${field}"]`).replaceChildren(document.createTextNode(strings[value[field + 'Source']] + ' · '), outcome);
      // Synchronize only the preference inputs; never rebuild an ongoing form.
      for (const input of document.querySelectorAll(`[data-presentation-setting="${field}"]`)) {
        input.value = value.device[field] || '';
        const labels = new Map(optionLabels(field, value));
        for (const option of input.options) {
          option.textContent = labels.get(option.value);
          option.lang = ['en', 'ar'].includes(option.value) ? option.value : value.language;
        }
      }
    }
    bar.querySelector('[role="status"]').textContent = [value.persistence === 'memory' ? strings.memory : announce ? strings.updated : ''].filter(Boolean).join(' ');
    document.dispatchEvent(new CustomEvent('presentationchange', { detail: value }));
  }
  bar.addEventListener('change', event => {
    const field = event.target.dataset.presentationField;
    if (field) event.target.value ? controller.set(field, event.target.value) : controller.clear(field);
  });
  controller.subscribe(value => update(value, true));
  window.addEventListener('languagechange', () => controller.setBrowserLanguages([...(navigator.languages || [])]));
  systemTheme?.addEventListener('change', () => update(controller.snapshot()));
  update(controller.snapshot());
  return controller;
}

function optionLabels(field, value) {
  const strings = copy[value.language];
  return field === 'language' ? [['', strings.default], ['system', strings.systemLanguage], ['en', 'English'], ['ar', strings.arabic]] : [['', strings.default], ['system', strings.system], ['light', strings.light], ['dark', strings.dark]];
}

export function presentationOptions(field) {
  const value = presentationPreferences().snapshot();
  return optionLabels(field, value).map(([key, label]) => `<option value="${key}" lang="${key === 'en' || key === 'ar' ? key : value.language}" ${value.device[field] === key || !Object.hasOwn(value.device, field) && key === '' ? 'selected' : ''}>${label}</option>`).join('');
}
