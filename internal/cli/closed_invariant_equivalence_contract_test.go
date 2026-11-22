package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestClosedInvariantCLIJSONProjectionPreservesNumbersAndAdmission(t *testing.T) {
	cases := []struct {
		data string
		want any
	}{
		{`9007199254740993`, json.Number("9007199254740993")},
		{`-0.50e+2`, json.Number("-0.50e+2")},
		{`{"n":1.25,"v":[true,null]}`, map[string]any{"n": json.Number("1.25"), "v": []any{true, nil}}},
	}
	for _, test := range cases {
		data := []byte(test.data)
		got, err := strictJSON(data)
		if err != nil || !reflect.DeepEqual(got, test.want) || string(data) != test.data {
			t.Fatal("strict numeric projection changed value, spelling or caller bytes")
		}
	}
	for _, text := range []string{`{"x":1,"x":2}`, `1 2`, `1e400`, `{"broken":`} {
		got, err := strictJSON([]byte(text))
		if got != nil || !fault.Is(err, fault.Invalid) || err.Error() != "invalid_policy: "+errorConfigJSON {
			t.Fatal("strict admission lost its zero-result refusal")
		}
	}
}

// These are direct private-helper contracts, not post-validation mutations or
// claims that an admitted jsonPointerDelete call can reach the second refusal.
func TestClosedInvariantCLIArrayReplacementRetainsDefensiveRefusals(t *testing.T) {
	cases := []struct {
		parts []string
		kind  fault.Kind
	}{
		{[]string{"missing", "leaf"}, fault.Missing},
		{[]string{"list", "5"}, fault.Invalid},
		{[]string{"scalar", "leaf"}, fault.Invalid},
	}
	for _, test := range cases {
		var document any = map[string]any{"list": []any{"first"}, "scalar": "owned"}
		before, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		err = replaceArray(&document, test.parts, []any{"replacement"})
		if !fault.Is(err, test.kind) {
			t.Fatal("array parent refusal changed category")
		}
		after, err := json.Marshal(document)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("array parent refusal mutated its input")
		}
	}
}

func TestClosedInvariantCLIReadSelectionRetainsMissingContracts(t *testing.T) {
	config := model.Policy{Workloads: map[string]model.Workload{"worker": {Bindings: []model.Binding{{Name: "TOKEN", Source: "data"}}}}}
	cases := []struct {
		id, key string
		kind    fault.Kind
		op      string
	}{
		{"missing", "TOKEN", fault.Invalid, "workload"},
		{"worker", "missing", fault.Denied, "MCP binding"},
	}
	for _, request := range cases {
		selected, workload, err := selectedReadPolicy(config, request.id, request.key)
		if !fault.Is(err, request.kind) || err.Error() != string(request.kind)+": "+request.op {
			t.Fatal("missing private selection changed exact refusal")
		}
		if !reflect.DeepEqual(selected, model.Policy{}) || !reflect.DeepEqual(workload, model.Workload{}) {
			t.Fatal("missing private selection returned a partial declaration")
		}
	}
	if len(config.Workloads["worker"].Bindings) != 1 {
		t.Fatal("selection refusal mutated policy")
	}
}

func TestClosedInvariantCLIActionFingerprintKeepsExactClosedProjection(t *testing.T) {
	workload := model.Workload{Mode: "offline", Bindings: []model.Binding{{Name: "TOKEN", Source: "data", AllowOffline: true}}, Environment: map[string]string{"SYNTHETIC": "λ\x00value"}}
	config := model.Policy{Sources: map[string]model.Source{"data": {Type: "json", Path: "repo:values.json"}}, Repository: model.Repository{URL: "https://example.invalid/data.git", Ref: "refs/heads/data"}}
	repository := config.Repository
	encoded, err := json.Marshal(struct {
		Workload   model.Workload          `json:"workload"`
		Sources    map[string]model.Source `json:"sources"`
		Repository *model.Repository       `json:"repository,omitempty"`
	}{workload, config.Sources, &repository})
	if err != nil {
		t.Fatal(err)
	}
	got, err := actionScopeFingerprint(config, workload)
	if err != nil || got != sha256.Sum256(encoded) {
		t.Fatal("closed scope encoding changed its fingerprint")
	}
	delete(config.Sources, "data")
	got, err = actionScopeFingerprint(config, workload)
	if got != ([sha256.Size]byte{}) || !fault.Is(err, fault.Invalid) || err.Error() != "invalid_policy: action source" {
		t.Fatal("missing declaration lost the real source refusal")
	}
}

func TestClosedInvariantCLIRunIdentifierKeepsStatusAndDrawShape(t *testing.T) {
	for _, code := range []int{0, 17} {
		closedInvariantCLIRunIdentifierCase(t, code)
	}
}

func closedInvariantCLIRunIdentifierCase(t *testing.T, code int) {
	t.Helper()
	got, err := actionRunResult(code)
	if err != nil || got.ExitCode != code {
		t.Fatal("run status lost original exit code")
	}
	identifier, err := hex.DecodeString(got.ID)
	if err != nil || len(identifier) != 16 {
		t.Fatal("run identifier changed its exact draw size")
	}
	want := "completed"
	if code != 0 {
		want = "failed"
	}
	if got.Status != want {
		t.Fatal("run status changed")
	}
}

func TestClosedInvariantCLIResolvedFloorStillPrecedesDispatch(t *testing.T) {
	app, data := cliCompletionConfig(t)
	app.buildVersion = "1.0.0"
	var candidate model.Policy
	if err := json.Unmarshal(data, &candidate); err != nil {
		t.Fatal(err)
	}
	candidate.Updates = &model.Updates{Check: "never", MinimumVersion: "2.0.0"}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.WriteProtected(app.config, encoded); err != nil {
		t.Fatal(err)
	}
	got, err := app.preflightOperationalPolicy(context.Background())
	if !fault.Is(err, fault.Denied) || err.Error() != "access_denied: minimum runtime version" || !reflect.DeepEqual(got, model.Policy{}) {
		t.Fatal("delegating preflight bypassed the root/resolved runtime floor")
	}
}
