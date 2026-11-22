package source

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

func TestSourceInvariantBoundaryContractLiteralOwnNULRefusal(t *testing.T) {
	value, err := literal("synthetic", true)
	if err != nil || value != (model.Value{Data: "synthetic", AuditVerified: true}) {
		t.Fatal("literal component control failed")
	}
	value, err = literal("synthetic\x00tail", true)
	if value != (model.Value{}) {
		t.Fatal("refused literal returned partial metadata or data")
	}
	requireProviderInputError(t, err, fault.Trust, "value_type")
	// Direct component validation, not a claim selectValue can emit NUL.
}

func TestSourceInvariantBoundaryContractInfisicalOwnCapabilityRefusal(t *testing.T) {
	calls := 0
	client := transport.Client{HTTP: &http.Client{Transport: cloudRoundTripper(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("invalid component capability selected a request")
		return nil, errors.New("unselected HTTP")
	})}}
	value, err := readInfisical(context.Background(), client,
		model.Source{Type: "not-a-versioned-provider"}, model.Binding{Version: "1"}, "synthetic-token")
	if value != (model.Value{}) || calls != 0 {
		t.Fatal("capability refusal returned data or attempted transport")
	}
	requireProviderInputError(t, err, fault.Invalid, "source_version_unsupported")
	// This is the adapter's own pre-effect defensive input contract. Registry
	// dispatch does not manufacture this mismatched provider/type combination.
}

func TestSourceInvariantBoundaryContractClosedSnapshotEncoding(t *testing.T) {
	directory := privateSourceDirectory(t, "invariant-snapshot")
	config := model.Snapshot{Path: filepath.Join(directory, "snapshot.json"), SOPS: filepath.Join(directory, "inert-sops"), Recipients: []string{"age1synthetic"}}
	values := map[string]string{"KEY": "synthetic<&>\n"}
	want := []byte("{\"KEY\":\"synthetic\\u003c\\u0026\\u003e\\n\"}")
	calls := 0
	run := func(ctx context.Context, executable string, args []string, plaintext []byte, identity, token string, allowlist []string) ([]byte, error) {
		calls++
		assertInvariantSnapshotInvocation(t, ctx, config, executable, args, plaintext, want)
		if identity != "" || token != "" || allowlist != nil {
			t.Fatal("snapshot encoding added credential or provider selectors")
		}
		return nil, fault.New(fault.Denied, "invariant-stop-before-write")
	}
	err := encryptSnapshotWithRunner(context.Background(), config, values, run)
	requireProviderInputError(t, err, fault.Denied, "invariant-stop-before-write")
	if calls != 1 || !reflect.DeepEqual(values, map[string]string{"KEY": "synthetic<&>\n"}) {
		t.Fatal("snapshot changed caller values or runner cardinality")
	}
	assertProviderInputAbsent(t, config.Path)
	assertProviderInputAbsent(t, config.SOPS)
}

func assertInvariantSnapshotInvocation(t *testing.T, ctx context.Context, config model.Snapshot, executable string, args []string, plaintext, want []byte) {
	t.Helper()
	if ctx.Err() != nil || executable != config.SOPS || !bytes.Equal(plaintext, want) {
		t.Fatal("snapshot lost its exact owned encoding or declared helper")
	}
	expected := append(sopsEncryptArgs(), "--age", "age1synthetic")
	if !reflect.DeepEqual(args, expected) {
		t.Fatal("snapshot changed its fixed argument vector")
	}
}

func TestSourceInvariantBoundaryContractNilSnapshotStillRefusesBeforeRunner(t *testing.T) {
	config := model.Snapshot{Recipients: []string{"age1synthetic"}}
	run := func(context.Context, string, []string, []byte, string, string, []string) ([]byte, error) {
		t.Fatal("nil primitive map bypassed semantic snapshot validation")
		return nil, errors.New("unselected runner")
	}
	err := encryptSnapshotWithRunner(context.Background(), config, nil, run)
	requireProviderInputError(t, err, fault.Invalid, "string document required")
}
