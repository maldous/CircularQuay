package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInvalidAuditArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"version"}, {"run", "--unknown"}, {"run", "extra"}, {"run", "--profile", "unknown"}} {
		var out, err bytes.Buffer
		if code := run(args, &out, &err); code != 2 || err.Len() == 0 {
			t.Fatal("invalid arguments accepted")
		}
		if strings.Contains(err.String(), "--unknown") {
			t.Fatal("untrusted arguments reflected")
		}
	}
}
