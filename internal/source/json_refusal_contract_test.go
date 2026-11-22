package source

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// Direct adapter checks complement the registry's version guard: a future
// internal caller must not turn a file metadata hint into a version selector.
func TestJSONRefusalContractVersionPrecedesLocalInput(t *testing.T) {
	const resource = "TEST_JSON_REFUSAL_VERSION"
	t.Setenv(resource, "SYNTHETIC_VERSION_INPUT")
	tests := []struct {
		name   string
		reader ReadFunc
		source model.Source
	}{
		{"json relative path", ReadFunc(readJSON), model.Source{Type: "json", Path: "synthetic-relative.json"}},
		{"selected environment", ReadFunc(readEnvironment), model.Source{Type: "env"}},
		{"absent dotenv", ReadFunc(readEnvironment), model.Source{Type: "env", Path: privateLiteralFile(t, "absent.env")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := test.reader.Read(context.Background(), test.source, model.Binding{Resource: resource, Field: "PRIMARY", Version: "synthetic-version"})
			assertJSONRefusalValue(t, value, err, fault.Invalid, "source_version_unsupported")
			if value != (model.Value{}) {
				t.Fatal("unsupported selector returned local source metadata")
			}
		})
	}
}

func TestJSONRefusalContractAbsentEnvironmentAndDotenvDoNotFallback(t *testing.T) {
	const resource = "TEST_JSON_REFUSAL_ABSENT"
	t.Setenv(resource, "SYNTHETIC_ENVIRONMENT_VALUE")
	path := privateLiteralFile(t, "selected.env")
	writeJSONRefusalFixture(t, path, "OTHER=SYNTHETIC_SIBLING_VALUE\n")
	// A live selected environment variable does not replace a missing dotenv
	// key. A missing selected variable remains a missing-value error.
	value, err := readEnvironment(context.Background(), model.Source{Type: "env", Path: path}, model.Binding{Resource: resource})
	assertJSONRefusalValue(t, value, err, fault.Missing, "dotenv_value")
	if err := os.Unsetenv(resource); err != nil {
		t.Fatal(err)
	}
	value, err = readEnvironment(context.Background(), model.Source{Type: "env"}, model.Binding{Resource: resource})
	assertJSONRefusalValue(t, value, err, fault.Missing, "environment_value")
}

func TestJSONRefusalContractAmbiguousDocumentsWithholdSelectedValue(t *testing.T) {
	tests := []struct {
		name, document, operation string
	}{
		{"duplicate selected", `{"PRIMARY":"SYNTHETIC_FIRST","PRIMARY":"SYNTHETIC_SECOND"}`, "json_duplicate"},
		{"duplicate sibling", `{"PRIMARY":"SYNTHETIC_SELECTED","OTHER":"one","OTHER":"two"}`, "json_duplicate"},
		{"broken sibling value", `{"PRIMARY":"SYNTHETIC_SELECTED","OTHER":}`, "json_source"},
		{"broken sibling key", `{"PRIMARY":"SYNTHETIC_SELECTED","unterminated`, "json_source"},
		{"missing closing brace", `{"PRIMARY":"SYNTHETIC_SELECTED"`, "json_source"},
		{"array not object", `[{"PRIMARY":"SYNTHETIC_SELECTED"}]`, "json_source"},
		{"trailing scalar", `{"PRIMARY":"SYNTHETIC_SELECTED"} 17`, "json_source"},
		{"nul selected string", `{"PRIMARY":"SYNTHETIC\u0000SELECTED"}`, "value_type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := privateLiteralFile(t, "ambiguous.json")
			writeJSONRefusalFixture(t, path, test.document)
			value, err := readJSON(context.Background(), model.Source{Type: "json", Path: path, AuditVerified: true}, model.Binding{Field: "PRIMARY"})
			assertJSONRefusalValue(t, value, err, fault.Trust, test.operation)
		})
	}
}

func TestJSONRefusalContractFieldSelectionDoesNotCoerceOrReturnDocument(t *testing.T) {
	const resource = "TEST_JSON_REFUSAL_FIELD"
	tests := []struct {
		name, document, field string
		kind                  fault.Kind
		operation             string
	}{
		{"absent", `{"OTHER":"SYNTHETIC_SIBLING"}`, "PRIMARY", fault.Missing, "source_field"},
		{"object", `{"PRIMARY":{"OTHER":"SYNTHETIC_SIBLING"}}`, "PRIMARY", fault.Trust, "value_type"},
		{"boolean", `{"PRIMARY":true}`, "PRIMARY", fault.Trust, "value_type"},
		{"nul", `{"PRIMARY":"SYNTHETIC\u0000VALUE"}`, "PRIMARY", fault.Trust, "value_type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(resource, test.document)
			binding := model.Binding{Resource: resource, Field: test.field}
			value, err := readEnvironment(context.Background(), model.Source{Type: "env", AuditVerified: true}, binding)
			assertJSONRefusalValue(t, value, err, test.kind, test.operation)
			path := privateLiteralFile(t, "field.env")
			writeJSONRefusalFixture(t, path, resource+"="+test.document+"\n")
			value, err = readEnvironment(context.Background(), model.Source{Type: "env", Path: path, AuditVerified: true}, binding)
			assertJSONRefusalValue(t, value, err, test.kind, test.operation)
		})
	}
}

