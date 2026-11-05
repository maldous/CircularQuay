// IDs mirror the reviewed authoring producer's public-schema naming rule.
// This receives a public schema pattern, never a user's concrete map key.
export async function schemaHelpTopic(pointer) {
  if (typeof pointer !== 'string' || !/^\/(?:[a-z0-9_]+|\{id\}|\*)(?:\/(?:[a-z0-9_]+|\{id\}|\*))*$/.test(pointer) && pointer !== '/') return null;
  const hash = [...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(pointer)))].map(value => value.toString(16).padStart(2, '0')).join('');
  const slug = pointer.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
  return `control-schema-${slug}-${hash.slice(0, 12)}`;
}

let installed = false;
let helpModule;
const topics = { configuration: 'policy', policy: 'policy', workbench: 'workbench', sources: 'providers', workloads: 'workbench', inheritance: 'policy', windows: 'availability', custody: 'availability', migration: 'availability', fleet: 'management', integrations: 'enterprise', telemetry: 'operations', identity: 'enterprise', settings: 'support', billing: 'management', profiles: 'management', usage: 'operations', support: 'support', members: 'management', groups: 'management', customers: 'management', associations: 'management', grants: 'management', roles: 'management', operations: 'operations', audit: 'operations' };
const currentTopic = () => topics[location.hash.replace(/^#\//, '').split('/').at(-1)] || 'start';

export function installHelp() {
  if (installed) return;
  installed = true;
  const launcher = document.createElement('button');
  launcher.lang = 'en';
  launcher.id = 'help-launcher'; launcher.className = 'help-launcher'; launcher.type = 'button';
  launcher.textContent = 'Help & search'; launcher.setAttribute('aria-haspopup', 'dialog'); launcher.setAttribute('aria-keyshortcuts', 'F1');
  const status = document.createElement('p'); status.lang = 'en'; status.id = 'help-launcher-status'; status.className = 'sr-only'; status.setAttribute('role', 'status');
  document.body.append(launcher, status); document.body.classList.add('has-help');
  async function open(topic, trigger, schemaPointer) {
    launcher.disabled = true;
    try {
      if (schemaPointer) topic = await schemaHelpTopic(schemaPointer) || topic;
      helpModule ||= import('./help.js').catch(error => { helpModule = null; throw error; });
      (await helpModule).openHelp(topic || currentTopic(), trigger);
      status.textContent = '';
    } catch { status.textContent = 'Help could not load. Your workspace is unchanged. Try Help & search again.'; }
    finally { launcher.disabled = false; }
  }
  launcher.onclick = () => open(currentTopic(), launcher);
  document.addEventListener('click', event => {
    const trigger = event.target.closest('[data-help-topic]');
    if (trigger) { event.preventDefault(); open(trigger.dataset.helpTopic, trigger); }
  });
  document.addEventListener('keydown', event => {
    if (event.key === 'F1' && !document.querySelector('.help-dialog[open]')) { event.preventDefault(); open(document.activeElement?.dataset.helpTopic || currentTopic(), document.activeElement, document.activeElement?.closest('[data-help-schema]')?.dataset.helpSchema); }
  });
}
