# Sender-constraint model

SenderConstraint.tla is a finite TLC model of the CircularQuay DPoP resource
server’s nonce and replay store. A token and proof that have already passed
cryptographic validation are modeled as opaque inputs. The model checks the
state transitions that bind an accepted request to the current authority
fingerprint, DPoP key binding, nonce, JTI, and token/proof expiry
fences. Commit is an abstraction of the resource authorization boundary: it
combines the nonce/replay consumption and the current-authority release
decision. Tick between Verify and Commit is the bounded TOCTOU case.

Lifetime is the earlier of the token and proof expiry, matching proofExpiry in
internal/dpop/dpop.go. The finite Start domain includes <<1,2>> and <<2,1>> as
well as equal pairs. Consequently TLC can issue an old nonce for a long-lived
request, then consume it for a new request whose token or proof expires sooner.
The store’s expired-entry pruning is modeled: the replay property says an
acceptance cannot repeat while an independent acceptance-history witness is
live. It deliberately does not assert a permanent, global JTI ban after the
record’s expiry.

The checked configuration is two authority fingerprints, one DPoP key, one
JTI, three nonce values, one fixed resource, and a clock from 0 through 2.
HTTP origin/path matching and JWK parsing are outside this state model. These are coverage
bounds, not a proof for larger deployments or unbounded execution. The model
checks type preservation, one acceptance per live recorded expiry window, no
acceptance with expired token/proof fences, no acceptance after authority
revocation between Verify and Commit, consumed nonce non-reappearance, and
preservation of live replay records across restart. The restart obligation uses
a separate expiry-carrying witness: a mutant that clears durable live replay
records during Restart violates it.

The model assumes a trusted monotonic server clock, authenticated authority
fingerprints from the identity layer, a trustworthy OS and private filesystem,
a native single-replica process lock with durable protected writes, correct
cryptography/JWT validation and secure random nonce generation, and correct
token issuance. It does not model or prove host-administrator resistance,
sandbox escape, clustered/shared-store behavior, cryptography, token issuance,
filesystem/OS durability, or liveness.

The audit bundle records the full TLC command, tool hashes and raw logs. The required private mutants remove the commit
replay, expiry, and authority guards, and clear live replays on restart; each
must produce a counterexample for the corresponding property. This is a
bounded behavioral model, not an exact whole-program refinement; the Go
trace/revocation tests are separate bounded conformance evidence.
