# Frontend control and interaction coverage

Snapshot: schema SHA-256 `1d34bfc0d3c93e6078390fe4318e957e49178aac580cf8aaf11a70b25d50dffd`; canonical Linux example SHA-256 `1d3b73c654f5371729e6005434c5cd15ca1c87d65cfecf1603ba6991c6429276`. Both source-owner files were rechecked unchanged before handoff.

All 187 reachable schema paths were found as actual typed controls/groups in a real browser across nine sections. A later action pass changed all 131 editable scalar controls and compared the complete proposed document after every edit; six schema constants remained single-choice controls. `typed-field-mutations.json` records exact pointers. Boolean telemetry controls now retain both alternatives while full-document validation enforces the selected branch. The root is represented by section navigation. A deliberately invalid synthetic all-fields draft supplies mutually incompatible provider branches for reachability and editing fidelity only: its 50 validation findings remained visible, and its export exactly matched its input. It is not a runnable or qualified policy.

The canonical valid Linux fixture passed browser schema/reference checks and a no-op semantic round trip. Seven representative edits across source, systemd target, telemetry, MCP, grant, repository and update controls matched the exact expected exported object; untouched siblings were equal. Export/re-import/export equality passed. Native protected-file, signature, include graph, lifecycle, service authority and provider checks were not run. The all-fields action pass covers scalar editing fidelity independently of the seven representative valid Linux edits. It does not establish every valid provider combination, graph/trust behavior or native execution. A fresh canonical Linux actual download/re-import/download semantic round trip also passed after the corrections.

## Interaction matrix

