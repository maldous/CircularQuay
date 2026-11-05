package configuration

import (
	"encoding/json"
	"strings"

	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/model"
)

func validFragmentDocument(data []byte, child model.Policy) bool {
	fields, ok := topLevelFields(data)
	return ok && child.Version == 1 && !forbiddenTopLevel(fields) && validFragmentState(child) && validFragmentSources(fields, child.Sources) && validFragmentWorkloads(fields)
}

func validFragmentState(child model.Policy) bool {
	return child.Composition == nil && len(child.Grants) == 0 && emptyMCP(child.MCP) && emptyRepository(child.Repository) && child.Observability == nil && child.Updates == nil
}

func topLevelFields(data []byte) (map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	if jsondocument.Decode(data, &fields) != nil || fields == nil {
		return nil, false
	}
	return fields, true
}

func forbiddenTopLevel(fields map[string]json.RawMessage) bool {
	for _, name := range []string{"grants", "mcp", "repository", "observability", "updates", "composition"} {
		if _, present := fields[name]; present {
			return true
		}
	}
	return false
}

func emptyMCP(value model.MCP) bool {
	return value.Resource == "" && len(value.Authorities) == 0 && value.Issuer == "" && value.Audience == "" && value.JWKSURL == "" && value.Address == "" && len(value.Origins) == 0 && len(value.Hosts) == 0 && value.AuditPath == ""
}

func emptyRepository(value model.Repository) bool { return value == (model.Repository{}) }

func validFragmentSources(fields map[string]json.RawMessage, sources map[string]model.Source) bool {
	raw, present := fields["sources"]
	if !present {
		return len(sources) == 0
	}
	if len(sources) == 0 {
		return false
	}
	var declarations map[string]map[string]json.RawMessage
	if jsondocument.Decode(raw, &declarations) != nil || len(declarations) != len(sources) {
		return false
	}
	for name, declaration := range declarations {
		source, found := sources[name]
		if !found || !validFragmentSource(name, source, declaration) {
			return false
		}
	}
	return true
}

func validFragmentSource(name string, source model.Source, fields map[string]json.RawMessage) bool {
	return validFragmentIdentifier(name) && dataOnlySource(source) && !forbiddenSourceFields(source, fields)
}

func validFragmentIdentifier(value string) bool {
	return validIdentifier(value) && !strings.Contains(value, "__")
}

var fragmentSourceTypes = map[string]bool{
	"json": true, "env": true, "openbao": true, "vault": true, "aws-sm": true, "aws-ssm": true,
	"azure": true, "gcp": true, "doppler": true, "infisical": true, "akeyless": true, "sops": true,
}

func dataOnlySource(source model.Source) bool { return fragmentSourceTypes[source.Type] }

func forbiddenSourceFields(source model.Source, fields map[string]json.RawMessage) bool {
	if _, present := fields["sops"]; present {
		return true
	}
	if source.Type == "bitwarden" || source.Type == "onepassword" {
		_, present := fields["path"]
		return present
	}
	return source.Path != "" && strings.HasPrefix(source.Path, "repo:")
}

func validFragmentWorkloads(fields map[string]json.RawMessage) bool {
	raw, present := fields["workloads"]
	if !present {
		return true
	}
	var declarations map[string]map[string]json.RawMessage
	if jsondocument.Decode(raw, &declarations) != nil {
		return false
	}
	for name, declaration := range declarations {
		if !validFragmentIdentifier(name) || forbiddenWorkloadFields(declaration) {
			return false
		}
	}
	return true
}

func forbiddenWorkloadFields(fields map[string]json.RawMessage) bool {
	for _, name := range []string{"command", "command_sha256", "untrusted_code", "directory", "snapshot", "targets"} {
		if _, present := fields[name]; present {
			return true
		}
	}
	return false
}
