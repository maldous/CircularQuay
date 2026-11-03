package cli

import "testing"

func TestDoctorDependenciesClassifiesOnlyDeclaredAbsolutePaths(t *testing.T) {
	dependencies := doctorDependencies([]string{"relative-tool", "/definitely/missing/cquay-tool", "relative-tool"})
	if len(dependencies) != 2 || dependencies[0].State != "invalid_name" || dependencies[1].State != "missing" {
		t.Fatal("doctor dependency classification changed")
	}
}

func TestDoctorMetadataDoesNotReadMissingFile(t *testing.T) {
	metadata := doctorMetadata([]string{"/definitely/missing/cquay-bootstrap"})
	if len(metadata) != 1 || metadata[0].Kind != "protected_file" || metadata[0].State != "missing" {
		t.Fatal("doctor metadata classification changed")
	}
}