| Requirement | Route/control | Actual evidence | Remaining limit |
| --- | --- | --- | --- |
| End-user demo entry | 5181 / #/landing → #/workbench | No login; end-user-only grid/inspector; old overview/variables URLs route to workbench | Original icon 13 raster reference preserved; no finished vector asset |
| .env key/value editing | Workbench Import .env / Add variable / inspector | Two variables imported; masked edit; invalid name (prior editor), duplicate import atomic rejection; no duplicate rows on restore | Simple literal .env parser; no shell evaluation |
| Independent windows and zones | Inspector Availability / preview UTC clock | Melbourne 08:00–18:00, Los Angeles independent window; UTC previews; invalid end-before-start denied; valid timezone-aware mock run generation 2 | No live issuer/workload authority; mock only |
| Providers and key profiles | Inspector Value & source | OpenBao vs AWS Secrets Manager and different KMS/age profile IDs retained after reload/backup restore | Synthetic profile choices; no provider connections or actual keys |
| Inheritance, effective provenance and locks | Inspector Inheritance / Show conflicts | Group conflict denied whole candidate; approved simulated revision resolved it; organization-locked row read-only | Not a signed hierarchy runtime; native checks separate |
| Local persistence and export | Save workspace / Export / Import workbench JSON | Workspace export/re-import/export equality; six rows, three zones and profiles retained | Browser demo store; not protected native storage |
| Complete supported configuration | Complete configuration, nine section groups | 187/187 control paths; 131 scalar mutations with complete-document equality; six constants retained | Valid provider combinations and structural cases are not exhaustively enumerated; see scoped evidence |
| Source / proposal fidelity | Complete configuration import/save/discard | Original no-op equality; seven exact edited paths; malformed/duplicate JSON rejection; unknown false/empty/null/list preservation; invalid numeric text blocks save/export | Unknown content retained as invalid draft, not accepted active policy |
| Discarded proposal persistence | Complete configuration Save locally / Discard proposal | Reproduced saved 9.9.9 proposal returning after discard/navigation; fixed persisted state. Original 1.0.1 remains across navigation and reload. Invalid draft preserves unknown false/empty/null/list values; accepted policy unchanged | Browser-only synthetic regression; no native activation |
| Optional sections, maps and lists | Composition / workload bindings / named environment entry | List reorder round trip, append/remove, map duplicate rejection and sibling preservation; all six explicit value types; exact discard. Invalid numeric input follows reorder/removal and parent replacement clears obsolete errors | No actual fragment reads/signature validation |
| Cross-view policy consistency | JSON, guided and complete configuration | Matching typed editor synchronizes after JSON/guided saves; separate invalid draft retained with notice and explicit Load saved policy; typed saves refresh guided sources. Raw duplicate JSON rejected before replacement; JSON source creates no unsupported identity field | Client-only state, no native activation |
| Mock apply/run and failure handling | Workbench Validate / Adapter scenario / Mock apply-run | Conflict denial; complete generation; access-denied failure preserved prior generation; timezone-aware successful run | No process launch; no real snapshot/key/audit release |
| Customer profiles / usage / billing | 5180 #/account/* | HSM denial; dual association; committed age +1; suspended profile denied +0; two retries preserve history; financial cancel/confirm/repeated portal with original plan restored | No billing provider, actual tariff, account or crypto backend |
| Operator grants / RBAC | 5180 #/operator/* / demo role selector | Auditor Create absent; invalid window and wrong tenant denied; global presentation created synthetic bounded grant | Presentation only; no authentication or real server authority |
| Account/operator route coverage | Account eight child routes; operator eleven child routes | Every route navigated; tables/details/forms present; client assets unavailable to 5180 | Phone localhost access user-confirmed through existing Termux SSH tunnel |
| Service telemetry | Operator Service health Start/Pause/Reset | Three-second updates observed; pause preserved table across later checks; reset restored 42/18/29 ms seeds | Synthetic metrics; no durable immutable audit claim |
| Selected devices | Local Devices / Inspect selected / Simulate selected refresh | Empty selection denied; explicit online + offline selection returns separate ready/unavailable-no-dispatch results | Synthetic results; no agent endpoint |
| Migration interruption/resume | Local Migration / confirmed Advance / Simulate interruption | planned → authorized; interruption retains authorized/blocked after reload; confirmed resume advances staged/simulating | Synthetic stage machine only; no key movement, hardware or recovery proof |
| Integrations | Local Integrations typed mock forms | Create/cancel, saved error state/reload, duplicate denial, edit, delete cancel/confirm and error-view retry; temporary record removed | Local plan metadata only; no provider endpoint |
| Support lifecycle | Account Support | Synthetic ticket create → resolved → reload; auditor has no Create/Edit; confirmed cleanup | No message or diagnostics transmitted |
| Scoped service operations | Operator Service operations / Key associations | Service operator can create maintenance but cannot edit custody; uncertain intent persists across reload, temporary record removed | Maintenance intent metadata only; no command/reconciliation executed |
| Theme/profile/images/IdP | Settings / Identity & access | Theme/name; avatar and brand local data URLs, reload, oversized/MIME rejection preserving state, removal/reload; saved provider matches selector on both planes; all four synthetic result states and disconnect | No issuer contact, image upload or actual authentication |
| Responsive/accessibility | 390×700 viewport; semantic labels/dialogs/tabs | No page overflow at width 390 for workbench, account, operator and typed configuration; local/account visible controls labelled; tab ArrowRight selects Availability; mobile screenshots retained | No full screen-reader or assistive-device certification |
| Privacy boundary | Disjoint asset roots, CSP, HTTP method/query denial | Local edit/discard CDP network observation empty and untruncated; HTTP tests reject configuration endpoint/uploads/local assets on remote | No production security/tenant isolation qualification |

## Every supported schema path

`{id}` is a map key and `*` is a list index. All controls are at `#/configuration`, grouped by their root section. “Rendered” proves the control exists in the browser, not a native execution or an individual mutation test. Exact DOM pointers/tags and detailed outcomes are retained in `output/playwright/schema-control-coverage.json`.

