# Delivery state model

`Delivery.tla` is a finite specification of exact user/client/workload grants,
complete generations, disclosure, fallback, revocation before activation and
crashes. `Delivery.cfg` fixes two users, clients, workloads and keys. Changing
these bounds changes what was checked and must be recorded with results.

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
authorization/TLS/expiry failure, and raw disclosure without exact current
grants. A crash discards incomplete resolution while retaining only a previously
complete active generation. Policy may be revoked between resolution steps and
activation, which must recheck the current grant.

This model abstracts provider values as key names. It does not prove TLS,
cryptography, SDK behavior, application logging, OS isolation, the correctness of
the specification, or the Go implementation. No liveness/availability guarantee
is claimed: provider recovery and scheduler fairness are external assumptions.
The policy tests exercise the corresponding implementation decisions, and the
integration suites must exercise the I/O boundaries. A successful model run is
one piece of evidence, not whole-system formal verification.

## Trusted assumptions

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
