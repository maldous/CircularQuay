# CircularQuay browser development

Synthetic frontend development only. The local client is an end-user environment
workbench; account and operator consoles are separate presentation planes. No
backend, account connection, real credentials, crypto service or payment service
is connected. All financial/key/fleet/identity actions are explicitly simulated.

Use Node.js 22 or later. There are no package dependencies and no install step.
From the repository root:

```sh
cd web
npm run dev
```

One small Node process listens on two **loopback-only** ports:

| Surface | Development URL |
| --- | --- |
| Local end-user workbench | http://127.0.0.1:5181/#/workbench |
| Brief demo entry | http://127.0.0.1:5181/#/landing |
| Complete typed configuration | http://127.0.0.1:5181/#/configuration |
| Customer account | http://127.0.0.1:5180/#/account/overview |
| Operator console | http://127.0.0.1:5180/#/operator/overview |

Phone previews can use authenticated Termux SSH forwarding to the loopback ports.
Forward phone `127.0.0.1:5181` to Foo `127.0.0.1:5181`, and phone
`127.0.0.1:5180` to Foo `127.0.0.1:5180`, using the existing Foo SSH target.
Phone reachability is operator-reported; no automated physical-device test is
recorded. Browser storage is device/browser/origin scoped; desktop drafts do not automatically
transfer to the phone. Loopback forwarding requires no public listener or service installation.
`Ctrl+C` stops the process. Coordinate port use on shared development hosts.

Editing source JavaScript, CSS or bundled JSON reloads connected previews. An
unsaved draft suppresses automatic reload and asks the user to save/discard it.
Artifacts under `output/` do not trigger reload. There is no bundler/build step.

```sh
npm run check
npm test
```

The syntax check covers all frontend JavaScript. Focused Node tests cover
HTTP privacy boundaries, JSON fidelity/duplicate handling, schema branch/type/
reference rejection, policy/editor synchronization, numeric draft reindexing,
conditional controls, seed validity and control inventory reachability. These do not execute Go,
native workloads, real providers or the repository's production qualification.
The hosted browser workflow runs these checks across pinned Node 22/24/26 on
Linux, macOS and Windows. A workflow definition is not a passing result.

## Local workbench

Add/import synthetic `.env` variables, choose exact source references and key
profiles, edit masked values, set individual IANA-zone windows, inspect inherited
effective values/locks and resolve an explicitly labeled simulated group conflict.
The preview-time clock checks required windows before mock apply/run. Failed
candidates preserve the preceding complete generation. No program is launched.
`local`/`remote`/`hybrid` and custody controls are local design models, not new
native flags. Required independent/HSM custody denies the software model.

Workspace save/reload and JSON export/restore preserve windows, zones, profiles
and provenance. Plain `.env` exports cannot represent those policies and contain
literal values only. Export requires an explicit local acknowledgment. Native
configuration is preserved separately during workbench backup restoration.

## Complete configuration

Load the bundled canonical Linux example or explicitly select/paste local JSON.
Every reachable schema path has a typed control: grouped objects/maps, explicit
optional fields, enum/boolean selectors, references, numeric/path/text values,
lists with order controls and arbitrary non-null values with explicit types.
Progressive disclosure and path search expose advanced nested controls.

The untouched imported source and proposed document are distinct. JSON/guided
saves update a matching typed editor. A separate saved editor draft is retained
with a notice; “Load saved policy” explicitly replaces it. Typed saves refresh
the guided source/workload views. Discard persists the restored source across
navigation and reload. Invalid numeric drafts follow moved list items, shift
after removals and clear when their parent value is explicitly replaced.
Rendering never
adds defaults. Duplicate keys/malformed JSON reject before replacement. Unknown
fields, explicit nulls, false and empty collections survive; unsupported content
is reported and can be retained as an **invalid draft** without replacing the
active policy. Invalid numeric text blocks save/export until corrected/discarded.

Bundled schema and Linux fixture are pinned copies from the implementation
checkout. Hashes and 187-path control coverage are in `coverage.md` and the retained
`output/playwright/` evidence. Browser validation implements the keywords used by
that snapshot and bounded local reference checks. Full native permissions,
signature/include resolution, credential lifecycle and provider checks remain
separate; no browser check is a native validation or runtime qualification claim.

## Privacy and synthetic presentation

All configuration, local profiles, source references, images and workspace data
stay in browser storage. Static GETs load the bundled schema/example; user files
are read only after explicit file selection. There is no configuration upload,
hosted fetch proxy, analytics or configuration-bearing mock transport. The server
rejects non-GET/HEAD methods and query strings, serves explicit asset roots and
cannot expose local-client assets on the account/operator listener. CSP restricts
network access to that origin. Browser storage is a demo store, not a secret vault;
use synthetic data only.

Account/operator state uses a different origin/storage key. The remote role
selector changes synthetic presentation/capabilities only. Real integrations must
verify credentials and server-issued tenant/resource roles independently. UI role
selection is not authentication or a grant. The software-custody UI states the
shared runtime/KEK exposure risk; key/private values cannot be displayed here.
Revocation cannot recall copied private keys or plaintext.

