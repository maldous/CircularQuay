package policy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func privatePolicyDirectory(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(base, "cquay-policy-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	if err := platform.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	return directory
}

func policyBytes(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(validPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLoadStrictJSONAndRelativePaths(t *testing.T) {
	dir := privatePolicyDirectory(t)
	p := validPolicy()
	changeSource(&p, func(s *model.Source) {
		s.Type = "sops"
		s.Path = "values.json"
		s.AgeIdentityFile = "age"
		s.SOPS = "./bin/sops"
		s.Identity = "none"
	})
	changeWorkload(&p, func(w *model.Workload) {
		w.Directory = "work"
		w.Command = []string{"./bin/worker"}
		w.Snapshot.Path = "cache"
		w.Snapshot.Recipients = []string{"age1cy0su9fwf3gf9mw868g5yut09p6nytfmmnktexz2ya5uqg9vl9sss4euqm"}
		w.Snapshot.IdentityFile = "snapshot-age"
		w.Snapshot.SOPS = "./bin/sops"
		w.Bindings[0].AllowPublish = true
		w.Targets = map[string]model.Target{"sink": {Type: "file", Path: "out", AcceptSnapshot: true}}
	})
	p.MCP.AuditPath = "events.jsonl"
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "policy.json")
	if err = os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{loaded.Sources["fixture"].Path, loaded.Sources["fixture"].AgeIdentityFile, loaded.Sources["fixture"].SOPS, loaded.Workloads["demo"].Directory, loaded.Workloads["demo"].Command[0], loaded.Workloads["demo"].Snapshot.Path, loaded.Workloads["demo"].Snapshot.IdentityFile, loaded.Workloads["demo"].Snapshot.SOPS, loaded.Workloads["demo"].Targets["sink"].Path, loaded.MCP.AuditPath} {
		if !filepath.IsAbs(path) || !strings.HasPrefix(path, dir+string(filepath.Separator)) {
			t.Fatal("relative path was not anchored")
		}
	}
}

func TestLoadRejectsInvalidPinnedAuthorityTrustStore(t *testing.T) {
	directory := privatePolicyDirectory(t)
	trustStore := filepath.Join(directory, "authority-roots.pem")
	trustBytes := []byte("synthetic non-certificate trust store")
	if err := platform.WriteProtected(trustStore, trustBytes); err != nil {
		t.Fatal(err)
	}
	policy := namedIdentityPolicy()
	authority := policy.MCP.Authorities["workforce"]
	digest := sha256.Sum256(trustBytes)
	authority.TrustStoreFile = trustStore
	authority.TrustStoreSHA256 = hex.EncodeToString(digest[:])
	policy.MCP.Authorities["workforce"] = authority
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "policy.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("invalid pinned authority trust store accepted at load")
	}
}

func TestParseBindsPathWithoutFilesystemEffects(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "not-created")
	path := filepath.Join(directory, "policy.json")
	policy := validPolicy()
	changeSource(&policy, func(source *model.Source) { source.Path = "values.json" })
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data, path)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Sources["fixture"].Path != filepath.Join(directory, "values.json") {
		t.Fatal("relative source was not bound to intended policy path")
	}
	if _, err = os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("Parse changed the filesystem")
	}
}

func TestParseRejectsOversizedDataAndDestinationCollision(t *testing.T) {
	if _, err := Parse(bytes.Repeat([]byte(" "), maximumPolicyBytes+1), "/private/policy.json"); err == nil {
		t.Fatal("oversized policy accepted")
	}
	policy := validPolicy()
	changeWorkload(&policy, func(workload *model.Workload) {
		workload.Bindings[0].AllowPublish = true
		workload.Targets = map[string]model.Target{"sink": {Type: "file", Path: "policy.json", AcceptSnapshot: true}}
	})
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(data, filepath.Join(t.TempDir(), "policy.json")); err == nil {
		t.Fatal("policy destination writer accepted")
	}
}

func TestDecodeRejectsAmbiguousPolicies(t *testing.T) {
	valid := string(policyBytes(t))
	invalidUTF8 := bytes.Replace(policyBytes(t), []byte(`"fixture"`), []byte{'"', 0xff, '"'}, 1)
	inputs := []string{
		`{"version":1,"version":2}`, `{"unknown":"SYNTHETIC_DO_NOT_ECHO"}`,
		valid + valid, valid + ` false`, `{"version":`, `{"version":1,"sources":[]}`,
		`[`, "", "---\nversion: 1\n---\nversion: 1\n",
		"version: 1\nversion: 2\n", "version: &value 1\n", "version: *missing\n",
		"version: !untrusted 1\n", "unknown: rejected\n",
		strings.Repeat("[", 34) + strings.Repeat("]", 34),
		strings.Replace(valid, `"version":1`, `"Version":1`, 1),
		strings.Replace(valid, `"version":1`, `"version":1,"Version":1`, 1),
		strings.TrimSuffix(valid, "}") + `,"grants":null}`,
	}
	for _, input := range inputs {
		var result model.Policy
		if err := decodePolicy([]byte(input), &result); err == nil {
			t.Fatalf("unsafe policy accepted (%d bytes)", len(input))
		}
	}
	var result model.Policy
	if err := decodePolicy(invalidUTF8, &result); err == nil {
		t.Fatal("invalid UTF-8 policy accepted")
	}
}

