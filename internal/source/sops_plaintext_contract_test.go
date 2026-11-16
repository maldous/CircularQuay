package source

import (
	"context"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func TestSnapshotPlaintextRejectsAmbiguity(t *testing.T) {
	cases := map[string][]byte{
		"duplicate envelope": []byte(`{"cq_snapshot":"first","cq_snapshot":"second"}`),
		"escaped duplicate":  []byte(`{"KEY":"first","\u004bEY":"second"}`),
		"null string":        []byte(`{"KEY":null}`),
		"invalid UTF-8 key":  append([]byte{'{', '"', 0xff}, []byte(`":"value"}`)...),
		"invalid UTF-8 value": append([]byte(`{"KEY":"`),
			append([]byte{0xff}, []byte(`"}`)...)...),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			values, err := decodeStrings(data)
			if !fault.Is(err, fault.Invalid) || values != nil {
				t.Fatal("ambiguous plaintext must be denied without returning values")
			}
		})
	}
}

func TestSnapshotPlaintextPreservesExactStrings(t *testing.T) {
	values, err := decodeStrings([]byte(`{"KEY":"synthetic-✓\nsecond line","EMPTY":"","key":"case-sensitive"}`))
	if err != nil || len(values) != 3 || values["KEY"] != "synthetic-✓\nsecond line" || values["EMPTY"] != "" || values["key"] != "case-sensitive" {
		t.Fatal("valid plaintext strings changed or were rejected")
	}
}

func TestSOPSPlaintextNullDeniedAfterHelper(t *testing.T) {
	config, file := sopsFixture(t, `{"mac":"synthetic","age":[{"recipient":"age1synthetic"}]}`)
	// The synthetic helper returns this payload successfully. Neither source
	// reads nor snapshot reads may turn the null value into an empty secret.
	data := []byte(`{"sops":{"mac":"synthetic","age":[{"recipient":"age1synthetic"}]},"test_data":{"KEY":null}}`)
	if err := platform.WriteProtected(file, data); err != nil {
		t.Fatal(err)
	}
	value, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
	if !fault.Is(err, fault.Invalid) || value != (model.Value{}) {
		t.Fatal("source read released coerced plaintext")
	}
	snapshot := model.Snapshot{Path: file, SOPS: config.SOPS}
	values, err := DecryptSnapshot(context.Background(), snapshot)
	if !fault.Is(err, fault.Invalid) || values != nil {
		t.Fatal("snapshot read released coerced plaintext")
	}
}
