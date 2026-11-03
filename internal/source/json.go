package source

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
)

func readJSON(_ context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	data, version, err := readLiteralFileVersion(s.Path)
	if err != nil {
		return model.Value{}, err
	}
	object, err := literalObject(data)
	if err != nil {
		return model.Value{}, err
	}
	field := b.Field
	if field == "" {
		field = b.Name
	}
	value, err := stringField(object, field)
	return model.Value{Data: value, Version: version, AuditVerified: s.AuditVerified}, err
}

func readEnvironment(_ context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	if s.Path == "" {
		return readEnvironmentVariable(b.Resource, b.Field, s.AuditVerified)
	}
	return readDotenv(s, b)
}

func readEnvironmentVariable(name, field string, auditVerified bool) (model.Value, error) {
	if err := validateEnvironmentInput(name); err != nil {
		return model.Value{}, err
	}
	value, ok := os.LookupEnv(name)
	if !ok {
		return model.Value{}, fault.New(fault.Missing, "environment_value")
	}
	return selectedLiteral(value, field, auditVerified)
}

func readDotenv(s model.Source, b model.Binding) (model.Value, error) {
	if err := validateEnvironmentInput(b.Resource); err != nil {
		return model.Value{}, err
	}
	data, version, err := readLiteralFileVersion(s.Path)
	if err != nil {
		return model.Value{}, err
	}
	values, err := dotenvValues(data)
	if err != nil {
		return model.Value{}, err
	}
	value, ok := values[b.Resource]
	if !ok {
		return model.Value{}, fault.New(fault.Missing, "dotenv_value")
	}
	v, err := selectedLiteral(value, b.Field, s.AuditVerified)
	v.Version = version
	return v, err
}

func validateEnvironmentInput(name string) error {
	if policy.IsProtectedEnvironmentName(name) {
		return fault.New(fault.Invalid, "environment_input")
	}
	return nil
}

func selectedLiteral(data, field string, auditVerified bool) (model.Value, error) {
	value, err := selectValue(data, field)
	if err != nil {
		return model.Value{}, err
	}
	return literal(value, auditVerified)
}

func dotenvValues(data []byte) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1<<20)
	for scanner.Scan() {
		if err := addDotenvLine(values, scanner.Text()); err != nil {
			return nil, err
		}
	}
	if scanner.Err() != nil {
		return nil, fault.New(fault.Trust, "dotenv_size")
	}
	return values, nil
}

func addDotenvLine(values map[string]string, line string) error {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	name, value, ok := strings.Cut(line, "=")
	if !ok {
		return fault.New(fault.Invalid, "dotenv_literal")
	}
	name = strings.TrimSpace(name)
	if _, exists := values[name]; exists {
		return fault.New(fault.Invalid, "dotenv_duplicate")
	}
	if strings.ContainsAny(value, "`\x00") || strings.Contains(value, "$(") || strings.Contains(value, "${") {
		return fault.New(fault.Invalid, "dotenv_expansion")
	}
	values[name] = value
	return nil
}

func literal(value string, audit bool) (model.Value, error) {
	if strings.ContainsRune(value, 0) {
		return model.Value{}, fault.New(fault.Trust, "value_type")
	}
	return model.Value{Data: value, AuditVerified: audit}, nil
}

// readLiteralFileVersion returns a file metadata hint, never a secret value hash.
// Equality is not authoritative and must not suppress refresh or publication.
func readLiteralFileVersion(path string) ([]byte, string, error) {
	if !filepath.IsAbs(path) {
		return nil, "", fault.New(fault.Invalid, "source_path")
	}
	f, err := platform.OpenProtected(path, true)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || opened.Size() > 4<<20 {
		return nil, "", fault.New(fault.Trust, "source_file")
	}
	return readStableLiteralFile(f, opened)
}

func readStableLiteralFile(f *os.File, opened os.FileInfo) ([]byte, string, error) {
	data, err := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return nil, "", fault.New(fault.Trust, "source_file")
	}
	if err := unchangedLiteralFile(f, opened); err != nil {
		return nil, "", err
	}
	return data, fmt.Sprintf("file:%d:%d", opened.ModTime().UnixNano(), opened.Size()), nil
}

func unchangedLiteralFile(file *os.File, opened os.FileInfo) error {
	after, err := file.Stat()
	if err != nil || !after.ModTime().Equal(opened.ModTime()) || after.Size() != opened.Size() {
		return fault.New(fault.Trust, "source_file_changed")
	}
	return nil
}

// literalObject rejects ambiguous duplicate keys and non-object documents.
func literalObject(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fault.New(fault.Trust, "json_source")
	}
	object, err := decodeLiteralObject(dec)
	if err != nil {
		return nil, err
	}
	if token, err = dec.Token(); err != nil || token != json.Delim('}') {
		return nil, fault.New(fault.Trust, "json_source")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, fault.New(fault.Trust, "json_source")
	}
	return object, nil
}

func decodeLiteralObject(dec *json.Decoder) (map[string]any, error) {
	object := make(map[string]any)
	for dec.More() {
		key, err := dec.Token()
		name, ok := key.(string)
		if err != nil || !ok {
			return nil, fault.New(fault.Trust, "json_source")
		}
		if _, exists := object[name]; exists {
			return nil, fault.New(fault.Trust, "json_duplicate")
		}
		var value any
		if dec.Decode(&value) != nil {
			return nil, fault.New(fault.Trust, "json_source")
		}
		object[name] = value
	}
	return object, nil
}

func stringField(object map[string]any, field string) (string, error) {
	v, ok := object[field]
	if !ok {
		return "", fault.New(fault.Missing, "source_field")
	}
	s, ok := v.(string)
	if !ok || strings.ContainsRune(s, 0) {
		return "", fault.New(fault.Trust, "value_type")
	}
	return s, nil
}

func selectValue(data string, field string) (string, error) {
	if field == "" {
		if strings.ContainsRune(data, 0) {
			return "", fault.New(fault.Trust, "value_type")
		}
		return data, nil
	}
	object, err := literalObject([]byte(data))
	if err != nil {
		return "", err
	}
	return stringField(object, field)
}
