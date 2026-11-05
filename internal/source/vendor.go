package source

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/transport"
)

func readVendor(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	if err := validateVendorExecutable(s.Path); err != nil {
		return model.Value{}, err
	}
	token, err := transport.ReadCredentialFile(s.TokenFile)
	if err != nil {
		return model.Value{}, err
	}
	args, key, err := vendorCommand(s, b)
	if err != nil {
		return model.Value{}, err
	}
	home, err := os.MkdirTemp("", "cq-vendor-")
	if err != nil {
		return model.Value{}, fault.New(fault.Apply, "vendor_state")
	}
	defer os.RemoveAll(home)
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, s.Path, args...)
	cmd.Dir = home
	cmd.Env = []string{"HOME=" + home, "USERPROFILE=" + home, "APPDATA=" + home, "LOCALAPPDATA=" + home, "NO_COLOR=1", key + "=" + token}
	if root, ok := os.LookupEnv("SystemRoot"); ok {
		cmd.Env = append(cmd.Env, "SystemRoot="+root)
	}
	var output limitedBuffer
	cmd.Stdout = &output
	cmd.Stderr = &discardBuffer{}
	if cmd.Run() != nil {
		return model.Value{}, fault.New(fault.Denied, "vendor_request")
	}
	if output.overflow {
		return model.Value{}, fault.New(fault.Trust, "vendor_response_size")
	}
	return parseVendorOutput(s, b, output.Bytes())
}

func validateVendorExecutable(path string) error {
	if !filepath.IsAbs(path) {
		return fault.New(fault.Invalid, "vendor_executable")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fault.New(fault.Trust, "vendor_executable")
	}
	if strings.EqualFold(filepath.Ext(path), ".cmd") || strings.EqualFold(filepath.Ext(path), ".bat") {
		return fault.New(fault.Trust, "vendor_executable")
	}
	return platform.ValidateExecutable(path)
}

func parseVendorOutput(s model.Source, b model.Binding, output []byte) (model.Value, error) {
	if s.Type == "onepassword" {
		data, err := selectValue(string(output), b.Field)
		return model.Value{Data: data, AuditVerified: s.AuditVerified}, err
	}
	var result struct {
		ProjectID    string  `json:"projectId"`
		Value        *string `json:"value"`
		RevisionDate string  `json:"revisionDate"`
	}
	if json.Unmarshal(output, &result) != nil {
		return model.Value{}, fault.New(fault.Trust, "vendor_response")
	}
	if result.ProjectID != s.Project {
		return model.Value{}, fault.New(fault.Denied, "vendor_project")
	}
	if result.Value == nil {
		return model.Value{}, fault.New(fault.Missing, "vendor_value")
	}
	data, err := selectValue(*result.Value, b.Field)
	return model.Value{Data: data, Version: result.RevisionDate, AuditVerified: s.AuditVerified}, err
}

func vendorCommand(s model.Source, b model.Binding) ([]string, string, error) {
	if strings.ContainsAny(b.Resource, "\x00\r\n") {
		return nil, "", fault.New(fault.Invalid, "vendor_resource")
	}
	switch s.Type {
	case "bitwarden":
		return bitwardenCommand(s, b)
	case "onepassword":
		return onePasswordCommand(s, b)
	default:
		return nil, "", fault.New(fault.Unsupported, "vendor_type")
	}
}

func bitwardenCommand(s model.Source, b model.Binding) ([]string, string, error) {
	if s.Project == "" || b.Resource == "" || strings.HasPrefix(b.Resource, "-") {
		return nil, "", fault.New(fault.Invalid, "vendor_scope")
	}
	return []string{"secret", "get", b.Resource, "--output", "json"}, "BWS_ACCESS_TOKEN", nil
}

func onePasswordCommand(s model.Source, b model.Binding) ([]string, string, error) {
	if s.Environment != "" {
		return nil, "", fault.New(fault.Unsupported, "onepassword_environments")
	}
	if !onePasswordResourceAllowed(s.Project, b.Resource) {
		return nil, "", fault.New(fault.Denied, "vendor_scope")
	}
	return []string{"read", b.Resource, "--no-newline"}, "OP_SERVICE_ACCOUNT_TOKEN", nil
}

func onePasswordResourceAllowed(project, resource string) bool {
	parts := strings.Split(strings.TrimPrefix(resource, "op://"), "/")
	return strings.HasPrefix(resource, "op://") && len(parts) == 3 && project != "" && parts[0] == project && parts[1] != "" && parts[2] != "" && !strings.ContainsAny(resource, "?#")
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	overflow bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > 4<<20 {
		b.overflow = true
		return len(p), nil
	}
	return b.buffer.Write(p)
}

func (b *limitedBuffer) Bytes() []byte { return b.buffer.Bytes() }

type discardBuffer struct{}

func (*discardBuffer) Write(p []byte) (int, error) { return len(p), nil }
