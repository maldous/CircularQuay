-------------------------- MODULE SenderConstraint --------------------------
EXTENDS Integers, Naturals, FiniteSets, TLC

\* A deliberately small, finite instance for TLC.
Authorities == {"fp-a", "fp-b"}
Keys == {"key-a"}
JTIs == {"jti-1"}
Nonces == {"nonce-1", "nonce-2", "nonce-3"}
Times == 0..2
NoExpiry == -1

Bindings == Authorities \X Keys
BindingJTIs == Bindings \X JTIs

VARIABLES
    now, currentAuthority, nonces, replays, issued, consumed, acceptedUntil,
    restartReplayUntil, phase, reqAuthority, reqKey, reqJTI, reqNonce,
    tokenEnd, proofEnd, checked, duplicateAccept, expiredAccept,
    staleAuthorityAccept

vars ==
    <<now, currentAuthority, nonces, replays, issued, consumed, acceptedUntil,
      restartReplayUntil, phase, reqAuthority, reqKey, reqJTI, reqNonce,
      tokenEnd, proofEnd, checked, duplicateAccept, expiredAccept,
      staleAuthorityAccept>>

Binding == <<reqAuthority, reqKey>>
Replay == <<Binding, reqJTI>>
Live(expiry) == expiry > now
RequestLive == Live(tokenEnd) /\ Live(proofEnd)

\* The implementation stores the earlier of token expiry and proof expiry.
Lifetime == IF tokenEnd <= proofEnd THEN tokenEnd ELSE proofEnd

Init ==
    /\ now = 0
    /\ currentAuthority = "fp-a"
    /\ nonces = [p \in (Bindings \X Nonces) |-> NoExpiry]
    /\ replays = [r \in BindingJTIs |-> NoExpiry]
    /\ issued = {}
    /\ consumed = {}
    /\ acceptedUntil = [r \in BindingJTIs |-> NoExpiry]
    /\ restartReplayUntil = [r \in BindingJTIs |-> NoExpiry]
    /\ phase = "idle"
    /\ reqAuthority = "fp-a"
    /\ reqKey = "key-a"
    /\ reqJTI = "jti-1"
    /\ reqNonce = ""
    /\ tokenEnd = 0
    /\ proofEnd = 0
    /\ checked = FALSE
    /\ duplicateAccept = FALSE
    /\ expiredAccept = FALSE
    /\ staleAuthorityAccept = FALSE

\* Mixed pairs let an old long-lived nonce be presented with a new request
\* whose token or proof is shorter. Thus TLC exercises min(token, proof).
Start ==
    /\ phase = "idle"
    /\ phase' = "received"
    /\ reqAuthority' \in Authorities
    /\ reqKey' \in Keys
    /\ reqJTI' \in JTIs
    /\ reqNonce' \in
        ({""} \cup {n \in Nonces :
            Live(nonces[<<<<reqAuthority', reqKey'>>, n>>])})
    /\ ((tokenEnd' = 1 /\ proofEnd' = 1) \/
        (tokenEnd' = 2 /\ proofEnd' = 2) \/
        (tokenEnd' = 1 /\ proofEnd' = 2) \/
        (tokenEnd' = 2 /\ proofEnd' = 1))
    /\ checked' = FALSE
    /\ UNCHANGED
        <<now, currentAuthority, nonces, replays, issued, consumed,
          acceptedUntil, restartReplayUntil, duplicateAccept, expiredAccept,
          staleAuthorityAccept>>

Verify ==
    /\ phase = "received"
    /\ reqAuthority = currentAuthority
    /\ RequestLive
    /\ phase' = "verified"
    /\ checked' = TRUE
    /\ UNCHANGED
        <<now, currentAuthority, nonces, replays, issued, consumed,
          acceptedUntil, restartReplayUntil, reqAuthority, reqKey, reqJTI,
          reqNonce, tokenEnd, proofEnd, duplicateAccept, expiredAccept,
          staleAuthorityAccept>>

Challenge ==
    /\ phase = "verified"
    /\ reqNonce = ""
    /\ \E n \in Nonces \ issued:
        /\ phase' = "idle"
        /\ nonces' = [nonces EXCEPT ![<<Binding, n>>] = Lifetime]
        /\ issued' = issued \cup {n}
        /\ UNCHANGED
            <<now, currentAuthority, replays, consumed, acceptedUntil,
              restartReplayUntil, reqAuthority, reqKey, reqJTI, reqNonce,
              tokenEnd, proofEnd, checked, duplicateAccept, expiredAccept,
              staleAuthorityAccept>>

