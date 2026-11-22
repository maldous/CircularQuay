package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestParseResolvesPrivateFragmentWithoutProvenance(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	result, err := parse(context.Background(), masterDocument(t, []model.Include{include}), configurationPath(t), resolverFor(fragment, include.File))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	assertPrivateResult(t, result, include.BaseDirectory)
}

func TestParseFailsAtomicallyOnFragmentPinMismatch(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	include.SHA256 = stringsOf('0', sha256.Size*2)
	result, err := parse(context.Background(), masterDocument(t, []model.Include{include}), configurationPath(t), resolverFor(fragment, include.File))
	if !fault.Is(err, fault.Trust) || len(result.Policy.Sources) != 0 {
		t.Fatalf("partial result=%#v error=%v", result, err)
	}
}

func TestResolveMasterRejectsPostOverrideDuplicateIdentity(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	rootPath := fixturePath(t, "master", "root.json")
	master := masterDocumentWithRoot(t, []model.Include{include}, []model.PolicyEdit{{Pointer: "/sources/sales__data/path", Value: rawString(t, rootPath)}}, rootPath)
	result, err := parse(context.Background(), master, configurationPath(t), resolverFor(fragment, include.File))
	if !fault.Is(err, fault.Invalid) || len(result.Policy.Sources) != 0 {
		t.Fatalf("post-override identity conflict escaped: result=%#v error=%v", result, err)
	}
}

func TestParseFailsAtomicallyWhenLaterFragmentIsRejected(t *testing.T) {
	first := fragmentDocument(t)
	second := fragmentDocument(t)
	firstInclude := localInclude(t, first)
	secondInclude := namedLocalInclude(t, "finance", "finance", second)
	secondInclude.SHA256 = stringsOf('0', sha256.Size*2)
	files := map[string][]byte{firstInclude.File: first, secondInclude.File: second}
	result, err := parse(context.Background(), masterDocument(t, []model.Include{firstInclude, secondInclude}), configurationPath(t), resolverFiles(files))
	if !fault.Is(err, fault.Trust) || len(result.Policy.Sources) != 0 {
		t.Fatalf("partial result=%#v error=%v", result, err)
	}
}

func TestPlanRejectsInvalidDependencyAndPrivateOverlap(t *testing.T) {
	first := localInclude(t, []byte("first"))
	second := namedLocalInclude(t, "finance", "finance", []byte("second"))
	second.DependsOn = []string{"finance"}
	if _, err := newPlan([]model.Include{first, second}); err == nil {
		t.Fatal("self dependency accepted")
	}
	second = namedLocalInclude(t, "finance", "finance", []byte("second"))
	second.BaseDirectory = first.BaseDirectory
	second.File = filepath.Join(second.BaseDirectory, "finance.json")
	if _, err := newPlan([]model.Include{first, second}); err == nil {
		t.Fatal("overlapping private bases accepted")
	}
}

func TestLocksRejectChangedCompiledValue(t *testing.T) {
	path := fixturePath(t, "master", "root.json")
	compiled := model.Policy{Version: 1, Sources: map[string]model.Source{"root": {Type: "json", Path: path}}, Workloads: map[string]model.Workload{"root": {Mode: "offline", Bindings: []model.Binding{{Name: "ROOT_VALUE", Source: "root", Field: "value", AllowOffline: true}}}}}
	composition := &model.Composition{Locks: []model.PolicyEdit{{Pointer: "/sources/root/path", Value: rawString(t, fixturePath(t, "other.json"))}}}
	if _, err := applyEdits(compiled, composition, configurationPath(t)); !fault.Is(err, fault.Denied) {
		t.Fatalf("changed lock error=%v", err)
	}
}

func TestResolveMasterRejectsOfflineGraphExpiredBeforeExposure(t *testing.T) {
	expires := time.Unix(100, 0).UTC()
	fragment := fragmentDocument(t)
	include := offlineRepositoryInclude(t, fragment, &expires)
	clockCalls := 0
	resolver := resolver{
		current: func(context.Context, model.Repository) (string, error) {
			return filepath.Join(include.Repository.Directory, "releases", include.Commit), nil
		},
		readFile: func(string, bool) ([]byte, error) { return fragment, nil },
		now: func() time.Time {
			clockCalls++
			if clockCalls == 1 {
				return expires.Add(-time.Second)
			}
			return expires
		},
	}
	result, err := parse(context.Background(), masterDocument(t, []model.Include{include}), configurationPath(t), resolver)
	if !fault.Is(err, fault.Expired) || len(result.Policy.Sources) != 0 {
		t.Fatalf("expired graph admitted: result=%#v error=%v", result, err)
	}
}