func TestJSONNullDetectionTraversesArraysAndObjects(t *testing.T) {
	for _, test := range []struct {
		name string
		data string
		want bool
	}{
		{"scalar", `{"value":"present"}`, false},
		{"array", `{"value":["present",null]}`, true},
		{"object", `{"value":{"nested":null}}`, true},
		{"nested", `{"value":[{"nested":[null]}]}`, true},
		{"malformed", `{`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := hasJSONNull([]byte(test.data)); got != test.want {
				t.Fatalf("hasJSONNull() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestLoadYAML(t *testing.T) {
	dir := privatePolicyDirectory(t)
	file := filepath.Join(dir, "policy.yaml")
	data := "version: 1\nsources:\n  fixture:\n    type: json\n    path: values.json\nworkloads:\n  demo:\n    mode: offline\n    bindings:\n      - name: DEMO_MESSAGE\n        source: fixture\n        field: message\n        allow_offline: true\n"
	if err := os.WriteFile(file, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsUntrustedFiles(t *testing.T) {
	dir := privatePolicyDirectory(t)
	rejectMissingAndDirectory(t, dir)
	file := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(file, policyBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	rejectInsecurePolicyFile(t, dir, file)
	rejectInvalidPolicyContents(t, file)
}

func rejectMissingAndDirectory(t *testing.T, directory string) {
	t.Helper()
	if _, err := Load(filepath.Join(directory, "missing")); !fault.Is(err, fault.Trust) {
		t.Fatalf("missing policy error = %v", err)
	}
	if _, err := Load(directory); err == nil {
		t.Fatal("directory accepted")
	}
}

func rejectInsecurePolicyFile(t *testing.T, directory, file string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}

	if err := os.Chmod(file, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err == nil {
		t.Fatal("writable policy accepted")
	}
	if err := os.Chmod(file, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlink policy accepted")
	}
}

func rejectInvalidPolicyContents(t *testing.T, file string) {
	t.Helper()
	for _, test := range []struct {
		name string
		data []byte
	}{
		{name: "oversized", data: bytes.Repeat([]byte(" "), maximumPolicyBytes+1)},
		{name: "invalid", data: []byte(`{"version":2}`)},
		{name: "malformed", data: []byte("{")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(file, test.data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(file); err == nil {
				t.Fatalf("%s policy accepted", test.name)
			}
		})
	}
}

func FuzzDecodeNeverLeaksInput(f *testing.F) {
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte("version: 1\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maximumPolicyBytes {
			t.Skip()
		}
		var result model.Policy
		if err := decodePolicy(data, &result); err != nil && len(err.Error()) > 100 {
			t.Fatal("diagnostic contains upstream detail")
		}
	})
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("SYNTHETIC_PRIVATE_IO_DIAGNOSTIC")
}

func TestBoundedIOFaultsAreSanitized(t *testing.T) {
	for _, reader := range []io.Reader{failingReader{}, strings.NewReader(strings.Repeat("x", maximumPolicyBytes+1))} {
		if _, err := readBounded(reader); err == nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
			t.Fatal("unsafe I/O diagnostic")
		}
	}
	file := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(file, policyBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPolicyWithOpen(file, func(string, bool) (*os.File, error) { return nil, errors.New("SYNTHETIC_PRIVATE_OPEN_DIAGNOSTIC") }); err == nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
		t.Fatal("unsafe open diagnostic")
	}
}

func TestLoadRequiresNativeProtectedPolicyPath(t *testing.T) {
	directory := privatePolicyDirectory(t)
	path := filepath.Join(directory, "policy.json")
	if err := os.WriteFile(path, policyBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("protected public-readable policy rejected: %v", err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	if err := os.Chmod(directory, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); !fault.Is(err, fault.Trust) {
		t.Fatalf("policy below writable ancestor error = %v", err)
	}
}

func TestLoadRejectsPolicySymlinkAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symbolic links requires additional Windows privileges")
	}
	directory := privatePolicyDirectory(t)
	target := filepath.Join(directory, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "policy.json")
	if err := os.WriteFile(path, policyBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(link, "policy.json")); !fault.Is(err, fault.Trust) {
		t.Fatalf("policy symlink ancestor error = %v", err)
	}
}

func TestRemotePathsAreNotLocalFiles(t *testing.T) {
	policy := validPolicy()
	policy.Sources["remote"] = model.Source{Type: "infisical", Path: "/environments/development", Resources: []string{"API_KEY"}}
	bindPaths(&policy, t.TempDir())
	if policy.Sources["remote"].Path != "/environments/development" {
		t.Fatal("remote path was rewritten")
	}
}

func TestLoadRejectsUnknownWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows prevents removing the process working directory")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("policy.json"); err == nil {
		t.Fatal("unresolvable relative path accepted")
	}
}
