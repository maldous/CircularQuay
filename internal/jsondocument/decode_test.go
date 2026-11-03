package jsondocument

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeRejectsAmbiguousDocuments(t *testing.T) {
	type nested struct {
		Name string `json:"name"`
	}
	type document struct {
		Items map[string]nested `json:"items"`
		At    time.Time         `json:"at"`
	}
	valid := `{"items":{"one":{"name":"synthetic"}},"at":"2026-11-03T00:00:00Z"}`
	var result document
	if err := Decode([]byte(valid), &result); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		valid + ` {}`, valid + ` junk`, `null`,
		`{"items":{"one":{"name":"a","name":"b"}}}`,
		`{"items":{"one":{"Name":"b"}}}`,
		`{"items":{"one":{"unknown":"b"}}}`,
		`{"items":{"one":null}}`,
		`{"items":{"one":{"name":1}}}`,
		`{"items":{},"items":{}}`,
		`{"Items":{}}`, `{"at":"invalid"}`,
		strings.Repeat(`[`, 34) + `0` + strings.Repeat(`]`, 34),
	}
	for _, data := range cases {
		if Decode([]byte(data), &result) == nil {
			t.Errorf("unsafe document accepted: %q", data)
		}
	}
	if Decode([]byte(valid), nil) == nil || Decode([]byte(valid), result) == nil {
		t.Fatal("non-pointer destination accepted")
	}
}

func TestDecodeRejectsInvalidUTF8(t *testing.T) {
	type document struct {
		Name string `json:"name"`
	}
	var result document
	invalid := []byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}
	if err := Decode(invalid, &result); err == nil {
		t.Fatal("invalid UTF-8 document accepted")
	}
}
