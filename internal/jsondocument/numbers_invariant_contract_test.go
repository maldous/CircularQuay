package jsondocument

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeNumbersInvariantContractPreservesLexemesAndDefault(t *testing.T) {
	input := []byte(`{"number":9007199254740993,"exponent":1.2300e+2,"values":[null,true,"text"]}`)
	var precise, ordinary map[string]any
	if err := DecodeNumbers(input, &precise); err != nil {
		t.Fatal(err)
	}
	if err := Decode(input, &ordinary); err != nil {
		t.Fatal(err)
	}
	if precise["number"] != json.Number("9007199254740993") || precise["exponent"] != json.Number("1.2300e+2") {
		t.Fatal("strict numeric decoding lost original lexemes")
	}
	if _, ok := ordinary["number"].(float64); !ok {
		t.Fatal("default Decode changed its existing numeric representation")
	}
	if string(input) != `{"number":9007199254740993,"exponent":1.2300e+2,"values":[null,true,"text"]}` {
		t.Fatal("strict decoding mutated the caller's document")
	}
}

func TestDecodeNumbersInvariantContractKeepsEveryDocumentRefusal(t *testing.T) {
	for _, input := range [][]byte{
		[]byte(`{"key":1,"key":2}`),
		[]byte(`1 2`),
		[]byte(`1e400`),
		[]byte(`{"number":1e400}`),
		{0xff},
		[]byte(strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)),
	} {
		assertNumbersInvariantRefusal(t, input)
	}
	var value any
	if err := DecodeNumbers([]byte("0"), value); err != errDocument {
		t.Fatal("nonpointer result changed its original refusal")
	}
}

func assertNumbersInvariantRefusal(t *testing.T, input []byte) {
	t.Helper()
	var precise, ordinary any
	if DecodeNumbers(input, &precise) != errDocument || Decode(input, &ordinary) != errDocument {
		t.Fatal("numeric mode relaxed an original strict document refusal")
	}
}

func TestDecodeNumbersInvariantContractKeepsStructFieldsAndNull(t *testing.T) {
	var value struct {
		Name string `json:"name"`
	}
	for _, input := range []string{`{"name":"ok","other":1}`, `{"Name":"ok"}`, `{}`} {
		if DecodeNumbers([]byte(input), &value) != errDocument {
			t.Fatal("numeric mode relaxed strict struct field admission")
		}
	}
	if err := DecodeNumbers([]byte(`{"name":"ok"}`), &value); err != nil || value.Name != "ok" {
		t.Fatal("valid strict struct control failed")
	}
	var nullable any
	if err := DecodeNumbers([]byte(`null`), &nullable); err != nil || nullable != nil {
		t.Fatal("nullable interface semantics changed")
	}
}

func TestDecodeNumbersInvariantContractRetainsExactDepthBoundary(t *testing.T) {
	input := []byte(strings.Repeat("[", 32) + "0" + strings.Repeat("]", 32))
	var precise, ordinary any
	if DecodeNumbers(input, &precise) != nil || Decode(input, &ordinary) != nil {
		t.Fatal("previously supported depth boundary was reduced")
	}
}
