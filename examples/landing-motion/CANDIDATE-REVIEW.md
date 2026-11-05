# CircularQuay engineering review — 5 November 2026

Engineering provenance record; product guidance is in README.md and Explore.

The focused eight-node landing replaces the superseded full-flock experiment. Under the heading “Get the right secrets ready.”, it links directly to the first featured Explore document, and an upward wheel or swipe opens the grouped Explore home. The current delta keeps continuous diagram/story animation and removes only the horizontal divider between them. Desktop, tablet and foldable layouts include the same Explore orientation used on the documentation home. The removed Explore-home introductory paragraph does not appear in the deployed index. Login presents 30 source-inventory identity choices, followed by account actions and the real footer in one document scroll; local-preview feedback appears only after interaction.

## Review and polish

Two independent Impeccable assessments were used: visual/design Assessment A and detector/source/browser-evidence Assessment B. A reviewed final light/dark phone, foldable and desktop captures before B findings were synthesized. The product's supplied branding was preserved.

| Surface | Mode | Design assessment | Disposition |
| --- | --- | --- | --- |
| Landing | Persuade | 28/32; heuristics 7 and 10 not applicable | Assessment A approved the full-catalogue delta |
| Login | Operate | 33/40 | Assessment A approved the material/layout delta |
| Help / Search | Read | 33/36; heuristic 7 not applicable | Earlier assessment remains applicable |
| Explore / Markdown | Read | 34/36; heuristic 7 not applicable | Assessment A approved the compact-index delta |

The current presentation delta uses 2px emphasis for active/delivered routes and node overlays, with lighter idle route bases, a borderless hub base, 18px text padding, and a mobile icon/name/purpose face grid with an 18px minimum row gutter. The motto stays single-line. Headers use common stable gutters and padding, with language before theme through 760px. Help is an outlined text button without an icon. The full-viewport loader centres the logo with a neutral-grey opacity breath, fades it out as the page fades in, and remains still for reduced motion. Escape closes a search dropdown without pausing that flow; the diagram keeps its own Escape pause behavior.

The motion catalogue uses all 113 local entries: 30 identity providers, 35 encrypted sources and 48 applications, represented by 108 distinct SVGs. Fair variant bags rotate 24 actor roles and 12 contexts. Each source position draws from the full encrypted-source pool; no source is forced. App intents and actions are curated in the motion catalogue, while names remain the full local integration names in accessible labels and titles.

## Motion and layout evidence

The product flow uses a balanced shuffled cycle of optional-feature masks: none, first, second, or both. The first scene includes one optional workload feature, and optional feature types are randomized with distinct types when both are present. An absent optional step is absent from the sentence, route, completion highlight and consecutive numbering together. Cases have six to eight numbered clauses and twelve to sixteen 1200ms legs. Each starts at CircularQuay, visits an IDP and returns, visits an encrypted source and returns, then reaches a target and returns; selected optionals make their own request/response round trip. The default generator’s independent 0.25 API remains a historical test detail, not product presentation behavior. Reduced motion uses a complete basic static example.

Earlier story-reservation and animation evidence is historical for the superseded presentation. It does not assess the current landing presentation delta.

Current refined responsive evidence covers eight viewports. The motto remains one line; Help and theme controls have equal 44px targets; no icon/name overlap or 12px optional-route conflict was observed; mobile gutters are at least 18px and the 1024px desktop gutter is 16.25px. The footer is 33px at document bottom. At 844×390, a natural 52px scroll preserves readable nodes; the observed footer viewport coordinate reflects `window.scrollY`, while its document bottom equals footer bottom minus `body.y`.

`full-catalog-390.json` and `full-catalog-responsive.json` record a current-model probe of 1,000 scenes at two weights across 320×740, 390×844, 673×841 and 1440×900: 8,000 renders in total. It covered all 113 entries, all 24 actors and all 12 contexts, with no overflow. The maximum natural story heights were 199.92, 181.67, 129.78 and 117.80px against fixed heights of 221, 185, 151 and 156px respectively. Assessment B found four empty detector results. The final navigation proof passed all 900 local HTTP and mocked-Worker routes for bytes and MIME; six current canonical and three alias live routes matched source hashes. This does not claim a live check of every public route.

## Explore source and naming

