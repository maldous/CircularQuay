# Governed lifecycle tasks

The launch custody profile uses software KMS and age-key custody on Cloudflare
or Netlify. A protected runtime wrapping key stays outside the database, and
each profile has an independently random data key. Scoped authorization,
tenant isolation and required audit apply. Runtime or shared-key compromise
can expose reachable profile keys; software custody does not provide independent
HSM or non-exportability guarantees.

Each deployment must verify provider plans, issuer metadata, entitlements and
required custody. Additional services or upgrades require separate operator
approval. Hosting forecasts exclude labor, CRM/payment fees and customer pricing.
Synthetic frontend previews do not establish a connected backend or production
service. Implementation and qualification evidence must match the selected
release and deployment.

Read the selected release's capability contract before acting. The complete-v1
product design includes the following profiles; this file does not add CLI
verbs, MCP tools, SDK methods or schema fields to an installed release. If the
capability is absent, return the bounded unsupported result and owner next step.
Do not substitute an arbitrary provider script or mutate accounts directly.

## One policy and separate grants

Use the canonical organisation/tenant policy and its compiled effective revision.
Defaults fill unconstrained settings, required controls and locks propagate,
and multi-group conflicts follow typed approved rules or deny. No client-local
file, environment override, group ordering or model instruction may widen the
ceiling. CLI/API/UI/SDK/MCP share validation, authorization, approvals, revision
preconditions and required audit. Read again after a conflict.

Optional SCIM provisions/deprovisions metadata for an enrolled tenant/directory;
it is not login or a secret grant. Workload OIDC/SPIFFE proves only the selected
issuer/trust-domain identity, not permission to select a tenant or provider.
Device-posture/Conditional Access needs authenticated fresh evidence under a
supported user-device or workload profile. Caller health flags are not proof.
An optional external evaluator such as OPA can only narrow current permission;
HTTP success, missing/undefined result or a stale cached allow grants nothing.
Required unavailable/unknown evidence denies. Recheck after waits before effects.

Use independently protected directory, GitHub, GitGuardian, evaluator and issuer
credential references. Account connection, repository enrollment, purchases,
scan-content transmission, incident mutation, exposure exceptions and provider
rotation/revocation have distinct authority; an instruction file supplies none.
Never read an operational value or compute/send its hash to bind an incident.
GitGuardian's internal `secret_hash` is not a local hash lookup interface.

Single-file and explicitly trusted master/include layouts resolve to the same
canonical model. Master-defined pins, namespaces, source order and typed
conflict/group rules preserve enterprise locks and each source's authority
ceiling. Reject cycles/duplicates/escape, bound loading and validate all inputs
before atomic activation. Corporate and personal grants, decryption keys,
cache and scoped provenance stay separate; no personal uploads through remote
administration, explain or telemetry. Explicit refresh and optional finite
offline policy preserve known denials and never infer production access.
Do not invent syntax or bypass an unsupported resolver with a manual merge.

## Exposure and issuer response

Associate evidence with the exact tenant/account/source and immutable credential
version through authorized metadata and provenance. Scanner responses/webhooks
can contain secrets even with read-only access. GitHub list/get profiles require
`hide_secret=true`; reject unsupported hiding or an unhidden-secret response.
Only a bounded allowlisted DTO
may enter model-visible status; raw bodies, tokens, matched values and replacement
material stay outside logs, telemetry and agent context. Unknown associations
need review and cannot target an automatic issuer mutation.

Distinguish exposure, scanner workflow closure, validity observation/override,
issuer revocation, replacement activation and verified rotation. A resolved or
ignored incident does not clear known exposure. Apply canonical block/warn/
approved-exception policy. A known exposed version remains blocked unless a
separately approved finite exact-version exception permits the action; a warning
presents that exception rather than creating permission. Exceptions never
override issuer revocation, expiry, locks or required audit. Preserve
known blocks under stale evidence, offline use, cache restore and retries.

Execute a provider-native rotation/revocation only for its supported exact
profile and separately approved intent. A new store version is not necessarily
a new issuer credential. Return safe operation IDs/outcomes. Reconcile a
timeout or cancelled/uncertain effect before retrying; never create repeated
keys or claim revocation was undone. The incident policy determines routine
overlap versus revoke-first compromise handling. Verify consumer activation and
old-version disposition separately; copied keys/plaintext cannot be recalled.

## Export and forward migration

