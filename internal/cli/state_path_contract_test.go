package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/state"
)

func TestDefaultStatePathUsesCQuayWhenLegacyStateIsAbsent(t *testing.T) {
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	preferred := filepath.Join(configDirectory, "cquay", "state")
	requireMissingPath(t, legacy)
	requireMissingPath(t, preferred)

	store, err := (application{}).store()
	if err != nil || store.Directory != preferred {
		t.Fatalf("default store = %#v, %v; want %q", store, err, preferred)
	}
	requireMissingPath(t, legacy)
	requireMissingPath(t, preferred)
}

func TestDefaultStatePathRejectsLegacyFenceWithoutMutatingEitherStore(t *testing.T) {
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	preferred := filepath.Join(configDirectory, "cquay", "state")
	legacyStore := state.Store{Directory: legacy}
	if err := legacyStore.Write(state.Record{Workload: "demo", State: "blocked", Mode: "hybrid", FallbackFence: true}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(legacy, "demo.json"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = (application{}).store(); !fault.Is(err, fault.Trust) {
		t.Fatalf("legacy state did not fail closed: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(legacy, "demo.json"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("legacy record changed while denying migration: %v", err)
	}
	record, err := legacyStore.Read("demo")
	if err != nil || !record.FallbackFence {
		t.Fatalf("legacy fence was not retained: %#v, %v", record, err)
	}
	requireMissingPath(t, preferred)
}

func TestExplicitLegacyStatePathRemainsAvailableWithoutMigration(t *testing.T) {
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	legacyStore := state.Store{Directory: legacy}
	if err := legacyStore.Write(state.Record{Workload: "demo", State: "blocked", Mode: "offline", FallbackFence: true}); err != nil {
		t.Fatal(err)
	}

	store, err := (application{directory: legacy}).store()
	if err != nil || store.Directory != legacy {
		t.Fatalf("explicit legacy store = %#v, %v; want %q", store, err, legacy)
	}
	record, err := store.Read("demo")
	if err != nil || !record.FallbackFence {
		t.Fatalf("explicit legacy fence was not retained: %#v, %v", record, err)
	}
	requireMissingPath(t, filepath.Join(configDirectory, "cquay", "state"))
}

func TestDefaultStatePathRejectsAmbiguousLegacyPath(t *testing.T) {
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("synthetic-only"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := (application{}).store(); !fault.Is(err, fault.Trust) {
		t.Fatalf("ambiguous legacy path did not fail closed: %v", err)
	}
	requireMissingPath(t, filepath.Join(configDirectory, "cquay", "state"))
}

func TestDefaultStatePathRejectsUnreadableLegacyLookup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL fixtures require platform-specific test ownership")
	}
	configDirectory := syntheticUserConfigDirectory(t)
	legacyParent := filepath.Join(configDirectory, "circularquay")
	if err := os.MkdirAll(legacyParent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(legacyParent, 0); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(legacyParent, 0o700); err != nil {
			t.Errorf("restore legacy fixture permissions: %v", err)
		}
	}()

	if _, err := (application{}).store(); !fault.Is(err, fault.Trust) {
		t.Fatalf("unreadable legacy lookup did not fail closed: %v", err)
	}
}

func syntheticUserConfigDirectory(t *testing.T) string {
	t.Helper()
	home := filepath.Join(privateCLIDirectory(t), "synthetic-home")
	xdg := filepath.Join(home, "xdg-config")
	appData := filepath.Join(home, "appdata")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("APPDATA", appData)
	switch runtime.GOOS {
	case "windows":
		return appData
	case "darwin", "ios":
		return filepath.Join(home, "Library", "Application Support")
	default:
		return xdg
	}
}

func requireMissingPath(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path unexpectedly exists or cannot be checked: %q: %v", path, err)
	}
}
