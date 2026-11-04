# Governed lifecycle tasks

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
