import { installPresentation } from './presentation.js';
import { mountHelp } from './help.js';
installPresentation();
const root = document.querySelector('#help-page');
root.lang = 'en';
const topic = location.hash.slice(1);
const viewer = mountHelp(root, { topic: /^[a-z][a-z0-9.-]{0,100}$/.test(topic) ? topic : '', standalone: true });
viewer.focus();
