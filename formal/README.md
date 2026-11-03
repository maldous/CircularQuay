# Delivery state model

`Delivery.tla` is a finite specification of exact user/client/workload grants,
complete generations, disclosure, fallback, revocation before activation and
crashes. `Delivery.cfg` fixes two users, clients, workloads and keys. Changing
these bounds changes what was checked and must be recorded with results.

Run the pinned TLC distribution from the release audit environment:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config formal/Delivery.cfg formal/Delivery.tla
```

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
