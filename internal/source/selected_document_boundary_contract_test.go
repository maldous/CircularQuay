package source

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestSelectedDocumentBoundaryContractWrongObjectTerminatorHasNoPartialMap(t *testing.T) {
	data := []byte(`{"PRIMARY":"owned"]`)
	before := bytes.Clone(data)
	object, err := literalObject(data)
	requireProviderInputError(t, err, fault.Trust, "json_source")
	if object != nil || !bytes.Equal(data, before) {
		t.Fatal("wrong terminator released a partial object or changed input")
	}
	object, err = literalObject([]byte(`{"PRIMARY":"owned"}`))
	if err != nil || !reflect.DeepEqual(object, map[string]any{"PRIMARY": "owned"}) {
		t.Fatal("complete object positive control failed", err)
	}
}
