# Credential exposure and incident response

Use this route for scanner reports, accidental publication or suspected unauthorized retrieval. Keep protected evidence private and never paste a detected value into an issue, agent conversation or public report.

1. Record the incident ID, repository/commit/location, detector type, affected workload alias and known exposure interval using metadata only.
2. Confirm affected credential ownership and consumers without publishing the value.
3. Revoke or rotate at the issuer according to its service/incident policy.
4. Deliver replacements through the approved generation and verify consumer activation.
5. Fence compromised host/service/MCP identities and queued work as required.
6. Review native provider and toolkit audit evidence, including gaps and retention limits.
7. Remove plaintext from the current source/artifacts and prevent recurrence.
8. Decide whether history rewriting or cached-view removal is necessary and coordinate it separately.
9. Record recovery evidence and any remaining affected copies.

Git deletion, module retraction, registry deprecation and history rewriting do not recall earlier copies. Published Go content may remain cached after a repository/tag is removed. Issuer revocation is the credential containment step.

Use local redacted scanner results by default. Optional GitGuardian scanning is explicit because scan content can be sent to its service. Do not use `--show-secrets`, public unredacted reports or command-line values. A provider finding's raw API response may contain the detected secret; reduce it to approved metadata before logging or feeding it to an agent.

Do not broadly revoke unrelated credentials, alter production service state, rewrite history or publish an advisory without the relevant owner's authorization. Emergency-stop paths and audit preservation must remain available.

## Directory, exposure-gate and issuer workflows

Use [the lifecycle workflow](../credential-lifecycle.md) and
[canonical gate policy](../configuration-governance.md) to bind the incident to
an exact tenant, subject or credential version. Directory offboarding fences
sessions, delegation and queued effects under its declared propagation bounds.
GitHub/GitGuardian reads need separate scopes and protected references; raw
alerts may contain plaintext and must not enter incident tickets or telemetry.
GitHub list/get metadata profiles require `hide_secret=true` and reject responses
with unhidden secrets. Do not infer GitGuardian identifiers by hashing a value.

Preserve known exposure blocks when a vendor incident is resolved, ignored,
unavailable or stale. Block/warn/approved-exception decisions use the canonical
locks. A known exposed version stays blocked unless an explicitly approved
finite exception authorizes its exact use, presented with a warning; no exception
overrides issuer revocation or grants access. Routine
rotation can prepare/activate a replacement before retiring the old version;
a separately authorized compromise policy may require revocation first. Use
only supported issuer-native operations, reconcile uncertain effects before
retry and retain partial consumer outcomes. Closing the incident, revoking the
issuer credential and verifying rotation are independent recorded actions.
Secret-free OTLP events assist diagnosis; exporter success is not immutable
history or required-audit durability.

## SDK, telemetry and distribution incidents

Identify the exact package/native version, immutable artifact digest, safe fault
category and affected collector/plugin/registry scope. Preserve required audit
and redacted correlation records; exclude raw exception bodies, process streams,
request headers and token files. Disable a compromised exporter or update feed
under its explicit scope without waiving workload authorization or required audit.

Revoke stolen service/client grants at their issuing authority. A plugin removal,
package yank, feed correction or SDK-wrapper redaction does not erase delivered
values. Publish a newly identified correction with its own qualification and
registry readback, retaining the original incident and affected-version evidence.
