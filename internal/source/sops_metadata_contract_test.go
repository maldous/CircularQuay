package source

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

func ambiguousSOPSDocuments() map[string][]byte {
	return map[string][]byte{
		"duplicate metadata object":           []byte(`{"sops":{"mac":"synthetic","kms":[{}]},"sops":{"mac":"synthetic","age":[{}]}}`),
		"duplicate recipient backend":         []byte(`{"sops":{"mac":"synthetic","age":[{}],"kms":[{}],"kms":[]}}`),
		"escaped duplicate recipient backend": []byte(`{"sops":{"mac":"synthetic","age":[{}],"kms":[{}],"\u006bms":[]}}`),
		"duplicate encrypted field":           []byte(`{"KEY":"first","KEY":"second","sops":{"mac":"synthetic","age":[{}]}}`),
		"noncanonical metadata field":         []byte(`{"SOPS":{"mac":"synthetic","age":[{}]}}`),
		"invalid UTF-8":                       append([]byte(`{"KEY":"`), append([]byte{0xff}, []byte(`","sops":{"mac":"synthetic","age":[{}]}}`)...)...),
	}
}

func TestSOPSReaderRejectsAmbiguousDocumentBeforeHelper(t *testing.T) {
	for name, data := range ambiguousSOPSDocuments() {
		t.Run(name, func(t *testing.T) {
			directory := privateSourceDirectory(t, "ambiguous-sops-source")
			file := filepath.Join(directory, "snapshot.json")
			if err := platform.WriteProtected(file, data); err != nil {
				t.Fatal(err)
			}
			config := model.Source{Type: "sops", Path: file, SOPS: filepath.Join(directory, "missing-helper"), AgeIdentityFile: filepath.Join(directory, "missing-identity")}
			_, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
			if !fault.Is(err, fault.Invalid) {
				t.Fatalf("ambiguous document must fail before helper or identity access: %v", err)
			}
		})
	}
}

func TestSnapshotRejectsAmbiguousDocumentBeforeHelper(t *testing.T) {
	for name, data := range ambiguousSOPSDocuments() {
		t.Run(name, func(t *testing.T) {
			directory := privateSourceDirectory(t, "ambiguous-sops-snapshot")
			file := filepath.Join(directory, "snapshot.json")
			if err := platform.WriteProtected(file, data); err != nil {
				t.Fatal(err)
			}
			config := model.Snapshot{Path: file, SOPS: filepath.Join(directory, "missing-helper"), IdentityFile: filepath.Join(directory, "missing-identity")}
			_, err := DecryptSnapshot(context.Background(), config)
			if !fault.Is(err, fault.Invalid) {
				t.Fatalf("ambiguous snapshot must fail before helper or identity access: %v", err)
			}
		})
	}
}

func TestSOPSMetadataPreservesOrdinaryAgeRecipientsAndExtensions(t *testing.T) {
	data := []byte(`{"KEY":"ENC[synthetic]","sops":{"mac":"synthetic","age":[{"recipient":"age1first"},{"recipient":"age1second"}],"version":"synthetic-version","lastmodified":"synthetic-time"}}`)
	if err := validateSOPSMetadata(data, model.Source{}, false); err != nil {
		t.Fatalf("ordinary age alternatives and upstream metadata were rejected: %v", err)
	}
}
