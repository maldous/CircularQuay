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
	"github.com/maldous/CircularQuay/internal/model"
	"gopkg.in/yaml.v3"
)

const maximumPolicyBytes = 1 << 20

// Load reads one regular, non-writable-by-others policy. Relative file references
// are bound to the policy directory, rather than the caller's working directory.
func Load(path string) (model.Policy, error) {
	var result model.Policy
	absolute, err := filepath.Abs(path)
	if err != nil {
		return result, fault.New(fault.Trust, "policy_location")
	}
	path = absolute
	data, err := readPolicy(path)
	if err != nil {
		return result, err
	}
	if err = decodePolicy(data, &result); err != nil {
		return model.Policy{}, err
	}
	bindPaths(&result, filepath.Dir(path))
	if err = Validate(result); err != nil {
		return model.Policy{}, err
	}
	return result, nil
}

func readPolicy(path string) ([]byte, error) {
	return readPolicyWithOpen(path, os.Open)
}

func readPolicyWithOpen(path string, open func(string) (*os.File, error)) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0o022 != 0 {
		return nil, fault.New(fault.Trust, "policy_file")
	}
	file, err := open(path)
	if err != nil {
		return nil, fault.New(fault.Trust, "policy_file")
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) || after.Size() > maximumPolicyBytes {
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
	checker := json.NewDecoder(bytes.NewReader(data))
	if err := checkJSONValue(checker, 0); err != nil {
		return fault.New(fault.Invalid, "policy_json")
	}
	if _, err := checker.Token(); err != io.EOF {
		return fault.New(fault.Invalid, "policy_json")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return fault.New(fault.Invalid, "policy_json")
	}
	return nil
}

func checkJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return fault.New(fault.Invalid, "policy_depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, collection := token.(json.Delim)
	if !collection {
		return nil
	}
	if delimiter == '{' {
		return checkJSONObject(decoder, depth)
	}
	if delimiter != '[' {
		return fault.New(fault.Invalid, "policy_json")
	}
	for decoder.More() {
		if err := checkJSONValue(decoder, depth+1); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func checkJSONObject(decoder *json.Decoder, depth int) error {
	seen := make(map[string]bool)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, valid := token.(string)
		if !valid || seen[name] {
			return fault.New(fault.Invalid, "duplicate_policy_field")
		}
		seen[name] = true
		if err = checkJSONValue(decoder, depth+1); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
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
	policy.Repository.Directory = absolutePath(base, policy.Repository.Directory)
	policy.Repository.AllowedSignersFile = absolutePath(base, policy.Repository.AllowedSignersFile)
	policy.Repository.TokenFile = absolutePath(base, policy.Repository.TokenFile)
	for name, source := range policy.Sources {
		if localFileSource(source.Type) || source.Type == "env" {
			source.Path = absolutePath(base, source.Path)
		}
		source.TokenFile = absolutePath(base, source.TokenFile)
		source.AgeIdentityFile = absolutePath(base, source.AgeIdentityFile)
		source.SOPS = executablePath(base, source.SOPS)
		policy.Sources[name] = source
	}
	for name, workload := range policy.Workloads {
		workload.Directory = absolutePath(base, workload.Directory)
		if len(workload.Command) > 0 {
			workload.Command[0] = executablePath(base, workload.Command[0])
		}
		workload.Snapshot.Path = absolutePath(base, workload.Snapshot.Path)
		workload.Snapshot.IdentityFile = absolutePath(base, workload.Snapshot.IdentityFile)
		workload.Snapshot.SOPS = executablePath(base, workload.Snapshot.SOPS)
		for targetName, target := range workload.Targets {
			if target.Type == "file" || target.Type == "systemd" || target.Type == "windows" {
				target.Path = absolutePath(base, target.Path)
			}
			target.TokenFile = absolutePath(base, target.TokenFile)
			workload.Targets[targetName] = target
		}
		policy.Workloads[name] = workload
	}
	policy.MCP.AuditPath = absolutePath(base, policy.MCP.AuditPath)
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
