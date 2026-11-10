import { helpRelease } from './help-release.js';
import { readHelpJson, validateHelpManifest, validateHelpCatalog, validateHelpPostings, validateHelpArticle } from './help-contract.js';
import { searchHelp } from './help-search.js';

// Static assets only. A query is never part of a URL, request body, browser
// storage or diagnostic event. Fetch/ranking failures do not touch editor state.
export function createHelpStore(fetcher = fetch) {
  let manifestPromise;
  let catalogPromise;
  let indexPromise;
  const requests = new Map();
  const articles = new Map();
  let cachedBytes = 0;
  const maximumCachedBytes = 2 * 1024 * 1024;
  const maximumCachedArticles = 32;
  const fetchJson = async descriptor => readHelpJson(await fetcher(descriptor.path, { credentials: 'same-origin', cache: 'default', referrerPolicy: 'no-referrer' }), descriptor);
  const manifest = () => {
    manifestPromise ||= readHelpJsonResponse().catch(error => { manifestPromise = null; throw error; });
    return manifestPromise;
  };
  async function readHelpJsonResponse() {
    return validateHelpManifest(await readHelpJson(await fetcher(helpRelease.manifest.path, { credentials: 'same-origin', cache: 'no-store', referrerPolicy: 'no-referrer' }), helpRelease.manifest));
  }
  const catalog = () => {
    catalogPromise ||= manifest().then(async current => validateHelpCatalog(await fetchJson(current.catalog), current.contentVersion)).catch(error => { catalogPromise = null; throw error; });
    return catalogPromise;
  };
  async function search(query) {
    if (!String(query).trim()) return [];
    const current = await manifest();
    const documentsPromise = catalog();
    // A cold search downloads metadata and postings concurrently after the
    // tiny manifest. Ordinary browsing still never requests the search index.
    indexPromise ||= Promise.all([fetchJson(current.search), documentsPromise]).then(([value, documents]) => validateHelpPostings(value, current.contentVersion, documents.documents)).catch(error => { indexPromise = null; throw error; });
    const [documents, index] = await Promise.all([documentsPromise, indexPromise]);
    return searchHelp(index.postings, query).map(hit => ({ ...documents.documents[hit.document], score: hit.score }));
  }
  async function article(id) {
    const current = await manifest();
    const documents = await catalog();
    const item = documents.documents.find(document => document.id === id);
    if (!item) throw new Error('This help topic is not in the current catalog.');
    const key = item.article.path;
    if (articles.has(key)) {
      const cached = articles.get(key); articles.delete(key); articles.set(key, cached);
      return cached.value;
    }
    if (!requests.has(key)) requests.set(key, fetchJson(item.article).then(value => {
      const checked = validateHelpArticle(value, current.contentVersion, id);
      articles.set(key, { value: checked, bytes: item.article.bytes }); cachedBytes += item.article.bytes;
      while (cachedBytes > maximumCachedBytes || articles.size > maximumCachedArticles) {
        const oldest = articles.keys().next().value;
        cachedBytes -= articles.get(oldest).bytes; articles.delete(oldest);
      }
      return checked;
    }).finally(() => requests.delete(key)));
    return requests.get(key);
  }
  return { catalog, search, article };
}
