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

Matt confirmed phone access through his existing authenticated Termux SSH tunnel.
Forward phone `127.0.0.1:5181` to Foo `127.0.0.1:5181`, and phone
`127.0.0.1:5180` to Foo `127.0.0.1:5180`, using the existing Foo SSH target.
This is user-confirmed access, not an automated physical-phone test. Browser
storage is device/browser/origin scoped; desktop drafts do not automatically
transfer to the phone. No firewall, public tunnel, production hosting or service
installation was created. `Ctrl+C` stops the process. Ports and ownership were
agreed with the implementation owner.

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
`/tmp/cquay-frontend-before-workbench/web`.

All 131 editable scalar controls were individually changed with exact whole-document
comparisons on the deliberately invalid all-fields draft; six true constants
remain single-choice controls. This proves editing fidelity, not every valid
provider combination or native execution. The canonical Linux fixture also
passed a fresh actual download/re-import/download semantic round trip.

The retained coverage matrix distinguishes actual browser actions, rendered
control reachability and remaining native/provider/public-preview gaps. Publication
and integration remain coordinated with the source owner; this frontend never
modifies backend, production or BAU state.

Owner-led UIF handover must preserve this checkout/branch, pinned schema/example,
loopback ports, browser-local drafts, Node 22+ dependency-free toolchain and
synthetic-only adapters. Integration/publication stays with the implementation
owner; existing evidence is under `output/playwright/`. No new lease, harness
assignment, service exposure or production authority follows from the frontend.

The configuration editor groups destinations and native option menus, keeps
secondary actions under More actions, and offers Undo removal until the next
edit/save/document replacement. Undo restores the whole previous proposal and
invalid numeric inputs. Browser validation remains separate from native
qualification. `PRODUCT.md` and `DESIGN.md` record the authorized browser scope.
