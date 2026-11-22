package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/update"
)

func cliCompletionConfig(t *testing.T) (application, []byte) {
	t.Helper()
	path := filepath.Join(privateCLIDirectory(t), "cq.json")
	data := bootstrapPolicy()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	return application{config: path, directory: filepath.Join(filepath.Dir(path), "state"), runner: Runner{Out: io.Discard, Err: io.Discard}}, data
}

func TestCLICompletionMutationCommitFencesPreserveNamedBytes(t *testing.T) {
	for _, phase := range []string{"changed", "cancel", "write"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionConfigUpdateCompletionCase1(t, phase)
		})
	}
}

func TestCLICompletionFormatterCASCancelAndWriteFailure(t *testing.T) {
	for _, phase := range []string{"read", "changed", "cancel", "write"} {
		t.Run(phase, func(t *testing.T) {
			cliCompletionConfigUpdateCompletionCase2(t, phase)
		})
	}
}

func TestCLICompletionAdvisoryOnlyCheckerKeepsContinuationContract(t *testing.T) {
	for _, status := range []string{"error", "available", "current"} {
		t.Run(status, func(t *testing.T) {
			cliCompletionConfigUpdateCompletionCase3(t, status)
		})
	}
}

func TestCLICompletionAdvisoryStoreRefusalPrecedesChecker(t *testing.T) {
	app := application{runner: Runner{In: bytes.NewReader(nil), Err: io.Discard}, paths: &applicationPathOperations{configurationDirectory: func() (string, error) { return "", os.ErrNotExist }}}
	err := app.checkUpdatesWithOperations(context.Background(), "plan", &model.Updates{Check: "interactive"}, func(io.Reader, io.Writer) bool { return true }, func(context.Context, *model.Updates, update.Options) (update.Result, error) {
		t.Fatal("missing store reached update provider")
		return update.Result{}, nil
	})
	cliCompletionFault(t, err, fault.Apply, stateDirectoryDiagnostic)
}

func cliCompletionConfigUpdateCompletionCase1(t *testing.T, phase string) {
	t.Helper()
	p := newCLIConfigCompletionProbe(t, phase)
	code, err := p.app.mutateConfigWithOperations(p.ctx, "", false, false, func(*any) error { return nil }, configurationCommitOperations{p.unchanged, p.context, p.write})
	p.assertRefusal(err)
	expected := map[string][]string{"changed": {"unchanged"}, "cancel": {"unchanged", "context"}, "write": {"unchanged", "context", "write"}}[phase]
	p.assertNamedBytesAndCalls(expected)
	if code != 0 {
		t.Fatal("mutation inner code changed")
	}
}
func cliCompletionConfigUpdateCompletionCase2(t *testing.T, phase string) {
	t.Helper()
	p := newCLIConfigCompletionProbe(t, phase)
	changed, err := p.app.formatConfigWithOperations(p.ctx, configurationFormatOperations{p.read, p.context, p.write})
	if phase == "read" {
		cliCompletionFault(t, err, fault.Denied, errorConfigChanged)
	} else {
		p.assertRefusal(err)
	}
	expected := map[string][]string{"read": {"read"}, "changed": {"read"}, "cancel": {"read", "context"}, "write": {"read", "context", "write"}}[phase]
	p.assertNamedBytesAndCalls(expected)
	if changed {
		t.Fatal("refused formatting reported a change")
	}
}

type cliConfigCompletionProbe struct {
	t        *testing.T
	phase    string
	app      application
	ctx      context.Context
	cancel   context.CancelFunc
	expected []byte
	calls    []string
}

