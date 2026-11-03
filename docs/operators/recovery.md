# Rotation, backup and recovery

## Separate rotations

Provider credentials, bootstrap identities, age recipients, SOPS data keys and OpenBao wrapping keys have different lifetimes. For a normal credential rotation, deliver and verify the replacement before revoking the old value when the issuer supports overlap. An incident may require revocation first.

Removing an age recipient does not invalidate historical ciphertext. Rotating a wrapping key does not rotate the actual provider credential. A backend denial cannot erase plaintext already delivered to a running process or MCP client.

## Managed-key recovery and revocation

Follow the [managed-key custody contract](../managed-key-service.md). Revoke the
compromised principal/client mapping to stop future service fetches, then assess
all recipients, data keys, and plaintext already released. A shared organization
key needs recipient/data-key replacement for future bundles; an old key plus an
old ciphertext remains usable. Rotate exposed application credentials at their
issuer. Recovery recipients are separately held and never an unreviewed bypass
for ordinary clients. Restore current deny state and policy versions before
re-enabling fetches; restoring a backup must not resurrect revoked grants.

## Backups

Back up canonical encrypted artifacts, protected policy/trust anchors, required native store state and independently protected OpenBao data/configuration/seal material. Machine-bound credential blobs alone are not sufficient.

Export upstream manager values only when explicitly permitted. Native upstream backup responsibility stays with that manager's operator; CircularQuay does not silently clone every secret store.

Keep recovery authority off ordinary workload hosts. Recovery shares do not replace permanently lost auto-unseal key material. Verify both backup integrity and an isolated decryption/restore before relying on retention.

The default objective is daily backup plus restore qualification before operational enrollment and after material recovery changes. Actual capacity, retention and custody must be recorded by the operator.

## Replacement host

1. Install authenticated native artifacts.
2. Recover protected policy/trust and approved encrypted releases.
3. Generate/select a fresh host identity.
4. On a trusted recovery station, re-encrypt or grant only its required workloads.
5. Restore OpenBao with the independent seal/data/configuration material when applicable.
6. Regenerate native systemd/DPAPI projections; do not require old TPM/DPAPI master state.
7. Prove synthetic startup, denial, rotation and offline behavior before operation.

## Rollback

Use a retained verified executable and known-good projection or a new authenticated forward release restoring prior configuration. Do not weaken downgrade protection or reuse published versions. No rollback may reactivate known revoked, expired or compromised material.

Native cloud stores and deployment revisions have their own update/rollback behavior. Preserve complete generation/version receipts, and report partial destination failure rather than inventing cross-system atomicity.
