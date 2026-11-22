//go:build !windows

package platform

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestFinalResidualReceiptPreservesValidatedBytesAndRefusals(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "unused.jsonl")
	input := []byte(" { \"nested\": [1, true, null], \"text\": \"a b\" } ")
	before := append([]byte(nil), input...)
	got, err := normalizeReceipt(path, input)
	if err != nil || string(got) != "{\"nested\":[1,true,null],\"text\":\"a b\"}\n" {
		t.Fatalf("exact compact receipt: %q %v", got, err)
	}
	if !bytes.Equal(input, before) {
		t.Fatal("normalization mutated caller bytes")
	}
	for _, invalid := range []string{"null", "[]", "{", "{} {}"} {
		got, err := normalizeReceipt(path, []byte(invalid))
		if got != nil || !fault.Is(err, fault.Invalid) {
			t.Fatal("invalid receipt admitted", err)
		}
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pure normalization created output")
	}
}

func TestFinalResidualAncestorRootLookupRefusesAfterOwnedTraversal(t *testing.T) {
	directory := secureTempDir(t)
	root := string(filepath.Separator)
	calls := []string{}
	inspect := func(path string) (os.FileInfo, error) {
		calls = append(calls, path)
		if path == root {
			return nil, errors.New("owned root metadata refusal")
		}
		return os.Stat(path)
	}
	if trustedAncestorsWithInspect(directory, inspect) {
		t.Fatal("failed root lookup was trusted")
	}
	if len(calls) == 0 || calls[len(calls)-1] != root {
		t.Fatal("root guard was not reached", calls)
	}
	if !trustedAncestorsWithInspect(directory, os.Stat) {
		t.Fatal("real private ancestry positive control failed")
	}
}

func TestFinalResidualArchiveMetadataCannotAttestNativeOwner(t *testing.T) {
	header := &tar.Header{Name: "owned", Mode: 0700, Typeflag: tar.TypeDir, Uid: os.Geteuid()}
	info := header.FileInfo()
	if info.Sys() != header {
		t.Fatal("stdlib archive metadata contract changed")
	}
	if ownerID(info) != -1 || trustedOwner(info) || trustedDirectory(info) {
		t.Fatal("archive UID was trusted as native ownership")
	}
	actual, err := os.Stat(secureTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if ownerID(actual) != os.Geteuid() || !trustedOwner(actual) {
		t.Fatal("actual owned metadata control failed")
	}
}

func TestFinalResidualAliasesPropagateActualRootMetadataRefusal(t *testing.T) {
	path := completionFile(t, "inert")
	root := string(filepath.Separator)
	calls := 0
	ancestors := func(directory string) bool {
		if directory != root {
			t.Fatal("alias root admission changed")
		}
		return trustedAncestorsWithInspect(directory, func(inspected string) (os.FileInfo, error) {
			calls++
			if inspected != root {
				t.Fatal("root refusal continued traversal")
			}
			return nil, errors.New("owned root lookup error")
		})
	}
	if trustedExecutableAliasesWithAncestors(path, ancestors) || calls != 1 {
		t.Fatal("alias accepted failed root lookup")
	}
	if !trustedExecutableAliasesWithAncestors(path, trustedAncestors) {
		t.Fatal("real owned alias positive control failed")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "owned" {
		t.Fatal("alias validation changed owned bytes")
	}
}