func TestJSONRefusalContractOversizedDotenvLineRefusesWholeFile(t *testing.T) {
	const resource = "TEST_JSON_REFUSAL_LINE"
	path := privateLiteralFile(t, "long.env")
	// Even an ignored comment after a usable key must satisfy the scanner's
	// bound. A partial parse must not expose the earlier selected value.
	writeJSONRefusalFixture(t, path, resource+"=SYNTHETIC_SELECTED\n#"+strings.Repeat("x", 1<<20)+"\n")
	value, err := readEnvironment(context.Background(), model.Source{Type: "env", Path: path}, model.Binding{Resource: resource})
	assertJSONRefusalValue(t, value, err, fault.Trust, "dotenv_size")
}

func TestJSONRefusalContractStableDescriptorWithholdsReadAndMutationFailures(t *testing.T) {
	t.Run("stable positive control", func(t *testing.T) {
		file, opened := openJSONRefusalFixture(t)
		assertJSONRefusalStableDescriptor(t, file, opened)
	})
	t.Run("closed read", func(t *testing.T) {
		file, opened := openJSONRefusalFixture(t)
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		data, version, err := readStableLiteralFile(file, opened)
		assertJSONRefusalBytes(t, data, version, err, "source_file")
	})
	t.Run("closed metadata recheck", func(t *testing.T) {
		file, opened := openJSONRefusalFixture(t)
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		assertJSONRefusalError(t, unchangedLiteralFile(file, opened), fault.Trust, "source_file_changed")
	})
	t.Run("in-place size mutation", func(t *testing.T) {
		file, opened := openJSONRefusalFixture(t)
		// A deterministic real in-place mutation, not a sleep/race or invented
		// FileInfo: the already-open descriptor sees the changed same file.
		if err := os.WriteFile(file.Name(), []byte("SYNTHETIC_FILE_VALUE_CHANGED"), 0o600); err != nil {
			t.Fatal(err)
		}
		data, version, err := readStableLiteralFile(file, opened)
		assertJSONRefusalBytes(t, data, version, err, "source_file_changed")
	})
}

func assertJSONRefusalStableDescriptor(t *testing.T, file *os.File, opened os.FileInfo) {
	t.Helper()
	data, version, err := readStableLiteralFile(file, opened)
	if err != nil || !bytes.Equal(data, []byte("SYNTHETIC_FILE_VALUE")) {
		t.Fatal("stable descriptor did not preserve its synthetic bytes")
	}
	want := fmt.Sprintf("file:%d:%d", opened.ModTime().UnixNano(), opened.Size())
	if version != want {
		t.Fatal("file version is not the declared metadata hint")
	}
}

func TestJSONRefusalContractFileBoundsApplyBeforeAndAfterOpen(t *testing.T) {
	const limit = 4 << 20
	path := privateLiteralFile(t, "bounded.json")
	writeJSONRefusalFixture(t, path, "")
	if err := os.Truncate(path, limit); err != nil {
		t.Fatal(err)
	}
	data, version, err := readLiteralFileVersion(path)
	if err != nil || len(data) != limit || !strings.HasPrefix(version, "file:") {
		t.Fatal("exact file byte limit was rejected")
	}
	file, err := platform.OpenProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, limit+1); err != nil {
		t.Fatal(err)
	}
	// The stream cap still applies when a file grows after its opening stat;
	// size metadata alone must not authorize returning an oversized body.
	data, version, err = readStableLiteralFile(file, opened)
	assertJSONRefusalBytes(t, data, version, err, "source_file")
	data, version, err = readLiteralFileVersion(path)
	assertJSONRefusalBytes(t, data, version, err, "source_file")
}

func writeJSONRefusalFixture(t *testing.T, path, text string) {
	t.Helper()
	if err := platform.WriteProtected(path, []byte(text)); err != nil {
		t.Fatal(err)
	}
}

func openJSONRefusalFixture(t *testing.T) (*os.File, os.FileInfo) {
	t.Helper()
	path := privateLiteralFile(t, "descriptor.json")
	writeJSONRefusalFixture(t, path, "SYNTHETIC_FILE_VALUE")
	file, err := platform.OpenProtected(path, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	opened, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	return file, opened
}

func assertJSONRefusalValue(t *testing.T, value model.Value, err error, kind fault.Kind, operation string) {
	t.Helper()
	assertJSONRefusalError(t, err, kind, operation)
	if value.Data != "" {
		t.Fatal("refusal returned value-bearing data")
	}
	// Some selection failures retain the source's metadata hint; this does
	// not promote it to a version selector or claim that the value succeeded.
}

func assertJSONRefusalBytes(t *testing.T, data []byte, version string, err error, operation string) {
	t.Helper()
	assertJSONRefusalError(t, err, fault.Trust, operation)
	if data != nil || version != "" {
		t.Fatal("failed stable file read returned bytes or a version hint")
	}
}

func assertJSONRefusalError(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	if !fault.Is(err, kind) || err.Error() != string(kind)+": "+operation {
		t.Fatalf("wrong or unsafe failure for %s", operation)
	}
}
