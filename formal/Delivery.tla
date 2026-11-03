------------------------------ MODULE Delivery ------------------------------
EXTENDS Naturals, FiniteSets, TLC

CONSTANTS Users, Clients, Workloads, Keys, InitialUseGrants,
          InitialReadGrants

VARIABLES phase, mode, actor, client, workload, selectedKey, operation,
          useGrants, readGrants, resolved, active, failure, usedCache,
          disclosed, upstreamCalls

vars == <<phase, mode, actor, client, workload, selectedKey, operation,
          useGrants, readGrants, resolved, active, failure, usedCache,
          disclosed, upstreamCalls>>

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
  /\ UNCHANGED <<useGrants, readGrants, active>>

RequestAllowed == UseAllowed /\ (operation = "use" \/ ReadAllowed)

ResolveOne ==
  /\ phase \in {"pending", "resolving"}
  /\ RequestAllowed
  /\ mode \in {"online", "hybrid"}
  /\ failure = "none"
  /\ resolved # Keys
  /\ \E key \in Keys \ resolved: resolved' = resolved \cup {key}
  /\ phase' = "resolving"
  /\ upstreamCalls' = upstreamCalls + 1
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure, usedCache, disclosed>>

ResolveCache ==
  /\ phase = "pending"
  /\ RequestAllowed
  /\ ((mode = "offline" /\ failure = "none")
      \/ (mode = "hybrid" /\ failure = "availability"))
  /\ phase' = "resolving"
  /\ resolved' = Keys
  /\ usedCache' = TRUE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure, disclosed, upstreamCalls>>

Activate ==
  /\ phase = "resolving"
  /\ RequestAllowed
  /\ resolved = Keys
  /\ operation = "use"
  /\ phase' = "complete"
  /\ active' = resolved
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, failure, usedCache,
                disclosed, upstreamCalls>>

Disclose ==
  /\ phase = "resolving"
  /\ RequestAllowed
  /\ resolved = Keys
  /\ operation = "read"
  /\ ReadAllowed
  /\ phase' = "complete"
  /\ disclosed' = TRUE
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, resolved, active, failure, usedCache,
                upstreamCalls>>

RevokeUse ==
  /\ phase \in {"pending", "resolving"}
  /\ UseAllowed
  /\ useGrants' = useGrants \ {Identity}
  /\ UNCHANGED <<phase, mode, actor, client, workload, selectedKey,
                operation, readGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls>>

RevokeRead ==
  /\ phase \in {"pending", "resolving"}
  /\ ReadAllowed
  /\ readGrants' = readGrants \ {<<actor, client, workload, selectedKey>>}
  /\ UNCHANGED <<phase, mode, actor, client, workload, selectedKey,
                operation, useGrants, resolved, active, failure, usedCache,
                disclosed, upstreamCalls>>

Crash ==
  /\ phase # "idle"
  /\ phase' = "idle"
  /\ resolved' = {}
  /\ usedCache' = FALSE
  /\ disclosed' = FALSE
  /\ upstreamCalls' = 0
  /\ UNCHANGED <<mode, actor, client, workload, selectedKey, operation,
                useGrants, readGrants, active, failure>>

Next == Start \/ ResolveOne \/ ResolveCache \/ Activate \/ Disclose
        \/ RevokeUse \/ RevokeRead \/ Crash

TypeOK ==
  /\ phase \in {"idle", "pending", "resolving", "complete"}
  /\ mode \in {"offline", "online", "hybrid"}
  /\ resolved \subseteq Keys
  /\ active \subseteq Keys
  /\ upstreamCalls \in 0..Cardinality(Keys)

NoPartialActivation == active = {} \/ active = Keys
OnlineNeverCaches == mode = "online" => ~usedCache
NoUnsafeFallback == usedCache =>
  ((mode = "offline" /\ failure = "none")
   \/ (mode = "hybrid" /\ failure = "availability"))
DisclosureRequiresGrant == disclosed =>
  (operation = "read" /\ UseAllowed /\ ReadAllowed /\ resolved = Keys)

Spec == Init /\ [][Next]_vars
=============================================================================
