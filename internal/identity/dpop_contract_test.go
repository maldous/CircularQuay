package identity

import (
	"context"
	"strings"
	"testing"
)

const validConfirmationJKT = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func TestDPoPConfirmationClaimContract(t *testing.T) {
	for _, profile := range []Profile{ProfileRFC9068, ProfileMicrosoftEntraV2} {
		t.Run(string(profile), func(t *testing.T) {
			fixture := newAuthorityFixture(t)
			authority := fixture.authority(profile)
			assertBearerRejectsConfirmation(t, fixture, authority)
			assertDPoPConfirmationContract(t, fixture, authority)
		})
	}
}

func assertBearerRejectsConfirmation(t *testing.T, fixture *authorityFixture, authority Authority) {
	t.Helper()
	verifier := newFixtureVerifier(t, fixture, authority)
	for _, confirmation := range []any{
		nil,
		map[string]any{"jkt": validConfirmationJKT},
	} {
		changes := dpopProfileClaims(authority)
		changes["cnf"] = confirmation
		if _, err := verifier(context.Background(), fixture.token(t, authority, changes)); err != ErrInvalidToken {
			t.Fatalf("bearer accepted cnf %T: %v", confirmation, err)
		}
	}
}

func assertDPoPConfirmationContract(t *testing.T, fixture *authorityFixture, authority Authority) {
	t.Helper()
	authority.RequireDPoP = true
	verifier := newFixtureVerifier(t, fixture, authority)
	for _, test := range []struct {
		name    string
		changes map[string]any
	}{
		{name: "omitted", changes: map[string]any{}},
		{name: "null", changes: map[string]any{"cnf": nil}},
		{name: "extra member", changes: map[string]any{"cnf": map[string]any{"jkt": validConfirmationJKT, "kid": "extra"}}},
		{name: "malformed", changes: map[string]any{"cnf": map[string]any{"jkt": "not-base64url"}}},
		{name: "wrong digest length", changes: map[string]any{"cnf": map[string]any{"jkt": strings.Repeat("A", 42)}}},
		{name: "noncanonical", changes: map[string]any{"cnf": map[string]any{"jkt": validConfirmationJKT + "="}}},
		{name: "noncanonical unused bits", changes: map[string]any{"cnf": map[string]any{"jkt": strings.Repeat("A", 42) + "B"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := verifier(context.Background(), fixture.token(t, authority, dpopTokenChanges(authority, test.changes))); err != ErrInvalidToken {
				t.Fatalf("accepted invalid cnf: %v", err)
			}
		})
	}

	principal, err := verifier(context.Background(), fixture.token(t, authority, dpopTokenChanges(authority, map[string]any{
		"cnf": map[string]any{"jkt": validConfirmationJKT},
	})))
	if err != nil {
		t.Fatalf("valid cnf rejected: %v", err)
	}
	if principal.ConfirmationJKT != validConfirmationJKT {
		t.Fatalf("confirmation JKT = %q", principal.ConfirmationJKT)
	}
}

func dpopTokenChanges(authority Authority, changes map[string]any) map[string]any {
	merged := dpopProfileClaims(authority)
	for name, value := range changes {
		merged[name] = value
	}
	return merged
}

func dpopProfileClaims(authority Authority) map[string]any {
	if authority.Profile != ProfileMicrosoftEntraV2 {
		return map[string]any{}
	}
	return map[string]any{
		"ver": "2.0", "tid": tenantID, "oid": objectID, "azp": clientID, "scp": "cquay.read",
	}
}
