# CircularQuay landing-motion proof of concept

This is a local-only, synthetic landing-page experiment. It tests whether a
short, legible motion attractor can introduce CircularQuay before a visitor
reaches the existing workbench or account surfaces. It is not a replacement for
the owned `web/` preview, a production site, or a claim that a backend exists.

The starting reference is the [CircularQuay motion study](https://circularquay-motion-study.matthewaldous.chatgpt.site/).
The concept to retain is a visually playful explanation of a key travelling from
intent to an approved application. The concept to change is its overloaded
iconography: the POC uses one readable manual-to-coordinated sequence, rather
than multiple competing symbols, so the visitor can understand the story before
interacting with it.

This direct user-requested proof of concept is an exemption from normal issue
intake. It adds no Linear issue and makes no product, CIX, infrastructure or
production change. Existing UIF-170 work is treated as read-only context. Any
future integration belongs to the current CircularQuay frontend/source owner.

## Review locally

Use Node.js 22 or later; there are no dependencies to install.

```sh
node examples/landing-motion/server.mjs
```

Review at [http://127.0.0.1:5182/](http://127.0.0.1:5182/). The server binds
only to loopback, accepts GET and HEAD, and only serves the declared POC assets.
It does not serve this README, briefs, design material, evidence or repository
files. Stop it with `Ctrl+C`. Syntax-only checks, with no dependencies, are:

```sh
node --check examples/landing-motion/server.mjs
node --check examples/landing-motion/motion.js
node --check examples/landing-motion/preference-bootstrap.js
```

## Device and interaction plan

- **Desktop and large tablet:** use the animation as the hero’s spatial
  attractor, with a stable text column and an explicit replay affordance.
- **Tablet and folded devices:** reduce the stage to a single focal lane when
  the available width is 900 CSS pixels or less; preserve the explanation in text.
  Short viewports retain ordinary scrolling rather than shrinking the diagram.
- **Unfolded foldables:** progressively enhance into the wider composition only
  when the effective viewport is actually available. A hinge must never contain
  essential copy or controls; CSS viewport segments can refine placement when
  supported, while the one-column composition remains the baseline.
- **Phones:** show the concise story and direct controls first. Do not require
  precise dragging, hover or landscape orientation to understand it.

The automatic story runs once for fourteen seconds of active playback, then
settles on an intelligible final frame. Visitors can pause, replay, or choose a chapter.
When the browser requests reduced motion, the page begins in its static settled
state. Animation also pauses when hidden or off-screen. The story only depicts
synthetic, clearly labelled approved and denied paths; it never represents a
real key, credential, provider or authorization decision.

## Visual and integration boundaries

The POC uses the established CircularQuay theme tokens and font stack from the
existing browser preview: Inter, `ui-sans-serif` and system fallbacks; light
`#f7f8fa` / `#192834` with accent `#126b5b` and feature `#102f36`; dark
`#111b24` / `#e5eced` with accent `#7dd0b5` and feature `#0e3035`. Use only the
exact icon-13-derived SVG assets and their retained provenance; it is not
permission to create or claim a new final logo. The public product name is
**CircularQuay**; **CQ** is an internal compact label and `cq` is reserved for
owned technical names.

If the experiment is adopted, the frontend owner can port the isolated motion
module or a Web Component into `web/` after a source-owner handoff. That later
work needs its own review against the existing UI, privacy/server boundary,
accessibility, performance budget and actual desktop/tablet/foldable/phone
device evidence. This POC does not establish those qualifications.

## Realisation and later integration

1. **Retain the story, simplify the stage — implemented here.** Three sources
   and three workload examples replace the original moving provider cloud.
   Manual handoffs give way to a single local-policy route. The denied chapter
   removes every outgoing particle and approval mark. All content is synthetic.
2. **Make the animation optional — implemented here.** The full route and
   explanation are in HTML before scripts run. Autoplay yields to navigation,
   scrolling, explicit chapter selection and reduced motion. No signup, provider
   connection, preload sequence or scroll lock sits in front of the page.
3. **Adapt the composition — implemented here.** Above 900 CSS pixels the route
   is horizontal; at narrower widths the diagram has independently authored
   vertical geometry. Foldable segment reporting also selects the compact route.
   Ordinary scrolling is preserved at short heights. Device models are not
   inferred from a user agent.
4. **Review the attractor with Matt — next.** Use Replay and the three chapters
   to judge the story, the entertainment value and how quickly the local-policy
   boundary becomes clear. Compare light/dark and folded/unfolded viewport sizes.
   The experiment is removable without changing the workbench.
5. **Port the chosen treatment through the existing frontend owner — later.**
   Move the diagram and timeline into the owned landing route using the actual
   presentation/preferences modules, existing language controls and shared tokens.
   Keep those modules authoritative; the small standalone theme bootstrap here
   is for this independent review origin. Preserve current drafts, CSP, help,
   client/server separation and ownership. No new backend is needed for motion.
6. **Qualify the integrated result — later.** Exercise real touch devices,
   Safari/Firefox, RTL, forced colours, reduced motion, slow CPUs and real
   foldable posture/hinges; profile frame work and activation/network cost.
   This local Chromium review does not establish that broader acceptance.

The inherited visual authority remains `../../web/DESIGN.md` and the actual
tokens/font stack in `../../web/style.css`. No existing product/design file was
rewritten. The selected light/dark SVGs are exact copies of `web/logo-*.svg`.

## Review evidence

The retained evidence lives in the repository's existing
`.local/qualification/checkpoint57/landing-motion-review/` store. It is not a new
controller or work tracker. Full-page Chromium captures cover desktop 1440,
tablet 820, unfolded-width 673, folded-width 360, mobile 390 and narrow-phone 320
CSS pixels, with no document-width overflow in the captured layouts. Foldable
names describe viewport emulation; no physical hinge was tested.

The independent source/visual review returned **ship** for this POC. Bounded
verification includes JavaScript syntax, eleven HTTP asset/privacy/method checks,
exact logo-byte comparison, light/dark, a static reduced-motion route, pause,
keyboard chapter selection and JavaScript-disabled content. There is no font,
library, video or third-party asset request. These are local review checks,
not a measured frame-rate or production performance claim.

The design hook's small-text finding was fixed by increasing interface copy to
at least 12px and the story caption to 14px. Nothing was suppressed. Existing
UI font choice follows the user's explicit consistency instruction.
