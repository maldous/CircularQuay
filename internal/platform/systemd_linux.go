//go:build linux && !android

package platform

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func applySystemd(ctx context.Context, target model.Target, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/systemd-creds", "encrypt", "--name="+filepath.Base(target.Path), "-", "-")
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
