package cq

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestSDKInvariantBoundaryContractGeneralOutcomeMapper(t *testing.T) {
	if outcome(nil) != "success" {
		t.Fatal("successful outcome changed")
	}
	if outcome(fmt.Errorf("component wrapper: %w", &Error{Category: CategoryDenied})) != string(CategoryDenied) {
		t.Fatal("safe typed component category was lost")
	}
	if outcome(errors.New("PRIVATE_COMPONENT_DETAIL")) != "unavailable" {
		t.Fatal("unknown component error was disclosed")
	}
	// Public SDK methods still normalize errors first; this checks the general
	// error mapper, not a fabricated raw error from a concrete SDK operation.
}

func TestSDKInvariantBoundaryContractIJSONTraversalDefenses(t *testing.T) {
	nodes := 0
	if !validIJSON(nil, 0, &nodes) || nodes != 1 {
		t.Fatal("owned I-JSON traversal control failed")
	}
	if validIJSON(nil, 0, nil) {
		t.Fatal("missing traversal budget accepted")
	}
	if validIJSON(nil, maxIJSONDepth+1, new(int)) {
		t.Fatal("excess traversal depth accepted")
	}
	if validIJSON(struct{}{}, 0, new(int)) {
		t.Fatal("unsupported component Go type accepted")
	}
	// The typed traversal takes any; these are its own defensive contracts,
	// not a claim encoding/json produces structs or a nil owned node counter.
}

func TestSDKInvariantBoundaryContractIJSONNumberDefenses(t *testing.T) {
	for _, value := range []json.Number{"not-a-number", "1e400", "9007199254740992"} {
		if validIJSONNumber(value) {
			t.Fatal("nonportable component number accepted")
		}
	}
	for _, value := range []json.Number{"1.5", "9007199254740991"} {
		if !validIJSONNumber(value) {
			t.Fatal("portable component number refused")
		}
	}
	// Invalid json.Number strings are direct validator inputs, not forged
	// results from the preceding strict JSON decoder.
}

func TestSDKInvariantBoundaryContractHTTPSAlreadyImpliesAbsolute(t *testing.T) {
	for _, value := range []string{"https://synthetic.example.test", "HTTPS://synthetic.example.test/policy"} {
		if !validConfigSource(value) {
			t.Fatal("valid absolute HTTPS source refused")
		}
	}
	for _, value := range []string{"//synthetic.example.test", "http://synthetic.example.test", "https:opaque", "https://synthetic.example.test?", "https://synthetic.example.test#"} {
		if validConfigSource(value) {
			t.Fatal("noncanonical or non-HTTPS source accepted")
		}
	}
}
