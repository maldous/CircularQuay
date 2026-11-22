package source

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

const sopsOrchestrationDocument = `{"KEY":"ENC[SYNTHETIC]","sops":{"mac":"synthetic","age":[{}]}}`

func TestSOPSOrchestrationBoundaryContractInvalidPlaintextCannotReleaseValue(t *testing.T) {
	config := sopsOrchestrationSource(t)
	calls := 0
	run := func(context.Context, string, []string, []byte, string, string, []string) ([]byte, error) {
		calls++
		return []byte(`{"KEY":42}`), nil
	}
	value, err := readSOPSWithRunner(context.Background(), config, model.Binding{Name: "KEY"}, run)
	assertRejectedProviderValue(t, value, err, fault.Invalid, "PRIVATE_DATA")
	if calls != 1 {
		t.Fatalf("invalid plaintext was retried: %d", calls)
	}
	assertSOPSOrchestrationBytes(t, config.Path, []byte(sopsOrchestrationDocument))
}

func TestSOPSOrchestrationBoundaryContractSnapshotRefusalsKeepOwnedBytes(t *testing.T) {
	config := model.Snapshot{Path: sopsOrchestrationSource(t).Path, Recipients: []string{"age1synthetic"}}
	for _, returned := range []string{`{"sops":{"age":[{}]}}`, `not-json`} {
		run := func(context.Context, string, []string, []byte, string, string, []string) ([]byte, error) {
			return []byte(returned), nil
		}
		err := encryptSnapshotWithRunner(context.Background(), config, map[string]string{"KEY": "SYNTHETIC_VALUE"}, run)
		if err == nil {
			t.Fatal("invalid encrypted metadata replaced the owned snapshot")
		}
		assertSOPSOrchestrationBytes(t, config.Path, []byte(sopsOrchestrationDocument))
	}
}

func TestSOPSOrchestrationBoundaryContractSnapshotRunnerFailureIsNotPlaintext(t *testing.T) {
	config := model.Snapshot{Path: sopsOrchestrationSource(t).Path}
	wanted := fault.New(fault.Denied, "SOPS operation failed")
	run := func(context.Context, string, []string, []byte, string, string, []string) ([]byte, error) {
		return []byte("PRIVATE_DATA"), wanted
	}
	values, err := decryptSnapshotWithRunner(context.Background(), config, run)
	if values != nil || err != wanted || strings.Contains(err.Error(), "PRIVATE_DATA") {
		t.Fatalf("runner failure released partial plaintext or changed safe error: %v", err)
	}
	assertSOPSOrchestrationBytes(t, config.Path, []byte(sopsOrchestrationDocument))
}

func TestSOPSOrchestrationBoundaryContractSyntheticRunnerKeepsArgumentsAndInput(t *testing.T) {
	config := sopsOrchestrationSource(t)
	calls := 0
	run := func(ctx context.Context, executable string, arguments []string, input []byte, ageFile, tokenFile string, allowlist []string) ([]byte, error) {
		calls++
		assertSOPSOrchestrationRequest(t, ctx, config, executable, arguments, input, ageFile, tokenFile, allowlist)
		return []byte(`{"KEY":"SYNTHETIC_VALUE"}`), nil
	}
	value, err := readSOPSWithRunner(context.Background(), config, model.Binding{Name: "KEY"}, run)
	if err != nil || value.Data != "SYNTHETIC_VALUE" || value.Version != sopsVersion([]byte(sopsOrchestrationDocument)) || !value.AuditVerified || calls != 1 {
		t.Fatalf("synthetic SOPS orchestration result changed: %#v, %v", value, err)
	}
}

func assertSOPSOrchestrationRequest(t *testing.T, ctx context.Context, config model.Source, executable string, arguments []string, input []byte, ageFile, tokenFile string, allowlist []string) {
	t.Helper()
	if ctx.Err() != nil || executable != config.SOPS || !reflect.DeepEqual(arguments, sopsDecryptArgs()) {
		t.Fatal("orchestration changed executable selection, context or fixed decrypt arguments")
	}
	if string(input) != sopsOrchestrationDocument || ageFile != "" || tokenFile != "" || len(allowlist) != 0 {
		t.Fatal("orchestration changed admitted document or discovered credentials")
	}
}

func TestSOPSOrchestrationBoundaryContractExecutionRefusalsCleanOnlyOwnedHome(t *testing.T) {
	for _, test := range []struct {
		stage string
		kind  fault.Kind
	}{
		{"validate", fault.Trust},
		{"home", fault.Apply},
		{"environment", fault.Trust},
		{"run", fault.Denied},
	} {
		t.Run(test.stage, func(t *testing.T) {
			assertSOPSExecutionRefusal(t, test.stage, test.kind)
		})
	}
}

func assertSOPSExecutionRefusal(t *testing.T, stage string, kind fault.Kind) {
	t.Helper()
	fixture := newSOPSExecutionFixture(t)
	fixture.fail = stage
	data, err := runSOPSWithOperations(context.Background(), fixture.executable, sopsDecryptArgs(), []byte("SYNTHETIC_INPUT"), "", "", nil, fixture.operations())
	if data != nil || !fault.Is(err, kind) || strings.Contains(err.Error(), "PRIVATE_DATA") {
		t.Fatalf("execution refusal released output or unsafe text: %q, %v", data, err)
	}
	if fixture.steps[len(fixture.steps)-1] != stage {
		t.Fatalf("execution continued after refusal: %v", fixture.steps)
	}
	fixture.assertHomeLifetime(t, stage != "validate" && stage != "home")
}

