import { initialiseBrowserServices } from './browser-services.js';
import { SURFACES } from './service-config.js';
const surface = document.body.classList.contains('login-page') ? SURFACES.AUTH_PAGE : SURFACES.PUBLIC_LANDING;
export const browserServices = initialiseBrowserServices({ surface });
window.addEventListener('pagehide', () => browserServices.dispose(), { once: true });