func newCLIConfigCompletionProbe(t *testing.T, phase string) *cliConfigCompletionProbe {
	t.Helper()
	app, original := cliCompletionConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &cliConfigCompletionProbe{t: t, phase: phase, app: app, ctx: ctx, cancel: cancel, expected: append([]byte(nil), original...)}
}
func (p *cliConfigCompletionProbe) drift(path string) {
	if p.phase != "changed" {
		return
	}
	p.expected = append(p.expected, ' ')
	if err := platform.WriteProtected(path, p.expected); err != nil {
		p.t.Fatal(err)
	}
}
func (p *cliConfigCompletionProbe) unchanged(path, digest string, repair bool) error {
	p.calls = append(p.calls, "unchanged")
	p.drift(path)
	err := p.app.ensureConfigUnchanged(path, digest, repair)
	if p.phase == "cancel" {
		p.cancel()
	}
	return err
}
func (p *cliConfigCompletionProbe) read(path string, secret bool) ([]byte, error) {
	p.calls = append(p.calls, "read")
	if path != p.app.config || secret {
		p.t.Fatal("wrong final read")
	}
	if p.phase == "read" {
		return nil, os.ErrPermission
	}
	p.drift(path)
	data, err := platform.ReadProtected(path, false)
	if p.phase == "cancel" {
		p.cancel()
	}
	return data, err
}
func (p *cliConfigCompletionProbe) context(ctx context.Context) error {
	p.calls = append(p.calls, "context")
	return configurationWriteContext(ctx)
}
func (p *cliConfigCompletionProbe) write(path string, data []byte) error {
	p.calls = append(p.calls, "write")
	if path != p.app.config || len(data) == 0 {
		p.t.Fatal("write arguments changed")
	}
	return fault.New(fault.Apply, "owned config write")
}
func (p *cliConfigCompletionProbe) assertRefusal(err error) {
	switch p.phase {
	case "changed":
		cliCompletionFault(p.t, err, fault.Denied, errorConfigChanged)
	case "cancel":
		cliCompletionFault(p.t, err, fault.Unavailable, "configuration cancelled")
	case "write":
		cliCompletionFault(p.t, err, fault.Apply, "owned config write")
	}
}
func (p *cliConfigCompletionProbe) assertNamedBytesAndCalls(expected []string) {
	data, err := os.ReadFile(p.app.config)
	if err != nil || !bytes.Equal(data, p.expected) || !reflect.DeepEqual(p.calls, expected) {
		p.t.Fatal("refused write changed named bytes or ordering")
	}
}

func cliCompletionConfigUpdateCompletionCase3(t *testing.T, status string) {
	t.Helper()
	var diagnostics bytes.Buffer
	app := application{directory: filepath.Join(privateCLIDirectory(t), "state"), buildVersion: "1.0.0", runner: Runner{In: bytes.NewReader(nil), Out: io.Discard, Err: &diagnostics}}
	p := &cliCompletionUpdateProbe{t: t, status: status, config: &model.Updates{Check: "interactive"}, directory: app.directory}
	err := app.checkUpdatesWithOperations(context.Background(), "plan", p.config, func(io.Reader, io.Writer) bool { return true }, p.check)
	if err != nil || p.calls != 1 {
		t.Fatal("advisory result changed command admission")
	}
	p.assertNotice(diagnostics.String())
}

type cliCompletionUpdateProbe struct {
	t                 *testing.T
	status, directory string
	config            *model.Updates
	calls             int
}

func (p *cliCompletionUpdateProbe) check(_ context.Context, config *model.Updates, options update.Options) (update.Result, error) {
	p.calls++
	if config != p.config || options.Version != "1.0.0" || options.StateDirectory != p.directory {
		p.t.Fatal("checker input changed")
	}
	if p.status == "error" {
		return update.Result{}, fault.New(fault.Unavailable, "synthetic private update failure")
	}
	return update.Result{Status: p.status, Available: "1.0.1", InstructionURL: "https://example.invalid/upgrade"}, nil
}
func (p *cliCompletionUpdateProbe) assertNotice(got string) {
	switch p.status {
	case "error":
		if got != "CircularQuay could not check for updates; continuing.\n" {
			p.t.Fatal("error text disclosed")
		}
	case "available":
		if !strings.Contains(got, "A newer CircularQuay release is available: 1.0.1\n") {
			p.t.Fatal("public update notice missing")
		}
	case "current":
		if got != "" {
			p.t.Fatal("current version emitted notice")
		}
	}
}

func TestCLICompletionPublicUpdateTrustFailurePrecedesAnyRequest(t *testing.T) {
	root := privateCLIDirectory(t)
	app := application{directory: filepath.Join(root, "state"), buildVersion: "1.0.0", runner: Runner{Out: io.Discard, Err: io.Discard}}
	policy := model.Policy{Updates: &model.Updates{Check: "interactive", Channel: "stable", IntervalHours: 1, TimeoutSeconds: 1, TrustStoreFile: filepath.Join(root, "missing.pem"), TrustStoreSHA256: strings.Repeat("a", 64)}}
	code, err := app.updateCheckCommand(context.Background(), nil, policy)
	cliCompletionFault(t, err, fault.Trust, "trust store file")
	if code != 0 {
		t.Fatal("public update trust refusal changed code")
	}
}