Shared UIF migration also preserves project identity and custody. Use the
[existing project route](operations.md#shared-uif-project-profiles), accepted
tree/toolchain and canonical impact/checklist bindings; CircularQuay and OffMenu remain
different products, not aliases for a default OffMenu job. Keep their private
state, source rights, credentials, review/publication and evidence separate.
Shared changes need actual evidence for both affected profiles. Continue their
independent work in parallel while preserving current leases and controlled
unit handover; a qualified candidate or Linear project record is not installed
runtime, task admission or a completed migration. No duplicate tracker or
abrupt cutover is authorized.

Preview migration/enrollment/rotation forms model the intended lifecycle; they
do not execute a provider ceremony or qualify recovery. The internal native
key-service core also needs real authorization/storage/provisioning and public
interface integration before it becomes callable service functionality. Retain
separate source, synthetic interaction and actual backend/native receipts. A
new schema or saved browser proposal requires renewed validation before active
policy changes; a previous control inventory is not an all-path mutation pass.

Export only approved operation/outcome/duration and bounded freshness/health
signals through the canonical OTel policy. No scanner/provider bodies, values,
value hashes, credential references or uncontrolled identity labels belong in
those signals. Collector delivery is neither immutable audit nor durable
required audit, and failure cannot authorize a repeated issuer effect.

Use reviewed versioned migrations and corrected forward revisions for schema,
policy and state changes. Preserve locks, authority generations, directory
removal, exposure/revocation history, replay/approvals and required audit. Never
make a feature appear usable by dropping an unsupported field, inventing an
endpoint, restoring a permissive policy or resetting incident state. The
release's qualification record establishes actual profile/interface support.

Optional enterprise lifecycle/evaluator/scanner profiles above are not default
hosted-subscription components or costs. The latest hosted baseline includes
genuine KMS and authenticated age-private-key service. Authorized profile
enrollment automatically provisions BOTH associations, with safe metering and
an upgrade opportunity, not config orchestration. It does not release offline
keys or activate hybrid by itself.
First-run initialization honors CLI/config disable before any hosted network;
locked conflicts deny. Registration supplies neither arbitrary key access nor
permission to import local keys/config or charge automatically. CI repeats
reconcile one workload-identity enrollment. KMS is configured per profile; it is
distinct from hosted age retrieval and does not imply a runtime flag. Copied keys/ciphertext
remain usable outside future service-release decisions.

Track dual provisioning state and reconcile KMS/age resources separately before
ready. Explicit hybrid activation needs compatible ciphertext/recipients and
key-version/window/scope checks; association creation downloads no offline key.
Retain separate logical events for associations, KMS operations and age fetches;
only explicit tariff policy may assign customer billing, never a provider retry.

Meter age hosted access events, not profiles/key slots. Distinct host fetches are
distinct successful uses; local reuse and denied/precommit failures add zero.
Retries of one committed logical access reconcile the same durable usage event,
without reauthorizing revoked access or creating repeated charges. Extra production custom admin UI/screens remain unapproved; a separate frontend
checkout may build synthetic/mock previews and loopback local-configuration
simulations only. Reuse existing platform services and the confirmed CLI/text
flow for actual operations.

All config stays on the client: user-owned network/VPN/source credentials,
complete single/multiple-source resolution/validation/explain and local save.
No contents/source credentials reach hosted backend/proxy/analytics/logs.
Browser CORS/file restrictions require permitted native/local CLI/SDK integration,
not a server fetch proxy. Hosted state is profile→KMS/age plus necessary
service metadata; keys are never logged/displayed. CLI/config disable resolves
before hosted network. KMS profile selection is separate from hosted age
registration; no guessed flag, production UI authority or automatic subscription
follows. The separately authorized previews remain synthetic/mock and
loopback-only.

For clarified user-profile KMS, apply [custody and availability rules](operations.md#user-profile-kms-and-availability-modes).
Existing local keys/assets require the separate [complete migration workflow](operations.md#local-to-managed-migration),
not a mode flag. Rewrap/import/re-encrypt and later retirement have separate
permissions, real backend/hardware limits, resumable per-asset evidence and
retained-copy risks under the existing security/assurance requirements.
Online-only cannot bypass required KMS; explicit hybrid retains authorized
availability-only encrypted fallback within its approved window. Denial, trust
failure, malformed response and expiry do not permit downgrade. Provider revocation
cannot recall retained data keys/plaintext or an age fallback key.

Customer account/usage/invoice/plan/quota/subscription/support operations require
exact tenant/account/field grants; hosting dashboard/admin APIs are not customer
billing or CRM. Reconcile signed provider events and lost/duplicate mutations
idempotently. Future unattended financial actions need explicit bounded user
authority and applicable confirmation; capability alone never authorizes charges.
Use actual qualified provider-hosted payment entry, not card custody/admin keys
or guessed SDK/MCP/CLI methods. No provider setup/purchase/config upload or production custom UI implementation
is authorized by design documentation. Synthetic/mock previews retain the
separate no-backend/no-provider/no-key boundary.