| Schema path | Group / control | Browser evidence | Execution gap |
| --- | --- | --- | --- |
| `/` | root / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/version` | version / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/sources` | sources / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/sources/{id}` | sources / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/sources/{id}/type` | sources / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/endpoint` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/account` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/region` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/project` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/environment` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/namespace` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/mount` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/path` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/resources` | sources / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/sources/{id}/resources/*` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/identity` | sources / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/token_file` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/profile` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/sops` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/age_identity_file` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/vault_allowlist` | sources / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/sources/{id}/vault_allowlist/*` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/vault_keys` | sources / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/sources/{id}/vault_keys/*` | sources / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/sources/{id}/audit_verified` | sources / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/mode` | workloads / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings` | workloads / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/bindings/*` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/bindings/*/name` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/source` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/resource` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/field` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/version` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/allow_offline` | workloads / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/bindings/*/allow_publish` | workloads / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/environment` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/environment/{id}` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/command` | workloads / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/command/*` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/command_sha256` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/untrusted_code` | workloads / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/directory` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/snapshot` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/snapshot/path` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/snapshot/recipients` | workloads / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/snapshot/recipients/*` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/snapshot/identity_file` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/snapshot/sops` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/targets/{id}` | workloads / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/workloads/{id}/targets/{id}/type` | workloads / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/path` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/endpoint` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/account` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/project` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/environment` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/resource` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/token_file` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/accept_snapshot` | workloads / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/workloads/{id}/targets/{id}/service` | workloads / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/targets/{id}/activation` | workloads / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/workloads/{id}/audit_required` | workloads / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants` | grants / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/grants/*` | grants / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/grants/*/subject` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/client` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/workload` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/operations` | grants / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/grants/*/operations/*` | grants / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/keys` | grants / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/grants/*/keys/*` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/not_before` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/expires_at` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/grants/*/authority` | grants / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/issuer` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/audience` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/jwks_url` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/address` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/origins` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/origins/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/hosts` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/hosts/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/audit_path` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/resource` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/issuer` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/jwks_url` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/trust_store_file` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/trust_store_sha256` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/audience` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/profile` | mcp / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/clients` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/clients/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/subjects` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/subjects/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/required_scopes` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/required_scopes/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/required_roles` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/required_roles/*` | mcp / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/permissions` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/permissions/{id}` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/permissions/{id}/scopes` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/permissions/{id}/scopes/*` | mcp / text/path/reference input | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/permissions/{id}/roles` | mcp / list + typed item controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/permissions/{id}/roles/*` | mcp / text/path/reference input | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint` | mcp / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint/type` | mcp / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint/proof_max_age_seconds` | mcp / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint/clock_skew_seconds` | mcp / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint/nonce` | mcp / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/mcp/authorities/{id}/sender_constraint/replay_store` | mcp / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/repository` | repository / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/repository/url` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/repository/directory` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/repository/ref` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/repository/allowed_signers_file` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/repository/initial_commit` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/repository/token_file` | repository / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability` | observability / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/enabled` | observability / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/service_name` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/environment` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/diagnostic_level` | observability / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/resource_attributes` | observability / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/resource_attributes/service_namespace` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/resource_attributes/service_instance_id` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/metrics` | observability / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/metrics/enabled` | observability / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/metrics/interval_seconds` | observability / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/traces` | observability / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/traces/enabled` | observability / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/traces/sample_ratio` | observability / numeric input | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/logs` | observability / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/observability/logs/enabled` | observability / boolean select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/endpoint` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/token_file` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/trust_store_file` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/trust_store_sha256` | observability / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/timeout_seconds` | observability / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/export_interval_seconds` | observability / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/batch_size` | observability / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/observability/queue_size` | observability / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates` | updates / group + optional field/map controls | Rendered in real browser; common typed-control behavior exercised, individual mutation not claimed | Native/provider not run |
| `/updates/check` | updates / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/channel` | updates / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/interval_hours` | updates / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/timeout_seconds` | updates / numeric input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/feed_endpoint` | updates / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/token_file` | updates / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/trust_store_file` | updates / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/trust_store_sha256` | updates / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/updates/minimum_version` | updates / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition` | composition / group + optional field/map controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/includes` | composition / list + typed item controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/includes/*` | composition / group + optional field/map controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/includes/*/id` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/namespace` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/sha256` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/file` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository` | composition / group + optional field/map controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/includes/*/repository/url` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository/directory` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository/ref` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository/allowed_signers_file` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository/initial_commit` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/repository/token_file` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/path` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/commit` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/base_directory` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/depends_on` | composition / list + typed item controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/includes/*/depends_on/*` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/visibility` | composition / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/refresh` | composition / select | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/includes/*/cache_expires_at` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/conflicts` | composition / select | Single schema constant retained; no alternative invented | Native/provider not run |
| `/composition/overrides` | composition / list + typed item controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/overrides/*` | composition / group + optional field/map controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/overrides/*/pointer` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/overrides/*/value` | composition / explicit type selector + typed value control | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/locks` | composition / list + typed item controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/locks/*` | composition / group + optional field/map controls | Optional section/list add and discard exercised; individual descendant edits not all exercised | Native/provider not run |
| `/composition/locks/*/pointer` | composition / text/path/reference input | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |
| `/composition/locks/*/value` | composition / explicit type selector + typed value control | Typed edit + exact full-document comparison on invalid all-fields draft; native validity not implied | Native/provider not run |

