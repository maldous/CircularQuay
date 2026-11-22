package configuration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestClosedInvariantConfigurationMergeOwnsPreallocatedMaps(t *testing.T) {
	master := model.Policy{Version: 1}
	resolution := newMasterResolution(master, "owned-master", 0, includePlan{})
	if resolution.compiled.Sources == nil || resolution.compiled.Workloads == nil {
		t.Fatal("master resolution did not allocate both mutable declaration maps")
	}
	child := model.Policy{Sources: map[string]model.Source{"data": {Type: "env"}}}
	if err := mergeFragment(&resolution.compiled, child, model.Include{Namespace: "team"}, includePlan{}); err != nil {
		t.Fatal("preallocated merge positive control refused", err)
	}
	if len(resolution.compiled.Sources) != 1 || resolution.compiled.Sources["team__data"].Type != "env" {
		t.Fatal("merge changed its qualified declaration")
	}
	if master.Sources != nil || master.Workloads != nil || len(child.Sources) != 1 {
		t.Fatal("resolution map allocation or merge mutated caller-owned declarations")
	}
}

// These are pure parsing/helper inputs, not manufactured mismatches between
// an admitted serialized fragment and its typed DecodeDocument result.
func TestClosedInvariantConfigurationRawObjectHelperKeepsRefusals(t *testing.T) {
	for _, raw := range []string{`{invalid`, `[]`, `null`} {
		fields, ok := topLevelFields([]byte(raw))
		if ok || fields != nil {
			t.Fatal("non-object fragment helper returned partial fields")
		}
	}
	fields, ok := topLevelFields([]byte(`{"version":1}`))
	if !ok || string(fields["version"]) != "1" {
		t.Fatal("raw object positive control changed")
	}
}

func TestClosedInvariantConfigurationOptionalFragmentHelperContracts(t *testing.T) {
	if !validFragmentSources(map[string]json.RawMessage{}, nil) || !validFragmentWorkloads(map[string]json.RawMessage{}) {
		t.Fatal("private optional-declaration helper contract changed")
	}
	fields := map[string]json.RawMessage{"workloads": json.RawMessage(`[]`)}
	if validFragmentWorkloads(fields) {
		t.Fatal("non-object workload declarations accepted")
	}
}

func TestClosedInvariantConfigurationForbiddenVendorPathKeepsPresenceContract(t *testing.T) {
	fields := map[string]json.RawMessage{"path": json.RawMessage(`""`)}
	for _, kind := range []string{"bitwarden", "onepassword"} {
		if !forbiddenSourceFields(model.Source{Type: kind}, fields) {
			t.Fatal("private vendor helper lost explicit path-presence refusal")
		}
		if validFragmentSource("data", model.Source{Type: kind}, fields) {
			t.Fatal("data-only admission accepted an executable-backed vendor")
		}
	}
}

func TestClosedInvariantConfigurationEditParentKeepsPointerAdmission(t *testing.T) {
	document := map[string]any{"sources": map[string]any{"data": map[string]any{"path": "owned"}}}
	before, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, pointer := range []string{"not-a-pointer", "/sources", "/sources//path", "/sources/data/~2"} {
		parent, leaf, ok := editParent(document, pointer)
		if ok || parent != nil || leaf != "" {
			t.Fatal("invalid private pointer retained a partial parent")
		}
	}
	after, err := json.Marshal(document)
	if err != nil || string(after) != string(before) {
		t.Fatal("pointer admission changed its input document")
	}
}