func TestResolveMasterRejectsCancellationDuringLastRead(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resolver := resolver{
		readFile: func(string, bool) ([]byte, error) { cancel(); return fragment, nil },
		fetch:    func(context.Context, model.Repository, string) (string, error) { return "", nil },
		current:  func(context.Context, model.Repository) (string, error) { return "", nil },
		now:      time.Now,
	}
	result, err := parse(ctx, masterDocument(t, []model.Include{include}), configurationPath(t), resolver)
	if !fault.Is(err, fault.Unavailable) || len(result.Policy.Sources) != 0 {
		t.Fatalf("cancelled graph admitted: result=%#v error=%v", result, err)
	}
}

func TestReadRepositoryRejectsMismatchedPinnedRelease(t *testing.T) {
	include := repositoryInclude(t, "online", nil)
	called := ""
	resolver := resolver{
		fetch: func(ctx context.Context, repo model.Repository, commit string) (string, error) {
			called = commit
			return fixturePath(t, "cache", "releases", "other"), nil
		},
		readFile: func(string, bool) ([]byte, error) { t.Fatal("read unpinned release"); return nil, nil },
	}
	if _, _, _, err := resolver.readRepository(context.Background(), include); !fault.Is(err, fault.Trust) || called != include.Commit {
		t.Fatalf("called=%q error=%v", called, err)
	}
}

func TestReadRepositoryRejectsOfflineMismatchedPinnedRelease(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	include := repositoryInclude(t, "offline", &expires)
	resolver := resolver{
		current: func(context.Context, model.Repository) (string, error) {
			return fixturePath(t, "cache", "releases", "other"), nil
		},
		readFile: func(string, bool) ([]byte, error) { t.Fatal("read unpinned cached release"); return nil, nil },
		now:      time.Now,
	}
	if _, _, _, err := resolver.readRepository(context.Background(), include); !fault.Is(err, fault.Trust) {
		t.Fatalf("error=%v", err)
	}
}

func TestDecodeFragmentUsesMasterAuthorizedBase(t *testing.T) {
	fragment := fragmentDocument(t)
	include := localInclude(t, fragment)
	child, err := decodeFragment(fragment, include)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if child.Sources["data"].Path != filepath.Join(include.BaseDirectory, "values.json") {
		t.Fatalf("bound path=%q", child.Sources["data"].Path)
	}
	child.Sources["data"] = model.Source{Type: "json", Path: fixturePath(t, "release", "values.json")}
	if fragmentPathsWithin(child, include.BaseDirectory) {
		t.Fatal("release path escaped base-directory custody")
	}
}

func TestSourceIdentityIncludesScopeAndCustodyMetadata(t *testing.T) {
	first := model.Source{Type: "openbao", Endpoint: "https://vault.example.test", Mount: "team", Resources: []string{"alpha", "beta"}, Identity: "token_file", TokenFile: fixturePath(t, "private", "token-a")}
	second := first
	second.Resources = []string{"beta", "gamma"}
	if duplicateSourceIdentity(map[string]model.Source{"first": first}, map[string]model.Source{"second": second}) {
		t.Fatal("distinct resource scope was treated as duplicate")
	}
	second = first
	second.TokenFile = fixturePath(t, "private", "token-b")
	if duplicateSourceIdentity(map[string]model.Source{"first": first}, map[string]model.Source{"second": second}) {
		t.Fatal("distinct custody reference was treated as duplicate")
	}
	second.Resources = []string{"beta", "alpha"}
	second.TokenFile = first.TokenFile
	if !duplicateSourceIdentity(map[string]model.Source{"first": first}, map[string]model.Source{"second": second}) {
		t.Fatal("equivalent declaration escaped duplicate detection")
	}
}

func TestPlanRejectsReservedIDAndUnsafeBase(t *testing.T) {
	include := localInclude(t, []byte("fragment"))
	include.ID = "root"
	if _, err := newPlan([]model.Include{include}); err == nil {
		t.Fatal("reserved include ID accepted")
	}
	include = localInclude(t, []byte("fragment"))
	include.BaseDirectory += "\n"
	if _, err := newPlan([]model.Include{include}); err == nil {
		t.Fatal("control-character base accepted")
	}
}

