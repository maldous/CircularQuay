> Preserved, superseded experiment. The human rejected the full-flock hero for excessive detail, especially on mobile. This snapshot retains the complete work and critiques for reuse. The next revision restores a focused professional hero and moves the full icon inventory into a static secondary ecosystem section.

# CircularQuay landing motion proof of concept

A local-only exploration of the original [motion study](https://circularquay-motion-study.matthewaldous.chatgpt.site/), preserving its full integration breadth and the existing CircularQuay identity.

The corrected attractor retains **all 113 entries**: 30 authentication, 35 encryption/source and 48 application examples, represented by 108 distinct local SVG assets. It follows the user's selected treatment: a flocking field that settles into groups as the journey becomes organized. Icons stay visual; named examples rotate automatically during active playback. Every entry remains in static HTML on desktop, tablet, folded/unfolded and mobile layouts.

The earlier three-generic-source/three-app POC at `d894597` is retained in Git history, but was **not accepted against the required breadth**. Its static-polish verdict did not establish compliance with that requirement. The correction's evidence chain is:

1. [Source-bound critique](CRITIQUE.md): independent A design and B detector/browser assessments of that exact baseline.
2. [Justified correction plan](CORRECTION-PLAN.md): issue → change → acceptance evidence.
3. [Visual critique before polish](POLISH-CRITIQUE.md): the 23/32 assessment that rejected the broad but generic dark-card composition.
4. [Surface brief](BRIEF.md): confirmed user choices, motion, responsive and custody contracts.
5. `review.json`: exact runtime/asset hashes and scoped before/after verification.

## Review locally

Use Node.js 22 or later; there are no dependencies to install:

```sh
node examples/landing-motion/server.mjs
```

Open [http://127.0.0.1:5182/](http://127.0.0.1:5182/). The one small server binds only to loopback and serves an exact allowlist of HTML/CSS/JS/logo/integration SVGs. It does not serve briefs, catalogs, source provenance or private evidence. Stop it with `Ctrl+C`. Existing workbench/account previews on 5180/5181 remain owned and unchanged.

The catalog is the authoritative inventory for this isolated example. Regenerate/check its static HTML with:

```sh
node examples/landing-motion/render-integrations.mjs
node examples/landing-motion/render-integrations.mjs --check
node --check examples/landing-motion/motion.js
node --check examples/landing-motion/preference-bootstrap.js
node --check examples/landing-motion/server.mjs
```

## Interaction and device behavior

The 14-second active story begins with the complete mixed flock, organizes it into the three complete fields, demonstrates denial, and settles on the coordinated route. Each flock can roam the entire animation area; final role groups are destinations, not movement lanes. Cached geometry, global separation and protected readable content keep the field usable. Direct eased organization avoids spring overshoot under dropped frames. Pause, replay and chapters are native controls placed before the field. Playback suspends while hidden/offscreen. Name changes do not announce continuously to assistive technology.

Reduced motion shows the complete grouped field, without flocking or automatic name rotation. With JavaScript disabled, all 113 icons, names in image alternatives, explanation and workbench links remain available. No interaction with an individual icon is required.

The human-selected fourth reference panel drives the pale composition: copy at left, real integration marks arranged around the local-policy pane at right. Desktop (1440×900 and 1280×800) and tablet (820×1180) fit one viewport. Narrow/folded/mobile layouts recompose the **same 113 entries** into a compact field. Where the readable inventory needs more height, the page uses ordinary vertical scrolling with gentle proximity snapping at the intro/diagram boundary. Reduced motion disables smooth scrolling and snapping. There is no nested scrolling or mandatory full-screen snap. Viewport-segment handling is progressive enhancement, not physical-foldable qualification.

## Branding, assets and truth

The exact name/slogan, selected logo13-derived light/dark SVGs, existing Inter/platform font stack, and product theme tokens remain in use. The visual authority stays in `../../web/DESIGN.md` and `../../web/style.css`; no existing product/design file is rewritten.

Integration vectors and verified brand colours are hosted locally, with per-entry upstream URL, licence and digest in `integrations.json`. Simple Icons colour transforms retain both upstream and final SHA-256 values. Historical Microsoft/IBM vectors retain their source context. The local-filesystem symbol is explicitly non-brand; shared vendor marks retain separate role names. See [asset provenance](assets/integrations/PROVENANCE.md) and [licence notices](assets/integrations/LICENSES.md).

**The scene is an illustrative ecosystem, not a supported-adapter catalogue.** Logos do not prove a qualified, configured, connected or authorized integration. No actual credentials, source retrieval, workload, enrollment or backend is connected. Configuration stays client-local.

## Evidence and handoff

Use the existing `.local/qualification/checkpoint57/landing-motion-review/` store. `panel-four-after/` holds the corrected composition. Earlier motion traces are retained as historical failing evidence and are excluded from current acceptance; the final trace is explicitly source-bound in `review.json`. Before captures are source-bound to `d894597`; the intermediate breadth-only view was independently rejected for visual craft; corrected captures and runtime/asset hashes are recorded separately. The supplied desktop/tablet/foldable/mobile evidence is Chromium viewport emulation. Actual hardware/hinges, Safari/Firefox, screen readers, RTL, forced colours and device performance need their own acceptance.

The scoped Inter detector exceptions in `.impeccable/config.json` preserve the user's explicit existing-font instruction; they apply only to this example's HTML/CSS. They do not disable whole rules or whole files. A clean detector does not establish design or product acceptance.

This directly authorized landing exemption creates no Linear issue, resumes no BAU/CIX work and publishes nothing. All corrections are committed on the current feature branch for later use. Any adoption into `web/` belongs to the existing frontend/source owner: reuse the actual presentation/language/theme modules, preserve drafts, CSP/help and privacy boundaries, and qualify the integrated result separately.
