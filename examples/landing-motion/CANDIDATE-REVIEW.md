# CircularQuay engineering review — 5 November 2026

Engineering provenance record; product guidance is in README.md and Explore.

The focused eight-node landing replaces the superseded full-flock experiment. Under the heading “Get the right secrets ready.”, it now links directly to the first featured Explore document while Explore home remains the grouped overview. The current delta keeps continuous diagram/story animation and removes only the horizontal divider between them. Desktop, tablet and foldable layouts include the same Explore orientation used on the documentation home. Login presents the thirty source-inventory identity choices with explicit local-preview feedback.

## Review and polish

Two independent Impeccable assessments were used: visual/design Assessment A and detector/source/browser-evidence Assessment B. A reviewed final light/dark phone, foldable and desktop captures before B findings were synthesized. The product's supplied branding was preserved.

| Surface | Mode | Design assessment | Disposition |
| --- | --- | --- | --- |
| Landing | Persuade | 28/32; heuristics 7 and 10 not applicable | Approved for local review |
| Login | Operate | 34/40 | Approved for frontend preview |
| Explore | Read | 34/36; heuristic 7 not applicable | Approved for local reading |

The current presentation delta uses 2px overlays and routes, a borderless hub base, 18px text padding, and a mobile icon/name/purpose face grid with an 18px minimum row gutter. The motto stays single-line. Headers use common stable gutters and padding, with language before theme through 760px. Help is an outlined text button without an icon. The full-viewport loader centres the logo with a neutral-grey breath and a clockwise arc clipped to the C-shaped logo band, fades it out as the page fades in, and remains still for reduced motion. Escape closes a search dropdown without pausing that flow; the diagram keeps its own Escape pause behavior.

## Motion and layout evidence

Each workload includes an optional feature with probability 0.25. An absent optional step is absent from the sentence, route, completion highlight and consecutive numbering together. Cases have six to eight numbered clauses and twelve to sixteen 1200ms legs. Each starts at CircularQuay, visits an IDP and returns, visits an encrypted source and returns, then reaches a target and returns; selected optionals make their own request/response round trip. Reduced motion uses a complete basic static example.

Earlier story-reservation and animation evidence is historical for the superseded presentation. It does not assess the current landing presentation delta.

Prior viewport evidence remains historical. The parent browser/detector review for the current delta is in progress; this document makes no new Assessment A approval claim until that review completes.

## Explore source and naming

Explore now uses the current local documentation captured on 5 November, rather than an older published GitHub revision. The portable source freeze contains 59 source files—55 Markdown documents and four referenced SVGs—rendered as 85 reader parts. Exact raw bytes and SHA-256 values are committed under explore-source; normal builds/checks use that freeze. Help distribution and autocomplete are implemented in the Explore surface. An explicit --refresh-local-source is required to replace it.

The README displays CQ and the cq command. The final source freeze is based on canonical commit `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`, with manifest SHA-256 `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Current guides use `cq-sdk`, Python `from cq import` and TypeScript `@maldous/cq-sdk` directly. Genuine historical scan, loader and security receipts retain their revision-specific paths and versions; the reader identifies these engineering records and links to current SDK guidance. Repository-context links explicitly disclose that their contents may differ from this local snapshot. All 611 reader links resolve; no inert anchors remain.

## Verification and remaining limits

The final bounded checks pass: seven flow tests, five service tests, JavaScript syntax, reproducible documentation generation, native static-page/catalog checks, source/public-file hashes, and all 211 public HTTP paths byte-identical to local files. Eight private/source probes return 404. The self-only CSP and noindex profile remain. Login search/provider feedback and preview account forms were exercised without persisting an email or starting authentication.

The final markup detector returns no findings for all three pages. An earlier landing scan reported three compact 11px text warnings; the live diagram still deliberately uses compact noninteractive labels as a documented readability tradeoff. Narrow persistent exceptions preserve user-pinned Inter, warm ivory, and the editorial row's child-link padding. No rule or file is globally ignored. An external detector overlay was not injected through the self-only CSP; no overlay success is claimed.

This is a local frontend candidate. Real authentication/provisioning, legal content, production domains, configured service accounts/IDs, consent qualification and vendor adapters remain pending. Eighteen discovered service hooks are recorded and disabled/fail-closed. Device, assistive-technology and physical animation-performance qualification remain outstanding; browser captures are emulation evidence.

## Custody

Direct user authority permits this landing exemption and its scoped current-branch commit without a Linear issue. No push, deployment, Linear mutation, BAU/CIX pickup or paid service action follows. The landing owner changes only examples/landing-motion and scoped Impeccable records. Native/root rename and documentation cleanup are separately owned. The initial reviewed checkpoint is `67dbd38`; a separate scoped consumer commit records the canonical-source refresh and initial-story correction, without native/root source changes.

Private reproducible review evidence remains in the existing checkpoint57/landing-motion-review/candidate-after store. Earlier full-flock critiques and snapshots remain historical and superseded.

Questions skipped: the user already authorized these refinements and requested a concrete local review candidate with a commit.

## Historical initial-story correction

The former initial-story correction, its animation traces and its Assessment A approval apply only to the superseded animated presentation. They do not approve or qualify the current landing presentation delta.

Screenshot 1000015445.png could not be materialized: the coordinator reported the supported Library helper failed with “download failed,” with no executor-local readable pixels. The correction used the supplied text and actual browser verification. This intake limit is separate from the successful behavior evidence.

The builder reads frozen metadata for normal build/check, so advancing Git HEAD cannot invalidate a snapshot. Only explicit source refresh reads current HEAD/branch. Unreferenced generated assets are pruned; four remain. Current consumer README/brief/implementation guidance is factual, without personal authorization or assistant-workflow narration; dated review history and custody evidence remain in engineering provenance.
