//go:build linux && !android

package platform

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

var errSystemdOutputTooLarge = errors.New("systemd credential output too large")

func applySystemd(ctx context.Context, target model.Target, data []byte) error {
	return applySystemdWithRunner(ctx, target, data, runSystemdCommand)
}

func applySystemdWithRunner(ctx context.Context, target model.Target, data []byte, run systemCommandRunner) error {
	name, err := systemdCredentialName(target.Path)
	if err != nil {
		return err
	}
	if run == nil {
		return fault.New(fault.Apply, "systemd credential encryption")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// The executable and every flag are fixed. name is a basename constrained to
	// the systemd credential alphabet; no shell or inherited environment exists.
	out := &boundedSystemdOutput{}
	if err := run(ctx, systemCommand{
		path: "/usr/bin/systemd-creds", args: []string{"encrypt", "--name=" + name, "-", "-"},
		env: []string{}, stdin: bytes.NewReader(data), stdout: out, stderr: io.Discard,
	}); err != nil {
		return fault.New(fault.Apply, "systemd credential encryption")
	}
	if out.buffer.Len() == 0 {
		return fault.New(fault.Apply, "empty systemd credential output")
	}
	return WriteProtected(target.Path, out.Bytes())
}

type boundedSystemdOutput struct{ buffer bytes.Buffer }

func (output *boundedSystemdOutput) Write(data []byte) (int, error) {
	if output.buffer.Len()+len(data) > MaxDocumentSize {
		return 0, errSystemdOutputTooLarge
	}
	return output.buffer.Write(data)
}

func (output *boundedSystemdOutput) Bytes() []byte { return output.buffer.Bytes() }

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
