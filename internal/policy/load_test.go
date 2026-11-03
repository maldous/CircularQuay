package policy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func policyBytes(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(validPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLoadStrictJSONAndRelativePaths(t *testing.T) {
	dir := t.TempDir()
	p := validPolicy()
	changeSource(&p, func(s *model.Source) {
		s.Path = "values.json"
		s.TokenFile = "token"
		s.AgeIdentityFile = "age"
		s.SOPS = "./bin/sops"
	})
	changeWorkload(&p, func(w *model.Workload) {
		w.Directory = "work"
		w.Command = []string{"./bin/worker"}
		w.Snapshot.Path = "cache"
		w.Snapshot.IdentityFile = "snapshot-age"
		w.Snapshot.SOPS = "sops"
		w.Bindings[0].AllowPublish = true
		w.Targets = map[string]model.Target{"sink": {Type: "file", Path: "out", TokenFile: "target-token", AcceptSnapshot: true}}
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
	for _, path := range []string{loaded.Sources["fixture"].Path, loaded.Sources["fixture"].TokenFile, loaded.Sources["fixture"].AgeIdentityFile, loaded.Sources["fixture"].SOPS, loaded.Workloads["demo"].Directory, loaded.Workloads["demo"].Command[0], loaded.Workloads["demo"].Snapshot.Path, loaded.Workloads["demo"].Snapshot.IdentityFile, loaded.Workloads["demo"].Targets["sink"].Path, loaded.Workloads["demo"].Targets["sink"].TokenFile, loaded.MCP.AuditPath} {
		if !filepath.IsAbs(path) || !strings.HasPrefix(path, dir+string(filepath.Separator)) {
			t.Fatal("relative path was not anchored")
		}
	}
	if loaded.Workloads["demo"].Snapshot.SOPS != "sops" {
		t.Fatal("executable name must stay a name")
	}
}

func TestDecodeRejectsAmbiguousPolicies(t *testing.T) {
	valid := string(policyBytes(t))
	inputs := []string{
		`{"version":1,"version":2}`, `{"unknown":"SYNTHETIC_DO_NOT_ECHO"}`,
		valid + valid, valid + ` false`, `{"version":`, `{"version":1,"sources":[]}`,
		`[`, "", "---\nversion: 1\n---\nversion: 1\n",
		"version: 1\nversion: 2\n", "version: &value 1\n", "version: *missing\n",
		"version: !untrusted 1\n", "unknown: rejected\n",
		strings.Repeat("[", 34) + strings.Repeat("]", 34),
	}
	for _, input := range inputs {
		var result model.Policy
		if err := decodePolicy([]byte(input), &result); err == nil {
			t.Fatalf("unsafe policy accepted (%d bytes)", len(input))
		}
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
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
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing policy accepted")
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("directory accepted")
	}
	file := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(file, policyBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(file, 0o666); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(file); err == nil {
			t.Fatal("writable policy accepted")
		}
		if err := os.Chmod(file, 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(file, link); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(link); err == nil {
			t.Fatal("symlink policy accepted")
		}
	}
	if err := os.WriteFile(file, bytes.Repeat([]byte(" "), maximumPolicyBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err == nil {
		t.Fatal("oversized policy accepted")
	}
	if err := os.WriteFile(file, []byte(`{"version":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err == nil {
		t.Fatal("invalid declaration accepted")
	}
	if err := os.WriteFile(file, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file); err == nil {
		t.Fatal("malformed declaration accepted")
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
	if _, err := readPolicyWithOpen(file, func(string) (*os.File, error) { return nil, errors.New("SYNTHETIC_PRIVATE_OPEN_DIAGNOSTIC") }); err == nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
		t.Fatal("unsafe open diagnostic")
	}
}

func TestJSONTokenStateAndDepth(t *testing.T) {
	var result model.Policy
	deep := `{"nested":` + strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34) + "}"
	if decodeJSON([]byte(deep), &result) == nil {
		t.Fatal("deep JSON accepted")
	}
	if decodeJSON([]byte(`{"nested":[invalid]}`), &result) == nil {
		t.Fatal("malformed nested JSON accepted")
	}
	decoder := json.NewDecoder(strings.NewReader("[]"))
	if _, err := decoder.Token(); err != nil {
		t.Fatal(err)
	}
	if checkJSONValue(decoder, 0) == nil {
		t.Fatal("unexpected collection close accepted as value")
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
