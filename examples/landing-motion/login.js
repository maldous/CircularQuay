const localeCopy = {
  en: { provider: 'Sign-in with {name} is not connected in this local preview.', email: 'Email address', createMode: 'New account', resetMode: 'Reset password', createCta: 'Create account', resetCta: 'Send reset link', createPending: 'Account creation is not connected in this local preview.', resetPending: 'Password reset is not connected in this local preview.' },
  ar: { provider: 'تسجيل الدخول عبر {name} غير متصل في هذه المعاينة المحلية.', email: 'عنوان البريد الإلكتروني', createMode: 'حساب جديد', resetMode: 'إعادة تعيين كلمة المرور', createCta: 'إنشاء حساب', resetCta: 'إرسال رابط إعادة التعيين', createPending: 'إنشاء الحساب غير متصل في هذه المعاينة المحلية.', resetPending: 'إعادة تعيين كلمة المرور غير متصل في هذه المعاينة المحلية.' },
};

let mode = 'create';
let selectedProvider = '';
const language = () => document.documentElement.lang === 'ar' ? 'ar' : 'en';
const strings = () => localeCopy[language()];
function renderProviderStatus() { document.querySelector('#provider-status').textContent = selectedProvider ? strings().provider.replace('{name}', selectedProvider) : ''; }
function setMode(nextMode) {
  mode = nextMode === 'reset' ? 'reset' : 'create'; const copy = strings();
  document.querySelectorAll('[data-mode]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.mode === mode)));
  document.querySelector('#account-submit').textContent = mode === 'create' ? copy.createCta : copy.resetCta;
  document.querySelector('#account-status').textContent = '';
}
function applyLoginLocale() {
  const copy = strings(); document.querySelectorAll('[data-i18n-login]').forEach(element => { element.textContent = copy[element.dataset.i18nLogin]; });
  document.querySelector('#email').setAttribute('aria-label', copy.email);
}
function init() {
  const choices = [...document.querySelectorAll('.provider-choice')];
  document.querySelector('.provider-grid').disabled = false; document.querySelector('#account-form fieldset').disabled = false;
  choices.forEach(choice => choice.addEventListener('click', () => { selectedProvider = choice.dataset.provider; renderProviderStatus(); }));
  document.querySelectorAll('[data-mode]').forEach(button => button.addEventListener('click', () => setMode(button.dataset.mode)));
  document.querySelector('#account-form').addEventListener('submit', event => { event.preventDefault(); const email = document.querySelector('#email'); if (!email.checkValidity()) { email.reportValidity(); return; } document.querySelector('#account-status').textContent = mode === 'create' ? strings().createPending : strings().resetPending; });
  window.addEventListener('localechange', () => { applyLoginLocale(); setMode(mode); renderProviderStatus(); }); applyLoginLocale(); setMode(mode);
}
if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init, { once: true }); else init();