func TestSOPSOrchestrationBoundaryContractOutputLimitAndPositiveControl(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "bounded", true: "overflow"}[overflow], func(t *testing.T) {
			fixture := newSOPSExecutionFixture(t)
			fixture.output = []byte("SYNTHETIC_OUTPUT")
			if overflow {
				fixture.output = bytes.Repeat([]byte("x"), platform.MaxDocumentSize+1)
			}
			data, err := runSOPSWithOperations(context.Background(), fixture.executable, sopsDecryptArgs(), []byte("SYNTHETIC_INPUT"), "", "", nil, fixture.operations())
			assertSOPSExecutionOutput(t, data, err, overflow)
			fixture.assertHomeLifetime(t, true)
		})
	}
}

func assertSOPSExecutionOutput(t *testing.T, data []byte, err error, overflow bool) {
	t.Helper()
	if overflow {
		if data != nil || !fault.Is(err, fault.Invalid) {
			t.Fatal("overflow returned partial plaintext")
		}
		return
	}
	if err != nil || string(data) != "SYNTHETIC_OUTPUT" {
		t.Fatalf("bounded synthetic output changed: %q, %v", data, err)
	}
}

func sopsOrchestrationSource(t *testing.T) model.Source {
	t.Helper()
	root := privateSourceDirectory(t, "sops-orchestration")
	path := filepath.Join(root, "source.json")
	if err := platform.WriteProtected(path, []byte(sopsOrchestrationDocument)); err != nil {
		t.Fatal(err)
	}
	return model.Source{Path: path, SOPS: filepath.Join(root, "inert-not-executed"), AuditVerified: true}
}

func assertSOPSOrchestrationBytes(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, expected) {
		t.Fatal("owned ciphertext bytes changed")
	}
}

type sopsExecutionFixture struct {
	t          *testing.T
	executable string
	home       string
	fail       string
	steps      []string
	output     []byte
	removed    bool
}

func newSOPSExecutionFixture(t *testing.T) *sopsExecutionFixture {
	t.Helper()
	root := privateSourceDirectory(t, "sops-execution")
	fixture := &sopsExecutionFixture{t: t, executable: filepath.Join(root, "inert"), home: filepath.Join(root, "home")}
	if err := platform.WriteProtected(fixture.executable, []byte("OWNED_NOT_AN_EXECUTABLE")); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func (fixture *sopsExecutionFixture) step(name string) error {
	fixture.steps = append(fixture.steps, name)
	if fixture.fail == name {
		return fault.New(fault.Trust, "synthetic_sops_refusal")
	}
	return nil
}

func (fixture *sopsExecutionFixture) operations() sopsExecution {
	return sopsExecution{validate: fixture.validate, makeHome: fixture.makeHome, removeHome: fixture.removeHome, environment: fixture.environment, run: fixture.run}
}

func (fixture *sopsExecutionFixture) validate(path string) error {
	fixture.t.Helper()
	if path != fixture.executable {
		fixture.t.Fatal("validation selected an unrelated path")
	}
	return fixture.step("validate")
}

func (fixture *sopsExecutionFixture) makeHome(parent, pattern string) (string, error) {
	fixture.t.Helper()
	if parent != "" || pattern != "cq-sops-*" {
		fixture.t.Fatal("isolation selected unexpected parameters")
	}
	if err := fixture.step("home"); err != nil {
		return "", err
	}
	return fixture.home, os.Mkdir(fixture.home, 0o700)
}

func (fixture *sopsExecutionFixture) removeHome(path string) error {
	fixture.t.Helper()
	if path != fixture.home {
		fixture.t.Fatal("cleanup selected an unowned path")
	}
	fixture.removed = true
	return os.RemoveAll(path)
}

func (fixture *sopsExecutionFixture) environment(home, ageFile, tokenFile string, allowlist []string) ([]string, error) {
	fixture.t.Helper()
	if home != fixture.home || ageFile != "" || tokenFile != "" || len(allowlist) != 0 {
		fixture.t.Fatal("isolation received undeclared identity material")
	}
	return []string{"SYNTHETIC_ENVIRONMENT=1"}, fixture.step("environment")
}

func (fixture *sopsExecutionFixture) run(ctx context.Context, command sopsCommand) error {
	fixture.t.Helper()
	if _, ok := ctx.Deadline(); !ok || command.executable != fixture.executable || command.home != fixture.home {
		fixture.t.Fatal("synthetic command lost deadline or owned selection")
	}
	if !reflect.DeepEqual(command.arguments, sopsDecryptArgs()) || string(command.input) != "SYNTHETIC_INPUT" || !reflect.DeepEqual(command.environment, []string{"SYNTHETIC_ENVIRONMENT=1"}) {
		fixture.t.Fatal("synthetic command changed input, argv or environment")
	}
	if err := fixture.step("run"); err != nil {
		return err
	}
	_, err := command.output.Write(fixture.output)
	return err
}

func (fixture *sopsExecutionFixture) assertHomeLifetime(t *testing.T, created bool) {
	t.Helper()
	if fixture.removed != created {
		t.Fatal("cleanup did not match the actually created isolation home")
	}
	if _, err := os.Lstat(fixture.home); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary home survived refusal/completion: %v", err)
	}
	assertSOPSOrchestrationBytes(t, fixture.executable, []byte("OWNED_NOT_AN_EXECUTABLE"))
}

var _ io.Writer = (*cappedBuffer)(nil)
