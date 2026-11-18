import { createHelpStore } from './help-store.js';

const sourcePrefix = '/help/generated/';
const previewPrefix = '/client-help/generated/';

// The vendored release retains its upstream absolute descriptors. This adapter
// changes only the static preview asset location; queries never enter a URL.
export function previewHelpAssetPath(resource) {
  const value = resource instanceof URL ? resource.pathname : String(resource);
  return value.startsWith(sourcePrefix) ? `${previewPrefix}${value.slice(sourcePrefix.length)}` : value;
}

export function createPreviewHelpStore(fetcher = fetch) {
  return createHelpStore((resource, options) => fetcher(previewHelpAssetPath(resource), options));
}