\* This is the atomic PersistentStore.Consume linearization point. Its
\* acceptance-history latches are specification witnesses, not store state.
Commit ==
    /\ phase = "verified"
    /\ checked
    /\ reqNonce \in Nonces
    /\ reqAuthority = currentAuthority
    /\ RequestLive
    /\ Live(nonces[<<Binding, reqNonce>>])
    /\ ~Live(replays[Replay])
    /\ \E next \in Nonces \ issued:
        /\ phase' = "idle"
        /\ nonces' = [nonces EXCEPT
              ![<<Binding, reqNonce>>] = NoExpiry,
              ![<<Binding, next>>] = Lifetime]
        /\ replays' = [replays EXCEPT ![Replay] = Lifetime]
        /\ issued' = issued \cup {next}
        /\ consumed' = consumed \cup {<<Binding, reqNonce>>}
        /\ acceptedUntil' = [acceptedUntil EXCEPT ![Replay] = Lifetime]
        /\ duplicateAccept' =
            (duplicateAccept \/ Live(acceptedUntil[Replay]))
        /\ expiredAccept' = (expiredAccept \/ ~RequestLive)
        /\ staleAuthorityAccept' =
            (staleAuthorityAccept \/ (reqAuthority # currentAuthority))
        /\ UNCHANGED
            <<now, currentAuthority, restartReplayUntil, reqAuthority,
              reqKey, reqJTI, reqNonce, tokenEnd, proofEnd, checked>>

Reject ==
    /\ phase \in {"received", "verified"}
    /\ phase' = "idle"
    /\ UNCHANGED
        <<now, currentAuthority, nonces, replays, issued, consumed,
          acceptedUntil, restartReplayUntil, reqAuthority, reqKey, reqJTI,
          reqNonce, tokenEnd, proofEnd, checked, duplicateAccept,
          expiredAccept, staleAuthorityAccept>>

\* Store pruning may remove expired records. Witnesses keep their expiry so
\* their obligations end at that same fence without promising eternal storage.
Tick ==
    /\ now < 2
    /\ now' = now + 1
    /\ nonces' =
        [p \in (Bindings \X Nonces) |->
            IF Live(nonces[p]) THEN nonces[p] ELSE NoExpiry]
    /\ replays' =
        [r \in BindingJTIs |->
            IF Live(replays[r]) THEN replays[r] ELSE NoExpiry]
    /\ UNCHANGED
        <<currentAuthority, issued, consumed, acceptedUntil,
          restartReplayUntil, phase, reqAuthority, reqKey, reqJTI, reqNonce,
          tokenEnd, proofEnd, checked, duplicateAccept, expiredAccept,
          staleAuthorityAccept>>

\* A revocation after Verify is admissible; Commit must recheck authority.
RevokeAuthority ==
    /\ phase \in {"received", "verified"}
    /\ \E a \in Authorities \ {currentAuthority}:
        /\ currentAuthority' = a
        /\ UNCHANGED
            <<now, nonces, replays, issued, consumed, acceptedUntil,
              restartReplayUntil, phase, reqAuthority, reqKey, reqJTI,
              reqNonce, tokenEnd, proofEnd, checked, duplicateAccept,
              expiredAccept, staleAuthorityAccept>>

\* The durable replay record stays intact. The witness records every replay
\* live at a restart so a restart-clearing mutant violates the invariant.
Restart ==
    /\ phase # "idle"
    /\ phase' = "idle"
    /\ checked' = FALSE
    /\ restartReplayUntil' =
        [r \in BindingJTIs |->
            IF Live(replays[r]) THEN replays[r] ELSE restartReplayUntil[r]]
    /\ UNCHANGED
        <<now, currentAuthority, nonces, replays, issued, consumed,
          acceptedUntil, reqAuthority, reqKey, reqJTI, reqNonce, tokenEnd,
          proofEnd, duplicateAccept, expiredAccept, staleAuthorityAccept>>

Next ==
    Start \/ Verify \/ Challenge \/ Commit \/ Reject \/ Tick \/
    RevokeAuthority \/ Restart

TypeOK ==
    /\ now \in Times
    /\ currentAuthority \in Authorities
    /\ nonces \in [(Bindings \X Nonces) -> (-1..2)]
    /\ replays \in [BindingJTIs -> (-1..2)]
    /\ issued \subseteq Nonces
    /\ consumed \subseteq (Bindings \X Nonces)
    /\ acceptedUntil \in [BindingJTIs -> (-1..2)]
    /\ restartReplayUntil \in [BindingJTIs -> (-1..2)]
    /\ phase \in {"idle", "received", "verified"}
    /\ reqAuthority \in Authorities
    /\ reqKey \in Keys
    /\ reqJTI \in JTIs
    /\ reqNonce \in ({""} \cup Nonces)
    /\ tokenEnd \in Times
    /\ proofEnd \in Times
    /\ checked \in BOOLEAN
    /\ duplicateAccept \in BOOLEAN
    /\ expiredAccept \in BOOLEAN
    /\ staleAuthorityAccept \in BOOLEAN

\* Acceptance may recur after its stored expiry has been pruned, but not while
\* this independent acceptance-history witness still records it as live.
AtMostOneAcceptPerLiveExpiryWindow == ~duplicateAccept
NoExpiredAcceptance == ~expiredAccept
NoStaleAuthorityAcceptance == ~staleAuthorityAccept
OldNonceCannotReappear == \A p \in consumed: nonces[p] = NoExpiry
ReplayPersistsAcrossRestart ==
    \A r \in BindingJTIs:
        Live(restartReplayUntil[r]) => Live(replays[r])

Spec == Init /\ [][Next]_vars
=============================================================================