Profile, theme, image and IdP controls remain local demo settings. No actual
issuer is contacted and no avatar/brand image is uploaded. The selected icon 13
contact sheet is preserved unmodified at `references/selected-icon-contact-sheet.png`,
with source details, size, SHA-256 and visual selection in
`references/selected-icon-provenance.json`. Number 13 is row 3, column 1.
The reference remains unchanged. `logo.png` is the exact isolated crop;
`logo-light.svg` and `logo-dark.svg` are transparent native-path reconstructions
of the selected C/key geometry. They share geometry with navy/off-white fills,
follow the application theme, and appear in all wordmarks including the local
landing page. They are derived vectors, not the original vector source or a
pixel-identical reproduction of the raster texture.
Dock/Guided were structural preferences, not a claimed approved final
layout. The pre-revision source is retained at
`/tmp/cq-frontend-before-workbench/web`.

All 131 editable scalar controls were individually changed with exact whole-document
comparisons on the deliberately invalid all-fields draft; six true constants
remain single-choice controls. This proves editing fidelity, not every valid
provider combination or native execution. The canonical Linux fixture also
passed a fresh actual download/re-import/download semantic round trip.

The retained coverage matrix distinguishes actual browser actions, rendered
control reachability and remaining native/provider/public-preview gaps. Publication
and integration remain coordinated with the source owner; this frontend never
modifies backend, production or BAU state.

Frontend integration must preserve this checkout/branch, pinned schema/example,
loopback ports, browser-local drafts, Node 22+ dependency-free toolchain and
synthetic-only adapters. Integration/publication stays with the implementation
owner; existing evidence is under `output/playwright/`. No new lease, harness
assignment, service exposure or production authority follows from the frontend.

The configuration editor groups destinations and native option menus, keeps
secondary actions under More actions, and offers Undo removal until the next
edit/save/document replacement. Undo restores the whole previous proposal and
invalid numeric inputs. Browser validation remains separate from native
qualification. `PRODUCT.md` and `DESIGN.md` record the authorized browser scope.

## Client-local help

Use **Help & search**, F1, or `/help.html` on either preview port. The launcher loads with the app; the viewer, public catalog and selected article load only when opened. The full-text index loads on the first nonblank search, with one shared request across concurrent searches. Ranking and subsequent queries run locally. Queries never enter request URLs, bodies, browser storage or diagnostic events. Browsing starts with 16 guides; full-text search also covers 317 schema and collection-field articles.

Run `npm run build:help` after an approved authoring-content update, then restart the existing preview to reload its exact asset allowlist. The build validates the closed v1 authoring schema, publishes only product instructions/references, and produces content-addressed articles, catalog, postings and precompressed Brotli alternatives. Hash-addressed assets have immutable cache headers; the small manifest is fetched without caching on a fresh viewer store. Its exact byte count, SHA-256 and expected content/source version are pinned in the reviewed executable module `help-release.js`; the bounded reader verifies raw bytes before JSON parsing. A self-consistent replacement manifest and corpus are rejected before catalog use. Review any regenerated pin together with its source and derived assets. This bundle pin does not protect against executable-bundle or host compromise. Browsers without Brotli receive the identical uncompressed content. Runtime checks reject mismatched versions, hashes, unsupported shapes and oversized streams. Article memory caching is bounded to 32 articles or 2 MiB.

`npm test` selects current root test files explicitly so retained historical evidence under `output/` is not counted as new test coverage. The English corpus is an authored draft; article presence/searchability is not acceptance of every native semantic branch. Initial language scope is English plus one Arabic locale. Arabic translation and full interface RTL acceptance are separate pending work, not implied by Arabic search normalization.

The English public source JSON is compactly serialized to remain below the existing 1 MiB first-party source-identity limit. Its parsed 333-article/header/reference semantics are identical to the independently accepted readable authoring artifact; the distribution version and bundle pin use the actual compact file hash. The build refuses larger source files.

## Language and theme

Language and theme controls work before entering either demo and on task and standalone help pages. English (`en`) is the fallback; Arabic (`ar`) is explicitly a preview for RTL testing while product pages and the current help corpus remain English. The preferences module loads independently of the application module. Changing a preference updates the document and preference controls in place, preserving open form drafts.

The single current storage record is `cq.presentation.v1`: `{version:1, language?:'system'|'en'|'ar', theme?:'system'|'light'|'dark'}`. Only explicit choices are saved. Automatic clears the field; Browser only and Device system explicitly follow this device. An optional verified profile value would apply between device choice and browser defaults, but no real profile service is connected. Workspace preferences are never read or converted. Blocked browser storage keeps the choice for this page and displays a warning. No geolocation service, language request or telemetry is involved.

Browser checks cover draft preservation, reload persistence, narrow RTL layouts, keyboard controls, forced-colors emulation and component contrast. They do not establish full Arabic translation, genuine identity transitions, physical-phone or assistive-technology acceptance.

## Measured public-asset delivery

The preview negotiates Brotli for allowlisted JavaScript/CSS of at least 1 KiB, with byte-identical identity fallback. Every request still passes the trusted-file read. Encodings are keyed by current content hash, with at most 32 retained entries / 2 MiB and four concurrent encodes; concurrent requests for the same bytes share work. Immutable help payloads retain their separately built/pinned Brotli representations. No configuration or search input enters compression keys or uploaded diagnostics.

Local Lighthouse reports, public-timing probes and rendered-route observations are under `output/playwright/performance/`. They identify source/tool/browser/emulation and limits; no homepage score, mobile viewport, automated accessibility score or local Event Timing sample establishes complete action coverage, physical-device/OS/AT acceptance or field INP. The current whole-interface Arabic translation and full platform/feedback acceptance remain pending.
