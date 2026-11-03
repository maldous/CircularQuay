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
)

func readJSON(_ context.Context, s model.Source, b model.Binding) (model.Value, error) {
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
	if s.Path == "" {
		value, ok := os.LookupEnv(b.Resource)
		if !ok {
			return model.Value{}, fault.New(fault.Missing, "environment_value")
		}
		return literal(value, s.AuditVerified)
	}
	data, version, err := readLiteralFileVersion(s.Path)
	if err != nil {
		return model.Value{}, err
	}
	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return model.Value{}, fault.New(fault.Invalid, "dotenv_literal")
		}
		name = strings.TrimSpace(name)
		if _, exists := values[name]; exists {
			return model.Value{}, fault.New(fault.Invalid, "dotenv_duplicate")
		}
		if strings.ContainsAny(value, "`\x00") || strings.Contains(value, "$(") || strings.Contains(value, "${") {
			return model.Value{}, fault.New(fault.Invalid, "dotenv_expansion")
		}
		values[name] = value
	}
	if scanner.Err() != nil {
		return model.Value{}, fault.New(fault.Trust, "dotenv_size")
	}
	value, ok := values[b.Resource]
	if !ok {
		return model.Value{}, fault.New(fault.Missing, "dotenv_value")
	}
	v, err := literal(value, s.AuditVerified)
	v.Version = version
	return v, err
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
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", fault.New(fault.Missing, "source_file")
	}
	if !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return nil, "", fault.New(fault.Trust, "source_file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fault.New(fault.Denied, "source_file")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, "", fault.New(fault.Trust, "source_file")
	}
	data, err := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return nil, "", fault.New(fault.Trust, "source_file")
	}
	after, err := f.Stat()
	if err != nil || !after.ModTime().Equal(opened.ModTime()) || after.Size() != opened.Size() {
		return nil, "", fault.New(fault.Trust, "source_file_changed")
	}
	return data, fmt.Sprintf("file:%d:%d", opened.ModTime().UnixNano(), opened.Size()), nil
}

// literalObject rejects ambiguous duplicate keys and non-object documents.
func literalObject(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fault.New(fault.Trust, "json_source")
	}
	object := map[string]any{}
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
	if token, err = dec.Token(); err != nil || token != json.Delim('}') {
		return nil, fault.New(fault.Trust, "json_source")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, fault.New(fault.Trust, "json_source")
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
