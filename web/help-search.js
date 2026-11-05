const maximumQueryLength = 256;
const maximumQueryTerms = 32;

// Preserve whole identifiers and their word components so ordinary language,
// CLI flags, snake_case and camelCase names all address the same local corpus.
export function searchTerms(text) {
  const normalized = String(text).normalize('NFKC').replace(/[\u0610-\u061a\u0640\u064b-\u065f\u0670\u06d6-\u06ed]/g, '').replace(/[أإآٱ]/g, 'ا').replace(/ى/g, 'ي');
  const identifier = /^[\p{L}\p{N}]+(?:[._/][\p{L}\p{N}]+)+$/u.test(normalized) ? [normalized.toLowerCase().replace(/[._/]/g, '')] : [];
  const complete = normalized.toLowerCase().match(/[\p{L}\p{N}]+/gu) || [];
  const components = normalized.replace(/([a-z0-9])([A-Z])/g, '$1 $2').toLowerCase().match(/[\p{L}\p{N}]+/gu) || [];
  return [...new Set([...complete, ...components, ...identifier])];
}

export function buildHelpPostings(entries) {
  const terms = new Map();
  for (const [document, entry] of entries.entries()) {
    const fields = [[entry.title, 12], [entry.summary, 5], [entry.keywords.join(' '), 8], [entry.audiences.join(' '), 1],  [entry.bodyMarkdown, 2]];
    const bindings = [...new Set((entry.controlFamilies || []).flatMap(searchTerms))];
    fields.push([bindings.join(' '), 24]);
    for (const [text, weight] of fields) for (const term of searchTerms(text)) {
      if (!terms.has(term)) terms.set(term, new Map());
      const documents = terms.get(term);
      documents.set(document, (documents.get(document) || 0) + weight);
    }
  }
  return Object.fromEntries([...terms].sort(([left], [right]) => left.localeCompare(right, 'en')).map(([term, documents]) => [term, [...documents]]));
}

export function searchHelp(postings, query, maximumResults = 30) {
  const terms = searchTerms(String(query).slice(0, maximumQueryLength)).slice(0, maximumQueryTerms);
  if (!terms.length) return [];
  let hits;
  for (const [position, term] of terms.entries()) {
    const matches = new Map(Object.hasOwn(postings, term) ? postings[term] : []);
    // Prefix completion applies only to an unfinished last word. It never
    // substitutes unrelated results when another query term is absent.
    if (!matches.size && position === terms.length - 1 && term.length >= 2) {
      for (const candidate of Object.keys(postings)) if (candidate.startsWith(term)) {
        for (const [document, weight] of postings[candidate]) matches.set(document, Math.max(matches.get(document) || 0, weight / 2));
      }
    }
    hits = hits ? new Map([...hits].filter(([document]) => matches.has(document)).map(([document, weight]) => [document, weight + matches.get(document)])) : matches;
    if (!hits.size) return [];
  }
  return [...hits].sort((left, right) => right[1] - left[1] || left[0] - right[0]).slice(0, Math.min(100, Math.max(1, maximumResults))).map(([document, score]) => ({ document, score }));
}
