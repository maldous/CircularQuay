# Delivery state model

`Delivery.tla` is a finite specification of exact user/client/workload grants,
complete generations, disclosure, fallback, revocation before activation,
durable pre-read fencing, and crashes. `Delivery.cfg` fixes two users (`alice`,
`bob`), two clients (`desktop`, `agent`), two workloads (`demo`, `other`), and
two keys (`KEY_A`, `KEY_B`), with the exact initial grants defined in the
module. Changing any of those bounds changes what was checked and must be
recorded with results.

Run a checksum-verified TLC distribution from the release audit environment.
`Delivery.cfg` selects the finite `Model*` definitions in `Delivery.tla`; the
tuple-valued grants are deliberately defined in TLA+ because the TLC
configuration value language does not accept those tuples directly.

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools.jar tlc2.TLC \
  -workers 1 -fp 0 -seed 0 -metadir ./audit-results/tlc-state \
  -config formal/Delivery.cfg formal/Delivery.tla
```

The official `v1.8.0` release channel is rolling, so its tag alone is not an
immutable tool identity. Record the TLC-reported build version, the downloaded
JAR digest, the upstream release digest, and the model/configuration digests
with each run. Keep the state directory isolated from source and remove it only
after retaining the result receipt needed for review.

The invariants reject partial activation, online cache reuse, fallback after
authorization/TLS/expiry failure, raw disclosure without exact current grants,
and provider reads that start without a durable fence. A cache use requires an
unfenced durable state. A crash discards volatile resolution state while
retaining the workload's durable fence; a fully complete fresh generation is the
only modeled transition that clears a fence that existed before the request.
Availability restores the exact pre-read fence (and can therefore restore an
initially unfenced state), while unsafe results and a failed restoration write
retain the fence.
Policy may be revoked between resolution steps and activation, which must
recheck the current grant.

This model abstracts provider values as key names. It does not prove TLS,
cryptography, SDK behavior, application logging, OS isolation, the correctness of
the specification, or the Go implementation. No liveness/availability guarantee
is claimed: provider recovery and scheduler fairness are external assumptions.
The policy tests exercise the corresponding implementation decisions, and the
integration suites must exercise the I/O boundaries. A successful model run is
one piece of evidence, not whole-system formal verification.

## Conformance trace mapping

The model is a small state-machine contract. These focused Go tests replay its
fence transitions against the protected state-store implementation:

- `TestCrashAfterPersistedProviderFenceDeniesSnapshotAfterRestart` maps
  `FencePersisted` followed by `Crash`: a restarted process cannot restore the
  snapshot.
- `TestUnavailableRestoreWriteFailureRetainsFenceAfterRestart` maps
  `RestoreWriteFails`: an availability result whose prior-state restoration
  cannot be written leaves the pre-read fence durable across restart.
- `TestProviderReadRequiresDurableFenceBeforeReader` maps `FenceWriteFails`:
  the provider reader is not called when the durable write cannot succeed.
- `TestPartialReadNeverActivatesAndDenialFences`,
  `TestProviderFailureFenceSurvivesRestart`, and
  `TestProviderFenceSurvivesFailureWriteAndRestart` map unsafe provider
  outcomes after partial provider progress, including failed final-state
  persistence.
- `TestCompleteGenerationClearsFenceAndPreservesHistory` and
  `TestReadKeyDoesNotClearWholeWorkloadFence` map the only clear transition and
  the prohibition on a single-key read clearing a workload-wide fence.

The POSIX permission-failure tests need independent native Windows durability
qualification; they do not establish it.

## Trusted assumptions

- Each modeled workload uses a configured protected persistent state store and
  its native workload lock; an in-memory-only engine is outside the durability
  model.
- The OS and filesystem uphold protected atomic writes and release process locks
  after crashes.
- The operator controls protected policy and bootstrap identities.
- Authentication supplies an already verified exact user/client tuple.
- A cache generation was explicitly approved for its workload and contains only
  eligible static values.
- Provider availability, denial, TLS and expiry classifications are accurate.
- Once disclosed, a value cannot be recalled from a client.
- Source retrieval and raw disclosure audit requirements remain separate from
  this authorization abstraction.

Publish the actual TLC version, model/config hashes, completed state count,
warnings and result in the audit bundle. Do not mark this model as checked merely
because its files exist or parse.

## Sender constraints

[SenderConstraint.tla](SenderConstraint.tla) and its
[finite configuration](SenderConstraint.cfg) model the optional DPoP nonce and
replay decisions separately. [The model contract](SenderConstraint.md) states
the bounds and trusted clock, cryptographic, store and operating-system
assumptions. The corresponding Go conformance trace uses signed synthetic
proofs and the persistent local store. This bounded model does not establish
cryptographic security, resistance to a host administrator or cluster safety.
