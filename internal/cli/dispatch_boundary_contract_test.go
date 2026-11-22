package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/update"
)

type dispatchBoundaryResult struct {
	code                int
	output, diagnostics string
}

func dispatchBoundaryRunner(output, diagnostics io.Writer) Runner {
	return Runner{In: bytes.NewReader(nil), Out: output, Err: diagnostics, BuildVersion: "1.2.3", BuildCommit: "synthetic-build-commit"}
}

func runDispatchBoundary(arguments []string) dispatchBoundaryResult {
	var output, diagnostics bytes.Buffer
	code := dispatchBoundaryRunner(&output, &diagnostics).Run(context.Background(), arguments)
	return dispatchBoundaryResult{code: code, output: output.String(), diagnostics: diagnostics.String()}
}

func TestDispatchBoundaryContractEmptyInvocationDoesNotLoadPolicy(t *testing.T) {
	root := privateCLIDirectory(t)
	config, state := filepath.Join(root, "missing-policy.json"), filepath.Join(root, "unused-state")
	result := runDispatchBoundary([]string{"--config", config, "--state-dir", state})
	want := "cq [--config PATH] [--state-dir PATH] <bootstrap|flow|init|config|doctor|plan|run|sync|snapshot|apply|status|update|providers|mcp|version>\n"
	if result.code != 0 || result.output != want || result.diagnostics != "" {
		t.Fatal("empty invocation changed its public usage or loaded a policy")
	}
	assertDispatchBoundaryAbsent(t, config, state)
}

func TestDispatchBoundaryContractVersionUsesEmbeddingIdentity(t *testing.T) {
	root := privateCLIDirectory(t)
	config := filepath.Join(root, "missing-policy.json")
	result := runDispatchBoundary([]string{"--config", config, "version"})
	if result.code != 0 || result.diagnostics != "" {
		t.Fatal("version metadata required a policy or wrote diagnostics")
	}
	var got map[string]string
	decodeDispatchBoundaryJSON(t, result.output, &got)
	want := map[string]string{"version": "1.2.3", "commit": "synthetic-build-commit", "go": "1.27", "maturity": "development"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("version output changed its exact public identity fields")
	}
	assertDispatchBoundaryAbsent(t, config)
}

type dispatchBoundaryCapability struct {
	Type      string `json:"type"`
	Read      bool   `json:"read"`
	Versioned bool   `json:"versioned"`
	Maturity  string `json:"maturity"`
}

func TestDispatchBoundaryContractProvidersRemainDeclarationOnly(t *testing.T) {
	root := privateCLIDirectory(t)
	config, state := filepath.Join(root, "missing-policy.json"), filepath.Join(root, "unused-state")
	result := runDispatchBoundary([]string{"--config", config, "--state-dir", state, "providers"})
	if result.code != 0 || result.diagnostics != "" {
		t.Fatal("provider capability metadata required operational inputs")
	}
	var got []dispatchBoundaryCapability
	decodeDispatchBoundaryJSON(t, result.output, &got)
	if !reflect.DeepEqual(got, dispatchBoundaryCapabilities()) {
		t.Fatal("provider output changed its fields, ordering or qualification limits")
	}
	assertDispatchBoundaryAbsent(t, config, state)
}

func dispatchBoundaryCapabilities() []dispatchBoundaryCapability {
	const contract = "contract-tested; live qualification pending"
	const sdk = "official SDK; scope/unit tests; live qualification pending"
	const vendor = "fixed vendor CLI; live qualification pending"
	return []dispatchBoundaryCapability{
		{"json", true, false, contract},
		{"env", true, false, contract},
		{"sops", true, false, contract},
		{"openbao", true, true, contract},
		{"vault", true, true, contract},
		{"doppler", true, false, contract},
		{"infisical", true, true, contract},
		{"akeyless", true, false, contract},
		{"aws-sm", true, true, sdk},
		{"aws-ssm", true, true, sdk},
		{"azure", true, true, sdk},
		{"gcp", true, true, sdk},
		{"bitwarden", true, false, vendor},
		{"onepassword", true, false, vendor},
	}
}

type dispatchBoundaryRefusingWriter struct{ called bool }

func (writer *dispatchBoundaryRefusingWriter) Write([]byte) (int, error) {
	writer.called = true
	return 0, errors.New("SYNTHETIC_PRIVATE_WRITER_DETAIL")
}

func TestDispatchBoundaryContractOutputFailureIsSanitized(t *testing.T) {
	for _, command := range []string{"version", "providers"} {
		t.Run(command, func(t *testing.T) {
			assertDispatchBoundaryOutputRefusal(t, command)
		})
	}
}

