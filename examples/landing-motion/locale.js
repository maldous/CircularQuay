const storageKey = 'circularquay-motion-language';

const copy = {
  en: {
    optionalConnections: 'Optional connections', title1: 'The right secrets,', title2: 'ready for your apps.', explore: 'Explore', login: 'Log in',
    navFlow: 'How it works', navEcosystem: 'Ecosystem', roleAuth: 'Authentication',
    roleLocal: 'Encrypted', roleRemote: 'Encrypted', roleShell: 'Shell',
    roleContainer: 'Containers', roleCloud: 'Cloud', roleCI: 'CI / CD', localFile: 'Local file',
    localPolicy: 'Local policy', onDevice: 'On your device', policyScope: 'Managed policies',
    ecosystemHeading: 'A familiar ecosystem.',
    ecosystemDescription: 'Bring identification, encryption and applications together through the policy you control.',
    authHeading: 'Authentication', authDescription: 'Who is asking? Your chosen identity providers.',
    encryptionHeading: 'Encryption', encryptionDescription: 'Where secrets live. Local files and remote sources.',
    applicationHeading: 'Application', applicationDescription: 'Where approved values go. Your tools and workloads.',
    catalogNote: 'Illustrative ecosystem examples. Integration availability depends on your setup and the selected adapter.',
    connectionHeading: 'Choose the connections you need.',
    connectionDescription: 'Key custody, policy trust and enterprise services are independent choices.',
    age: 'Age keys', ageDescription: 'Local key custody', kms: 'KMS / Transit', kmsDescription: 'Remote key operations',
    signedPolicy: 'Signed policy', signedDescription: 'Enrolled source trust', directory: 'Directory / SCIM',
    directoryDescription: 'User and group metadata', clients: 'API / SDK / MCP', clientsDescription: 'Scoped client access',
    audit: 'Audit & telemetry', auditDescription: 'Decisions and operational signals',
    connectionNote: 'Encryption, policy trust and directory metadata each have a separate role. This preview uses illustrative connections.',
    footerNote: 'Your configuration stays on your device.', previewNote: 'Local design preview. No services connected.',
    loginStatus: 'Sign-in isn’t connected in this local design preview.', languageLabel: 'Language',
  },
  ar: {
    optionalConnections: 'اتصالات اختيارية', title1: 'الأسرار الصحيحة،', title2: 'جاهزة لتطبيقاتك.', explore: 'استكشف', login: 'تسجيل الدخول',
    navFlow: 'كيف يعمل', navEcosystem: 'النظام البيئي', roleAuth: 'المصادقة',
    roleLocal: 'مشفّر', roleRemote: 'مشفّر', roleShell: 'سطر الأوامر',
    roleContainer: 'الحاويات', roleCloud: 'السحابة', roleCI: 'التكامل / النشر المستمر', localFile: 'ملف محلي',
    localPolicy: 'سياسة محلية', onDevice: 'على جهازك', policyScope: 'السياسات المُدارة',
    ecosystemHeading: 'نظام بيئي مألوف.',
    ecosystemDescription: 'اجمع الهوية والتشفير والتطبيقات عبر السياسة التي تتحكم بها.',
    authHeading: 'المصادقة', authDescription: 'من الذي يطلب؟ موفرو الهوية الذين تختارهم.',
    encryptionHeading: 'التشفير', encryptionDescription: 'حيث تحفظ الأسرار: ملفات محلية ومصادر بعيدة.',
    applicationHeading: 'التطبيقات', applicationDescription: 'إلى أين تذهب القيم المعتمدة: أدواتك وأحمالك.',
    catalogNote: 'أمثلة توضيحية للنظام البيئي. يعتمد توفر التكامل على إعدادك والمحوّل المحدد.',
    connectionHeading: 'اختر الاتصالات التي تحتاجها.',
    connectionDescription: 'حيازة المفاتيح وثقة السياسة وخدمات المؤسسة خيارات مستقلة.',
    age: 'مفاتيح Age', ageDescription: 'حيازة مفاتيح محلية', kms: 'KMS / Transit', kmsDescription: 'عمليات مفاتيح بعيدة',
    signedPolicy: 'سياسة موقّعة', signedDescription: 'ثقة مصدر مسجّل', directory: 'الدليل / SCIM',
    directoryDescription: 'بيانات المستخدمين والمجموعات', clients: 'API / SDK / MCP', clientsDescription: 'وصول عميل محدد النطاق',
    audit: 'التدقيق والقياس', auditDescription: 'قرارات وإشارات تشغيلية',
    connectionNote: 'للتشفير وثقة السياسة وبيانات الدليل أدوار منفصلة. تستخدم هذه المعاينة اتصالات توضيحية.',
    footerNote: 'يبقى إعدادك على جهازك.', previewNote: 'معاينة تصميم محلية. لا توجد خدمات متصلة.',
    loginStatus: 'تسجيل الدخول غير متصل في معاينة التصميم المحلية هذه.', languageLabel: 'اللغة',
  },
};

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
