package mcp

import (
	"net/http/httptest"
	"testing"
)

func TestMCPInvariantBoundaryContractCallbackWithoutTokenReturnsNoServer(t *testing.T) {
	request := httptest.NewRequest("POST", "https://synthetic.example.test/mcp", nil)
	service := &Service{}
	if service.httpSessionServer(request) != nil {
		t.Fatal("SDK callback constructed a server without token authority")
	}
	// This directly checks the callback's independent defensive contract. The
	// real outer middleware still refuses this request before SDK dispatch;
	// no private token context or grant is manufactured here.
}

func TestMCPInvariantBoundaryContractLegacyScopePredicate(t *testing.T) {
	if !legacyScopeAllows(Principal{}, "status") {
		t.Fatal("local component scope shortcut changed")
	}
	if !legacyScopeAllows(Principal{HTTP: true, Scopes: []string{"cq:status"}}, "status") {
		t.Fatal("matching HTTP component scope refused")
	}
	if legacyScopeAllows(Principal{HTTP: true, Scopes: []string{"cq:list"}}, "status") {
		t.Fatal("unrelated HTTP component scope accepted")
	}
	// Scope alone is never a grant: Principal.Authorize's policy, expiry and
	// authority checks remain unchanged and are not bypassed by this predicate.
}
