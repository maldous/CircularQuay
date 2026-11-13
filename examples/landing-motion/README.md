# CircularQuay frontend preview

A responsive landing page, provider-selection login preview and task-oriented documentation explorer. The shared identity uses the CircularQuay logo, warm ivory and forest palette, light/dark themes and existing typography. Its landing slogan is **“Get the right secrets ready.”**

## Landing

The compact landing goes directly to its first featured Explore document; Explore home remains the grouped overview. An upward wheel or swipe from the landing opens Explore. The single-line motto is retained at every viewport. The eight-node diagram continuously animates identity providers, encrypted sources and application environments through CircularQuay. The horizontal divider between diagram and story is removed.

The motion catalogue rotates all 113 local examples: 30 identity providers, 35 encrypted sources and 48 applications, represented by 108 distinct SVGs. Fair variant bags rotate across 24 actor roles and 12 contexts; both source slots draw from all 35 sources, without a forced source. Workloads use balanced shuffled masks (none, first action, second action or both), so half include an optional feature; the first scene includes one. A complete static fallback is available without JavaScript.

Active and delivered routes, plus node overlays, use 2px emphasis; idle route bases remain lighter. The hub base has no border. Text uses 18px padding. On phones, faces form an icon/name/purpose grid with an 18px minimum row gutter. Desktop, tablet and foldable layouts include the compact Explore menu; phones use the header link. Headers share a stable root gutter and padding; at widths up to 760px the language control precedes theme so the logo and wordmark fit.

Language choices use local England and Saudi Arabia flag assets under their recorded MIT terms; see `assets/FLAGS.md`.

The full-viewport loader centres the logo, uses a neutral-grey opacity breath, then fades the logo out as the main page fades in. Its 50vw size is capped by viewport height. Reduced motion keeps the loader still. Escape closes an open search dropdown without pausing its flow; elsewhere it retains the diagram pause behavior.

## Explore

Explore opens at `/explore`. Its compact index keeps the existing six task groups and 55 documents, removes redundant horizontal rules and repeated labels, and uses the first real manifest heading for each document summary. It has breadcrumbs, title/heading search and genuine multipart guide navigation across 85 readable parts. Help distribution and autocomplete are available through the implemented Explore surface; Help is an outlined text button without an icon. Long articles scroll inside the reader. Wide tables retain a 9rem minimum cell width and scroll horizontally on narrow screens.

The portable local snapshot contains 59 source files, including 55 Markdown documents and four SVGs, under `explore-source/`. Its canonical base is `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`; its freeze SHA-256 is `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Per-file hashes preserve the exact local bytes. Current guides use `cq`, `cq-sdk`, Python `from cq import` and TypeScript `@maldous/cq-sdk` directly. Dated engineering records preserve revision-specific paths and tool versions and point readers to the current SDK guides.

All 611 reader links resolve. Normal builds and checks use frozen sources; `--refresh-local-source` deliberately captures a new local snapshot. Documentation generation requires the packages pinned in `requirements-explore.txt`. Public documentation and asset inputs are explicitly listed in `explore-documents.json`; maintenance policies, session history and qualification-session records are excluded. New Markdown files are not automatically exported.

## Run locally

```sh
node examples/landing-motion/server.mjs
```

Open `http://localhost:5182/`, `/login` or `/explore`. The loopback preview requires Node.js 22+ and no package installation.

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
