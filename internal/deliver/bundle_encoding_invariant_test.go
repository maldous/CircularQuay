package deliver

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestBundleEncodingInvariantRetainsClosedStringMapSemantics(t *testing.T) {
	want := map[string]string{"FIRST": "quoted \"\n<value>\u2028", "SECOND": "café"}
	values := map[string]model.Value{}
	for name, data := range want {
		values[name] = model.Value{Data: data}
	}
	encoded, err := bundle(model.Generation{Values: values})
	if err != nil {
		t.Fatal("closed string bundle returned an encoding error")
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(encoded), &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("closed string bundle changed its JSON semantics")
	}
}
