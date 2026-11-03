//go:build linux && !android

package platform

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func applySystemd(ctx context.Context, target model.Target, data []byte) error {
	name, err := systemdCredentialName(target.Path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// The executable and every flag are fixed. name is a basename constrained to
	// the systemd credential alphabet; no shell or inherited environment exists.
	cmd := exec.CommandContext(ctx, "/usr/bin/systemd-creds", "encrypt", "--name="+name, "-", "-")
	cmd.Env = []string{}
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = io.Discard
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fault.New(fault.Apply, "systemd credential encryption")
	}
	return WriteProtected(target.Path, out.Bytes())
}

func systemdCredentialName(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fault.New(fault.Invalid, "absolute systemd credential path required")
	}
	name := filepath.Base(path)
	if name == "." || name == "" || len(name) > 200 || strings.HasPrefix(name, "-") {
		return "", fault.New(fault.Invalid, "systemd credential name")
	}
	for _, character := range name {
		if !systemdCredentialRune(character) {
			return "", fault.New(fault.Invalid, "systemd credential name")
		}
	}
	return name, nil
}

func systemdCredentialRune(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		strings.ContainsRune("._-", character)
}