func assertDispatchBoundaryOutputRefusal(t *testing.T, command string) {
	t.Helper()
	writer := &dispatchBoundaryRefusingWriter{}
	var diagnostics bytes.Buffer
	code := dispatchBoundaryRunner(writer, &diagnostics).Run(context.Background(), []string{command})
	if code != 1 || !writer.called || diagnostics.String() != "{\"error\":\"activation_failed\"}\n" {
		t.Fatal("output refusal lost its public status or disclosed the writer error")
	}
}

func TestDispatchBoundaryContractDoctorSerializationRetainsFailure(t *testing.T) {
	report := doctorReport{OS: "synthetic-os", Arch: "synthetic-arch", LocalSubject: "synthetic-subject", Config: "missing_default", Fix: "not_applicable"}
	var output, diagnostics bytes.Buffer
	app := application{runner: dispatchBoundaryRunner(&output, &diagnostics)}
	code, err := printDoctor(app, report, 1)
	if code != 1 || err != nil || diagnostics.Len() != 0 {
		t.Fatal("doctor metadata positive control changed its caller-supplied status")
	}
	var got doctorReport
	decodeDispatchBoundaryJSON(t, output.String(), &got)
	if !reflect.DeepEqual(got, report) {
		t.Fatal("doctor serialization changed its public report fields")
	}
	writer := &dispatchBoundaryRefusingWriter{}
	app.runner = dispatchBoundaryRunner(writer, &diagnostics)
	code, err = printDoctor(app, report, 1)
	assertDispatchBoundaryFault(t, err, fault.Apply, "output")
	if code != 0 || !writer.called || diagnostics.Len() != 0 {
		t.Fatal("doctor serialization failure retained a success result or raw diagnostics")
	}
}

func TestDispatchBoundaryContractDoctorRejectsBeforeIdentityOrPolicy(t *testing.T) {
	for _, arguments := range [][]string{{"--SYNTHETIC_PRIVATE_FLAG"}, {"SYNTHETIC_PRIVATE_ARGUMENT"}} {
		root := privateCLIDirectory(t)
		config, state := filepath.Join(root, "missing-policy.json"), filepath.Join(root, "unused-state")
		args := append([]string{"--config", config, "--state-dir", state, "doctor"}, arguments...)
		assertDispatchBoundaryCLIRefusal(t, runDispatchBoundary(args), 2, "invalid_policy")
		assertDispatchBoundaryAbsent(t, config, state)
	}
}

func TestDispatchBoundaryContractDeclaredExecutablesAreMetadataOnly(t *testing.T) {
	root := privateCLIDirectory(t)
	policy := model.Policy{
		Sources: map[string]model.Source{
			"encrypted": {Type: "sops", SOPS: filepath.Join(root, "declared-sops")},
			"vault":     {Type: "bitwarden", Path: filepath.Join(root, "declared-bw")},
			"account":   {Type: "onepassword", Path: filepath.Join(root, "declared-op")},
			"data":      {Type: "json", Path: filepath.Join(root, "not-an-executable.json")},
		},
		Workloads: map[string]model.Workload{
			"demo": {Snapshot: model.Snapshot{SOPS: filepath.Join(root, "snapshot-sops")}, Command: []string{filepath.Join(root, "declared-command"), "not-a-tool-argument"}, Targets: map[string]model.Target{"unit": {Type: "systemd"}}},
		},
	}
	before := encodeDispatchBoundaryPolicy(t, policy)
	sources := sourceExecutablePaths(policy.Sources)
	sort.Strings(sources)
	wantSources := []string{filepath.Join(root, "declared-bw"), filepath.Join(root, "declared-op"), filepath.Join(root, "declared-sops")}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatal("source tool declarations included value files or omitted fixed vendor helpers")
	}
	wantWorkloads := []string{filepath.Join(root, "snapshot-sops"), filepath.Join(root, "declared-command"), "/usr/bin/systemd-creds"}
	if !reflect.DeepEqual(workloadExecutablePaths(policy.Workloads), wantWorkloads) || !bytes.Equal(before, encodeDispatchBoundaryPolicy(t, policy)) {
		t.Fatal("workload tool declarations changed selector order or mutated the policy")
	}
	// These are declarations, not readiness probes: in particular, never stat
	// or launch the systemd path while collecting its fixed metadata name.
	assertDispatchBoundaryAbsent(t, sources...)
	assertDispatchBoundaryAbsent(t, wantWorkloads[:2]...)
}

func TestDispatchBoundaryContractOwnedMetadataDoesNotExecuteFiles(t *testing.T) {
	root := privateCLIDirectory(t)
	path := filepath.Join(root, "inert-owned-file")
	content := []byte("SYNTHETIC INERT METADATA FIXTURE; NEVER EXECUTE\n")
	if err := platform.CreateProtectedExclusive(path, content); err != nil {
		t.Fatal(err)
	}
	if executableState(path) != "ready" || executableState(root) != "untrusted" {
		t.Fatal("metadata classification confused a protected regular file and a directory")
	}
	want := []doctorDependency{{Kind: "protected_file", Path: path, State: "ready"}, {Kind: "protected_file", Path: root, State: "untrusted"}}
	if !reflect.DeepEqual(doctorMetadata([]string{path, "", path, root}), want) {
		t.Fatal("doctor metadata changed exact deduplication or nonregular refusal")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, content) {
		t.Fatal("metadata inspection modified the owned fixture")
	}
}

