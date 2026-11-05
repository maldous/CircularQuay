const localeCopy = {
  en: { preview: 'Local preview · no provider is connected', title: 'Login to CircularQuay', intro: 'Choose your identity provider.', providerTitle: 'Continue with your provider', providerDescription: 'Thirty identity options.', findProvider: 'Find your provider', provider: 'Sign-in with {name} is not connected in this local preview.', noMatch: 'No providers match that search.', accountTitle: 'New to CircularQuay?', accountDescription: 'Create an account or reset access in this local preview.', email: 'Email address', create: 'Create account', reset: 'Reset password', createLegend: 'Create an account', resetLegend: 'Reset password', createPending: 'Account creation is not connected in this local preview.', resetPending: 'Password reset is not connected in this local preview.' },
  ar: { preview: 'معاينة محلية · لا يوجد موفر متصل', title: 'سجّل الدخول إلى CircularQuay', intro: 'اختر موفر الهوية.', providerTitle: 'تابع عبر موفر الهوية', providerDescription: 'ثلاثون خيار هوية.', findProvider: 'ابحث عن موفر الهوية', provider: 'تسجيل الدخول عبر {name} غير متصل في هذه المعاينة المحلية.', noMatch: 'لا يوجد موفرو هوية يطابقون البحث.', accountTitle: 'هل أنت جديد في CircularQuay؟', accountDescription: 'أنشئ حساباً أو أعد تعيين الوصول في هذه المعاينة المحلية.', email: 'عنوان البريد الإلكتروني', create: 'إنشاء حساب', reset: 'إعادة تعيين كلمة المرور', createLegend: 'إنشاء حساب', resetLegend: 'إعادة تعيين كلمة المرور', createPending: 'إنشاء الحساب غير متصل في هذه المعاينة المحلية.', resetPending: 'إعادة تعيين كلمة المرور غير متصل في هذه المعاينة المحلية.' },
};

let mode = 'create';
let selectedProvider = '';
const language = () => document.documentElement.lang === 'ar' ? 'ar' : 'en';
const strings = () => localeCopy[language()];

function renderProviderStatus() {
  const status = document.querySelector('#provider-status');
  if (status) status.textContent = selectedProvider ? strings().provider.replace('{name}', selectedProvider) : '';
}

function setMode(nextMode) {
  mode = nextMode === 'reset' ? 'reset' : 'create';
  const copy = strings();
  document.querySelectorAll('[data-mode]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.mode === mode)));
  document.querySelector('#account-legend').textContent = mode === 'create' ? copy.createLegend : copy.resetLegend;
  document.querySelector('#account-submit').textContent = mode === 'create' ? copy.create : copy.reset;
  document.querySelector('#account-status').textContent = '';
}

function applyLoginLocale() {
  const copy = strings();
  document.querySelectorAll('[data-i18n-login]').forEach(element => { element.textContent = copy[element.dataset.i18nLogin]; });
  const search = document.querySelector('#provider-search');
  search.placeholder = copy.findProvider;
  search.setAttribute('aria-label', copy.findProvider);
}

function init() {
  const search = document.querySelector('#provider-search');
  const choices = [...document.querySelectorAll('.provider-choice')];
  const noProviders = document.querySelector('#no-providers');
  search.disabled = false;
  document.querySelector('.provider-grid').disabled = false;
  document.querySelector('#account-form fieldset').disabled = false;
  choices.forEach(choice => choice.addEventListener('click', () => { selectedProvider = choice.dataset.provider; renderProviderStatus(); }));
  search.addEventListener('input', () => {
    const term = search.value.trim().toLocaleLowerCase();
    let visible = 0;
    choices.forEach(choice => { const matches = choice.dataset.provider.toLocaleLowerCase().includes(term); choice.hidden = !matches; if (matches) visible += 1; });
    noProviders.hidden = visible !== 0;
    noProviders.textContent = strings().noMatch;
  });
  document.querySelectorAll('[data-mode]').forEach(button => button.addEventListener('click', () => setMode(button.dataset.mode)));
  document.querySelector('#account-form').addEventListener('submit', event => {
    event.preventDefault();
    const email = document.querySelector('#email');
    if (!email.checkValidity()) { email.reportValidity(); return; }
    document.querySelector('#account-status').textContent = mode === 'create' ? strings().createPending : strings().resetPending;
  });
  window.addEventListener('localechange', () => { applyLoginLocale(); setMode(mode); renderProviderStatus(); });
  applyLoginLocale();
  setMode(mode);
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init, { once: true }); else init();
