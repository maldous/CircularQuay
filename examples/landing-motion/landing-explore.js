import { exploreHomeMarkup } from './explore-navigation.js';

const mount = document.querySelector('[data-explore-menu]');
if (mount) mount.innerHTML = exploreHomeMarkup({ compact: true });