type dispatchBoundaryFixture struct {
	config, valuePath, state string
	policyBytes              []byte
}

func newDispatchBoundaryFixture(t *testing.T) dispatchBoundaryFixture {
	t.Helper()
	root := privateCLIDirectory(t)
	fixture := dispatchBoundaryFixture{config: filepath.Join(root, "policy.json"), valuePath: filepath.Join(root, "unopened-values.json"), state: filepath.Join(root, "unused-state")}
	policy := model.Policy{
		Version:   1,
		Sources:   map[string]model.Source{"fixture": {Type: "json", Path: fixture.valuePath}},
		Workloads: map[string]model.Workload{"demo": {Mode: "offline", Bindings: []model.Binding{{Name: "VALUE", Source: "fixture", Field: "VALUE", AllowOffline: true}}}},
		Updates:   &model.Updates{Check: "never"},
	}
	fixture.policyBytes = encodeDispatchBoundaryPolicy(t, policy)
	if err := platform.CreateProtectedExclusive(fixture.config, fixture.policyBytes); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestDispatchBoundaryContractUpdateArgumentsRefuseBeforeEffects(t *testing.T) {
	for _, arguments := range [][]string{
		nil,
		{"SYNTHETIC_PRIVATE_SUBCOMMAND"},
		{"check", "--SYNTHETIC_PRIVATE_FLAG"},
		{"check", "--json=false"},
		{"check", "SYNTHETIC_PRIVATE_ARGUMENT"},
	} {
		fixture := newDispatchBoundaryFixture(t)
		args := append([]string{"--config", fixture.config, "--state-dir", fixture.state, "update"}, arguments...)
		assertDispatchBoundaryCLIRefusal(t, runDispatchBoundary(args), 2, "invalid_policy")
		assertDispatchBoundaryFixtureUnchanged(t, fixture)
	}
}

func TestDispatchBoundaryContractDisabledUpdateNeedsNoValueOrState(t *testing.T) {
	fixture := newDispatchBoundaryFixture(t)
	result := runDispatchBoundary([]string{"--config", fixture.config, "--state-dir", fixture.state, "update", "check", "--json=true"})
	if result.code != 0 || result.diagnostics != "" {
		t.Fatal("valid disabled update check required a provider or state")
	}
	var got update.Result
	decodeDispatchBoundaryJSON(t, result.output, &got)
	if !reflect.DeepEqual(got, update.Result{Status: "disabled", CurrentVersion: "1.2.3"}) {
		t.Fatal("disabled update emitted cache, feed, availability or other non-disabled result")
	}
	assertDispatchBoundaryFixtureUnchanged(t, fixture)
}

func TestDispatchBoundaryContractUpdateMissingRootIsClosed(t *testing.T) {
	root := privateCLIDirectory(t)
	config, state := filepath.Join(root, "missing-policy.json"), filepath.Join(root, "unused-state")
	result := runDispatchBoundary([]string{"--config", config, "--state-dir", state, "update", "check"})
	assertDispatchBoundaryCLIRefusal(t, result, 3, "trust_failure")
	assertDispatchBoundaryAbsent(t, config, state)
}

func assertDispatchBoundaryFixtureUnchanged(t *testing.T, fixture dispatchBoundaryFixture) {
	t.Helper()
	after, err := os.ReadFile(fixture.config)
	if err != nil || !bytes.Equal(after, fixture.policyBytes) {
		t.Fatal("update command changed its protected policy")
	}
	assertDispatchBoundaryAbsent(t, fixture.valuePath, fixture.state)
}

func assertDispatchBoundaryCLIRefusal(t *testing.T, result dispatchBoundaryResult, code int, category string) {
	t.Helper()
	if result.code != code || result.output != "" || result.diagnostics != "{\"error\":\""+category+"\"}\n" {
		t.Fatal("CLI refusal changed its exact status/diagnostic or exposed private inputs")
	}
}

func assertDispatchBoundaryFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	typed, ok := err.(*fault.Error)
	if !ok || typed.Kind != kind || typed.Operation != operation || err.Error() != string(kind)+": "+operation {
		t.Fatal("application refusal did not preserve the direct safe fault")
	}
}

func assertDispatchBoundaryAbsent(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("metadata or refusal path created an input or state destination")
		}
	}
}

func decodeDispatchBoundaryJSON(t *testing.T, text string, destination any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		t.Fatal("metadata did not preserve its closed JSON shape")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatal("metadata emitted another result or trailing output")
	}
}

func encodeDispatchBoundaryPolicy(t *testing.T, policy model.Policy) []byte {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
