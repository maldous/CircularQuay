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
		`{"items":{}}`,
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

func TestDecodeEnforcesMaximumNestingDepth(t *testing.T) {
	var result any
	withinLimit := strings.Repeat(`[`, 32) + `0` + strings.Repeat(`]`, 32)
	if err := Decode([]byte(withinLimit), &result); err != nil {
		t.Fatalf("maximum permitted nesting rejected: %v", err)
	}
	beyondLimit := strings.Repeat(`[`, 33) + `0` + strings.Repeat(`]`, 33)
	if err := Decode([]byte(beyondLimit), &result); err == nil {
		t.Fatal("excessive nesting accepted")
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

func TestDecodeEnforcesSchemaAcrossCollections(t *testing.T) {
	type child struct {
		Name string `json:"name"`
	}
	type document struct {
		Default string           `json:""`
		Items   []child          `json:"items"`
		Fixed   [1]child         `json:"fixed"`
		ByName  map[string]child `json:"by_name"`
		Any     any              `json:"any"`
		Extra   *child           `json:"extra,omitempty"`
		ignored string
		Skip    string `json:"-"`
	}
	data := `{"Default":"default","items":[{"name":"list"}],"fixed":[{"name":"fixed"}],"by_name":{"one":{"name":"map"}},"any":{"nested":null}}`
	result := document{ignored: "private"}
	if err := Decode([]byte(data), &result); err != nil {
		t.Fatal(err)
	}
	if result.Default != "default" || result.Items[0].Name != "list" || result.Fixed[0].Name != "fixed" || result.ByName["one"].Name != "map" || result.Any == nil {
		t.Fatal("valid collection document was not decoded")
	}
	if result.ignored != "private" {
		t.Fatal("unexported field was modified")
	}
	for _, unsafe := range []string{
		strings.Replace(data, `"Default":`, `"ignored":"injected","Default":`, 1),
		strings.Replace(data, `"Default":`, `"Skip":"injected","Default":`, 1),
		`{"Default":"default","items":[{"name":"list","extra":true}],"fixed":[{"name":"fixed"}],"by_name":{"one":{"name":"map"}},"any":{}}`,
		`{"Default":"default","items":[{"name":"list"}],"fixed":[{"name":"fixed"}],"by_name":{"one":{"name":"map","name":"duplicate"}},"any":{}}`,
		`{"Default":"default","items":[{"name":"list"}],"fixed":[{"name":"fixed"}],"by_name":{"one":null},"any":{}}`,
	} {
		if err := Decode([]byte(unsafe), &result); err == nil {
			t.Fatal("unsafe collection document accepted")
		}
	}
}

func TestDecodeNullabilityAndMalformedCollections(t *testing.T) {
	type child struct {
		Name string `json:"name"`
	}
	type nullable struct {
		Map   map[string]string `json:"map"`
		List  []string          `json:"list"`
		Any   any               `json:"any"`
		Child *child            `json:"child"`
	}
	var result nullable
	if err := Decode([]byte(`{"map":null,"list":null,"any":null,"child":{"name":"present"}}`), &result); err != nil {
		t.Fatalf("nullable fields rejected: %v", err)
	}
	for _, unsafe := range []string{
		``,
		`{"map":null,"list":null,"any":null,"child":null}`,
		`{"map":{},"list":["value"],"any":{},"child":{"name":"present"}`,
		`{"map":{},"list":["value"],"any":{},"child":{"name":"present"}} trailing`,
	} {
		if err := Decode([]byte(unsafe), &result); err == nil {
			t.Fatal("unsafe nullable document accepted")
		}
	}
}

func TestDecodeDefersCustomScalarValidationToStandardDecoder(t *testing.T) {
	type document struct {
		At time.Time `json:"at"`
	}
	var result document
	if err := Decode([]byte(`{"at":{"unexpected":true}}`), &result); err == nil {
		t.Fatal("custom scalar accepted an object")
	}
}

func TestCheckCollectionRejectsUnexpectedDelimiter(t *testing.T) {
	if err := checkCollection(nil, nil, '(', 0); err == nil {
		t.Fatal("unexpected delimiter accepted")
	}
}