func assertPrivateResult(t *testing.T, result Result, base string) {
	t.Helper()
	guard := result.Policy.ConfigurationGuard
	if len(result.Provenance()) != 0 || guard == nil || guard.SourceDomains["sales__data"] != "sales" || guard.LocalRoots["sales__data"] != base {
		t.Fatalf("private resolver result=%#v guard=%#v", result, guard)
	}
}

func fragmentDocument(t *testing.T) []byte {
	t.Helper()
	return marshalValue(t, map[string]any{"version": 1, "sources": map[string]model.Source{"data": {Type: "json", Path: "values.json"}}, "workloads": map[string]model.Workload{}})
}

func masterDocument(t *testing.T, includes []model.Include) []byte {
	t.Helper()
	return masterDocumentWithComposition(t, includes, nil)
}

func masterDocumentWithComposition(t *testing.T, includes []model.Include, overrides []model.PolicyEdit) []byte {
	t.Helper()
	return masterDocumentWithRoot(t, includes, overrides, fixturePath(t, "master", "root.json"))
}

func masterDocumentWithRoot(t *testing.T, includes []model.Include, overrides []model.PolicyEdit, rootPath string) []byte {
	t.Helper()
	return marshalPolicy(t, model.Policy{
		Version:     1,
		Sources:     map[string]model.Source{"root": {Type: "json", Path: rootPath}},
		Workloads:   map[string]model.Workload{"root": {Mode: "offline", Bindings: []model.Binding{{Name: "ROOT_VALUE", Source: "root", Field: "value", AllowOffline: true}}}},
		Composition: &model.Composition{Includes: includes, Conflicts: "reject", Overrides: overrides},
	})
}

func localInclude(t *testing.T, fragment []byte) model.Include {
	t.Helper()
	return namedLocalInclude(t, "sales", "sales", fragment)
}

func namedLocalInclude(t *testing.T, id, namespace string, fragment []byte) model.Include {
	t.Helper()
	base := fixturePath(t, "private", id)
	return model.Include{ID: id, Namespace: namespace, SHA256: digest(fragment), File: filepath.Join(base, "fragment.json"), BaseDirectory: base, Visibility: "local"}
}

func repositoryInclude(t *testing.T, refresh string, expires *time.Time) model.Include {
	t.Helper()
	return model.Include{Commit: stringsOf('a', 40), Path: "configuration.json", Refresh: refresh, CacheExpiresAt: expires, Repository: &model.Repository{Directory: fixturePath(t, "cache")}}
}

func offlineRepositoryInclude(t *testing.T, fragment []byte, expires *time.Time) model.Include {
	t.Helper()
	commit := stringsOf('a', 40)
	directory := fixturePath(t, "repository")
	repository := &model.Repository{
		URL: "https://github.com/example/configuration.git", Directory: directory, Ref: "refs/heads/main",
		AllowedSignersFile: fixturePath(t, "trust", "allowed_signers"), InitialCommit: stringsOf('b', 40),
	}
	return model.Include{ID: "corporate", Namespace: "corporate", SHA256: digest(fragment), Repository: repository, Path: "fragment.json", Commit: commit, BaseDirectory: fixturePath(t, "data"), Visibility: "shared", Refresh: "offline", CacheExpiresAt: expires}
}

func resolverFor(fragment []byte, file string) resolver {
	return resolverFiles(map[string][]byte{file: fragment})
}

func resolverFiles(files map[string][]byte) resolver {
	return resolver{
		readFile: func(path string, private bool) ([]byte, error) { return files[path], nil },
		fetch:    func(context.Context, model.Repository, string) (string, error) { return "", nil },
		current:  func(context.Context, model.Repository) (string, error) { return "", nil },
		now:      time.Now,
	}
}

func configurationPath(t *testing.T) string { return fixturePath(t, "master", "policy.json") }

func fixturePath(t *testing.T, elements ...string) string {
	t.Helper()
	return filepath.Join(append([]string{t.TempDir()}, elements...)...)
}

func marshalPolicy(t *testing.T, policy model.Policy) []byte {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func marshalValue(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	return data
}

func rawString(t *testing.T, value string) json.RawMessage { return marshalValue(t, value) }

func digest(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func stringsOf(value rune, count int) string {
	result := make([]rune, count)
	for index := range result {
		result[index] = value
	}
	return string(result)
}
