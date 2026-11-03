// Package policy validates protected declarations before values are retrieved.
package policy

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"gopkg.in/yaml.v3"
)

const maximumPolicyBytes = 1 << 20

// Load reads one native-protected policy. Relative file references are bound to
// the policy directory, rather than the caller's working directory.
func Load(path string) (model.Policy, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return model.Policy{}, fault.New(fault.Trust, "policy_location")
	}
	data, err := readPolicy(absolute)
	if err != nil {
		return model.Policy{}, err
	}
	return Parse(data, absolute)
}

// Parse validates a policy document for its intended configuration pathname
// without reading or writing that pathname. Relative policy references bind to
// the supplied configuration directory, so callers can validate an atomic
// replacement before exposing it at that path.
func Parse(data []byte, configPath string) (model.Policy, error) {
	if len(data) > maximumPolicyBytes {
		return model.Policy{}, fault.New(fault.Invalid, "policy_size")
	}
	path, err := filepath.Abs(configPath)
	if err != nil || configPath == "" {
		return model.Policy{}, fault.New(fault.Invalid, "policy_location")
	}
	var result model.Policy
	if err = decodePolicy(data, &result); err != nil {
		return model.Policy{}, err
	}
	bindPaths(&result, filepath.Dir(path))
	if err = validatePolicyFileOwnership(result, path); err != nil {
		return model.Policy{}, err
	}
	if err = Validate(result); err != nil {
		return model.Policy{}, err
	}
	return result, nil
}

func readPolicy(path string) ([]byte, error) {
	return readPolicyWithOpen(path, platform.OpenProtected)
}

// readPolicyWithOpen accepts only an opener which returns an already verified
// descriptor. Production passes platform.OpenProtected, which checks native
// ownership, ACLs and ancestors, opens without following the final link, and
// verifies descriptor identity. The seam permits fault-sanitization tests.
func readPolicyWithOpen(path string, open func(string, bool) (*os.File, error)) ([]byte, error) {
	file, err := open(path, false)
	if err != nil {
		return nil, fault.New(fault.Trust, "policy_file")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumPolicyBytes {
		return nil, fault.New(fault.Trust, "policy_file")
	}
	return readBounded(file)
}

func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maximumPolicyBytes+1))
	if err != nil || len(data) > maximumPolicyBytes {
		return nil, fault.New(fault.Invalid, "policy_size")
	}
	return data, nil
}

func decodePolicy(data []byte, result *model.Policy) error {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return decodeJSON(data, result)
	}
	return decodeYAML(data, result)
}

func decodeJSON(data []byte, result *model.Policy) error {
	if jsondocument.Decode(data, result) != nil || hasJSONNull(data) {
		return fault.New(fault.Invalid, "policy_json")
	}
	return nil
}

func hasJSONNull(data []byte) bool {
	var document any
	return json.Unmarshal(data, &document) != nil || containsJSONNull(document)
}

func containsJSONNull(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case []any:
		return sliceContainsJSONNull(value)
	case map[string]any:
		return mapContainsJSONNull(value)
	}
	return false
}

func sliceContainsJSONNull(values []any) bool {
	for _, value := range values {
		if containsJSONNull(value) {
			return true
		}
	}
	return false
}

func mapContainsJSONNull(values map[string]any) bool {
	for _, value := range values {
		if containsJSONNull(value) {
			return true
		}
	}
	return false
}

func decodeYAML(data []byte, result *model.Policy) error {
	var node yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&node); err != nil || !safeYAML(&node, 0) {
		return fault.New(fault.Invalid, "policy_yaml")
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return fault.New(fault.Invalid, "policy_yaml")
	}
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(result); err != nil {
		return fault.New(fault.Invalid, "policy_yaml")
	}
	return nil
}

func safeYAML(node *yaml.Node, depth int) bool {
	if depth > 32 || node.Kind == yaml.AliasNode || node.Anchor != "" || strings.HasPrefix(node.Tag, "!") && !strings.HasPrefix(node.Tag, "!!") {
		return false
	}
	for _, child := range node.Content {
		if !safeYAML(child, depth+1) {
			return false
		}
	}
	return true
}

func bindPaths(policy *model.Policy, base string) {
	bindRepositoryPaths(&policy.Repository, base)
	for name, source := range policy.Sources {
		bindSourcePaths(&source, base)
		policy.Sources[name] = source
	}
	for name, workload := range policy.Workloads {
		bindWorkloadPaths(&workload, base)
		policy.Workloads[name] = workload
	}
	policy.MCP.AuditPath = absolutePath(base, policy.MCP.AuditPath)
}

func bindRepositoryPaths(repository *model.Repository, base string) {
	repository.Directory = absolutePath(base, repository.Directory)
	repository.AllowedSignersFile = absolutePath(base, repository.AllowedSignersFile)
	repository.TokenFile = absolutePath(base, repository.TokenFile)
}

func bindSourcePaths(source *model.Source, base string) {
	if localFileSource(source.Type) || source.Type == "env" {
		source.Path = absolutePath(base, source.Path)
	} else if source.Type == "bitwarden" || source.Type == "onepassword" {
		source.Path = executablePath(base, source.Path)
	}
	source.TokenFile = absolutePath(base, source.TokenFile)
	source.AgeIdentityFile = absolutePath(base, source.AgeIdentityFile)
	source.SOPS = executablePath(base, source.SOPS)
}

func bindWorkloadPaths(workload *model.Workload, base string) {
	workload.Directory = absolutePath(base, workload.Directory)
	bindCommandPath(workload.Command, base)
	workload.Snapshot.Path = absolutePath(base, workload.Snapshot.Path)
	workload.Snapshot.IdentityFile = absolutePath(base, workload.Snapshot.IdentityFile)
	workload.Snapshot.SOPS = executablePath(base, workload.Snapshot.SOPS)
	for name, target := range workload.Targets {
		bindTargetPaths(&target, base)
		workload.Targets[name] = target
	}
}

func bindCommandPath(command []string, base string) {
	if len(command) > 0 {
		command[0] = executablePath(base, command[0])
	}
}

func bindTargetPaths(target *model.Target, base string) {
	if target.Type == "file" || target.Type == "systemd" || target.Type == "windows" {
		target.Path = absolutePath(base, target.Path)
	}
	target.TokenFile = absolutePath(base, target.TokenFile)
}

func absolutePath(base, value string) string {
	if value == "" || filepath.IsAbs(value) || strings.HasPrefix(value, "repo:") {
		return value
	}
	return filepath.Clean(filepath.Join(base, value))
}

func executablePath(base, value string) string {
	if !strings.ContainsAny(value, `/\`) {
		return value
	}
	return absolutePath(base, value)
}