## Retained artifacts

The original `verification.json` and private Library ZIP bind the historical `0ecbc69` checkpoint. They are unchanged. Follow-up discard/navigation/reload, invalid-draft isolation, selected-device and migration evidence is in `output/playwright/verification-continuation.json`, with `configuration-discard-restored.jpg` and `migration-resumed.jpg`. Current schema/example hashes still match the owner checkout at `74e17dd62ac83d6504d6589e9d7a46b26ee77252`. Follow-up checks do not expand native/provider qualification.

`output/playwright/` contains actual desktop/guided/mobile screenshots, Linux typed-control and operator screenshots, original/expected/edited/re-imported exports, unknown-field preservation fixtures, all-fields DOM evidence, and the scoped verification manifest. This directory is ignored by Git to keep browser data out of publication. It is evidence for this exact local frontend, not a second project status ledger.

## Remaining owner obligations

- Phone localhost access is user-confirmed through the existing authenticated Termux SSH tunnel for ports 5181/5180. This is not an automated physical-phone test. No public URL, firewall change, new tunnel service or production deployment was created. Draft storage remains device/browser/origin scoped.
- Exact icon 13 contact sheet is preserved at `references/selected-icon-contact-sheet.png`; public source provenance and hash are in `references/selected-icon-provenance.json`. Private Library identifiers remain in the source owner's retained evidence. Original pixels were inspected: 13 is row 3, column 1. No isolated/vector logo or final approved-layout claim.
- Backend/Go/native/provider/production qualification and real identity/billing/custody/fleet remain outside this synthetic frontend. BAU resource holds remain intact.
- Publication/integration are coordinated with the implementation owner. No edits to the original checkout or canonical qualification/handoff files were made by this frontend worker.

The implementation owner integrates the frozen source with separate preview
serving, guided-reference, imported-JSON and module-boundary corrections.
The [qualification record](../docs/qualification.md#isolated-frontend-candidate-and-evidence-boundary)
keeps producer browser evidence distinct from those later source-only changes.
Evidence for the expanded action pass is retained in
`output/playwright/verification-actions.json`; the prior private ZIP remains
unchanged historical `0ecbc69` evidence. The integrated candidate requires
fresh hosted and browser verification. Real identity, billing, key service,
provider/fleet adapters, composition resolution and native/production qualification
retain their full implementation and exact-evidence requirements.

## Current CQ display projection

The current native schema title is CQ policy version 1; its exact source-owner bytes are SHA-256 `66634e5e23cb5d2268ca93b22bac3e6b76cb68307963ed39076c5f134b0d3f03`. Parsed schema equality after replacing only the historical title was verified. The earlier snapshot above remains historical evidence; current help source references and generated pins await the coherent owner update. The Linux example is unchanged.
