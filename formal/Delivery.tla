------------------------------ MODULE Delivery ------------------------------
EXTENDS Naturals, FiniteSets, TLC

CONSTANTS Users, Clients, Workloads, Keys, InitialUseGrants,
          InitialReadGrants

\* Finite model values selected by Delivery.cfg. Keep the tuple-valued grants
\* in TLA+ rather than the restricted TLC configuration value language.
ModelUsers == {"alice", "bob"}
ModelClients == {"desktop", "agent"}
ModelWorkloads == {"demo", "other"}
ModelKeys == {"KEY_A", "KEY_B"}
ModelInitialUseGrants == {<<"alice", "desktop", "demo">>, <<"bob", "agent", "other">>}
ModelInitialReadGrants == {<<"alice", "desktop", "demo", "KEY_A">>}

\* This model assumes each workload has a configured protected persistent state
\* store, serialized by its native lock, on a trusted OS/filesystem. An Engine
\* without State.Directory has only an in-memory fence and is intentionally
\* outside this durability model.
VARIABLES phase, mode, actor, client, workload, selectedKey, operation,
          useGrants, readGrants, resolved, active, failure, usedCache,
          disclosed, upstreamCalls, durableFence, priorFence, readInFlight

vars == <<phase, mode, actor, client, workload, selectedKey, operation,
          useGrants, readGrants, resolved, active, failure, usedCache,
          disclosed, upstreamCalls, durableFence, priorFence, readInFlight>>

Identity == <<actor, client, workload>>
UseAllowed == Identity \in useGrants
ReadAllowed == <<actor, client, workload, selectedKey>> \in readGrants

Init ==
  /\ phase = "idle"
  /\ mode = "online"
  /\ actor \in Users
  /\ client \in Clients
  /\ workload \in Workloads
  /\ selectedKey \in Keys
  /\ operation = "use"
  /\ useGrants = InitialUseGrants
  /\ readGrants = InitialReadGrants
  /\ resolved = {}
  /\ active = {}
  /\ failure = "none"
  /\ usedCache = FALSE
  /\ disclosed = FALSE
  /\ upstreamCalls = 0
  /\ durableFence = [w \in Workloads |-> FALSE]
  /\ priorFence = FALSE
  /\ readInFlight = FALSE

Start ==
  /\ phase = "idle"
  /\ phase' = "pending"
  /\ mode' \in {"offline", "online", "hybrid"}
  /\ actor' \in Users
  /\ client' \in Clients
  /\ workload' \in Workloads
  /\ selectedKey' \in Keys
  /\ operation' \in {"use", "read"}
  /\ resolved' = {}
  /\ failure' \in {"none", "availability", "denied", "tls", "expired"}
  /\ usedCache' = FALSE
  /\ disclosed' = FALSE
  /\ upstreamCalls' = 0
  /\ priorFence' = FALSE
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<useGrants, readGrants, active, durableFence>>

RequestAllowed == UseAllowed /\ (operation = "use" \/ ReadAllowed)

\* A provider read cannot begin until the durable fence write succeeds. The
\* saved prior value is restored only after an availability result, so a prior
\* unsafe/interrupted attempt remains a fence across a later process restart.
FencePersisted ==
  /\ phase = "pending"
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode \in {"online", "hybrid"}
  /\ phase' = "fenced"
  /\ priorFence' = durableFence[workload]
  /\ durableFence' = [durableFence EXCEPT ![workload] = TRUE]
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls, readInFlight>>

\* A failed fence write denies the request before any provider side effect.
FenceWriteFails ==
  /\ phase = "pending"
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode \in {"online", "hybrid"}
  /\ phase' = "complete"
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence, readInFlight>>

ResolveOne ==
  /\ phase \in {"fenced", "resolving"}
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode \in {"online", "hybrid"}
  /\ failure = "none"
  /\ durableFence[workload]
  /\ resolved # Keys
  /\ \E key \in Keys \ resolved: resolved' = resolved \cup {key}
  /\ phase' = "resolving"
  /\ upstreamCalls' = upstreamCalls + 1
  /\ readInFlight' = TRUE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure, usedCache, disclosed,
                durableFence, priorFence>>

\* An availability outcome restores exactly the pre-read fence. Fallback is
\* therefore possible only when that prior persistent state was unfenced.
ProviderUnavailable ==
  /\ phase \in {"fenced", "resolving"}
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode = "hybrid"
  /\ (failure = "availability" \/ (phase = "resolving" /\ failure = "none"))
  /\ phase' = "pending"
  /\ failure' = "availability"
  /\ durableFence' = [durableFence EXCEPT ![workload] = priorFence]
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, usedCache,
                disclosed, upstreamCalls, priorFence>>

