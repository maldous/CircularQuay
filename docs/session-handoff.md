# Development session handoff

Status: CircularQuay v1 development is active and **not production qualified**.
No release, package-index publication, production deployment, system/service
installation, credential migration, operational infrastructure change, or
live-provider qualification has occurred.
Do not claim a passed gate without its retained evidence.

## UIF migration and delivery closure

This is CircularQuay's canonical migration/impact checklist, within the existing
handoff rather than a second tracker. The complete accepted scope remains in
[the implementation plan](implementation-plan.md), its nineteen acceptance
scenarios and the linked product, management, identity, security, integration
and release contracts. Quiescence means delivered and verified agreed work,
not idle workers, authored documents, a mock UI or a successful subset of checks.

The current direction is to finish CircularQuay and prepare its controlled
adoption of the general **UIF** harness in parallel with OffMenu configuration,
data and BAU corrections. Neither project's completion gates the other's
independent work. Preserve immediate BAU priority on actual resource contention.
CIX execution, production, new paid resources and operational-key migration
remain held. USF V3 is a separate governed route; a workspace name does not
establish a selected USF manifest or permission to bypass its bindings.

### Authority and stable project identity

The verified native owner route is **CircularQuay — Fix code and CI**, chat
`01a1a102-5d55-7593-96ea-0c9ef716faa7`, parent of the existing `/root` subtask
tree. Runtime thread metadata establishes that identity, not a model SKU.
This owner coordinates integration and is the sole CircularQuay Linear writer.
Existing source, SDK, frontend and documentation ownership stays in place
until a separately verified per-unit handover; no second planner or scheduler
is introduced.

