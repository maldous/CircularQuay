# CircularQuay frontend preview

A responsive landing page, provider-selection login preview and task-oriented documentation explorer. The shared identity uses the CircularQuay logo, warm ivory and forest palette, light/dark themes and existing typography. Its landing slogan is **“Get the right secrets ready.”**

## Landing

The compact landing goes directly to its first featured Explore document; Explore home remains the grouped overview. An upward wheel or swipe from the landing opens Explore. The single-line motto is retained at every viewport. The eight-node diagram continuously animates identity providers, encrypted sources and application environments through CircularQuay. The horizontal divider between diagram and story is removed.

The motion catalogue rotates all 113 local examples: 30 identity providers, 35 encrypted sources and 48 applications, represented by 108 distinct SVGs. Fair variant bags rotate across 24 actor roles and 12 contexts; both source slots draw from all 35 sources, without a forced source. Workloads use balanced shuffled masks (none, first action, second action or both), so half include an optional feature; the first scene includes one. A complete static fallback is available without JavaScript.

Active and delivered routes, plus node overlays, use 2px emphasis; idle route bases remain lighter. The hub base has no border. Text uses 18px padding. On phones, faces form an icon/name/purpose grid with an 18px minimum row gutter. Desktop, tablet and foldable layouts include the compact Explore menu; phones use the header link. Headers share a stable root gutter and padding; at widths up to 760px the language control precedes theme so the logo and wordmark fit.

Language choices use local England and Saudi Arabia flag assets under their recorded MIT terms; see `assets/FLAGS.md`.

The full-viewport loader centres the logo, uses a neutral-grey opacity breath, then fades the logo out as the main page fades in. Its 50vw size is capped by viewport height. Reduced motion keeps the loader still. Escape closes an open search dropdown without pausing its flow; elsewhere it retains the diagram pause behavior.

## Explore

Explore opens at `/explore`. Its compact index keeps the existing six task groups and 53 product documents, removes redundant horizontal rules and repeated labels, and uses the first real manifest heading for each document summary. It has breadcrumbs, title/heading search and genuine multipart guide navigation across revision-bound readable parts. Help distribution and autocomplete are available through the implemented Explore surface; Help is an outlined text button without an icon. Long articles scroll inside the reader. Wide tables retain a 9rem minimum cell width and scroll horizontally on narrow screens.

The portable product snapshot contains 57 source files, including 53 Markdown documents and four SVGs, under `explore-source/`. Its recorded source base is `dfbcc7b0104f1e57c805892d5665c96443b9f38f`; its freeze SHA-256 is `f7132120d3ffdeb945efc5140a22fa45390bbc58090f77b47f37fa56e1db2580`. This is an explicit working-tree capture, not a claim that the captured updates are published at the base revision. Per-file hashes preserve the exact selected product bytes. Current guides use `cq`, `cq-sdk`, Python `from cq import` and TypeScript `@maldous/cq-sdk` directly. Product qualification evidence remains in the repository's `docs/qualification.md`; session history is not a public explorer input.

Reader links are checked against the selected product snapshot. Normal builds and checks use frozen sources; `--refresh-local-source` deliberately captures a new local snapshot. Documentation generation requires the packages pinned in `requirements-explore.txt`. Public documentation and asset inputs are explicitly listed in `explore-documents.json`; maintenance policies, session history and qualification-session records are excluded. New Markdown files are not automatically exported.

## Run locally

```sh
node examples/landing-motion/server.mjs
```

Open `http://localhost:5182/`, `/login` or `/explore`. The loopback preview requires Node.js 22+ and no package installation.

The twelve page HTML files are indexed authoring templates with one fixed header slot. Their single literal header is maintained in `site-header.html`; `render-site-header.mjs` composes only those named local pages, without browser-side fetching or evaluating includes. Open pages through this preview or the standalone export, not by opening an unexpanded raw template. Both paths deliver complete no-JavaScript HTML with the existing navigation and unchanged search markup, scripts, styles and accessibility references. The fragment and renderer are not public routes.

`build-public.mjs` writes complete rendered HTML to the existing ignored `.public-dist/` output and records its byte lengths and SHA-256 values in the existing upload manifest. `--check` compares every exported file with its expected rendered bytes (or unchanged raw bytes for other assets), then checks the exact manifest and Worker route module. The 902 routes and 554 public files are unchanged. Raw-template equality is not substituted for this rendered-byte check. Source templates are limited to 64 KiB and the header fragment to 8 KiB; invalid UTF-8, NUL, unknown page names, missing/duplicate slots and malformed literal headers refuse without output.


```sh
node examples/landing-motion/render-integrations.mjs --check
node examples/landing-motion/render-site-pages.mjs --check
node --test examples/landing-motion/services.test.mjs examples/landing-motion/scenario-flow.test.mjs
python examples/landing-motion/build-explore.py --check
```

## Preview boundaries

Login presents 30 identity choices in a single document scroll: provider wall, account actions, then the real footer. Provider and account-preview feedback appears only after interaction. Provider handoff, account creation and recovery require a real authentication backend. No credentials or email are persisted by the preview.

The public frontend is live at [circularquay.aldous.info](https://circularquay.aldous.info/) and [cq.aldous.info](https://cq.aldous.info/). Eighteen browser-service integration points are documented in `SERVICES.md`; adapters remain disabled until configuration and consent are qualified. The rest of the application, including backend services, is outside this deployment scope. Legal/CMP content, external accounts, real authentication, device and assistive-technology qualification remain deployment requirements. The site remains noindex.

`CANDIDATE-REVIEW.md` and `review.json` are engineering review/provenance records, separate from the product documentation. They record the design assessment, bounded checks and implementation limitations.
