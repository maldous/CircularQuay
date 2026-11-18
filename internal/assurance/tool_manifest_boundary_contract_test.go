package assurance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestToolManifestBoundaryContractParserWithholdsInvalidSizedDeclarations(t *testing.T) {
	for _, data := range [][]byte{nil, bytes.Repeat([]byte("x"), maxToolManifestBytes+1)} {
		assertToolManifestBoundaryParseRefused(t, data)
	}
}

func TestToolManifestBoundaryContractDigestShapePrecedesPathAccess(t *testing.T) {
	manifest := toolManifestBoundaryDeclaration(t)
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ParseToolManifest(data); err != nil {
		t.Fatalf("synthetic declaration shape rejected: %v", err)
	}
	for _, mutate := range []func(*ToolManifest){
		func(value *ToolManifest) { value.Tools[0].SHA256 = "short" },
		func(value *ToolManifest) { value.Tools[0].VersionOutputSHA256 = "short" },
	} {
		candidate := cloneManifest(t, manifest)
		mutate(&candidate)
		data, err = json.Marshal(candidate)
		if err != nil {
			t.Fatal(err)
		}
		assertToolManifestBoundaryParseRefused(t, data)
	}
}

func TestToolManifestBoundaryContractProtectedReaderHasInclusiveLimit(t *testing.T) {
	path := filepath.Join(privateAssuranceDirectory(t, "bounded-manifest"), "manifest.json")
	data := bytes.Repeat([]byte(" "), maxToolManifestBytes)
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
	canonical, got, err := readToolManifest(path)
	if err != nil || canonical != path || !bytes.Equal(got, data) {
		t.Fatalf("exact byte limit rejected: path=%q bytes=%d err=%v", canonical, len(got), err)
	}
	if err = platform.WriteProtected(path, append(data, 'x')); err != nil {
		t.Fatal(err)
	}
	canonical, got, err = readToolManifest(path)
	if err == nil || canonical != "" || got != nil {
		t.Fatalf("oversized manifest was not withheld: path=%q bytes=%d err=%v", canonical, len(got), err)
	}
}

func TestToolManifestBoundaryContractLoadRejectsParseAndPlatformBeforeTools(t *testing.T) {
	path := filepath.Join(privateAssuranceDirectory(t, "malformed-manifest"), "manifest.json")
	if err := platform.WriteProtected(path, []byte("{bad")); err != nil {
		t.Fatal(err)
	}
	assertToolManifestBoundaryLoadRefused(t, path, fixtureLocations(t), "invalid tool manifest")
	manifest := toolManifestBoundaryDeclaration(t)
	manifest.Platform = "not-current/fixture"
	assertToolManifestBoundaryLoadRefused(t, writeManifest(t, manifest), fixtureLocations(t), "tool manifest platform mismatch")
}

func TestToolManifestBoundaryContractInvalidLocationsPrecedeToolOpen(t *testing.T) {
	manifest := toolManifestBoundaryDeclaration(t)
	path := writeManifest(t, manifest)
	locations := fixtureLocations(t)
	file := filepath.Join(privateAssuranceDirectory(t, "not-directory"), "file")
	if err := platform.WriteProtected(file, []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []ToolLocations{
		{Source: "relative", Output: locations.Output},
		{Source: file, Output: locations.Output},
		{Source: locations.Source, Output: "relative"},
		{Source: locations.Source, Output: file},
	} {
		assertToolManifestBoundaryLoadRefused(t, path, invalid, "invalid tool manifest location")
	}
	// The otherwise admitted declaration reaches the deliberately absent tool.
	// No tool bytes exist, and none of these cases can execute a tool.
	assertToolManifestBoundaryLoadRefused(t, path, locations, "invalid tool protection")
}

func TestToolManifestBoundaryContractCanonicalPathsRefuseInvalidShapes(t *testing.T) {
	root := privateAssuranceDirectory(t, "canonical-paths")
	for _, path := range []string{"relative", root, filepath.Join(root, "missing")} {
		if got, err := canonicalRegularPath(path); err == nil || got != "" {
			t.Fatalf("invalid regular path accepted: %q, %v", got, err)
		}
	}
	if path, data, err := readToolManifest("relative"); err == nil || path != "" || data != nil {
		t.Fatalf("relative manifest read: %q, %d bytes, %v", path, len(data), err)
	}
}

func TestToolManifestBoundaryContractMissingToolHashWithholdsDigest(t *testing.T) {
	root := privateAssuranceDirectory(t, "missing-tool")
	for _, path := range []string{filepath.Join(root, "missing"), filepath.Join(root, "missing-directory", "tool")} {
		if got, err := executableSHA256(path); err == nil || got != "" {
			t.Fatalf("missing tool produced a digest: %q, %v", got, err)
		}
	}
}

func TestToolManifestBoundaryContractClosedDescriptorWithholdsDigest(t *testing.T) {
	path, file := toolManifestBoundaryFile(t)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if sum, count, err := boundedToolHash(file); err == nil || sum != "" || count != 0 {
		t.Fatalf("closed descriptor produced a digest: %q, %d, %v", sum, count, err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "synthetic-tool-bytes" {
		t.Fatalf("descriptor refusal changed the named fixture: %q, %v", data, err)
	}
}

func TestToolManifestBoundaryContractPostReadFenceRejectsSizeDrift(t *testing.T) {
	path, file := toolManifestBoundaryFile(t)
	opened, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err = unchangedToolFile(path, opened, file, opened.Size()); err != nil {
		t.Fatalf("stable descriptor rejected: %v", err)
	}
	if err = unchangedToolFile(path, opened, file, opened.Size()-1); err == nil {
		t.Fatal("partial read accepted as complete tool identity")
	}
	if err = file.Truncate(opened.Size() + 1); err != nil {
		t.Fatal(err)
	}
	if err = unchangedToolFile(path, opened, file, opened.Size()); err == nil {
		t.Fatal("changed descriptor size accepted as stable tool identity")
	}
}

func toolManifestBoundaryDeclaration(t *testing.T) ToolManifest {
	t.Helper()
	path := filepath.Join(privateAssuranceDirectory(t, "unopened-tool"), "git")
	tool := ToolEntry{ID: "git", Path: path, SHA256: digest([]byte("tool")), Version: "synthetic-v1", VersionOutputSHA256: digest([]byte("version"))}
	return ToolManifest{Version: 1, Platform: runtime.GOOS + "/" + runtime.GOARCH, Tools: []ToolEntry{tool}}
}

func assertToolManifestBoundaryParseRefused(t *testing.T, data []byte) {
	t.Helper()
	got, err := ParseToolManifest(data)
	if err == nil || err.Error() != "invalid tool manifest" || !reflect.DeepEqual(got, ToolManifest{}) {
		t.Fatalf("invalid declaration not withheld: %+v, %v", got, err)
	}
}

func assertToolManifestBoundaryLoadRefused(t *testing.T, path string, locations ToolLocations, message string) {
	t.Helper()
	got, err := LoadToolManifest(path, locations)
	if err == nil || err.Error() != message || !reflect.DeepEqual(got, VerifiedToolManifest{}) {
		t.Fatalf("invalid binding not withheld: %+v, %v", got, err)
	}
}

func toolManifestBoundaryFile(t *testing.T) (string, *os.File) {
	t.Helper()
	path := filepath.Join(privateAssuranceDirectory(t, "descriptor-tool"), "tool")
	if err := platform.WriteProtected(path, []byte("synthetic-tool-bytes")); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	return path, file
}