Explore now uses the current local documentation captured on 5 November, rather than an older published GitHub revision. The portable source freeze contains 59 source files—55 Markdown documents and four referenced SVGs—rendered as 85 reader parts. Exact raw bytes and SHA-256 values are committed under explore-source; normal builds/checks use that freeze. The presentation uses current `presentationCopy`, not the removed editorial overlay. Help distribution and autocomplete are implemented in the Explore surface. The compact index preserves the existing six groups and 55 documents while removing repeated labels and horizontal rules; its summaries use the first real manifest heading. An explicit --refresh-local-source is required to replace it.

The README displays CQ and the cq command. The final source freeze is based on canonical commit `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`, with manifest SHA-256 `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Current guides use `cq-sdk`, Python `from cq import` and TypeScript `@maldous/cq-sdk` directly. Genuine historical scan, loader and security receipts retain their revision-specific paths and versions; the reader identifies these engineering records and links to current SDK guidance. Repository-context links explicitly disclose that their contents may differ from this local snapshot. All 611 reader links resolve; no inert anchors remain.

## Verification and remaining limits

Validation passed 35 tests: 24 source TAP tests, five navigation tests and six Worker edge tests. JavaScript syntax, generators, reproducible documentation generation, native static-page/catalog checks and source/public-file hashes also passed. The self-only CSP and noindex profile remain. Login provider feedback and preview account forms were exercised without persisting an email or starting authentication. A 390px phone review verified that a 320px table area retains 9rem minimum cells and uses horizontal scrolling rather than collapsing its contents. A desktop upward wheel and a mobile upward touch gesture both opened Explore; the latter was validated after state reset, multitouch and header/control exclusion fixes. Assessment A and B are complete.

The final markup detector returns no findings for all three pages. An earlier landing scan reported three compact 11px text warnings; the live diagram still deliberately uses compact noninteractive labels as a documented readability tradeoff. Narrow persistent exceptions preserve user-pinned Inter, warm ivory, and the editorial row's child-link padding. No rule or file is globally ignored. An external detector overlay was not injected through the self-only CSP; no overlay success is claimed.

This public frontend is live at `https://circularquay.aldous.info/`, with proxied CNAME alias `https://cq.aldous.info/`. It is served by Worker version `32c9376b-5ea9-4c1a-9388-457e069b2e21`, deployment `9d5fb71d-b3ea-4bb7-ab27-ee728d0b37ba`, at 100%, using only an ASSETS binding; workers.dev and preview URLs are disabled. The rest of the application, including backend services, is outside this deployment scope. Real authentication/provisioning, legal content, configured service accounts/IDs, consent qualification and vendor adapters remain pending. Eighteen discovered service hooks are recorded and disabled/fail-closed. Device, assistive-technology and physical animation-performance qualification remain outstanding; browser captures are emulation evidence.

## Custody

Direct user authority permits this landing exemption, its scoped current-branch commit and public-frontend deployment to `circularquay.aldous.info` and `cq.aldous.info` without a Linear issue. The landing owner changes only examples/landing-motion and scoped Impeccable records. Native/root rename and documentation cleanup are separately owned. The final source checkpoint is `dfb51b1f64849746113d248d80a069d42a058117` (tree `f9696e6f97751c1550b394c3588ea40dbc33af54`); the deployed Worker receipt records ETag `58d9f7bcdd22648812a36f7ca061cf1a0611ba1a8258811e25d352b48b451c0e`.

Private reproducible review evidence remains in the existing checkpoint57/landing-motion-review/candidate-after store. Earlier full-flock critiques and snapshots remain historical and superseded.

Questions skipped: the user already authorized these refinements and requested a concrete local review candidate with a commit.

## Historical initial-story correction

The former initial-story correction, its animation traces and its Assessment A approval apply only to the superseded animated presentation. They do not approve or qualify the current landing presentation delta.

Screenshot 1000015445.png could not be materialized: the coordinator reported the supported Library helper failed with “download failed,” with no executor-local readable pixels. The correction used the supplied text and actual browser verification. This intake limit is separate from the successful behavior evidence.

The builder reads frozen metadata for normal build/check, so advancing Git HEAD cannot invalidate a snapshot. Only explicit source refresh reads current HEAD/branch. Unreferenced generated assets are pruned; four remain. Current consumer README/brief/implementation guidance is factual, without personal authorization or assistant-workflow narration; dated review history and custody evidence remain in engineering provenance.