[CircularQuay in Linear](https://linear.app/maldous/project/circularquay-656f1967b41c)
is project `P-UIF-17`, UUID `98c8070a-2fb0-438c-9358-634064bf3cca`, under
Universal Infrastructure Foundation (`UIF`), team UUID
`53f584e7-70a1-4b1e-b982-b815210e47d4`, in workspace
`af4f493a-b39f-4eaf-a9b4-ed11debdf98c`. It was created and independently
read back in Backlog after active/archived project inventory and a fresh name
check. `checkpoint52/linear-project-created-readback.json` retains the actual
provider response. Project identity supports a dynamic view; it is not harness
admission, execution or delivery acceptance. Do not classify this product under
OffMenu or the paused CIX queue. Reuse existing shared UIF issues for shared
harness defects; deduplicate product issues before creating them.

The complete active/archived team inventory covered 161 issues before creation.
The dedicated project now has 18 product/handover issues, independently read
back in Backlog with no assignee or dispatch. The provider responses and
relations are retained in `checkpoint52/linear-created-issues-readback.json`.

| Delivery scope | CircularQuay issue |
| --- | --- |
| Native policy, signed repositories and configuration | [UIF-162](https://linear.app/maldous/issue/UIF-162) |
| Offline state and availability | [UIF-163](https://linear.app/maldous/issue/UIF-163) |
| Providers and local contract environments | [UIF-164](https://linear.app/maldous/issue/UIF-164) |
| Native delivery, synchronization and recovery | [UIF-165](https://linear.app/maldous/issue/UIF-165) |
| Identity and enterprise credential lifecycle | [UIF-166](https://linear.app/maldous/issue/UIF-166) |
| Managed software KMS and age service | [UIF-167](https://linear.app/maldous/issue/UIF-167) |
| Signed administration, hierarchy and strict enforcement | [UIF-168](https://linear.app/maldous/issue/UIF-168) |
| Local-to-managed custody migration | [UIF-169](https://linear.app/maldous/issue/UIF-169) |
| Client, account and operator interfaces | [UIF-170](https://linear.app/maldous/issue/UIF-170) |
| Official SDK qualification | [UIF-171](https://linear.app/maldous/issue/UIF-171) |
| Observability, telemetry and release awareness | [UIF-172](https://linear.app/maldous/issue/UIF-172) |
| Native, SDK, OCI and client distribution | [UIF-173](https://linear.app/maldous/issue/UIF-173) |
| Whole-product security and independent qualification | [UIF-174](https://linear.app/maldous/issue/UIF-174) |
| Public support, release documentation and repository controls | [UIF-175](https://linear.app/maldous/issue/UIF-175) |
| Managed hosting and operational readiness | [UIF-176](https://linear.app/maldous/issue/UIF-176) |
| Controlled general-UIF adoption | [UIF-177](https://linear.app/maldous/issue/UIF-177) |
| Project isolation and read-only executor readiness | [UIF-178](https://linear.app/maldous/issue/UIF-178) |
| First read-only Sonar analyzer-version assessment | [UIF-179](https://linear.app/maldous/issue/UIF-179) |

UIF-178 is a child of UIF-177; UIF-179 is a child of UIF-174 and is blocked
only by the relevant UIF-178 readiness checks. Shared framework issues remain
in the existing UIF project and are related rather than duplicated. Neither
project creation nor these issue relations makes the harness ready.

### Complete obligation and current-owner reconciliation

Each row retains the full referenced acceptance scope. Passing static evidence
does not close native, provider, installed-artifact or production obligations.
Issue binding and status must follow actual Linear readback; this table does
not fabricate Ready/Done states or transfer existing leases.

| Obligation | Current owner and state | Next executable action and evidence needed | Exact remaining boundary |
| --- | --- | --- | --- |
| Foundation, policy, schema and complete generations | Source owner; native v1 model/schema and composition source exist | Integrate current corrections; exercise exact scopes, collisions, unknown/duplicate/null fields, graph composition, private ownership and complete-generation failure invariants | Current flat native schema does not implement the whole managed/strict contract; accepted fields must never be ignored |
| Signed repositories and local configuration management | Source owner; pinned includes and native CRUD/doctor source exist | Qualify signed/pinned fetch, hierarchy edits, lock/identity races, offline cache/freshness, interrupted writes and automated CRUD/doctor behavior | Full authenticated hierarchy/profile engine and cross-interface parity remain incomplete |
| Offline/SOPS and availability | Source owner; age-only snapshots and typed transport work exist | Execute strict recipient/snapshot validation, expiry/replay/clock and availability-only hybrid scenarios, including every denied downgrade | Cloud KMS/key groups remain rejected by current SOPS source; copied keys/plaintext cannot be recalled |
| Actual software KMS plus age service | Source owner; dependency-injected key-service core and adapters only | Wire real scoped authorization, durable association/idempotency storage, provisioning, required audit and age custody; integrate opt-outs-before-network and supported CLI/API/SDK/MCP operations; test denial, cancellation, races and uncertain effects | No hosted service, deployed KEK store, entitlement or live endpoint is qualified; source development is distinct from deployment authority |
| Local-to-managed migration | Source owner; complete contract, no implemented end-to-end workflow | Implement selected inventory and approved reference/import/rewrap/re-encryption, per-asset resume, staged verification, atomic activation and recovery; test interruption and cross-domain/old-version denial | No operational inventory, key inspection/upload or migration is authorized; non-exportable keys and retained old copies impose real limits |
| Provider and platform integrations | Source owner; connector-specific source and synthetic tests | Complete the declared provider capability map and selected local emulators/contract suites, then qualify exact native identity/version/lifecycle/audit profiles | Mock WireMock/LocalStack/other emulator success is not live-provider permission or compatibility; actual test endpoints/entitlements remain required |
| Identity, enterprise lifecycle and authorization | Source owner; selected identity/DPoP/MCP behavior and broader design contracts | Implement and test the agreed issuer/token/resource associations, lifecycle/revocation, SCIM, SPIFFE/SPIRE, posture/scanner/policy-engine and host-keychain/mobile/biometric profiles where declared | Login, software custody or a connector name cannot imply a different capability; each external profile requires its own actual authority and conformance evidence |
| Administration, governance and default/strict profiles | Source owner; final design and control register, no unified strict engine | Deliver separate policy/enrollment/deployment schemas and typed operation map, signed intent, approvals, monotonic requirements, replay/revocation/audit/recovery and downgrade-negative tests | Current local CRUD and four workload MCP tools do not establish remote administration; unsupported strict requirements must deny |
| Native delivery, activation and recovery | Source owner; per-target implementations/templates | Finish sync/application/activation semantics and qualify Linux, Windows, macOS, Termux, containers and declared service/workload targets, cold boot, restore and reconnect | Cross-builds and synthetic runner tests do not prove no-login recovery, host ACL/DPAPI/keychain/TPM or operational restart behavior; no Foo service changes |
| Official Go, TypeScript and Python SDKs | Existing SDK owners; source/native subsets with failed overall qualification | Publish current lifecycle tests and Windows fixture correction, then obtain matching safe-error/cancellation/reaping/native/install/coverage receipts on the full matrix | Fixed cryptography releases inspected lack macOS Intel wheels; vulnerable locks cannot be accepted, handwritten hashes or silent platform removal cannot close this gate |
| Frontend, branding and interaction completeness | Existing frontend owner in separate checkout; local `0ecbc69` then `6057e39`, further bounded fixes in progress | Review and integrate the latest frozen `web/` checkpoint, exact schema/example pins, selected icon provenance and all agreed local/account/operator/fleet/migration/image/integration interactions | Rendered controls are not every mutation or backend/native qualification; local browser state is not a secret vault, real server authority or shared device storage |
| Observability, telemetry and release awareness | Source owner; native source plus synthetic UI | Verify safe metrics/traces/log vocabulary, bounded queues/export/shutdown and collector failure, required-audit independence, interactive/noninteractive update behavior, RC/stable/air-gap trust and minimum-version denial | Synthetic service charts and version notices do not establish production telemetry, audit durability or trusted installation |
| Public CLI, MCP, plugin and portable skill lifecycle | Source owner plus six-path documentation handoff; actual catalog remains bounded | Complete the declared supported surfaces, then install/discover/invoke/deny/cancel/reload/update/remove the selected native/skill/plugin packages in each claimed client | Source metadata, archive globs, skill text and an invented manifest/App ID do not qualify a registry/client or grant raw-secret/administrative authority |
| Packaging, releases and registries | Source owner; release configuration and instructions in RELEASE | Produce immutable native/Go/TypeScript/Python/OCI/MCP/MCPB artifacts, RC/stable manifests, signatures/provenance/SBOM and clean consumer installs; retain exact registry/download receipts | Main merge, tagging, registry publication and production deployment remain gated; actual account/channel authority and administrative permission gaps must be recorded |
| Security, coverage and independent qualification | Source owner and independent reviewers; failed current source/security/SDK/Sonar workflows | Publish compile/complexity corrections; finish 100% owned/new/native coverage, tagged-lane provenance joining, zero warnings/findings, formal/fault/fuzz/race/conformance and independent audit | No gate is waived; native reboot/provider/installed-artifact trials need appropriate isolated executors and explicit operational authority |
| Hosting, commercial and operational readiness | Source owner; conditional hosting-only forecast | Verify the selected existing plan/project/issuer/storage tariff and capacity, qualify bounded budgets, tenant/account/operator lifecycle and controlled local/preview/production delivery | Paid Netlify Pro label is not entitlement/capacity proof; Cloudflare billing readback and Netlify storage price remain unverified; no purchase/account/production authority inferred |
| Documentation and community/enterprise support | Source owner integrating existing docs owner | Publish the six-path source/preview/onboarding/skill reconciliation, maintain honest qualification/release records, validate public support/security/release links and usable multi-platform instructions | Final product design is the North Star; reports and badges must reflect actual evidence rather than an intended working state |
| UIF project adoption | Existing CQ owner with existing UIF Root/122/Gateway | Complete issue mapping, actual current-schema project profile, source/toolchain/privacy/review/publication routing, cross-project denial and recovery proof, then one controlled unclaimed unit | Current harness source restrictions and publisher remain OffMenu-bound; authored configuration is not validated, admitted, executed or accepted work |

### Worktrees, leases and briefing coverage

The runtime checkout is `/home/user/src/2026-11-03/circularquay-dev` on
`feature/initial-implementation`; the selected published development checkpoint
is `74e17dd62ac83d6504d6589e9d7a46b26ee77252`, tree
`b47d902663d55c400498d0eebe6fbc68096613be`. It is not a qualified release.
Preserve all current dirty source/documentation and immutable review receipts.
Frontend ownership remains exclusively `web/` in
`/home/user/src/2026-11-04/circularquay-browser-dev`, branch
`feature/browser-dashboards`; preserve its later source checkpoints and
historical private evidence rather than overwriting or dispatching duplicate
work. The six supporting documentation/skill paths remain with their existing
handoff owner; root owns security, release, qualification and this handoff.

At the shared-direction brief, four relevant active subtasks received the
updated instruction through supported in-tree messaging: `/root/sdk_go`
(TypeScript lifecycle tests), `/root/release_awareness` (Windows SDK fixture),
`/root/sdk_python` (read-only Linear discovery) and
`/root/sdk_cross_language_review` (independent source review). Delivery is
recorded separately from acknowledgment and completed work. Their subsequent
source/discovery receipts retain their concrete scope. Completed/errored
workers were not resumed merely to acknowledge. Frontend and supporting-doc
owners are separately coordinated; do not count another session's claimed
briefing as this owner's observed send.

### Project input contract and actual UIF gaps

The fixed toolchain is Go `1.27.3` with explicitly pinned helpers/tool manifests.
Node/TypeScript and Python SDK/native matrices are declared in
`.github/workflows/sdk.yml`; source/native/cross-build, security, Sonar and
synthetic provider lanes remain their own fixed workflows. Do not replace them
with OffMenu's Python/Node checks. Local heavy builds/tests/scans/resolvers,
installers and containers remain held; one existing small frontend Node process
and bounded frontend checks do not authorize another SDK/runtime process.
Use synthetic fixtures, no shell-profile credentials or operational secret
values, no system/service installation, restarts or shared BAU changes.

The client configuration/credential boundary stays local. Browser previews are
loopback 5181 (client) and 5180 (account/operator); existing authenticated
Termux SSH forwarding is now human-confirmed. That is not public deployment or
automated physical-phone qualification. Source and evidence publication remain
scoped to the existing feature branch/draft PR, with expected-parent protection,
exact blob/tree readback and preservation of all in-flight bytes. No existing
OffMenu publisher installation, credential, path or grant may be substituted.

The existing UIF owner/122 packet is
`/home/user/src/offmenu-worktrees/uif122-discovery-observation/.delivery/uif122-staging-dependencies/general-harness-gap-proposal.json`,
SHA-256 `6decc43b5a60808eebc470efe9b601af2719071998eb73b482d2b78817beed95`,
auditing harness source `5b8ca36e3a212d60db7b74bb8ccb37e3993bb01e`.
It identifies three actual shared gaps: project/repository/source selection;
fixed toolchain/private workspace/preflight profiles; and publication,
metadata/refill/recovery routing. Its source/doc pins are inspection evidence,
not current runtime admission. Existing shared issue fits include UIF-120,
UIF-140/141/142/159/161; verify their actual records before updates or dependencies.

The existing UIF Root/122 owns the current-schema profile and shared corrections.
This handoff supplies actual CircularQuay identity and requirements; it does
not invent a second profile schema or pretend the OffMenu-only policy accepts
CircularQuay. Before admission, obtain the actual profile/schema/source
revision and validate both project identities, disjoint paths, toolchains,
authority, output privacy and review/publication routes. Wrong-project source,
test, permission, publisher or result must deny before effects. Lost replies,
recovery/refill and blocked-project isolation require separate observed proof.
Classify application/fixture/toolchain defects separately from shared-contract
defects; material shared changes require impact/regression evidence for both
CircularQuay and OffMenu. No live queue migration or old-lease takeover occurs
from project creation or authored documentation.

### First controlled handover candidate

The useful currently unassigned candidate is [UIF-179](https://linear.app/maldous/issue/UIF-179),
UUID `9679255d-9c8c-4248-9a16-a283b2a7d3b6`: the **Sonar Python analyzer-version
contract**: the exact current scanner warns that Python 3.15 is unsupported and
uses 3.14. No active coding owner is changing that configuration. Reserve this
bounded read-only investigation for the first UIF unit only after issue,
authorization, routing, isolation and executor readiness are verified; it has
not been claimed, dispatched or executed by UIF.

Inputs are the immutable selected development checkpoint above,
`.github/workflows/sonar.yml`, `sonar-project.properties`, Python SDK metadata
and the declared SDK runtime matrix, plus the exact retained Sonar warning and
primary released-analyzer documentation. Permitted output is a source-bound
assessment/change proposal under the existing qualification evidence store,
reviewed by the CQ owner; no product/source write is preauthorized for this
first unit. The output must distinguish analyzer/source-language configuration
from runtime compatibility, preserve the runtime matrix and every strict gate,
and identify an evidence-backed correction or exact upstream limitation.
It cannot suppress warnings, silently drop Python support or infer a pass.

This candidate needs no credential, live provider, Go/Java/Node execution,
container, installer or publisher effect. The actual harness read-only profile,
fixed input/output checks and executor suitability still require qualification;
do not replace missing capability with arbitrary command execution. Accepted
output returns to the existing CQ owner for review, source integration and
exact-candidate CI. Any source/project/rights mismatch or uncertain admission
returns to the current owner without duplicate dispatch. This is independent
of OffMenu completion while respecting real shared-resource holds.

The readiness scope for this first unit is limited to project/source identity,
read-only acquisition, isolation, bounded investigation and evidence return.
Go validation and GitHub publisher support are separate capabilities, not
prerequisites for a unit that has neither effect. The source-owner input
binding is retained under `checkpoint52/uif-179-source-input.json`; it is an
assessment contract for the existing UIF owner, not an invented harness
manifest, execution admission or transfer of a current lease.

## Published candidate and qualification boundary

Use `git rev-parse HEAD` to select the current checkout. Every receipt below
qualifies only its recorded revision, tree and execution scope.

Latest captured public development checkpoint is
`74e17dd62ac83d6504d6589e9d7a46b26ee77252`, tree
`b47d902663d55c400498d0eebe6fbc68096613be`. Source, security, SDK and Sonar
fail; WireMock and Dependency Review pass only their own scopes. The
[exact observation](qualification.md#exact-74e17dd-hosted-observation) records
the actual compile, coverage, scanner and dependency failures. Current
source-only corrections require fresh hosted qualification after publication.

The preceding captured public runtime/design commit is
`c18be5a76bc82ed7479e8ddbb6b4d28828890fa4`, tree
`e85408243741ccbf0483ca28e75fa87cf04bd973`. Its completed hosted metadata
records source, security, SDK and SonarQube failure; synthetic WireMock and
Dependency Review success applies only to their respective scopes. The
[exact observation](qualification.md#exact-c18be5a-hosted-observation) retains
the run links and bounded evidence. No production qualification follows.

Preceding public commit `10dde6d3b94fce98e3415e4ca8d8c0dcb649f0d0`,
tree `9174f780b61636b6f6e9820264bebe0b446910c6`. Its bounded hosted readback
records source/security/Sonar failure, SDK queued and three WireMock jobs plus
Dependency Review passing. Source Linux and CodeQL share a missing `platform`
import in `internal/cli/cli.go`; the following amendment restores it. Linux
also reports protected-path/trust-store and composition fixture mismatches.
Use the [exact observation](qualification.md#exact-10dde6d-hosted-observation)
and separate source receipts; no dirty source or new KMS/profile code is
qualified by those failed runs. Prior d9ff832 source/SDK cancellations remain
historical. No manual rerun or cancellation occurred during this readback.

The preceding complete historical analysis for
`275715525b74c99200ee9afc83ff069f6129b4a2`, tree
`2e7bb1cf20fda50c07107a5dccb81154d57433ad`, has retained hosted readback in
`checkpoint45-2757155-hosted-sonar.json`. Synthetic merge
`8ebd570715148137fa61e6e38443f83b82c9e786` uses the same tree. The Linux
complexity step passed. Sonar CE/upload/readbacks succeeded with zero issues,
hotspots and duplication, but the gate still fails 66.5% coverage, absent
new-code metrics and the Python-version warning. Go statement coverage is
7,295/8,738 (83.485923552%). Source assurance, security and SDK failures remain;
WireMock and Dependency Review pass only their recorded scopes. See
[exact qualification](qualification.md#exact-2757155-hosted-receipt).

### Following client-side composition and security work

The following source introduces `internal/configuration`, explicit pinned JSON
includes, bounded namespaces/dependencies, conflict rejection, root overrides
and equality locks. Single-file JSON/YAML remains supported through the
separate parser. The native CLI, linked Go SDK and native-process SDK paths
share the resolver; operational dispatch retains a complete snapshot.
`config get/list` return only the protected root declaration. Validation and
atomic writes resolve the whole candidate. MCP retains its existing four-tool
workload catalog and separate authorization/reload boundary.

Repository includes call `repository.FetchPinned`, which checks the exact
commit before acceptance while retaining signature/ancestry/quota/native Git
protections. Cache admission is not a distributed transaction across the
complete graph. Private-domain guards check declared data/credential and
executable paths, repository/writer locations, named remote grants, publication
and snapshots after root edits. They do not prove different key bytes behind
copied paths or constrain an authorized hostile plaintext consumer. The
[composition reference](composition-reference.md) and synthetic examples
contain the exact implemented descriptor vocabulary.

Independent source review, pinned formatting and bounded static checks are
complete; new tests have not run locally under the shared Foo BAU hold.
The source commit is `5b6770594bd50a8b7459b9e597215c0f0ebb22fe`, tree
`0e5c4de6f12cbcc7bcb8593996c78381027b029e`. It supersedes a never-ref-updated
source object after removal of an unused import caught by a bounded check.
Hosted/native/security/Sonar evidence for these new source bytes is required.
Do not run local builds, resolvers, tests, analyzers, installers or containers
without release of that resource hold. No production qualification follows.

The security-design review makes the required signed effective profile,
accepted generations, pre-effect reauthorization, approvals, durable audit,
downgrade-negative tests and cross-product control mapping explicit. Actual
source still lacks the signed strict-profile/hierarchy engine. Default/strict
are product contracts, not accepted runtime flags or certification claims.
Never advertise strict enforcement from this composition implementation alone.

The following development source adds a profile-bound `internal/keyservice`
core, optional native OpenBao/Vault Transit adapter and maintained Go JOSE
software adapter (`A256KW`/`A256GCM`, private KEK resolver, fixed flattened JWE).
Software is not HSM/non-exportable custody or a deployed hosted backend.
It requires separate KMS/age
actions, complete dual enrollment, conditional activation/reconciliation,
stable approved cryptographic context, safe error normalization and current
authorization after provider waits. This is not a hosted deployment or
policy/CLI/SDK/MCP integration. Real authorization/storage/provisioning,
hosted age custody, metering, strict-profile enforcement and migration remain
absent. New synthetic tests are source-reviewed, not executed under the hold;
exact provider, native and full engineering qualification remain required.

The same source batch fixes the missing CLI platform import, empty optional
reference aliases and intermediate composition JSON decoding. Ordinary JSON
root requirements and complete resolved validation remain. Source assurance
separates manifest-bound integration-tagged test-only inputs from untagged Go
commands and adds required `tagged_integration:not_run`; it cannot pass until
the missing fixture-lane evidence/join is implemented. No denominator or gate
is waived. Preserve the separate exact source reviews for scope, policy/model
and custody changes; subsequent hosted results must bind their published bytes.

Latest confirmed scope keeps **all configuration client-side**, including
source authentication through the user's own network/VPN, composition,
validation and local save. No raw/resolved config, source credentials or
protected references may reach a hosted backend, proxy, analytics or logs.
The intended hosted-service contract requires genuine remote key wrap/unwrap
and authenticated age-private-key retrieval. Authorized enrollment
provisions both associations; it does not download an offline key, activate hybrid or constitute
a successful-fetch billing event. KVM/PVM were clarified to mean KMS configured
in the application user profile, not OS USERPROFILE or ambient credentials.
First-run CLI/config opt-outs run before hosted network. Hosted initialization,
provisioning, retrieval and metering remain absent. Source/schema/CLI/SDK/MCP
integration and exact-provider qualification remain required. Current SOPS
rejects cloud KMS/key groups and writes age-only snapshots. Online-only has no
offline bypass; explicitly authorized hybrid retains positively classified
availability-only fallback with all scope/window/freshness/expiry/audit and
revocation checks. Denial, trust failures, malformed data, expiry, cancellation
and unknown failures never trigger fallback. Local age, hosted age retrieval
and remote KMS are distinct custody boundaries. Copied keys/plaintext cannot
be recalled. Synthetic frontend development/previews and a loopback local
client dashboard are authorized; real backend and production UI remain held.

The long-term enterprise North Star remains. A lean secure release can defer
optional premium infrastructure/features under its explicitly selected scope,
without lowering authentication, tenant isolation, key protection, abuse
controls or shipped-code quality gates. The existing managed-service forecast
contains sourced provider rates and hypothetical costs/proposals, not approved
tariffs, traction or purchased accounts. No verified deployed worker/site/
database/endpoint for this service is recorded; prepared settings are not
applied or tested infrastructure. The latest explicit frontend authorization
covers synthetic/mock user/account and operator/admin previews, local client
forms/validation/configuration editing and simulation. It supersedes the prior
frontend deferral, not backend integration or production deployment. Work in a
separate frontend checkout/branch, own only `web/`, keep client configuration
loopback/local and preserve the runtime publication snapshot. The separate
frontend candidate and its exact evidence are recorded below. The earlier
unbound-port readback is historical, not a reservation or current listener
check. Keep frontend checks bounded and retain the native heavy
execution hold. Reuse qualified hosting,
identity, payment and support services where appropriate; never expose their
operator credentials or treat hosting invoices as customer subscription billing.
No payment setup, charges, account connection, deployment or BAU change occurred.

The latest explicit launch decision authorizes software custody on one existing
paid Cloudflare or Netlify footprint, with no required paid external KMS or
upgrade. Protected runtime KEK stays separate from the database; independently
random profile DEKs do not isolate shared-KEK/runtime compromise. Strict policies
requiring non-exportability or independent native custody deny this profile.
Fundamental security and all engineering gates remain unchanged. No actual
host entitlement, spare allocation or software backend/deployment is qualified.
Paid Netlify Pro status alone does not establish billing family, database
allowance, credits or incremental cost; Cloudflare billing readback is denied.
Do not change shared BAU account limits, buy resources or infer account selection.

The forecast now answers hosting-only cost for product usage tiers: native
request counts and software-cryptography runtime, storage/database/audit/backup and egress, with
paid/free cohorts and public-plan inflection points. Separate incremental spend
from total allocated hosting. Exclude labor, external KMS, CRM and payment fees
from that subtotal. Earlier age-only/partial-KMS and labor-loaded proposals are
superseded, not valid current hosting prices or tariffs. Preserve historical
receipts and the latest supporting amendment without overwriting them.

Local-to-managed migration covers actual provider import/
export and non-exportable limits, rewrap versus re-encryption, idempotent
interruption recovery, destination activation and retained copies. No actual
inventory, key inspection/upload or migration was performed.

### Isolated frontend handoff

Local-only commit `0ecbc694f8454663fd9416d57a624f66314d75ee`, tree
`12c07b465845de2ad8f8cc17e24b0a4dce0d5820`, adds 19 files under `web/`
on `feature/browser-dashboards` in
`/home/user/src/2026-11-04/circularquay-browser-dev`, based on `10dde6d`.
It is not pushed or integrated into this runtime branch. The producer reports
one Node process on loopback 5180/5181, seven focused Node tests and synthetic
browser journeys. The workbench includes a masked grid/guided inspector,
local `.env` import, per-variable source/key/window/timezone/provenance controls,
typed configuration editing and simulated apply/run. Account/operator consoles
are separate synthetic presentation surfaces; configuration remains local.

Use the [scoped qualification record](qualification.md#isolated-frontend-candidate-and-evidence-boundary)
for exact commit, schema/fixture, report, coverage and independent-receipt
hashes. Independent readback verifies 19 source hashes and retained JSON
equalities; it does not rerun the producer's Node/browser tests. All 187
schema-path controls are reported rendered, with representative mutation
coverage only. No native/provider/backend/production claim follows. The
external preview route and selected icon 13 asset are unresolved.

Disposition: retain the local candidate and owner-held `web/coverage.md` and
`web/output/playwright/` evidence. Before integration, coordinate the exact
commit and current runtime base, recheck schema/Linux fixture hashes, review
the source/security and supported-command boundaries, and qualify the merged
candidate through the required gates. Do not infer remote authority from UI
role selectors, use operational values in browser demo storage, reuse BAU
routes, cherry-pick automatically, or expand the native heavy-execution hold.

### Evidence ownership

Supporting documentation is coordinated with the existing owner and retains
its exact composition/first-run/forecast and subsequent account-design receipts.
Root reserves governance, security/assurance, RELEASE, qualification and this
handoff. Preserve prior owner manifests and use final amended hashes for
publication rather than overwriting historical evidence.

Retain the composition security review, five-file source amendment, final
cache/cancellation admission review and separate unused-import amendment.
The worker overwrote the first draft cache-admission receipt after reporting
its hash; its original bytes were not retained. The separate checkpoint47
receipt records that transition and current hashes. Do not call that first
draft immutable. Final independent receipts remain separate and unchanged.
Supporting first-run and subsequent 13-path account/database/payment-design
manifests have their own static and formula checks, with no live qualification.

The following publications and receipts are historical.

Reviewed publication `fa767446175f76ae442ff81baff4b5d7ac2d73bf`, tree
`77843bdaec5eab412d1ed6a490b854af563c1b2b`, has readback for all 62 changed
files and preserves 392 working files. PR #1 remains draft to `develop`.
[Exact fa767446 evidence](qualification.md#exact-fa767446-hosted-receipt)
records source/security/Sonar failures and the explicitly partial SDK
observation. All four prior Sonar assurance findings are closed; coverage,
new-code metrics and the Python-version warning still fail. TypeScript fixture,
package and ESM consumer repairs pass in completed Ubuntu/Windows lanes;
the selected run has 18/18 tests but fails real coverage. Windows Python's new
diagnostic narrows the failure to protected-directory checks without relaxing
permissions or retrying consumers. Do not claim a cause/remedy from those codes.

The following OTel helper extraction addresses the remaining observed
`newOTelBackend` complexity 12. It preserves client/resource/URL prerequisites,
trace→metrics→logs ordering, error identity and cleanup. Independent source
review and pinned formatting pass; no post-extraction runtime/analyzer has run.
Retain exact `internal/observability/otel.go` SHA-256 and later hosted evidence.

The documentation owner handed off 29 stable lifecycle/support/skill paths in
`lifecycle-docs-20261104/assessment.json` and `scope-manifest.json`, with contained
patch, static validation and primary-source research. Root owns the new
canonical governance document and qualification/handoff. Preserve the owner's
manifest; root amendments get their own exact fences. The complete v1 now
includes canonical hierarchy/defaults/locks and deterministic group handling,
SCIM, authorized issuer-native rotation/revocation, GitHub and optional
GitGuardian metadata exposure gates, safe OTel events, OIDC/SPIFFE workload
identity, posture/Conditional Access and optional bounded external policy
decisions. These contracts do not add unsupported JSON fields or CLI/MCP verbs.

Required implementation remains explicit in qualification: canonical schema/
compiler and protected enrollment/deployment contracts, managed lifecycle and
session/credential-version state, common authorization/approval/audit and
typed API/CLI/SDK/MCP parity, safe metadata adapters and verified provider
mutation/reconciliation. All belong to v1. Workload/profile, scanner and issuer
evidence remain separate. Known compromised versions block unless an approved
finite exact-version exception permits a still-valid operation; warnings only
present that exception. Incident closure cannot prove rotation/revocation or
clear the block. No exception revives an expired/revoked credential. Age keys
and consumer plaintext cannot be recalled.

Official API research verified GitGuardian `Authorization: Token` and the
base read-only incident profile, excluded generic secret-bearing webhooks,
and required GitHub `hide_secret=true` with exact repository permissions.
No accounts, keys, secret scans, plaintext/value-hash queries, provider
mutations, purchases or infrastructure changes were performed. Endpoint/plan/
privacy and nonsecret association qualification precede live operation.
Use forward corrections/migrations that preserve locks, trust generations,
offboarding/compromise history, replay/approvals and audit.

New retained receipts include `configuration-governance-review.json`,
`checkpoint43-typescript-fa767-hosted.json`, `ci-fa767446-sonar.json`,
`checkpoint43-security-fa767446.json`, the source receipt and its separate
`checkpoint43-source-go-sdk-amendment.json`, and
`checkpoint40-windows-config-init-diagnostic.json` (its content binds fa767446
despite the historical filename prefix). The source reviewer corrected an
initial receipt in place; the later amendment explicitly records its old/new
hash transition and that the original bytes were not retained. Do not describe
that overwritten first receipt as immutable evidence.

The following checkpoint is historical and retains its own scope.

Publication `208bfe016a8e00c55d9e2394b2a930eec64cd606`, tree
`386b60dbdc59d95aee63c1ca7dfeee9e2a71368d`, has public blob readback
for all 20 changed files and preserves all 383 working files. Its six PR
workflows and three push-triggered workflows are terminal. No manual rerun or
cancellation occurred. PR #1 remains draft against `develop`. Use
[the exact receipt](qualification.md#exact-208bfe0-hosted-receipt); later source
and documentation changes require their own hosted evidence.

Source PR `41790956273`: 29 successful/two failed jobs. All 26 cross-builds,
their aggregate and macOS/Windows native suites pass. Linux normal/race tests,
vet and bounded fuzzing pass; the four earlier Go SDK complexity findings are
absent. Three new cyclomatic findings affect the bounded assurance parser and
its tests. Cognitive analysis is not reached. The following independently
reviewed extraction and command-spec object address those paths plus Sonar's
high-arity diagnostic without changing report version, redaction or gates.

The source auditor now attributes its failed unit command only to
`internal/integration`; vet remains `unknown`. Coverage is 7,273/8,707
(83.530492707%). Source review explains the mismatch: this package has only
integration-tagged tests, while the normal audit explicitly selects it without
that tag. The dedicated WireMock workflow runs the tagged suite separately.
Do not add an empty untagged file to conceal the deliberately failed scope.
A complete solution must represent and qualify the separate tagged scope.
The actual report binds merge `d5268bb25786ae5a54e142a81746e5ed15bd7c69`
and the publication tree; an earlier conversational reference to the previous
merge was corrected against the report, not adopted as evidence.

Security PR `41790956279`: CodeQL analysis/upload, Staticcheck and
govulncheck pass, while six G204 and two G103 findings remain unsuppressed.
No authoritative SHA-specific zero-alert readback exists. Three historical
GitGuardian incidents also retain current-scan readback/triage gaps. The bounded
public-source provenance review inspected no resolved operational values and
made no incident disposition.

SDK PR `41790956319`: four successful/28 failed jobs. Go SDK contracts pass
on all three native platforms, and Go lock reproducibility passes. All nine
TypeScript lanes pass lint, type checking and observed dependency audits.
The 18-test run has 15 passes and three failures in the new recording fixture,
which omitted the runtime prototype; the following fixture correction retains
public validation and its no-process dispatch seam. Windows now reports a real
coverage denominator (56.89% lines/statements), not the earlier zero-file pass.
Intermittent tar/grep SIGPIPE and installed-ESM resolution errors are separately
attributed fixture causes. The following workflow checks complete archive-member
output and imports the installed ESM package from its actual consumer directory.
Neither repair changes SDK exports or grants. The selected Ubuntu Node 24 MCP
step passes both official TypeScript and Go conformance scripts.

All nine new npm producer artifacts bind the current package input and the same
reviewed lock bytes. The following provenance update uses those actual receipts;
no dependency or integrity byte is invented. Retain the exact producer archive,
input, reviewed-lock and revision/tree hashes.

Python Ruff, mypy and Bandit pass on all three static platforms. The Windows
coverage artifact has all five canonical modules and exactly 76 ordered events
for all 38 source-defined tests, confirmed by AST comparison. An earlier
conversational 56-event count was a truncation mistake and was corrected before
recording this receipt. Coverage still fails the unchanged 100% threshold.
All five Linux and all five macOS installed/minimal consumers pass. Windows
3.11 consumers pass; 3.12–3.15 fail after native launch with `trust: config_init`.
Existing safe diagnostics cannot attribute the exact native creation check;
do not guess a permission relaxation. The selected Ubuntu Python 3.15 official
MCP check now passes after its required audit destination was configured.
The six vulnerable Intel macOS cryptography locks remain an upstream-wheel and
advisory blocker, with platform, wheel-only, hash and audit requirements intact.

Sonar PR `41790956278` completes CE/upload/readbacks at analysis
`c00f81c1-6a28-4cee-b463-b0c5dc8fa356`, merge `d5268bb25786ae5a54e142a81746e5ed15bd7c69`
and the same tree. Test/source classification takes effect. Four findings
(high arity, two parser complexities and one test complexity), 66.5% coverage,
absent new-code metrics and the Python 3.15 fallback warning still fail the gate.
Duplication, hotspots and CE warnings are zero. Python XML and TypeScript LCOV
production/ingestion remain absent; a classification correction is not coverage.

WireMock PR `41790956340` passes all three native synthetic suites; Dependency
Review `41790956284` passes. Push-triggered source/security/WireMock outcomes
agree at their own scope. Retain `checkpoint41-all-run-state.json`,
`checkpoint41-source-go-sdk-readonly.json`, `checkpoint41-security-208bfe0.json`,
`checkpoint41-python-sdk-qualification.json`, `checkpoint41-sonar-208bfe0.json`
and the exact source, fixture, workflow, lock and document-review receipts.

The documentation refresh is coordinated with the existing documentation owner.
Product and skill guides describe the complete v1 contract; this handoff and the
qualification matrix retain actual state and gaps. Local heavy execution,
resolvers, installs and containers remain held for shared-resource coordination.
No release, deployment, registry publication or production qualification follows
from a source correction or documentation update.

### Following reviewed design and diagnostic changes

The following Windows-only diagnostic runs only after an installed Python
consumer's original `trust: config_init` failure and a private request marker.
It preserves/re-raises the original failure, creates only fresh synthetic
children, rejects reparse ancestors and emits finite case/stage/category JSON.
It does not repair ACLs, retry the consumer to manufacture a pass or print
raw errors/paths. Source/format/syntax review is complete; no diagnostic test
body has run locally and no native creation-stage cause is yet established.
Retain `checkpoint42-config-init-windows-diagnostic.json` and
`checkpoint42-sdk-workflow-windows-diagnostic-review.json` with exact hashes.

The documentation owner handed off the full README/product/operator/SDK/skill
refresh in `replacement-documentation-20261104/assessment.json` and
`scope-manifest.json`. Root then added the complete v1 discovery/management,
administration-security and product-wide default/strict profile contracts,
linked them from README/navigation/enterprise/managed/agent guides, and expanded
the implementation plan and security review scope. Preserve original receipts
and the root amendment fences rather than overwriting the owner's manifest.
These are final-state requirements, not new executable command or tool names.

Actual gaps are explicit in [managed-service qualification](qualification.md#managed-service-design-status):
authoritative v1 management resource/error/event API contracts and distinct
enrollment/deployment schemas; remote administration and typed SDKs; signed
change/approval/revocation/HA/audit implementation; and unified profile selection
and downgrade enforcement. Existing policy schemas have no assurance-profile
field. Current four-tool/one-page workload MCP remains deliberately strict.
Do not introduce ignored schema fields or translate a local write into remote
authority. Review the new ADMIN controls and downgrade-negative matrix before
implementing the relevant trust boundary.

OffMenu/UIF restoration and useful parallel BAU work have immediate resource
priority. This bounded documentation/source handoff uses no additional workers,
local heavy execution, containers, installs or operational secret inspection.
Hosted candidate qualification may continue without changing the release holds.

### Preceding source-attribution checkpoint

Publication `20703ec8c1bd66eab4899ec6ce27d4f0385adc1b`, tree
`af4f26232a5dce50ecbc6e889defc7d75a7a1ea8`, has public blob readback
for all 19 changed files and preserves all 378 working files. Its six hosted
runs are terminal, with no pending, skipped or cancelled job. PR #1 remains
draft against `develop`; use [the exact receipt](qualification.md#exact-20703ec8-hosted-receipt)
for platform, phase and limitation distinctions. The following source changes
have independent source review only and require their own hosted results.

Source `41786298796`: 29 successful/two failed jobs. All 26 cross-builds,
their aggregate and both Windows/macOS native suites pass. Linux normal/race
tests, vet and bounded fuzzing pass before four Go SDK cyclomatic-complexity
findings; the cognitive analyzer is not reached. The following SDK helpers
preserve validation and observer behavior while addressing those four paths.
The source auditor independently fails unit tests and vet without individual
diagnostic attribution, and fails coverage at 7,168/8,592 statements
(83.426443203%). The following optional v1 failure metadata retains only
bounded manifest packages and tracked Go path/line references; raw diagnostics,
arguments, endpoints and values remain absent. It does not change failed
statuses or gate requirements.

Correction to the earlier a26e39e summary: its Linux job had nine cyclomatic
findings, not five. Five were outside the SDK; the same four SDK findings
persisted because their source blobs were unchanged. Preserve the historical
receipt and its explicit amendment rather than replacing prior evidence.

Security `41786298795`: CodeQL build, analysis and upload succeed; no
revision-bound zero-alert count is independently established. Staticcheck and
govulncheck pass. Gosec no longer reports the saturated heap conversion, but
six G204 process-boundary and two G103 native-ABI findings remain unresolved
and unsuppressed. Do not hide required subprocess or FFI operations to obtain
a cosmetic scanner pass.

SDK `41786298659`: four successful/28 failed jobs, including the failed
aggregate. Go SDK tests pass on all three native platforms; Go locks pass.
All nine npm lanes install reviewed dependencies and observed audits report
zero vulnerabilities. TypeScript exposes owned lint/type failures; its 12 tests
pass, but the recorded Linux sample has only 52.91% line/statement coverage.
Windows reports a zero denominator and exits successfully; this is not valid
coverage. The following corrections repair the attributed type/style issues,
explicit cross-platform source selection and bounded fail-closed coverage
validation, with additional public-boundary tests. Only package scripts changed;
the historical dependency-lock provenance has not been rebound to new inputs.
Review the next actual hosted producer receipt before changing provenance.

Python Ruff, mypy and Bandit pass on all three platforms. Linux coverage
captures all five owned modules: client 79%, local 62%, types 92%, and
initialization/errors 100%; the 100% gate correctly fails. Windows captures
all 56 progress events but no accepted source rows; the following parser
normalizes only the finite owned relative paths and requires all five rows.
Additional synthetic tests exercise public validation boundaries; they have
not run locally. Sampled installed-wheel, source-distribution, clean-environment
and no-optional-MCP local consumers pass on Linux, Windows and macOS. These
samples do not establish every platform/version combination.

The sole selected official Python MCP lane (Ubuntu/Python 3.15,
job `126429100521`) reaches the server and fails scoped `read_secret` as
`denied`; other lanes intentionally skip that step. Source tracing shows the
synthetic policy omitted the required disclosure auditor, so the service
correctly fails closed before provider retrieval. The following workflow
configures a relative audit path inside each existing protected synthetic root
for both official MCP fixtures. Grants and disclosure controls are unchanged;
there is no post-correction MCP result yet.

Six Intel macOS Python locks contain vulnerable cryptography 48.0.1. Fixed
releases lack compatible upstream wheels. Retain Intel, wheel-only, hash and
audit requirements; do not suppress advisories, remove the lane or invent locks.
This remains an external launch blocker, separate from source corrections.

Sonar `41786298735` completes upload/CE/readbacks at analysis
`4515e664-cfcf-4be6-8585-5cb923c4f9be`, tested merge
`621596917d7c489a2a40a759756a0d02cfa17518` with the same tree.
Open issues, hotspots, duplication and CE warnings are zero. The strict gate
still fails at 63.9% coverage, absent required new-code metrics and the Python
3.15-to-3.14 fallback warning. The missing duplication metric is not an observed
nonzero duplication result. Only Go coverage is imported; analyzed Python and
TypeScript have no language coverage input. The following scope correction
classifies actual tests as tests, while retaining implementation and evidence
validators as analyzed sources. It is not a coverage exemption. Revision-bound
Python XML/TypeScript LCOV production and ingestion remain unimplemented;
retain the reviewed same-run design rather than implying ingestion exists.

WireMock `41786298636` passes all three native synthetic suites. Dependency
Review `41786298740` passes. Retain `checkpoint40-candidate-run-state.json`,
`checkpoint40-source-security-review.json`, `ci-20703ec8-sonar.json` and the
exact SDK, helper, workflow and independent review receipts. Local
builds/tests/scans/resolvers, installs and containers remain held for
shared-resource coordination. A source-only correction is not a new test pass;
no duplicate run, release, deployment or BAU change is implied.

### Preceding native-fixture checkpoint

Publication `a26e39e46930893259b8385c65953b0896e5bdab`, tree
`ee50f613145dc21d1d6ea1db24452cd206b25a0f`, has public readback of
24 changed files and all 376 working files. Its six hosted runs are terminal;
[the exact historical receipt](qualification.md#exact-a26e39e-hosted-receipt)
retains their scope. Source has 27 successful/four failed jobs, including nine
cyclomatic findings, Windows SDK path fixtures and macOS symbolic temporary-root
fixtures. Source-profile unit/vet attribution is unavailable; coverage is
7,084/8,578 (82.583352763%). Security has nine gosec findings. SDK has three
successful/29 failed jobs; npm stops before installation at the reviewed-lock
gate, and Python parser/consumer/MCP setup fixtures fail. Sonar completes
analysis but fails at 63.7% coverage, 0.2% duplication, one test-complexity
issue, missing new-code metrics and a Python-version fallback warning.
WireMock and Dependency Review pass. Keep `checkpoint39-run-state.json`,
`checkpoint39-source-review.json`, `ci-a26e39e-sonar.json` and the explicit
cyclomatic-count amendment. The later 20703ec8 receipt, rather than these
results, records qualification after those corrections.

### Preceding compile-correction checkpoint

Publication `9d12512b11b462908f69159a16144dd80a0d0764`, tree
`bcfb89e4aa6974945163861f0f8bd3ce21ed0760`, has a complete public
readback of 35 changed files and all 374 working files. The checkout was
clean after publication; PR #1 remains draft against `develop`.

Source run `41776969369` and security run `41776969295` expose two further
compile causes: the MCP status/run methods redeclare named results with
`:=`, and an update-error constant collides with a test helper. The following
source uses assignment to the named results and a distinct operation-name
constant. Its package-wide source review found no analogous telemetry
declaration collision; compilation is not a source-review result.
Final metadata records all 31 source jobs and both security jobs failed.
Staticcheck and CodeQL are compilation-blocked; govulncheck, gosec and CodeQL
analysis are skipped. Fuzzing and complexity checks are not reached.

SDK run `41776969298` passes Go lock reproducibility. All three Go SDK jobs
fail before SDK tests at those MCP declarations. Reviewed npm locks install
successfully and the jobs reach actual lint, type, test, coverage and audit
checks. They expose owned type/style/test failures, 52.89% line coverage in
the recorded TypeScript sample, and 14 moderate toolchain dependency
advisories. Python static jobs install reviewed locks and pass helper tests;
their structured artifacts attribute 64 Ruff, 28 mypy and three Bandit
diagnostics to SDK source. Runtime closures install successfully where their
jobs have started. Python coverage/build/consumer/MCP results were still
pending at this receipt; do not infer a runtime pass or exact stalled test.

The subsequent authenticated Ubuntu static-coverage artifact `13298656230`
records a bounded 1,200,004 ms timeout and the exact incomplete test
`test_client.MCPClientTests.test_cancellation_propagates`. Its test double
rejects the adapter's `arguments` keyword before setting the start event,
so the fixture waits indefinitely. The following fixture correction uses
the actual keyword contract, a bounded wait, task cleanup and explicit
cancellation/observer assertions. This is not evidence that product
cancellation failed. Retain `checkpoint38-python-timeout.json`.

The following SDK source also aligns strict runtime-version DTO/SemVer
validation across Go, Python and TypeScript. Go exposes linked-core metadata;
the other local adapters inspect a selected native executable. TypeScript
toolchain fixes declare exact `glob` and `minimatch` overrides. The existing
lock/provenance remain evidence for the older inputs: hosted lock production
must retain the new closure, fail on drift, and await actual review before
installing it. No hand-edited integrity data supplies a pass.

Sonar run `41776969359` completes upload, CE and six HTTP-200 readbacks at
analysis `027a6c1b-e18e-4d48-ac45-4ecb0107e208`. Its synthetic merge is
`685444aa9a0ed7500b83954eec722a71c967101a`, with the public candidate's
tree. The previous 17 issues are absent; three new helper-regex issues
remain. Coverage is 0.0% after compilation fails, duplication is 0.2%,
hotspots and CE warnings are zero, and required new-code metrics are absent.
The pinned Python analyzer explicitly warns that it coerces declared 3.15
to 3.14. Keep that warning failed; current upstream source is not evidence
of released-tool support. The following source repairs the reported SDK and
regex causes without suppressions. Fresh qualification remains required.

WireMock `41776969322` passes the tagged synthetic provider suite on Linux,
macOS and Windows. Dependency Review `41776969351` passes. These do not
qualify live providers or the following source changes. Retain
`ci-9d12512-source-security.json`, `ci-9d12512-sonar.json` and
`ci-9d12512b-wiremock-dependency-review.json`.

### Preceding dependency checkpoint

Checkpoint `4c703c84fd9c1647b9a479c5525fdd3bfc653962`, tree
`f098a84ef22731a2cf9556ab664199154f11fb20`, is the preceding dependency
checkpoint. Public readback verified all 17 changed files and preserved all
354 working files. PR #1 remains draft against `develop`.

Its source run `41767625772` and security run `41767625840` fail on an
observability type/function name collision (`collectorURLs`). Tests and
native/cross builds cannot compile; Staticcheck and CodeQL cannot complete
analysis, and gosec/govulncheck are not reached. The following source
correction names the parser `parseCollectorURLs`; it requires a fresh hosted
compilation and analysis. Do not carry forward an earlier security pass.

The following source also enforces Python's shared 16,384-node I-JSON
budget, with exact boundary fixtures, and excludes semantic prereleases
from the stable update channel even when feed metadata labels them stable.
These are intentional contract corrections, separate from behavior-preserving
Sonar refactors. Their added tests have not run under the local heavy-work hold.

SDK run `41767625933` passes Go dependency-lock reproducibility. All three
Go SDK lanes then fail at that same compilation error. All nine TypeScript
lanes successfully generate identical npm locks and stop at the required
missing-reviewed-lock gate. All 15 Python runtime lanes generate native
wheel-only locks and stop at that gate, including Python 3.15. These resolver
results are not SDK tests. The following correction commits the exact
reviewed npm/runtime locks with artifact, input and hash provenance.

The three Python static jobs install the reviewed toolchain and pass the lock
helper tests. Ruff, mypy, Bandit and lock audit have completed failures;
coverage is interrupted when the jobs exceed their 30-minute limit. Job
logs are unavailable through the connector and the browser cannot recover
the analyzer diagnostics. The following CI correction preserves bounded
analyzer reports before coverage, records test progress and uses explicit
SDK mypy/coverage configuration. It does not guess source fixes, skip tests,
or lower a gate. The SDK aggregate fails.

Sonar run `41767625773` uploads analysis and reaches successful Compute
Engine completion, but fails the strict gate. The retained artifact reports
0.0% coverage after the Go compilation error, 0.2% duplication, 17 open
issues and zero hotspots. Required new-code metrics are absent and fail
closed. One Python compatibility-version warning appears in scanner and
analysis warnings. Its analysis revision is the PR synthetic merge
`45fe4e1d7c099d2be2022e910bd0666c9442ebc3`, whose tree matches the
public candidate. The following source corrections address the actual
reported literals, complexity, empty no-op rationale and SDK diagnostics;
it declares the actual Python 3.11–3.15 matrix for analyzer qualification.
Only a fresh analysis can establish their disposition. Retain
`ci-4c703c8-source-security.json` and `ci-4c703c8-sonar.json` with the exact
head/tree and phase distinctions. The hosted correction remains unqualified
until its own checks complete.

Dependency Review `41767625888` passes. WireMock run `41767625763` verifies
the pinned JAR on all three platforms, then fails suite compilation at the
same Go collision; it is not cancelled and establishes no provider-suite
pass. Retain `ci-4c703c84-wiremock-dependency-review.json`.

### Preceding SDK and telemetry checkpoint

Checkpoint `333a1c7e93cd8be93310f86633549fdc6a6891c9`, tree
`55e43daf8fa1818376f64bcdfeea7f7a4d0a7099`, publishes the SDK,
observability and update-awareness source. Public readback verified all 89
changed files and preserved all 349 working files; the guarded mixed reset
brought the checkout forward cleanly. Draft PR #1 targets `develop`. The
receipts below identify their exact source, rather than implying that an
older analysis qualifies the current feature-branch head.

SDK run `41765210485` generated and verified Go dependency metadata, then
correctly failed because those lock changes were not committed. Compilation
also caught an activation telemetry import cycle. TypeScript failed before
npm because its setup-node commit did not exist. Python 3.11–3.14 generated
12 native-platform resolver artifacts and failed the missing-reviewed-lock
gate; Python 3.15 failed resolution because Bandit's PyYAML dependency had
no eligible wheel. None of this establishes SDK runtime qualification.
Retain `checkpoint35-sdk-locks/receipt.json` and its Python manifest.

The working correction applies exactly the reviewed Go metadata and three
Python 3.14 static-tool locks with source/artifact/hash provenance. It moves
activation instrumentation above the platform primitives, fixes the official
setup-node pin, separates the analyzer interpreter from the 3.11–3.15 runtime
matrix, and excludes terminal-launched MCP servers from automatic advisory
update checks. Missing npm/runtime-test locks remain hard failures until
actual hosted output is reviewed and committed. No source distributions,
ignored findings or weakened coverage gates supply a bypass.

The preceding `19cdf833206708a99aee142f4dc7f5b335be5dbb` has tree
`373f28ab40c92ce7022718e8ff25e21957f59526` and the completed Sonar
receipt below. PR #1's base is the empty initial repository at
`faf8e0242847d9a0026139762d6558ecd39debac`. Do not fabricate a Go baseline
or assume that two analyses run now establish a historical new-code boundary.
The private proposed baseline patch is unmergeable until the pinned server's
actual API and correct initial-analysis semantics are demonstrated.

Sonar run `41762521270` successfully reached HTTP readiness, scanner upload,
Compute Engine analysis and all six revision-bound server readbacks. The sole
published container port was `9000/tcp` at `127.0.0.1:44533`; actual Java PID 1
limits were nofile 131072 and nproc 8192, soft and hard. Scanner warnings,
analysis warnings and hotspots were zero. The quality gate failed: Sonar
coverage 79.4%, Go profile 6248/7306 statements (85.518751711%), three open
code smells, and absent new-code coverage/duplication metrics on this fresh
project. Retain `ci-19cdf83-sonar.json`; scanner exit 3 followed a successful
analysis and expresses a quality-gate failure. It was not a scanner crash.

The matching `19cdf83` source run `41762521195` passed Linux normal/race
checks, vet and bounded fuzzing, plus native macOS/Windows and all declared
cross-builds. It failed cyclomatic complexity at `mcp.New` (11 versus 10).
The source-profile job also failed, but its sanitized log does not establish
an attributable unit-test/vet cause; do not call it coverage-only. Security
run `41762521288` passed Staticcheck, govulncheck and CodeQL and failed eight
unsuppressed gosec findings. Dependency Review and native WireMock passed.
Retain `ci-19cdf83-native-security.json`. Those receipts do not qualify the
following source candidate, including its extracted MCP claim-filter helper.

Checkpoint `333a1c7` addresses those three code smells, adds official Go,
TypeScript and Python SDKs, and implements configurable OTLP observability
and release awareness. SDK error/DTO/limit contracts and operational policy
have independent source reviews. Official MCP stdio fixtures and native SDK
qualification belong to the hosted matrix. No source review establishes a
compilation, runtime, coverage, interoperability or security pass.

The generated exporter module closure and Python static-tool hashes have
hosted resolver evidence; their committed reproducibility remains unpassed.
The npm and per-platform/interpreter runtime-test locks require hosted
resolution and review. The SDK workflow retains
actual generated lock evidence and fails until reviewed locks are committed;
normal qualification uses read-only Go modules and hash-checked Python/npm
locks. Do not invent hashes or call this bootstrap a passing dependency gate.

Interactive release awareness is an advisory, timeout-bounded notice with a
canonical upgrade URL; it reads no answer, installs nothing and continues on
check failure. It is disabled for automatic SDK/non-TTY use. Explicit
`update check --json` exposes structured results. A configured local minimum
version is enforced before telemetry credentials, provider access, operational
receipts and workload effects; recovery/inspection commands remain available.
Version comparison is not proof of executable authenticity.

Local heavy tests, builds, scans, package/module resolution, Java and Docker
remain held for shared-host coordination. Hosted CI is the qualification
route. Use synthetic fixtures and preserve the original infrastructure,
credentials and production/release prohibitions. Keep qualification/status
records candid while product guides describe the complete v1 contract.

## Resumption and retained scope

The user resumed CircularQuay delivery on 4 November 2026. The earlier pause
for BAU priority is revoked for this isolated project. Preserve the saved
source/security patch and existing task ownership. No BAU service,
operational credential, system/service installation, production deployment,
paid purchase, main merge, or release tag is authorized by the resumption.

The expanded v1 scope is published in the README and managed-service contract:
one-command session bootstrap, verified repository/tree configuration,
authenticated enterprise discovery, Azure Pipelines/GitHub Actions/GitLab CI
workload identity, complete overrides/CLI CRUD, optional MCP management and
delegation, and managed identity-to-age-key access with authorized UI/API.
The README and product/design/operator documents describe the final North Star;
actual implementation and qualification belong in status/evidence records.

## Published landing-page checkpoint

Checkpoint `d801d479ac68bdb3866beb0259f7a475286f1cfe` has tree
`a55a16f1bdcbc2c69918a199d9a1d2a780b9a9bd`. It publishes the 112-line
landing README and linked guides, preserving the author's opening story,
optional IdP/MCP wording, dual-platform quick starts, complete command examples
and continuous comparison table. Public readback verified all 17 selected
files; a guarded mixed reset preserved all 299 fenced working files, including
concurrent runtime work excluded from this documentation checkpoint.
GitHub rendering was inspected. Retain `checkpoint29-publication-readback.json`
and `readme-d801d47-rendered.png`.

GitHub's About description and 12 relevant topics were saved through its native
editor and verified through the repository API. The description begins
"Get the right secrets to the right apps" and states existing-provider delivery
with optional identity and MCP. No separate website was invented. This metadata
change is not a branch-rule or security-check enforcement change.

Exact-d801 source run `41749033589` passed all native jobs, normal/race
tests, vet, bounded fuzzing, both complexity checks and all 26 cross-builds.
The source auditor's coverage gate remained failed at 6,125/7,173 statements
(85.38966%) versus 100%. Security run `41749033545` passed CodeQL with zero
annotations and failed only the gosec step with eight findings (G204 ×6,
G103 ×2), zero suppressions. This is an analysis-gate result, not a newly
established exploit. The synthetic merge tree matches this documentation
checkpoint; none of these results qualifies the following source candidate.
Retain `ci-d801d47-native-cross.json`.

## Published integration checkpoint

Checkpoint `98c9caa7c5ed6b20f28cec720c73ca4939c4e63c` has tree
`01e7251bdb02b10b5f6304c0a4dfcf5f277442af`. Public readback verified all
43 changed files and all 299 working files; the checkout was clean. PR #1
remains draft against `develop`. WireMock run `41753111836` passed all three
native platforms on the matching PR merge tree. Source run `41753111878`
passed macOS and all 26 cross-builds, but Windows exposed a trust-store schema
path mismatch and Linux found cyclomatic complexity 11. Coverage remained
6,246/7,303 statements (85.52650%). Security still reports eight unsuppressed
gosec findings. Retain the three `ci-98c9caa-*` receipts.

Sonar run `41753111704` failed its file-descriptor prerequisite before analysis;
cleanup incorrectly attempted Compose before its environment file existed.
Dependency Review run `41753111900` flagged three real upstream go-jose 4.1.4
advisories; 4.1.6 is the minimal combined fix. The following candidate corrects
schema grammar, extracts a cohesive claim-filter helper, updates that dependency
without changing its RSA bound, and repairs hosted prerequisite/cleanup paths.
It does not waive any gate. Go metadata is retained under
`checkpoint31-dependency-metadata`; no module ZIP or local Go resolver was run.

The user additionally requires official Go, TypeScript and Python SDKs with
registry/operator release directions, and v1 configurable remote observability,
metrics, traces and diagnostics. Enterprise identity lifecycle, access review,
audit retention/export, interoperability and long-term operational requirements
are part of the acceptance scope. ISO 27001 support is a technical-control and
evidence goal, not a certification claim. Managed UI support follows the runtime
capabilities; no running managed service or SDK publication is implied.

## Integration source follow-up

The next reviewed candidate adds per-authority protected, digest-pinned private
CA roots for JWKS; the actual tagged WireMock process suite with Linux/macOS/
Windows CI; a fresh hosted Sonar diagnostic producer; and dependency review.
The linked integration contracts cover cloud providers/emulators, Keycloak,
SPIFFE/SPIRE, hosting adapters and a separate optional compatibility gateway.
Only fields with runtime enforcement enter the v1 schema. Managed-service,
gateway, LocalStack and real Keycloak/OpenBao service qualification remain
separate, incomplete work; no design is a passing test receipt.

Independent source reviews found and corrected native path/ACL, schema parity,
test-oracle, process-reaping, cancellation and strict Sonar readback defects
before publication. Source review is not compilation or execution. Exact-next
hosted normal/race/native/security/integration/Sonar results are required.
Local heavy work remains held. An integration worker nevertheless performed
two compile-only/vet batches without an exact source/environment fence; no test
bodies, WireMock, Java or Docker ran, and the root stopped further execution.
Retain `wiremock-compile-only-execution-20261104.json`; it is not qualification.

## Completed 060639 hosted receipt

Checkpoint `060639f36c1c7309769f34e3b88812e42d02cabf` has tree
`aef7be1f1b11f112ed04120ed58e61b17f864e46`; public readback verified
all seven changed files and all 274 working files. Hosted source run
`41744753747` passed native Linux/macOS/Windows, normal/race tests, vet,
bounded fuzzing, both complexity checks, all 26 cross-builds and the aggregate
checks. The synthetic merge tree matches. The source auditor passed unit tests
and vet; coverage alone failed at 6,125/7,173 statements (85.38966%) versus 100%.

Security run `41744754352` passed Staticcheck with all checks, govulncheck and
CodeQL with zero annotations. Gosec `-nosec` failed eight findings (G204 ×6,
G103 ×2), with zero suppressions. Retain
`.local/qualification/ci-060639f-native-cross.json`. No exact060 Sonar,
managed security audit, live-provider or production qualification is claimed.
The cloud-response changes below now have this bounded execution receipt.

## Completed 018fea hosted receipt

Checkpoint `018fea744d8dfb959072e1e95047c966b80ea011` has tree
`ebf49136ab6f5b248214e0353a7dae53a307a14d`; public readback verified
all four changed files and all 273 working files. Source run `41742555776`
passed native Linux/macOS/Windows, normal/race tests, vet, bounded fuzzing,
both complexity checks, all 26 cross-builds and the stable aggregate. Synthetic
merge `d6129d9648a41b37126016544d73f4d8c4bf2e89` has the same tree.
The prior Azure-fixture compilation failure is closed by actual execution.

The source auditor's unit tests, vet, probes, helper integrity and source fences
passed. Coverage is its only required source-profile failure: 6,076/7,129
statements (85.22934%), below 100%. Security run `41742555781` passed
Staticcheck with all checks, govulncheck and CodeQL with zero annotations.
Gosec `-nosec` still failed eight findings (G204 ×6, G103 ×2), zero
suppressions. Retain `.local/qualification/ci-018fea7-native-cross.json`.
No exact018 Sonar or managed security diff review is claimed. This is a
bounded development receipt and does not qualify a release or production.

## Cloud response identity follow-up

The next source candidate binds returned provider identity before field
selection or value release. AWS Secrets Manager and Parameter Store require
the exact declared canonical ARN. Azure requires a non-nil response ID with
the configured HTTPS origin, exact secret path and a nonempty version; DNS
case and default port 443 are equivalent, while distinct IPv6 addresses and
ports remain distinct. Ambiguous paths, encoded components, empty query or
fragment delimiters and scoped hostnames fail closed. Azure source-origin
validation also precedes credential selection and SDK construction.

GCP now binds the returned project/secret base even without a pinned version,
requires concrete version evidence and retains CRC and exact pinned-version
checks. Partial ARN, project-ID/number or secret-name aliases are not inferred.
This is response-integrity hardening of an authenticated provider boundary,
not a demonstrated hostile-provider exploit or a claim that a provider cannot
lie about its data. Canonicalization needs live-provider qualification before
any broader alias support is claimed.

Synthetic SDK/protobuf cases include positive neighbors, missing and substituted
identities, URI ambiguity, both IPv6 address and port mismatch, GCP unpinned
substitution and a pre-credential Azure rejection route. Existing version/CRC
fixtures retain correct identity evidence so they still reach their intended
boundaries. Pinned formatter parsing, root review and an independent source-only
review are separate from execution. Exact-candidate hosted tests/static/native
checks remain required; no local heavy test or scan is claimed. Retain
`checkpoint28-transfer.json` and `checkpoint28-cloud-review.md`.

## Retained dec332 test failure

Published test-only checkpoint `dec332ad1e60cde1f1400e9eb87c08a49aaccd1e`
has tree `d5f7546f14f42c39e4b65b11214f391993b33b06`; all four changed
files and all 273 working files were read back and preserved. Hosted source
run `41741675497` and security run `41741675310` caught the same compilation
error in both new Azure fixtures: the SDK transport interface requires `Do`,
not only `RoundTrip`. Linux tests and Staticcheck failed; govulncheck/gosec
were not reached in the security job. CodeQL passed with zero annotations.
The manual source review missed this interface mismatch and is not a passing
execution receipt. Do not represent this revision as coverage-only failure.

Retain `.local/qualification/ci-dec332a-native-cross.json`, including its
at-collection pending jobs and the checks not reached after compilation failed.

The next correction supplies an `http.Client` around each memory transport,
with a one-second fixture timeout. The pinned SDK interface and standard
client method were checked directly. Local heavy qualification remains held;
formatter parsing and source review do not establish compilation or tests.
A fresh exact-candidate hosted run must confirm correction. The production
adapter, policy/schema, workflow and security gates are unchanged.

## Completed 65be10 hosted receipt

Published checkpoint `65be10a48257b1207df411b857b2a67c68280ee1` has tree
`c756b0af4ed3e5bbc7009e7299d704ea17d0d18c`. Public readback verified
all 11 changed files and all 271 working files; draft PR #1 still targets
`develop`. Hosted source run `41739246845` passed native Linux/macOS/Windows,
normal/race tests, vet, bounded fuzzing, both complexity checks and all 26
cross-builds. The new aggregate check also passed. The source auditor accepts
the tool/helper manifest and passes unit tests, vet, formatting and source
identity. Its only required source-profile failure is coverage at 6,054/7,129
statements (84.92075%). Security run `41739247007` passed Staticcheck with all
checks, govulncheck and CodeQL with zero annotations. Gosec `-nosec` still
fails eight findings (G204 ×6, G103 ×2), with zero suppressions. Retain
`.local/qualification/ci-65be10a-native-cross.json`. No exact65 Sonar or
managed diff scan is claimed.

This candidate splits the two preceding cyclomatic violations without raising
the ceiling. Hosted Staticcheck now selects all checks; gosec ignores nosec
annotations; pinned gocognit uses exact recursion and rejects diagnostics.
The stable `Cross-build matrix` job requires the whole build matrix to succeed.
Workflow configuration alone does not enforce GitHub protections. Saving the
prepared additional required checks and all-severity CodeQL rules still waits
for GitHub's user-completed `Confirm access` authentication.

Local heavy tests/scans/builds remain held for BAU disk reserve. Three exactly
identified, completed CircularQuay Go build caches were cleared with pinned
`go clean -cache`, removing 3,018,936,320 allocated bytes (2.81 GiB). Every
tracked source byte, module/toolchain cache and nonreproducible qualification
artifact was preserved. At 2026-11-03 23:32:59 UTC, filesystem free space was
35,657,736,192 bytes; the reported 70 GiB reserve was still short by
39,504,191,488 bytes. Other host cleanup is not attributed to CircularQuay.
Retain `.local/qualification/resource-hold-latest-20261104.json` and its
exact ownership, cleanup and source-integrity receipts. A later 23:41:09 UTC
readback measured 618,975,252,480 available bytes (576.47 GiB), above the
reserve; the much larger external host recovery is not attributed to this
project. The bounded additional inventory identifies shared global caches,
active pinned project dependencies and completed task cache portions without
changing them. Retain `resource-reserve-readback-20261104.json` and
`resource-go-cache-additional-20261104.json`. Local heavy work remains held
for shared-resource coordination; hosted CI consumes separate build resources.
No new local heavy qualification is claimed.

## Previous revision evidence

Previous published checkpoint `6b0888307ebb15ea3ca13a4da42d4056102fe1e4`
has tree `35781cc4c4eca892e156e7cbc19abca8a0f82507`. Public readback verified
all nine changed files and all 268 working files. These receipts qualify only
that revision, not later source edits.

Source workflow `41733653552` passed Linux/macOS native tests and all 26
cross-builds. Windows job `126257137168` still fails
`TestToolInputOverlapAndPostReadNameFence` at protected-file activation;
the prior environment-path failure is absent. Source assurance accepts its
manifest and all Git/Go/gofmt probes, then fails `unit_tests` (exit 1) and
coverage at 5,810/7,055 statements (82.35294%). Do not call this coverage-only.
A separate restricted-PATH reproduction isolates the signed-repository
fixture's missing `ssh-keygen` dependency. CodeQL passes with zero annotations;
actual Linux gosec still reports eight unsuppressed findings. Retain
`.local/qualification/ci-6b08883-native-cross.json` and
`source-profile-helper-closure-20261104/before.json`.

Exact-head direct Sonar CE `707d4da6-6bd9-4042-b4a5-56a854f3e8f0` succeeded
with analysis `d936850a-ee82-4581-8a03-429f94e20fb6` and the requested SCM
revision. All open issues, including S1192, are zero. Gate ERROR is solely
coverage: 78.3% overall / 79.7% new versus the unchanged 100% gate. Fresh Go
union statements match 5,953/7,055 (84.37987%). Scanner warnings are zero.
Current-analysis and hotspot-list APIs still return 403; do not claim an
independent latest-analysis fence or hotspot-list clearance. Retain the
prior Nov10 token-expiry warning and `.local/sonar/summary-6b0888307ebb.txt`.
This diagnostic does not satisfy the auditor's runtime-binding gate.

Fresh managed scan `e53999d5-b8a8-4190-95a4-2b40c48bd43d` sealed all nine
files of `2d1f100` to `6b088830`, with actual populated and closed inventory,
zero candidates/findings and no exclusions. Focused assurance validation
passed; CLI validation exceeded its observation window and is not counted.
An earlier sealed scan with zero populated review rows is invalid qualification
evidence, regardless of its zero findings or semantic completeness claim.
Canonical evidence remains under the task's Codex Security scan store. This
is diff evidence, not a complete independent production audit.

Previous published checkpoint `bfaa66cc116093588dd1053a301bdf90a8ecef28`
has tree `0afc2f5f7fcaf00f1547b29f849540cf78cfc89f`; its native Linux/macOS/
Windows, all 26 cross-builds and CodeQL passed. Its synthetic merge
`3be5764a6b551982bfa4962fc7c35a8c7000df2f` has the same tree. Those results
do not qualify 2d1f or later edits. Retain `ci-bfaa66-native-cross.json`.

The exact-head scanner uploaded its report with the requested SCM revision.
CE `f8564a8c-d33e-470a-a829-a9b2558305dd` succeeded and returned analysis
`8a844c31-e83c-4be8-ba85-2bb12be93b69`. Gate ERROR reports 79.9% overall /
82.4% new coverage and four open code smells (three argument-count findings
and one shadowed builtin). Fresh statements are 5,554/6,441 (86.22885%).
The current-analysis and hotspot-list APIs returned HTTP 403; do not claim
an independent latest-server-analysis fence or hotspot-list clearance from
these receipts. The scanner's hotspot metric is zero. The separate old audit
route also failed tool checks and protected token reading because the shared
cache ancestor is group-writable. No permission was relaxed; direct scanning
used only the existing task analysis credential in process environment.
Retain `.local/sonar/summary-bfaa66cc1160.txt`, its artifact hashes, and the
prior 2026-11-10 token-expiry warning. The four source issues have readable fixes guardedly integrated from a
separate bounded checkout; only a fresh exact-head scan can confirm closure.

Managed diff scan `56553a87-7692-4deb-9432-d0f716652dc3` sealed all 37 files
of `97b757a` → `bfaa66c` with zero findings/candidates and complete coverage.
Pinned Go 1.27.3 focused DPoP/identity/MCP/policy validation passed. Artifacts
are under `.codex/state/plugins/codex-security/scans/repo/` in the task host's
Codex home; this is a complete diff review, not a full production audit.

Frozen local checkpoint21 adds optional named-authority DPoP, strict issuer
`cnf.jkt` binding, required persistent nonce/replay state and a request-local
SDK verifier handoff. Whole-module normal/race tests, vet, Staticcheck with all
checks, formatting, module checks, both complexity ceilings, actionlint,
13 Windows and four Darwin test compilations passed with identical source
manifests. Coverage is 5,554/6,441 (86.22885%), below 100%. Retain
`.local/qualification/checkpoint21-20261104/` and the separate failed gosec
receipts; exact committed native/Sonar/security qualification is separate.

The profile advertises its supported algorithms and all-authority requirement
through protected-resource metadata, checks the actual request host/path and
any absolute-form scheme, and never derives its trust origin from forwarded
headers. A reproduced alias/scheme/port gap failed before the fix and passes
afterward. Signed issuer-to-SDK tests prove raw-read actor/deadline/audit
behavior, replay denial, and that a rejected fresh Bearer proof remains usable
under the required DPoP scheme. The cache is single replica/local filesystem,
with required one-use nonces, expiry and fixed global/per-binding capacity.

The final sender-constraint model has independent acceptance/restart witnesses,
mixed token/proof expiries and live-window replay semantics. TLC checked
127,067 distinct states at depth 20; all four private guard mutations produced
actual counterexamples. Earlier weaker models and malformed mutation attempts
are historical evidence, not qualifying results. Retain raw final logs under
`.local/qualification/sender-constraint-final/`. The model is a bounded
abstraction with trusted host/clock/store/crypto assumptions, not exact program
refinement or proof against host administrators.

The first combined capture failed one new policy test's cognitive ceiling.
That test was split into readable helpers and the fresh combined capture
passed; retain `checkpoint21-20261104-before-test-refactor` as historical
failure evidence. Schema review also caught optional-skew decoder tags and
missing cross-field age/skew constraints; regression tests failed before the
fixes and cover both sides of every small-age boundary afterward.

Typed audit-tool inputs have been guardedly integrated into the primary
checkout from a frozen isolated slice. Protected tools and typed rules inputs,
fixed probes, private canonical Git/Go helpers, pre/post rules checks and
source-waiver denials are implemented. An actual tiny synthetic source audit
passed with protected tools under a poisoned ambient PATH. The primary merge
preserved other files and three-way merged RELEASE.md; focused README audit
commands now name the required manifest without changing the author's opening.

Integration review caught CI gofmt help being used as a multiline version
label and an incorrect expected help exit code. The corrected exact workflow
block generates an accepted protected manifest. Rooted bounded helper copies
stay non-executable until complete digest verification. Source-profile Sonar
is explicitly `not_run`, and enterprise Sonar is required failure, until full
scanner/runtime/remote-rules binding exists; the direct diagnostic route does
not satisfy that gate. Full runtime provenance, atomic use and child-process
isolation remain unresolved. Combined checkpoint22 checks and publication
are separate from the earlier isolated receipts; consult actual HEAD/status.

The primary checkout also integrates `flow run` dispatch for GitHub Actions,
GitLab CI and Azure Pipelines environment contexts. Explicit local config
suppresses ambient remote selection; explicit source fields override their
individual environment defaults. Remote retrieval requires matching immutable
commit/digest pins, uses a protected per-run temporary policy and removes it
on return. Tests cover selection, real declared child delivery, error cleanup,
pre/post-fetch cancellation and inherited CI markers. A sanitized stderr
receipt identifies configuration provenance and labels the runner context as
`hint_not_identity`. Child stdout/stderr remains workload-controlled, not
redacted. The README unattended example now supplies both pins. No CI token
verification, mutable/enrolled discovery, later-step export or interactive
login is added by this wrapper. These remain separate v1 requirements.

The first combined checkpoint22 capture failed the new flow integration
test's cognitive ceiling (20 versus 10). Root split the test into cohesive
fixture helpers and retained the failed capture. An expanded offline zizmor
run with the auditor persona also required a rationale for CodeQL's existing
`security-events: write` grant; the permission is now documented in place,
with no scope or threshold change. Fresh combined source/static receipts,
not isolated worker claims, determine publication readiness.

Frozen combined checkpoint22 passed all 31 recorded source/test/static/
format/module/workflow/compile checks with identical before/after source
manifests. It includes whole-module normal/race tests, vet, Staticcheck with
all checks, cyclomatic and exact-recursion cognitive ceilings of 10,
13 Windows and six Darwin package test compilations. Coverage is
5,953/7,055 statements (84.37987%), below the unchanged 100% gate. Final scoped
gosec still fails with eight Linux and nine Windows findings, zero Go errors
and zero suppressions. Offline config-free/no-ignore zizmor auditor-persona
collection now reports zero findings. The exact workflow-generated manifest
and public auditor passed a real tiny synthetic smoke under a poisoned PATH;
unselected tools were explicitly not run. These receipts do not qualify
later native CI, exact-head Sonar, a complete audit, providers or production.
Retain `.local/qualification/checkpoint22-20261104/`,
`checkpoint22-final-static-20261104`, `checkpoint22-workflow-security`, and
`workflow-manifest-auditor-green-20261104`. Publication and exact-revision
readback remain separate guarded steps.

Frozen local checkpoint23 passes all 31 recorded checks with identical source
manifests after the fixture, provisioning and duplicated-label fixes. Native
statement coverage is unchanged at 5,953/7,055 (84.37987%); scoped gosec still
fails eight Linux/nine Windows findings with zero Go errors/suppressions.
No threshold or platform guard was relaxed. Windows/Darwin compilation is
not native proof; the actual next-candidate CI must confirm the fixes.

The old shared writable tool layout was reproduced as `tool_manifest_invalid`
with no coverage. A private copied full Go distribution/Git layout rebuilt
the actual auditor with the copied compiler and passed a real poisoned-PATH
tiny source audit (1/1 fixture statements). These are fixture receipts, not
CircularQuay's coverage gate. Root also checked the pinned setup-go source:
for modern Go it adds PATH and does not export GOROOT. The workflow derives
the canonical distribution from that PATH and passes copied GOROOT only to
children. Selection tests pass with both absent and stale ambient GOROOT.
Offline strict/no-ignore zizmor auditor collection reports zero findings.
Retain `checkpoint23-20261104`, `checkpoint23-static-20261104`,
`checkpoint23-workflow-security`, `audit-runner-provisioning-20261104`, and
`setup-go-path-contract-20261104` under `.local/qualification`.

The host filesystem exhausted space during an isolated worker's extra test.
Root cleared only this checkout's disposable `.local/cache/gobuild` using
pinned `go clean -cache`, freeing about 16 GiB; every working source byte,
module cache and retained receipt was preserved. No unrelated data, service
or system configuration was touched. The subsequent full capture passed.
The intermediate capture with a workflow-only source-fence change is retained
as `checkpoint23-20261104-before-setup-go-path-contract`; it is not the final
combined qualification.

## Exact 20eac0 development receipt

Checkpoint `20eac0a2a99faf5d9f51c126d66ece3db107bf01` has tree
`1871522215dd75f2692b2833333095181d5d1b91`; public readback verified all
15 changed files and all 271 working files. Source workflow `41736690731`
passed normal/race tests, vet and bounded fuzzing; native Windows/macOS and
all 26 cross-builds passed. Linux's job nevertheless failed its cyclomatic
step: `validManifest` and `qualifiedTools.environment` each scored 11 versus
10. They are split into cohesive shape/copy helpers in the following slice;
the ceiling is unchanged and exact later execution must confirm the correction.

The actual source auditor now prepares and verifies its declared private SSH
helper and passes `unit_tests`. Its only source-profile failure is coverage:
6,050/7,125 statements (84.91228%) versus 100%. This does not replace a fresh
Sonar profile or qualify later changes. Security run `41736690751` passed
CodeQL with zero annotations, Staticcheck and govulncheck; gosec still failed
eight unsuppressed findings. Retain
`.local/qualification/ci-20eac0a-native-cross.json`. No exact20 Sonar or managed
security diff scan was run under the local capacity hold.

The next workflow adds a stable `Cross-build matrix` aggregate that passes
only when the entire matrix succeeds. It is a named result suitable for
GitHub-required checks; source configuration alone is not server enforcement.
The branch-rule browser is usable again. Stronger Go-security/CodeQL and all-
severity code-scanning rules are prepared, but GitHub's fresh `Confirm access`
authentication currently blocks saving; the API still reads the old four
required checks. Do not call the prepared change enforced until saved and
read back independently. The visible in-app GitHub tab is retained for user
authentication, with no credentials or codes requested in chat.

## Bounded continuation after the 6b088 receipt

The next source slice declares `ssh-keygen` as an optional fixed native helper
in the v1 audit manifest, distinct from probed analyzers. Its protected original
and private canonical copy are checked around commands; case-insensitive PATH
and GOROOT aliases are removed. No ambient helper directory or invented
version probe is accepted. The source workflow declares the copied dependency.
Schema/parser, drift and private-PATH tests accompany the implementation.

The Windows name-fence fixture now renames its open original aside and creates
a same-size/mode replacement, proving the original descriptor remains readable
and the named identity differs. Production sharing/ACL protections are unchanged.
A separate workload lifecycle fix preserves context cancellation/deadline errors
and bounds waiting for inherited output pipes with a 250ms WaitDelay. This is
not descendant containment or a bound on an arbitrary blocked caller reader or
writer.

The remote workflow mirrors the published analysis requirements with
Staticcheck `-checks=all`, gosec `-nosec`, and pinned gocognit v1.3.0 using
`-exact-recursion -over 10` on native owned directories. Nonempty diagnostic
output, including a type-aware fallback warning, fails that check. Workflow
configuration is not evidence of a passed execution or GitHub enforcement.

A BAU capacity notice arrived during this slice. Local heavy tests, scanners,
builds, downloads and cache copies were held; no task-owned Go/test/scanner
process remained at the resource readback. Free space was about 11.9GiB versus
the coordinator's 70GiB recovery reserve. The read-only inventory retained
exact ownership/size evidence under `.local/qualification/resource-hold-20261104.json`;
no cross-project cleanup, service or permission change occurred. Existing
source and nonreproducible evidence were preserved. No new complete local
qualification is claimed for this slice. GitHub-hosted CI can independently
qualify a published candidate without consuming Foo build storage; Sonar and
other Foo-heavy qualification remain subject to the capacity hold.

The earlier test-only candidate guardedly imports two synthetic provider
contract suites from `/home/user/src/2026-11-04/circularquay-provider-contracts`.
It exercises real AWS/Azure generated SDKs over memory transports and GCP
response evidence after the identity boundary. Positive adjacent cases prove
that stage/selector/version mismatches, missing payloads, denied/missing/error
responses and malformed JSON cannot disclose any returned value or version.
Failures require a complete zero `model.Value` and marker-free errors. Azure's
503 fixture disables SDK retries and preserves the current conservative
trust-failure classification. Every fixture is synthetic and makes no network
request. No provider adapter, policy/schema, permission or quality gate changes.

Pinned gofmt parsing and independent source-only review are separate from
execution. Local heavy validation remains held; exact-candidate hosted tests,
coverage, static checks and native runs are required before these assertions
can be reported as passing. Retain `checkpoint26-transfer.json` and
`checkpoint26-provider-review.md`; the transfer receipt records original worker
hashes, while root review records the final primary hashes and improvements.

## Latest bounded documentation result

The documentation-only commit `1d89ffbf65a03e4e6ef8f146944416b97356e38c`
is published and byte-verified on `feature/initial-implementation`. It adds the
standalone [managed key service](managed-key-service.md), a concise README link,
and architecture/operator/MCP/recovery references. The service uses an existing
trusted identity provider and explicit key mappings; scoped UI/API administration
manages allowed principals, grants, windows, changes, and revocation. Existing
paid Netlify/Cloudflare capabilities take preference over a custom UI; any missing
key-specific UI is hosted there. There is no email/onboarding suite. Distribution
cannot revoke already copied private keys; local custody and remote unlocking
remain explicitly distinct models. Financial scenarios are assumptions, not
chosen prices, benchmarks, traction, or deployment approval.

The README repository/tree bootstrap and CI/discovery scenario was already
published at `7b68c6116786779392e01024c102093c33254a0d`. The bounded product
presentation is a final v1 North Star, while actual managed-service status is
separate in [qualification](qualification.md#managed-service-design-status).
The updated user-supplied UIF diagram presents the same architecture without
promoting its old operational snapshot into a current BAU claim. Its artifact
and visual/structural receipts are retained in the private continuation record.
No implementation, service, credential, paid subscription, or BAU deployment
was changed. All existing uncommitted source was preserved during that documentation-only
turn. The later user resumption above reactivates isolated development and
qualification only.

## Start here

- Project/module: `CircularQuay` / `github.com/maldous/CircularQuay`. The app,
  binary, release-asset prefix, and skill are `cquay` ("see-key");
  `cquay-audit` is separate.
- Development branch: `feature/initial-implementation`. This continuation
  includes `4886fc8660706909376f3543ad7cab186d9256b9`; use `git rev-parse HEAD`
  and `git status` to establish the actual checkpoint and subsequent changes.
- Read [CONTRIBUTING.md](../CONTRIBUTING.md), [SECURITY.md](../SECURITY.md),
  the durable [implementation plan](implementation-plan.md), and
  [RELEASE.md](../RELEASE.md) before editing. The plan is the scope contract;
  this file records current evidence and continuation risk.
- Use Gitflow: `main` is released/stable, `develop` integrates, `feature/*`
  develops, `release/*` stabilizes, and `hotfix/*` starts from `main`. Do not
  merge to `main`, tag, publish, or deploy until the required gates genuinely
  pass.

## Isolation and authorization

Work only in an isolated developer checkout with synthetic fixtures. Do not
install CircularQuay into service or system paths, restart services, alter
infrastructure or existing credentials, inspect operational secret values or
shell-profile credentials, or interfere with OffMenu BAU recovery. Do not
create broad persistent credentials to bypass an unavailable GitHub permission.

The user authorized reversible development and free test tooling, not paid
resources, production deployments, or a universal credential daemon. Keep
publishing, account, registry, and operator procedures in `RELEASE.md`; keep
end-user/product material in `README.md`.

## Current implementation checkpoint

This checkpoint includes JSON policy CRUD and safe doctor
formatting; protected metadata/state and path-ownership hardening; strict
source, snapshot, target, grant, and repository contracts; repository quota
work; provider contract tests; and extended release cross-build configuration.
The expected policy/CLI contract is version 1 and the development version is
`1.0.0-dev`.

The CLI supports only policy operations `list`, `status`, `run`, and `read`.
`run` authorizes the enrolled workload; read keys restrict only `read`.
`cancel` is request-context cancellation, not a policy permission. Configuration
editing is JSON-only, protected, bounded, strict-decoded, validated, and
atomically replaced under a local lock plus digest recheck. YAML remains a
loadable policy format but is intentionally not mutated by the editor.

The v1 identity direction does not add a Keycloak dependency. Where a custom
API is supported, trust an existing authorization server/IdP directly (for
example Entra ID or Auth0) through named trust/configuration scope, role,
client, subject, resource, context, and grant declarations. Do not hard-code
development/production identity assumptions. The integrated optional DPoP
profile requires issuer-bound keys and persistent single-replica nonce/replay
state; native and issuer/client interoperability remain unqualified. mTLS,
private-key JWT and offline-artifact binding are not implemented or qualified.

Grant time windows and the six reviewed remediation patches are implemented
with focused synthetic tests. They are not qualified capabilities until fresh
combined, full, native, and Sonar checks pass.

The checkpoint6 source batch added named RFC 9068 and Entra v2 API authorities;
configurable all-of scope and roles; exact grants and current-trust
fingerprints; a strict shared legacy verifier; immutable GitHub bootstrap only
(full SHA plus trusted SHA-256); signed-repository CLI/MCP runtime with
`FetchCurrent` offline/fallback guards and hardened Git hashing/fsck; strict
snapshot validation and full-snapshot MCP reads; durable fallback pre-read
fencing across restart; and the Azure workload-identity transport constructor
test. It also includes Windows fixture repairs. Mutable branch/enrolled discovery and an authentication UI remain
unimplemented. Immutable `flow run` dispatch is integrated with the explicit
limitations above. This batch remains unqualified until
a combined candidate passes fresh native, Sonar, and independent checks.

README ordering was directly revised by the user and remains preserved.
The Android rendering investigation found the raw Markdown, fences, HTML, and
linked SVGs clean; browser rendering worked. Treat that as an app/cache issue
unless new evidence reproduces a source-side rendering defect. Do not reorder
the README without a user-directed reason.

## Recent local evidence

Checkpoint19 receipts under `.local/qualification/checkpoint19-20261104/`
record the 24 listed source/test/static/format/module/workflow/compile
checks passing with identical source manifests. Coverage is 5,224/6,082
(85.89280%), below the unchanged gate. Real synthetic children prove oversized
stdout denial, cancellation after startup and finite inherited-pipe waits;
reader-only `io.Copy` regressions exercise the former promoted `ReadFrom`
bypass. These mechanisms do not contain descendants or sandbox source code.
Repository tests prove invalid accepted/config/object/materialized state denies
before release, and cancellation returns no release while cleaning temporary
verification state. Linux seams verify fixed argv, empty environment and
failure/empty/oversized-output preservation of the prior projection without
starting a service. Native execution and exact-candidate Sonar remain separate.
The separate scoped gosec receipt under
`.local/qualification/checkpoint19-static-20261104/` fails: Linux has eight
findings (G204 ×6, G103 ×2), Windows has nine (G204 ×5, G103 ×4), with zero Go
errors and zero suppressions. The Linux count decreased because two systemd
invocations now share one execution boundary; it is not a cleared risk or
weakened rule set. Both scans preserve identical source manifests.

Checkpoint18 receipts under `.local/qualification/checkpoint18-20261104/`
record whole-repository normal/race tests, vet, Staticcheck, formatting,
tidy-diff/module verification, both complexity ceilings of 10, actionlint and
12 Windows test-compilation checks passing with identical source manifests.
Merged coverage is 5,133/6,048 statements (84.87103%), below the unchanged 100%
gate. The real signed-repository regression verifies protected offline Git
temporary state without fetch credentials and its cleanup after return.
Native Windows execution and exact-candidate Sonar remain separate evidence.

Checkpoint17 receipts under `.local/qualification/checkpoint17-20261104/`
record Go 1.27.3 whole-module normal/race tests, vet, Staticcheck, formatting,
tidy-diff/module verification, both complexity ceilings of 10, actionlint and
12 Windows test-compilation checks passing with identical before/after source
manifests. Coverage is 5,114/6,022 statements (84.92195%), below the unchanged
100% gate. The intermediate checkpoint15 test-helper cognitive-complexity
failure is retained; the refactored helper passes checkpoint17. These local
results do not establish native Windows, Sonar or independent qualification.

Checkpoint10 receipts are retained under
`.local/qualification/checkpoint10-20261104/`. Whole-repository normal/race tests,
vet, Staticcheck, formatting, tidy-diff/module verification, cyclomatic and
exact-recursion cognitive ceilings of 10, actionlint, and 12 Windows test
compilations passed with identical before/after source manifests. Merged
whole-module statement coverage is 4,899/5,765 (84.9783%), below 100%. An earlier
checkpoint8 diagnostic-helper complexity failure is retained; the refactored
helper passed checkpoint9 without changing the ceiling. Checkpoint10 includes
explicit DPAPI input/output bounds: Windows gosec no longer reports the two
length-conversion findings. Its nine remaining G103/G204 findings are retained;
they are distinct from the Linux lane's nine findings. Windows/macOS cross-static
checks do not establish native execution.

Historical checkpoint6 and checkpoint7 receipts remain retained. The earlier
checkpoint7 source receipt is
`.local/qualification/checkpoint7-20261104/`, with identical before/after
manifests for Go, JSON, workflows, and canonical model fences. These are local
evidence, not release evidence.
Using a fresh owner-only temporary directory under root-owned sticky `/tmp`,
Go 1.27.3 normal and race tests, vet, gofmt, `go mod tidy -diff`, `go mod
verify`, Staticcheck 2026.2.2, `gocyclo -over 10`, exact-recursion `gocognit
-over 10`, and actionlint passed. Compile-only Windows checks passed for the 12
affected packages. The checkpoint7 whole-module merged `-coverpkg` profile is
4,881/5,744 (84.97562674%), below the unchanged 100% gate. It succeeds older
checkpoint6 local coverage (4,787/5,742, 83.3682%) and is distinct from the
41a8 production-assurance CI profile (4,828/5,742, 84.0822%); none passes the
gate.
Pinned gitleaks v8.30.1 scanned a captured 216-file inventory before this receipt update (SHA-256
`0caf87f382e74e55bee6d7ef3f4f7ac9c839bb4d17183764ac3ff0e29752e74b`)
with zero findings. This remains bounded local working-tree evidence, not a
history, artifact, credential, or production certification. The old copied
213-file scan and its incorrect initial invocation remain historical evidence.
TLC also completed the durable-fence model with trusted-store and lock
assumptions: 14,818,232 generated states, 76,360 distinct states, depth 20,
an empty queue, and no error. It does not qualify native Windows permission
failure behavior, full implementation conformance, or host recovery.
The `internal/identity` package separately passed normal/race/vet and both
complexity checks with 183/183 statements covered; that package receipt does
not qualify the repository as a whole.
The same checks correctly reject a `.local` temporary directory with an unsafe
`0775` ancestor; do not relax that protection to make tests pass.

After source changes, run the applicable checks from protected temporary
storage and retain exact command, tool version, revision/tree state, and result.
The source-profile 100% owned-code coverage gate remains unchanged and is not
met by checkpoint19 or the last published CI receipt.

## Qualification and release blockers

Published candidate `54fb84a2d3ccd65d0cc7c3d7a780c7c5daa40507` has tree
`325e75f2016bfd055a1289e4203c583f062c29bc`. Public readback verified all 21
changed files and all 236 working files. Source run 41713357341 tested a
synthetic merge with the same public tree: Linux/Windows and all 26
cross-builds passed. macOS failed the audit bootstrap fixture because its
private temporary path included a system symlink; other listed packages
passed. Do not misclassify that assertion as a dependency/setup failure.
The captured checkpoint20 changes only that test fixture to use its physical
path, preserving production checks. Audit-package normal/race/vet/Staticcheck,
format/complexity and three Darwin/Windows test compilations passed with
unchanged source manifests. All production inputs match checkpoint19; native
macOS revalidation remains required. Retain
`.local/qualification/checkpoint20-20261104/`.

Security run 41713357268 passed CodeQL. Actual Linux gosec reported eight
findings; the private receipt preserves and corrects an earlier unmeasured
carried-forward count. Sonar CE `e6af4d51-c9be-4860-844d-89715fd6b4c5` and
current analysis `c8623fc7-2362-4019-9ff8-88546a7de9a6` have exact revision
fences and zero issues/smells/bugs/vulnerabilities/hotspots/duplication, but gate
ERROR at 79.4% overall / 81.8% new coverage. Keep the Nov10 token warning.
Go coverage remains 5,224/6,082 (85.89279842%), below 100%.

Managed delta `94b6f845-d4f0-4df1-b630-31c673f6294b` is sealed with complete
coverage of the 19 authoritative source/test diff items and zero findings.
Focused package validation passed. This does not qualify the whole repository,
host sandbox/tool provenance, live providers, native services or recovery.
Retain `.local/qualification/ci-54fb84a-native-cross.json` and
`.local/sonar/summary-54fb84a2d3cc.txt`. No production action occurred.

Published candidate `42a3806d68a39583584d22e51469278cfa698cac` has exact tree
`d592a8ba1f2c9acb1106982d6eb82a9d94af4921`. Source run 41706881432 tested
synthetic merge `834345b94f89aa3d1287cf747a85bbd9131f61eb`; public readback
confirms its tree equals the candidate tree. Native Linux/macOS/Windows and
all 26 cross-builds passed. Security run 41706881558 passed CodeQL but failed
gosec with nine findings. Coverage remains 5,133/6,048 (84.87103175%), below
100%. The source-profile job has no Sonar endpoint; it is separate from the
completed exact-head Sonar route.

Sonar CE `1b97ca13-5ab7-48ec-98b8-f2e0cc96e96b` completed with current
analysis `534d0cde-eb0b-45b7-9af4-72cd824f3ff0` and exact revision fences.
Both prior smells are resolved, with zero issues, bugs, vulnerabilities,
hotspots and duplication. The gate is ERROR at 78.3% overall / 81.5% new
coverage. Keep the 2026-11-10 token-expiry warning.

Exact-head Windows cross-static Staticcheck passed all 19 owned packages;
gosec retains nine unsuppressed G103/G204 findings. Before/after HEAD, tree,
index and file-content fences matched. Managed diff
`0100370b-bbe7-4a3c-91ad-5b78c011cd04` is sealed with zero reportable findings
and partial completeness. Its deferred native runtime and broader-audit scope
are not retroactively satisfied by the separate native test result.
Retained receipts are `.local/qualification/ci-42a3806-native-cross.json`,
`.local/qualification/static-42a3806-20261104/`, and
`.local/sonar/summary-42a3806d68a3.txt`. These are not production qualification.

Published candidate `4a485258c3a72a7227502f359b4a049996b4ba61` has exact tree
`f8c53890e348cf8b7fdb56a749b0f8aaca0d00d5`. Native Linux/macOS, all 26
cross-builds, and CodeQL passed. gosec failed with nine findings and source
assurance was 5,114/6,022 (84.92195284%), below the unchanged 100% gate.

Native Windows failed in more than one way: a googleapi module-proxy timeout
blocked CLI/resolve/source setup, while tests that ran failed trusted-directory
preflight and offline `Current` Git temporary-file creation. Do not attribute
all failures to the proxy or treat all tests as inconclusive.

Sonar CE task `79b7e0cd-e546-4d68-848e-be7ab4e90113` and analysis
`9c02dbab-ce75-4d3b-828e-de1bf483d2c9` completed with zero bugs,
vulnerabilities, hotspots, and duplication, but its gate is ERROR at 78.4%
overall and 81.6% new coverage, with `go:S1192` (five `rev-parse` literals)
and `go:S978` (rune issue). Its token-expiry warning for 2026-11-10 remains.
This is not a pass.

Sealed managed audit delta `cad20f55-3074-47e5-8dce-f7648dc9141b` from e553
to 4a has zero new reportable findings but partial completeness; host sandbox,
tool provenance, and native Windows are deferred. It is not an audit pass.

The follow-up published in `42a` replaces the repeated Git command literal and renames
the shadowing identifier. It adds protected working/executable directories to
positive platform/CLI fixtures and a protected per-call temporary directory for
offline `Current` verification, with prior-temp restoration and cleanup. Whole
checkpoint18 checks passed locally; later exact-candidate receipts are recorded
above. No native configuration, credentials or
infrastructure changed.

Published candidate `8ed18f35be811f722068a3cf62d3093038fa8d93` has exact tree
`b42f899e839d85febc7ff06105b60cf07c41e5bc`. Native Linux/macOS and all 26
cross-builds passed, as did CodeQL. Native Windows failed with two empty SSH
probes and a path-oracle failure; gosec retains nine findings. CI whole-module
coverage was 84.9957%, below the unchanged 100% gate.

Its Sonar CE task `dbd6d2fa-3814-4b06-a01e-e3bc7632a184` completed successfully
with analysis `81a9ef98-5a95-4e3d-a706-39362381c61e` and zero open issues, but
its quality gate is ERROR at 78.6% overall and 82.1% new coverage. The retained
receipt warns that its token expires 2026-11-10. This is not a pass.

The captured 4a follow-up integrated the v1 source manifest before and after target
checks, rejects unexpected/ignored files, disables Git replacement/global/system
configuration, and requires the actual repository root. It bounds file reads and
preserves worktree metadata support. A source-root alias is now rejected before
Go version probing; its prior later formatting failure already prevented source
tests, so the regression does not demonstrate arbitrary test execution. Manifest
checks do not provide an atomic snapshot, host sandbox, or analyzer provenance.

That 4a batch supplies the Windows OS-known ProgramData folder to native Git/SSH
without inheriting a caller override; compares canonical Windows directories by
filesystem identity; and adds a real SDK tool-catalog no-read-grant regression.
CI audit binaries/evidence and Sonar scratch are outside the reviewed source
inventory. Tests, coverage, vet, Staticcheck, govulncheck, gosec and formatting receive
manifest-owned packages/imports/files rather than traversing generated output.
Manifest directories without a native Go build fail closed instead of silently
reducing coverage; current packages provide native fallbacks. Native Windows execution, exact-candidate Sonar, and independent
qualification of this batch remain required.

Published candidate `e55327d513b0753ce08ae82adffbc4cb3cd5d2d4` has exact tree
`1523d6ac067ef94270bd015fa60ad296f086f021`. Native Linux/macOS, all 26
cross-builds, and CodeQL passed; native Windows failed, and the explicit
verifier hypothesis was refuted. gosec still has nine findings. Its Sonar CE
task `bb229fa5-0bb7-4db0-bb8e-f88b9f6d99c1` succeeded with analysis
`50ea05ae-0367-4ca7-aa16-e5111349b6c4` and zero open issues, but the quality
gate is ERROR at 79.1% overall and 83.4% new coverage; the receipt also warns
that its token expires 2026-11-10. This is not a pass.

The sealed `e553` managed scan retained eight validated findings: one high
(auditor-host execution), five medium (assurance binding/PATH, SIGTERM, Windows
DLL, and fresh-scope controls), and two low (UID and DPAPI ACL). Its completeness
is partial and it is not an audit pass. Three assurance issues remain open;
keep sensitive paths in the owner-held report.

The captured `8ed` remediation batch added live-generation-versus-snapshot,
legacy-state, mixed-UID, DPAPI staging ACL, Windows protected-loader-directory,
SIGTERM, and MCP base-context/active-HTTP-cancellation fixes. Its focused tests
and cross-builds are revision-scoped evidence only; later active work requires
its own frozen-candidate checks. Do not claim every audit finding is fixed.

The Compute Engine task
`e52a9f8f-b66c-48f2-8a8b-425a533cab12` for revision
`41a8da186c71d94b420303328fbe06fcee708602` succeeded with analysis ID
`dbe51b3f-8bae-4075-9ce0-0a72482e42c3`, but its gate is ERROR: 78.3% overall
coverage (6,134/7,833 lines), 82.5% new coverage, and two code smells
(`go:S107` and `go:S1192`); reported bugs, vulnerabilities, hotspots, and
duplication are zero. Sonar line coverage is distinct from the Go statement
profiles and does not change the 100% gate. At the 41a8 receipt, the two working-tree smell fixes required a fresh exact-candidate Sonar recheck; later e553 and 8ed scans record zero issues while coverage remains failing.
Sonar, full unchanged coverage, live providers, native reboot/recovery,
installers, registry publication, and independent audit remain incomplete.
The sealed static managed audit at this checkpoint reported zero validated
findings, but its completeness is partial and it is not an audit pass. It
rejected a documentation contradiction as non-exploitable, while the stale
repository-sync wording is corrected here. It does not qualify native behavior,
live providers, installations, or services. Keep detailed evidence in the
owner-held security channel rather than reproducing sensitive attack paths in
public documentation.

GitHub source run 41686510514 tested the open PR revision
`41a8da186c71d94b420303328fbe06fcee708602` through GitHub's synthetic merge
`6fd454e6bc77888f4e40d7eec37e6bbe4c7a9748`. Native Linux/macOS and all 26
cross-builds passed. Windows job 126093565982 failed only repository
`TestSignedRepositoryAcceptanceAndRollback` with `trust_failure:
repository_signature`; its platform package passed. Source assurance failed at
84.08220132358063% against the unchanged 100% gate. Security run 41686510624
passed CodeQL but failed gosec with nine unsuppressed findings. The frozen
static receipt found zero reachable govulncheck vulnerabilities and one required
module advisory, `GO-2026-5932` in `golang.org/x/crypto@v0.58.0`. Earlier
41ea/4886 evidence remains in the qualification matrix as historical evidence.
A later dependency follow-up updates that module to its fixed pseudo-version
and reports zero local govulncheck findings; the historical 41a8 advisory is
retained as revision-scoped evidence. Later e553 and 8ed Sonar scans record
zero issues but fail coverage. The 8ed native Windows SSH probes and path oracle
are failures requiring diagnosis, not an unverified signing-fixture hypothesis.

The current batch also fixes a listener-cleanup deadlock. Its retained
reproduction failed before the fix and the focused normal/race/vet checks pass
after it. This is bounded local evidence, not a full or native qualification.

The snapshot at revision `960` contains 25 native archives plus one source
archive (26 total) with exact checksum/provenance evidence; its Linux smoke is
only a smoke test, not production qualification. Cross-build/package work is
build evidence only.
Native Windows/macOS/Linux runtime, restart/recovery, installer, signed
artifact, anonymous download, SBOM/provenance, and registry qualification still
require real evidence. The
signed repository runtime is wired into `sync`, including `FetchCurrent` and
its offline/fallback guards. That integration is not deployment qualification:
quota, recovery, native, and adverse-path evidence remain required.

Actual GitHub ruleset readback confirms active Gitflow branch protections
with four required Actions checks, one approval, stale-review dismissal,
last-push approval and thread resolution, no bypass actors, plus active
immutable `v*` tag rules. The fetched ruleset does not include a code-scanning
rule or required Go security/CodeQL status. The earlier `Any` / `Any` UI note
is not sufficient evidence of enforcement. Legacy branch-protection reads
for main/develop both return 403 from the integration; do not infer those
settings. The read-only browser fallback failed before navigation with an undefined
`timeoutMs` binding, so remaining enforcement/metadata cannot be inferred
from UI state. No administrative change was made. Issues, Projects, Wiki and Discussions are enabled;
repository description/topics are empty. Advanced CodeQL has run following
the user-approved switch. Retain
`.local/qualification/github-hardening-readback-2d1f-20261104.json`. Earlier
administrative writes returned 403; preserve permission gaps without creating
broad credentials or claiming that committed settings are enforced.

The current continuation adds exclusive protected file creation for `init` and
Sonar test credentials: it validates the parent, anchors creation through
`os.Root`, refuses existing names, applies native protection before writing,
and syncs the file and directory. Audit coverage is opened through the protected
file boundary. Concurrent-creator, no-clobber, cleanup-identity, usable-init,
and Windows DACL tests cover the new behavior; native rechecks remain required.
Initialization writes a synthetic fixture and policy separately; a failure
between those creations can leave the synthetic fixture and is not reported
as successful initialization.

The native fixture corrections use JSON encoding for Windows paths, canonical
temporary roots for macOS, and in-memory synthetic signing-key generation while
retaining real Git signing/verification. Case-collision decisions have a pure
cross-platform oracle plus filesystem integration where distinct names are
representable. The Unix executable-alias test now creates the alias rather
than assuming a `/bin/true` layout. Native macOS CI prefers the protected OS
Git/OpenSSH paths; a rejected helper reports only sanitized file metadata.

The current continuation also hardens native policy and literal-source reads,
adds schema/Parse parity regressions and cloud/CLI/doctor negative tests, repairs
relative/default-path config replacement, and corrects native test fixtures.
Windows directory ACL inheritance preserves the private allowed principals
for ordinary child creation; these tests passed in the native Windows package
run at revision `960`. The new exclusive-create DACL tests require a fresh run. The bounded Go engineering review found the config-repair defect, which
failed both regression cases before the fix and passed afterward. This is a
scoped review, not the independent managed security recheck or release audit.

## Priority continuation

1. Inspect `git status`; preserve concurrent work and do not overwrite others.
   Format and run bounded tests with private temporary storage before changing
   behavior.
2. Integrate and review the current policy/CLI/provider/repository/platform
   changes. Keep strict validation and protected-file semantics; add focused
   adverse-path tests where contract behavior changed.
3. Re-run SonarQube against the exact candidate revision, resolve every owned
   issue and coverage gap without exclusions or weakened rules, and retain the
   quality-gate receipt.
4. Complete unresolved sync/application/activation/recovery behavior and its
   strict snapshot/repository validation; do not infer health from an attempted
   activation or a source timestamp.
5. Perform qualified native, provider, installer, release, registry, and
   independent-security evidence. Only then consider a `release/1.0.0` branch,
   `v1.0.0-rc.1`, and ultimately `v1.0.0` under the gates in `RELEASE.md`.

## Do not regress these boundaries

- No inline secret values, provider bootstrap tokens, private keys, or browser
  sessions in policy, fixtures, logs, screenshots, or documentation.
- No silent offline fallback after denial, trust/TLS failure, malformed input,
  expiry, or policy mismatch.
- No arbitrary command/provider URI through MCP; raw reads require an exact
  explicit grant and are deliberate disclosure.
- No quality-gate suppression, fabricated green badge, unconditional release
  claim, or broad credential created to evade a missing permission.
- No production assertion from compilation, cross-build, static analysis, or a
  synthetic fixture alone.
