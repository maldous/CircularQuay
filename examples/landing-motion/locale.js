const storageKey = 'circularquay-motion-language';

// Keep both translations beside one canonical key so locale key sets cannot drift.
const translations = {
  optionalConnections: ['Optional connections', 'اتصالات اختيارية'],
  title1: ['Get the right', 'جهّز الأسرار'],
  title2: ['secrets ready.', 'الصحيحة.'],
  explore: ['Explore', 'استكشف'],
  download: ['Download', 'تنزيل'],
  login: ['Login', 'تسجيل الدخول'],
  navFlow: ['How it works', 'كيف يعمل'],
  navEcosystem: ['Ecosystem', 'النظام البيئي'],
  roleAuth: ['Authentication', 'المصادقة'],
  roleLocal: ['Encrypted', 'مشفّر'],
  roleRemote: ['Encrypted', 'مشفّر'],
  roleShell: ['Shell', 'سطر الأوامر'],
  roleContainer: ['Containers', 'الحاويات'],
  roleCloud: ['Cloud', 'السحابة'],
  roleCI: ['CI / CD', 'التكامل / النشر المستمر'],
  localFile: ['Local file', 'ملف محلي'],
  localPolicy: ['Local policy', 'سياسة محلية'],
  onDevice: ['On your device', 'على جهازك'],
  policyScope: ['Managed policies', 'السياسات المُدارة'],
  ecosystemHeading: ['A familiar ecosystem.', 'نظام بيئي مألوف.'],
  ecosystemDescription: ['Bring identification, encryption and applications together through the policy you control.', 'اجمع الهوية والتشفير والتطبيقات عبر السياسة التي تتحكم بها.'],
  authHeading: ['Authentication', 'المصادقة'],
  authDescription: ['Who is asking? Your chosen identity providers.', 'من الذي يطلب؟ موفرو الهوية الذين تختارهم.'],
  encryptionHeading: ['Encryption', 'التشفير'],
  encryptionDescription: ['Where secrets live. Local files and remote sources.', 'حيث تحفظ الأسرار: ملفات محلية ومصادر بعيدة.'],
  applicationHeading: ['Application', 'التطبيقات'],
  applicationDescription: ['Where approved values go. Your tools and workloads.', 'إلى أين تذهب القيم المعتمدة: أدواتك وأحمالك.'],
  catalogNote: ['Illustrative ecosystem examples. Integration availability depends on your setup and the selected adapter.', 'أمثلة توضيحية للنظام البيئي. يعتمد توفر التكامل على إعدادك والمحوّل المحدد.'],
  connectionHeading: ['Choose the connections you need.', 'اختر الاتصالات التي تحتاجها.'],
  connectionDescription: ['Key custody, policy trust and enterprise services are independent choices.', 'حيازة المفاتيح وثقة السياسة وخدمات المؤسسة خيارات مستقلة.'],
  age: ['Age keys', 'مفاتيح Age'],
  ageDescription: ['Local key custody', 'حيازة مفاتيح محلية'],
  kms: ['KMS / Transit', 'KMS / Transit'],
  kmsDescription: ['Remote key operations', 'عمليات مفاتيح بعيدة'],
  signedPolicy: ['Signed policy', 'سياسة موقّعة'],
  signedDescription: ['Enrolled source trust', 'ثقة مصدر مسجّل'],
  directory: ['Directory / SCIM', 'الدليل / SCIM'],
  directoryDescription: ['User and group metadata', 'بيانات المستخدمين والمجموعات'],
  clients: ['API / SDK / MCP', 'API / SDK / MCP'],
  clientsDescription: ['Scoped client access', 'وصول عميل محدد النطاق'],
  audit: ['Audit & telemetry', 'التدقيق والقياس'],
  auditDescription: ['Decisions and operational signals', 'قرارات وإشارات تشغيلية'],
  connectionNote: ['Encryption, policy trust and directory metadata each have a separate role. This preview uses illustrative connections.', 'للتشفير وثقة السياسة وبيانات الدليل أدوار منفصلة. تستخدم هذه المعاينة اتصالات توضيحية.'],
  footerNote: ['Your configuration stays on your device.', 'يبقى إعدادك على جهازك.'],
  previewNote: ['Local design preview. No services connected.', 'معاينة تصميم محلية. لا توجد خدمات متصلة.'],
  loginStatus: ['Sign-in isn’t connected in this local design preview.', 'تسجيل الدخول غير متصل في معاينة التصميم المحلية هذه.'],
  languageLabel: ['Language', 'اللغة'],
};
const copy = { en: {}, ar: {} };
for (const [key, [english, arabic]] of Object.entries(translations)) {
  copy.en[key] = english;
  copy.ar[key] = arabic;
}

function readLocale() {
  try { return localStorage.getItem(storageKey) === 'ar' ? 'ar' : 'en'; } catch { return 'en'; }
}

function applyLocale(locale, persist = true) {
  const language = locale === 'ar' ? 'ar' : 'en';
  const strings = copy[language];
  document.documentElement.lang = language;
  document.documentElement.dir = language === 'ar' ? 'rtl' : 'ltr';
  document.querySelectorAll('[data-i18n]').forEach(element => {
    const value = strings[element.dataset.i18n];
    if (value) element.textContent = value;
  });
  const select = document.querySelector('#language');
  if (select) {
    select.value = language;
    select.setAttribute('aria-label', strings.languageLabel);
    select.disabled = false;
  }
  const flag = document.querySelector('.language-flag');
  if (flag) flag.src = language === 'ar' ? 'assets/flag-ar.svg' : 'assets/flag-en.svg';
  const status = document.querySelector('#login-status');
  if (status && !status.hidden) status.textContent = strings.loginStatus;
  if (persist) try { localStorage.setItem(storageKey, language); } catch { /* local preview remains usable */ }
  window.dispatchEvent(new CustomEvent('localechange', { detail: { locale: language } }));
}

function initialise() {
  const select = document.querySelector('#language');
  applyLocale(readLocale(), false);
  select?.addEventListener('change', event => applyLocale(event.currentTarget.value));

  const login = document.querySelector('#login');
  const status = document.querySelector('#login-status');
  let timer;
  if (login) login.disabled = false;
  login?.addEventListener('click', () => {
    if (!status) return;
    status.textContent = copy[document.documentElement.lang === 'ar' ? 'ar' : 'en'].loginStatus;
    status.hidden = false;
    clearTimeout(timer);
    timer = setTimeout(() => { status.hidden = true; }, 8000);
  });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', initialise, { once: true });
else initialise();
