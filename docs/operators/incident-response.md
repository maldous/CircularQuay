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
