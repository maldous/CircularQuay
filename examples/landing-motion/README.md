# CircularQuay landing motion proof of concept

This is a local production-candidate frontend review on the current branch. It is a direct user-authorized exemption: no Linear issue, BAU/CIX resumption, push or deployment is authorized. Final checks and review remain pending; it is not production-qualified or deployed.

## One-pane flow

The landing runs one active case at a time from 24 actor contexts. Each case uses two connected human goals joined by **and**, two workloads with distinct identities, both encrypted local and remote vaults, and two different application targets. Each workload independently has a 25% optional-feature chance. When optional content is absent, its flow, copy and number are absent together. Whole-case icons remain stable. Before the next case begins, its icon and story atomically fade in.

Numbered markers replace dots. Each case has 6–8 consecutive clause numbers and 12–16 legs. The matching number appears before its clause and completes the same numbered round trip for both workloads: CircularQuay → Authentication → CircularQuay → encrypted vault → CircularQuay → application → CircularQuay. A shared smoothstep transition advances each 1200ms leg and its text wipe. Both delivered paths pulse together for 2200ms, then the bold punchline appears.

Both vault labels read **Encrypted**. Vendor icons are larger and centred; in dark theme nodes use dark green with an off-white CircularQuay mark and small neutral vendor plates for legibility. The desktop headline is upper-positioned, with the topic mini-home below.

## Explore and navigation

Explore opens `/explore`, the canonical home pane for 55 raw public Markdown documents and four SVGs frozen under `explore-source/`. The portable local snapshot is based on the dirty local working tree at `b0abadd`; its manifest SHA-256 is `88a4d936a47bc09e2f909121c7265c8a9e6efd7a7b6e7f96b65267e7227b2377`. It renders 85 parts. The default builder and `--check` consume this frozen snapshot; only explicit `--refresh-local-source` recaptures it.

The editorial output applies the `cquay-sdk` → `cq-sdk` overlay in five files and 21 replacements. Raw source remains untouched and hash-recorded. Link count B is under current checking, so no earlier published link qualification is asserted here.

Explore home groups all 55 documents into six task groups with breadcrumbs and group rails. It has no native-document or Chapter dropdown. “In this guide” contains only genuine multipart links. The same compact six-group menu is shared by the landing on desktop, tablet and foldable layouts.

The shared footer has the planned safe-area treatment. Geometry has fit nine layouts; the newest 360px audit preserves primary curves and 12px clearance around optionals. The motion worker now reserves story space with 96 bounded rendered typographic envelopes plus one measured line per width/font/locale, yielding fixed story height. Parent CUA observed 106 samples across two stories at a stable 177px reservation and 467.8125px graph, without overflow. Final A/B review remains pending. The local loopback preview is served by the parent-owned process PID 1573071 using the nominal Node command below.

## Review locally

```sh
node examples/landing-motion/server.mjs
node examples/landing-motion/render-integrations.mjs --check
node examples/landing-motion/render-site-pages.mjs --check
node --test examples/landing-motion/services.test.mjs
node --test examples/landing-motion/scenario-flow.test.mjs
```

Node.js 22+ needs no dependency installation. Source/unit/static-layout evidence may be reviewed with these commands, but parent final checks and browser/CUA review are not yet complete. The root reviewer will write `CANDIDATE-REVIEW.md`.

The 113-entry/108-SVG inventory is source-only validation data for the 30-provider local login wall. The 18 browser-service hooks remain fail-closed: no vendor SDK, request, identifier, DSN, account or consent record is active. Real adapters, accounts, IDs, legal/CMP content, authentication, domains, device and assistive-technology qualification remain production blockers.
