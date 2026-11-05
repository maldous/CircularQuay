# CircularQuay frontend preview

A responsive landing page, provider-selection login preview and task-oriented documentation explorer. The shared identity uses the CircularQuay logo, warm ivory and forest palette, light/dark themes and existing typography. Its landing slogan is **“Get the right secrets ready.”**

## Landing

The compact landing goes directly to its first featured Explore document; Explore home remains the grouped overview. The single-line motto is retained at every viewport. The eight-node diagram continuously animates identity providers, encrypted sources and application environments through CircularQuay. The horizontal divider between diagram and story is removed.

All overlays and routes use 2px strokes; the hub base has no border. Text uses 18px padding. On phones, faces form an icon/name/purpose grid with an 18px minimum row gutter. Desktop, tablet and foldable layouts include the compact Explore menu; phones use the header link. Headers share a stable root gutter and padding; at widths up to 760px the language control precedes theme so the logo and wordmark fit.

Language choices use local England and Saudi Arabia flag assets under their recorded MIT terms; see `assets/integrations/FLAGS.md`.

The full-viewport loader centres the logo, uses a neutral-grey breath and a clockwise arc clipped within the original C-shaped logo band, then fades the logo out as the main page fades in. Its 50vw size is capped by viewport height. Reduced motion keeps the loader still. Escape closes an open search dropdown without pausing its flow; elsewhere it retains the diagram pause behavior.

## Explore

Explore opens at `/explore`. Six task groups organize 55 documents into 85 readable parts, with breadcrumbs, title/heading search and genuine multipart guide navigation. Help distribution and autocomplete are available through the implemented Explore surface; Help is an outlined text button without an icon. Long articles scroll inside the reader.

The portable local snapshot contains 59 source files, including 55 Markdown documents and four SVGs, under `explore-source/`. Its canonical base is `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`; its freeze SHA-256 is `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Per-file hashes preserve the exact local bytes. Current guides use `cq`, `cq-sdk`, Python `from cq import` and TypeScript `@maldous/cq-sdk` directly. Dated engineering records preserve revision-specific paths and tool versions and point readers to the current SDK guides.

All 611 reader links resolve. Normal builds and checks use frozen sources; `--refresh-local-source` deliberately captures a new local snapshot. Documentation generation requires the packages pinned in `requirements-explore.txt`.

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

Login presents thirty identity choices with search and local feedback. Provider handoff, account creation and recovery require a real authentication backend. No credentials or email are persisted by the preview.

Eighteen browser-service integration points are documented in `SERVICES.md`; adapters remain disabled until configuration and consent are qualified. Legal/CMP content, production domains, external accounts, real authentication, device and assistive-technology qualification remain deployment requirements. The site is noindex and has not been deployed.

`CANDIDATE-REVIEW.md` and `review.json` are engineering review/provenance records, separate from the product documentation. They record the design assessment, bounded checks and implementation limitations.