\* If the restoration write fails, the already-persisted pre-read fence stays
\* durable and the request cannot fall back to a snapshot.
RestoreWriteFails ==
  /\ phase \in {"fenced", "resolving"}
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode = "hybrid"
  /\ (failure = "availability" \/ (phase = "resolving" /\ failure = "none"))
  /\ phase' = "complete"
  /\ failure' = "availability"
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence>>

\* Denial, TLS/trust, and expiry failures retain the durable pre-read fence,
\* including when their final diagnostic state write fails.
UnsafeProviderFailure ==
  /\ phase \in {"fenced", "resolving"}
  /\ RequestAllowed
  /\ ~usedCache
  /\ mode \in {"online", "hybrid"}
  /\ (failure \in {"denied", "tls", "expired"}
      \/ (phase = "resolving" /\ failure = "none"))
  /\ phase' = "complete"
  /\ failure' \in {"denied", "tls", "expired"}
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence>>

CompleteFresh ==
  /\ phase = "resolving"
  /\ RequestAllowed
  /\ ~usedCache
  /\ resolved = Keys
  /\ failure = "none"
  /\ durableFence[workload]
  /\ durableFence' = [durableFence EXCEPT ![workload] = FALSE]
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<phase, mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls, priorFence>>

ResolveCache ==
  /\ phase = "pending"
  /\ RequestAllowed
  /\ ~durableFence[workload]
  /\ ((mode = "offline" /\ failure = "none")
      \/ (mode = "hybrid" /\ failure = "availability"))
  /\ phase' = "resolving"
  /\ resolved' = Keys
  /\ usedCache' = TRUE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure, disclosed, upstreamCalls,
                durableFence, priorFence, readInFlight>>

Activate ==
  /\ phase = "resolving"
  /\ RequestAllowed
  /\ resolved = Keys
  /\ ~durableFence[workload]
  /\ operation = "use"
  /\ phase' = "complete"
  /\ active' = resolved
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, failure, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence, readInFlight>>

Disclose ==
  /\ phase = "resolving"
  /\ RequestAllowed
  /\ resolved = Keys
  /\ ~durableFence[workload]
  /\ operation = "read"
  /\ ReadAllowed
  /\ phase' = "complete"
  /\ disclosed' = TRUE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, failure, usedCache,
                upstreamCalls, durableFence, priorFence, readInFlight>>

RevokeUse ==
  /\ phase \in {"pending", "fenced", "resolving"}
  /\ UseAllowed
  /\ useGrants' = useGrants \ {Identity}
  /\ UNCHANGED <<phase, mode, actor, client, workload, selectedKey,
                operation, readGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence, readInFlight>>

RevokeRead ==
  /\ phase \in {"pending", "fenced", "resolving"}
  /\ ReadAllowed
  /\ readGrants' = readGrants \ {<<actor, client, workload, selectedKey>>}
  /\ UNCHANGED <<phase, mode, actor, client, workload, selectedKey,
                operation, useGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls, durableFence, priorFence, readInFlight>>

\* A crash discards volatile work but deliberately retains the protected state
\* record's fence for the next process.
Crash ==
  /\ phase # "idle"
  /\ phase' = "idle"
  /\ resolved' = {}
  /\ usedCache' = FALSE
  /\ disclosed' = FALSE
  /\ upstreamCalls' = 0
  /\ priorFence' = FALSE
  /\ readInFlight' = FALSE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure, durableFence>>

Next == Start \/ FencePersisted \/ FenceWriteFails \/ ResolveOne
        \/ ProviderUnavailable \/ RestoreWriteFails \/ UnsafeProviderFailure
        \/ CompleteFresh \/ ResolveCache \/ Activate \/ Disclose
        \/ RevokeUse \/ RevokeRead \/ Crash

TypeOK ==
  /\ phase \in {"idle", "pending", "fenced", "resolving", "complete"}
  /\ mode \in {"offline", "online", "hybrid"}
  /\ resolved \subseteq Keys
  /\ active \subseteq Keys
  /\ upstreamCalls \in 0..Cardinality(Keys)
  /\ durableFence \in [Workloads -> BOOLEAN]
  /\ priorFence \in BOOLEAN
  /\ readInFlight \in BOOLEAN

NoPartialActivation == active = {} \/ active = Keys
OnlineNeverCaches == mode = "online" => ~usedCache
NoUnsafeFallback == usedCache =>
  ((mode = "offline" /\ failure = "none")
   \/ (mode = "hybrid" /\ failure = "availability"))
  /\ ~durableFence[workload]
ProviderReadHasDurableFence == readInFlight => durableFence[workload]
DisclosureRequiresGrant == disclosed =>
  (operation = "read" /\ UseAllowed /\ ReadAllowed /\ resolved = Keys)

Spec == Init /\ [][Next]_vars
=============================================================================
