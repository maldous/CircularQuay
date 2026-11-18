package repository

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestRepositoryInvariantBoundaryContractRelativeLookupRefusesBeforeIO(t *testing.T) {
	directory := privateRepositoryDirectory(t, "invariant-lookup")
	verifier := releaseVerifier{directory: directory, tree: map[string]string{"policy.json": "synthetic-blob"}}
	got, err := verifier.expectedBlob(filepath.Join(directory, "policy.json"))
	if err != nil || got != "synthetic-blob" {
		t.Fatal("owned lexical lookup control failed")
	}
	got, err = verifier.expectedBlob("relative-policy.json")
	var safe *fault.Error
	if got != "" || !errors.As(err, &safe) || safe.Kind != fault.Trust || safe.Operation != "release_path" {
		t.Fatal("incompatible relative lookup did not refuse with zero blob")
	}
	// expectedBlob is a lexical component; no release, Git, quota, subprocess
	// or foreign file is selected. WalkDir's public caller supplies descendants.
}

func TestRepositoryInvariantBoundaryContractAcceptedEncodingStaysClosed(t *testing.T) {
	typeOfState := reflect.TypeFor[accepted]()
	marshaler := reflect.TypeFor[json.Marshaler]()
	if typeOfState.NumField() != 3 || typeOfState.Implements(marshaler) || reflect.PointerTo(typeOfState).Implements(marshaler) {
		t.Fatal("accepted-state JSON type changed; requalify the infallible encoding proof")
	}
	for index := range typeOfState.NumField() {
		if typeOfState.Field(index).Type != reflect.TypeFor[string]() {
			t.Fatal("accepted-state field is no longer a built-in string")
		}
	}
	directory := privateRepositoryDirectory(t, "invariant-encoding")
	store := &store{repo: model.Repository{Directory: directory}}
	value := accepted{URL: "synthetic<&>", Ref: "line\n\u2028", Commit: "snowman☃"}
	if err := store.writeAccepted(value); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "accepted.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("{\"url\":\"synthetic\\u003c\\u0026\\u003e\",\"ref\":\"line\\n\\u2028\",\"commit\":\"snowman☃\"}")
	if !bytes.Equal(data, want) {
		t.Fatal("closed accepted-state serialization changed bytes")
	}
}
