package source

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestVendorExecutionBoundaryContractIsolationFailureNeverRunsCommand(t *testing.T) {
	fixture := newVendorExecutionFixture(t)
	fixture.failHome = true
	value, err := readVendorWithOperations(context.Background(), fixture.source, fixture.binding, fixture.operations())
	assertRejectedProviderValue(t, value, err, fault.Apply, "PRIVATE_DATA")
	if fixture.runs != 0 || fixture.removed {
		t.Fatal("failed home creation reached a command or removed an unowned path")
	}
}

func TestVendorExecutionBoundaryContractPartialFailedOutputIsNotAValue(t *testing.T) {
	fixture := newVendorExecutionFixture(t)
	fixture.failRun = true
	value, err := readVendorWithOperations(context.Background(), fixture.source, fixture.binding, fixture.operations())
	assertRejectedProviderValue(t, value, err, fault.Denied, "PRIVATE_DATA")
	fixture.assertRemoved(t)
}

func TestVendorExecutionBoundaryContractExplicitProjectionAndPositiveControl(t *testing.T) {
	fixture := newVendorExecutionFixture(t)
	value, err := readVendorWithOperations(context.Background(), fixture.source, fixture.binding, fixture.operations())
	if err != nil || value.Data != "SYNTHETIC_VENDOR_VALUE" || !value.AuditVerified {
		t.Fatalf("bounded vendor result changed: %#v, %v", value, err)
	}
	fixture.assertRemoved(t)
}

type vendorExecutionFixture struct {
	t        *testing.T
	source   model.Source
	binding  model.Binding
	home     string
	failHome bool
	failRun  bool
	runs     int
	removed  bool
}

func newVendorExecutionFixture(t *testing.T) *vendorExecutionFixture {
	t.Helper()
	root := privateSourceDirectory(t, "vendor-execution")
	return &vendorExecutionFixture{
		t: t, home: filepath.Join(root, "home"),
		source:  model.Source{Type: "onepassword", Project: "synthetic-vault", Path: filepath.Join(root, "inert-never-executed"), TokenFile: filepath.Join(root, "synthetic-token-not-read"), AuditVerified: true},
		binding: model.Binding{Resource: "op://synthetic-vault/item/field"},
	}
}

func (fixture *vendorExecutionFixture) operations() vendorExecution {
	return vendorExecution{validate: fixture.validate, credential: fixture.credential, makeHome: fixture.makeHome, removeHome: fixture.removeHome, lookup: fixture.lookup, run: fixture.run}
}

func (fixture *vendorExecutionFixture) validate(path string) error {
	fixture.t.Helper()
	if path != fixture.source.Path {
		fixture.t.Fatal("validation escaped the explicit synthetic selection")
	}
	// This fake is not executable validation or public readiness evidence.
	return nil
}

func (fixture *vendorExecutionFixture) credential(path string) (string, error) {
	fixture.t.Helper()
	if path != fixture.source.TokenFile {
		fixture.t.Fatal("credential collaborator selected an undeclared file")
	}
	return "SYNTHETIC_TOKEN", nil
}

func (fixture *vendorExecutionFixture) makeHome(parent, pattern string) (string, error) {
	fixture.t.Helper()
	if parent != "" || pattern != "cq-vendor-" {
		fixture.t.Fatal("vendor isolation parameters changed")
	}
	if fixture.failHome {
		return "", errors.New("PRIVATE_DATA")
	}
	return fixture.home, os.Mkdir(fixture.home, 0o700)
}

func (fixture *vendorExecutionFixture) removeHome(path string) error {
	fixture.t.Helper()
	if path != fixture.home {
		fixture.t.Fatal("vendor cleanup selected an unowned path")
	}
	fixture.removed = true
	return os.RemoveAll(path)
}

func (fixture *vendorExecutionFixture) lookup(name string) (string, bool) {
	fixture.t.Helper()
	if name != "SystemRoot" {
		fixture.t.Fatal("vendor environment requested an undeclared ambient name")
	}
	return "SYNTHETIC_SYSTEM_ROOT", true
}

func (fixture *vendorExecutionFixture) run(ctx context.Context, command vendorInvocation) error {
	fixture.t.Helper()
	fixture.runs++
	fixture.assertInvocation(ctx, command)
	if fixture.failRun {
		if _, err := command.output.Write([]byte("PRIVATE_DATA")); err != nil {
			fixture.t.Fatal(err)
		}
		return errors.New("PRIVATE_DATA")
	}
	_, err := command.output.Write([]byte("SYNTHETIC_VENDOR_VALUE"))
	return err
}

func (fixture *vendorExecutionFixture) assertInvocation(ctx context.Context, command vendorInvocation) {
	fixture.t.Helper()
	if _, ok := ctx.Deadline(); !ok || command.executable != fixture.source.Path || command.home != fixture.home {
		fixture.t.Fatal("vendor command lost deadline or owned selections")
	}
	if !reflect.DeepEqual(command.arguments, []string{"read", fixture.binding.Resource, "--no-newline"}) {
		fixture.t.Fatal("vendor command changed fixed argument boundaries")
	}
	expected := []string{"HOME=" + fixture.home, "USERPROFILE=" + fixture.home, "APPDATA=" + fixture.home, "LOCALAPPDATA=" + fixture.home, "NO_COLOR=1", "OP_SERVICE_ACCOUNT_TOKEN=SYNTHETIC_TOKEN", "SystemRoot=SYNTHETIC_SYSTEM_ROOT"}
	if !reflect.DeepEqual(command.environment, expected) {
		fixture.t.Fatal("vendor environment inherited undeclared values or changed isolated identity")
	}
}

func (fixture *vendorExecutionFixture) assertRemoved(t *testing.T) {
	t.Helper()
	if fixture.runs != 1 || !fixture.removed {
		t.Fatal("vendor operation retried or retained its owned home")
	}
	if _, err := os.Lstat(fixture.home); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("vendor home survived completion: %v", err)
	}
}
