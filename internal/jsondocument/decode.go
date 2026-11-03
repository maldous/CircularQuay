// Package jsondocument decodes bounded caller-owned JSON without ambiguous keys.
package jsondocument

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

var errDocument = errors.New("invalid JSON document")
var unmarshaler = reflect.TypeFor[json.Unmarshaler]()

// Decode accepts one document, rejecting duplicate keys, unknown or noncanonical
// struct fields, trailing input and excessive nesting. Errors never contain input.
// Fields without omitempty must be present. Callers own size bounds and semantic validation.
func Decode(data []byte, result any) error {
	t := reflect.TypeOf(result)
	if t == nil || t.Kind() != reflect.Pointer || !utf8.Valid(data) {
		return errDocument
	}
	checker := json.NewDecoder(bytes.NewReader(data))
	if checkValue(checker, t.Elem(), 0) != nil {
		return errDocument
	}
	if _, err := checker.Token(); err != io.EOF {
		return errDocument
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(result) != nil {
		return errDocument
	}
	return nil
}

func checkValue(decoder *json.Decoder, t reflect.Type, depth int) error {
	if depth > 32 {
		return errDocument
	}
	token, err := decoder.Token()
	if err != nil {
		return errDocument
	}
	if t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if token == nil && !nullable(t) {
		return errDocument
	}
	// Types such as time.Time own their scalar encoding. Their implementation
	// still validates contents in the standard decoder's second pass.
	if t != nil && reflect.PointerTo(t).Implements(unmarshaler) {
		t = nil
	}
	delimiter, collection := token.(json.Delim)
	if !collection {
		return nil
	}
	return checkCollection(decoder, t, delimiter, depth)
}

func nullable(t reflect.Type) bool {
	if t == nil {
		return true
	}
	switch t.Kind() {
	case reflect.Map, reflect.Slice, reflect.Interface:
		return true
	default:
		return false
	}
}

func checkCollection(decoder *json.Decoder, t reflect.Type, delimiter json.Delim, depth int) error {
	switch delimiter {
	case '{':
		if err := checkObject(decoder, t, depth); err != nil {
			return err
		}
	case '[':
		if err := checkArray(decoder, t, depth); err != nil {
			return err
		}
	default:
		return errDocument
	}
	_, err := decoder.Token()
	return err
}

func checkArray(decoder *json.Decoder, t reflect.Type, depth int) error {
	var element reflect.Type
	if t != nil && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) {
		element = t.Elem()
	}
	for decoder.More() {
		if err := checkValue(decoder, element, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func checkObject(decoder *json.Decoder, t reflect.Type, depth int) error {
	seen := make(map[string]bool)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return errDocument
		}
		name, ok := token.(string)
		if !ok || seen[name] {
			return errDocument
		}
		seen[name] = true
		element, valid := fieldType(t, name)
		if !valid {
			return errDocument
		}
		if err := checkValue(decoder, element, depth+1); err != nil {
			return err
		}
	}
	return checkRequiredFields(t, seen)
}

func checkRequiredFields(t reflect.Type, seen map[string]bool) error {
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := fieldName(field)
		optional := slices.Contains(strings.Split(field.Tag.Get("json"), ",")[1:], "omitempty")
		if name != "" && !optional && !seen[name] {
			return errDocument
		}
	}
	return nil
}

func fieldName(field reflect.StructField) string {
	if !field.IsExported() {
		return ""
	}
	tag := strings.Split(field.Tag.Get("json"), ",")[0]
	if tag == "-" {
		return ""
	}
	if tag == "" {
		return field.Name
	}
	return tag
}

func fieldType(t reflect.Type, name string) (reflect.Type, bool) {
	if t == nil {
		return nil, true
	}
	if t.Kind() == reflect.Map {
		return t.Elem(), true
	}
	if t.Kind() != reflect.Struct {
		return nil, true
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if fieldName(field) == name {
			return field.Type, true
		}
	}
	return nil, false
}
