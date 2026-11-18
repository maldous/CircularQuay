import { mountHelp } from './client-help/help.js';

const root = document.querySelector('#help-page');
let viewer;
function render() {
  viewer?.dispose();
  const topic = location.hash.slice(1);
  viewer = mountHelp(root, {
    topic: /^[a-z][a-z0-9.-]{0,100}$/.test(topic) ? topic : 'start',
    standalone: true,
  });
}
addEventListener('hashchange', render);
render();