func TestClosedInvariantConfigurationAdmittedCompositionProjection(t *testing.T) {
	fixture := newEditBoundaryFixture(t, nil, nil)
	root, err := ReadRoot(fixture.data, fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	compiled := copyPolicy(withoutComposition(root))
	document, err := compositionDocument(compiled)
	if err != nil {
		t.Fatal("admitted compiled policy did not project to JSON")
	}
	got, err := editedPolicy(document, fixture.path)
	if err != nil || !reflect.DeepEqual(got, compiled) {
		t.Fatal("unchanged composition projection lost an admitted declaration")
	}
}

func TestClosedInvariantConfigurationRepositoryContainmentPrecedesRead(t *testing.T) {
	release := filepath.Join(t.TempDir(), "release")
	data := []byte(`{"version":1}`)
	digest := sha256.Sum256(data)
	include := model.Include{Path: "../outside.json", SHA256: hex.EncodeToString(digest[:]), Repository: &model.Repository{Directory: "owned-cache"}}
	reader := &closedInvariantRepositoryReader{t: t, release: release, data: data}
	r := resolver{readFile: reader.read}
	got, name, roots, err := r.readRepositoryContent(include, release)
	reader.refusal(got, name, roots, err)
	include.Path = "values.json"
	got, name, roots, err = r.readRepositoryContent(include, release)
	reader.accepted(got, name, roots, err, include.Path)
}

type closedInvariantRepositoryReader struct {
	t       *testing.T
	release string
	data    []byte
	reads   int
}

func (r *closedInvariantRepositoryReader) read(path string, secret bool) ([]byte, error) {
	r.reads++
	if path != filepath.Join(r.release, "values.json") || secret {
		r.t.Fatal("read escaped the admitted owned data path")
	}
	return append([]byte(nil), r.data...), nil
}

func (r *closedInvariantRepositoryReader) refusal(got []byte, name string, roots []string, err error) {
	r.t.Helper()
	if !fault.Is(err, fault.Trust) || err.Error() != "trust_failure: configuration_include_path" {
		r.t.Fatal("private repository containment refusal changed")
	}
	if r.reads != 0 || got != nil || name != "" || roots != nil {
		r.t.Fatal("path refusal read data or returned partial output")
	}
}

func (r *closedInvariantRepositoryReader) accepted(got []byte, name string, roots []string, err error, path string) {
	r.t.Helper()
	if err != nil || r.reads != 1 || string(got) != string(r.data) {
		r.t.Fatal("contained positive read refused")
	}
	if name != filepath.Join(r.release, path) || !reflect.DeepEqual(roots, []string{"owned-cache"}) {
		r.t.Fatal("contained read changed provenance")
	}
}

func TestClosedInvariantConfigurationParsedZoneKeepsRealEncodingRefusal(t *testing.T) {
	for _, zone := range []string{"+24:00", "+23:60", "-24:00"} {
		t.Run(zone, func(t *testing.T) {
			closedInvariantConfigurationZoneCase(t, zone)
		})
	}
}

func closedInvariantConfigurationZoneCase(t *testing.T, zone string) {
	t.Helper()
	fixture := newEditBoundaryFixture(t, nil, nil)
	var document map[string]any
	if err := json.Unmarshal(fixture.data, &document); err != nil {
		t.Fatal(err)
	}
	document["grants"] = []any{map[string]any{
		"subject": "synthetic-user", "workload": "root", "operations": []string{"status"},
		"not_before": "2026-01-01T00:00:00" + zone,
	}}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	fixture.data = data
	original := append([]byte(nil), data...)
	admitted, err := ReadRoot(fixture.data, fixture.path)
	if err != nil || len(admitted.Grants) != 1 || admitted.Grants[0].NotBefore == nil {
		t.Fatal("serialized zone did not reach real typed master admission")
	}
	_, offset := admitted.Grants[0].NotBefore.Zone()
	if offset != 24*60*60 && offset != -24*60*60 {
		t.Fatal("fixture did not isolate the parsed non-marshalable zone")
	}
	result, err := resolveEditBoundaryFixture(t, fixture)
	completionConfigurationFailure(t, result, err, fault.Invalid, "composition_encode")
	if !bytes.Equal(original, fixture.data) {
		t.Fatal("encoding refusal mutated caller-owned master bytes")
	}
}
