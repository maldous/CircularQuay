package source

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

type vendorExecution struct {
	validate   func(string) error
	credential func(string) (string, error)
	makeHome   func(string, string) (string, error)
	removeHome func(string) error
	lookup     func(string) (string, bool)
	run        func(context.Context, vendorInvocation) error
}

type vendorInvocation struct {
	executable  string
	arguments   []string
	home        string
	environment []string
	output      io.Writer
}

func readVendor(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
	operations := vendorExecution{validate: validateVendorExecutable, credential: transport.ReadCredentialFile, makeHome: os.MkdirTemp, removeHome: os.RemoveAll, lookup: os.LookupEnv, run: executeVendorCommand}
	return readVendorWithOperations(ctx, s, b, operations)
}

func readVendorWithOperations(ctx context.Context, s model.Source, b model.Binding, operations vendorExecution) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	// Reject provider-specific scope before touching executable or bootstrap identity.
	args, key, err := vendorCommand(s, b)
	if err != nil {
		return model.Value{}, err
	}
	if err := operations.validate(s.Path); err != nil {
		return model.Value{}, err
	}
	token, err := operations.credential(s.TokenFile)
	if err != nil {
		return model.Value{}, err
	}
	home, err := operations.makeHome("", "cq-vendor-")
	if err != nil {
		return model.Value{}, fault.New(fault.Apply, "vendor_state")
	}
	defer operations.removeHome(home)
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	environment := []string{"HOME=" + home, "USERPROFILE=" + home, "APPDATA=" + home, "LOCALAPPDATA=" + home, "NO_COLOR=1", key + "=" + token}
	if root, ok := operations.lookup("SystemRoot"); ok {
		environment = append(environment, "SystemRoot="+root)
	}
	var output limitedBuffer
	command := vendorInvocation{executable: s.Path, arguments: args, home: home, environment: environment, output: &output}
	if operations.run(bounded, command) != nil {
		return model.Value{}, fault.New(fault.Denied, "vendor_request")
	}
	if output.overflow {
		return model.Value{}, fault.New(fault.Trust, "vendor_response_size")
	}
	return parseVendorOutput(s, b, output.Bytes())
}

func executeVendorCommand(ctx context.Context, command vendorInvocation) error {
	cmd := exec.CommandContext(ctx, command.executable, command.arguments...)
	cmd.Dir, cmd.Env = command.home, command.environment
	cmd.Stdout, cmd.Stderr = command.output, &discardBuffer{}
	return cmd.Run()
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
