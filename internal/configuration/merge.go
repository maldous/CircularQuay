package configuration

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func mergeFragment(master *model.Policy, child model.Policy, include model.Include, plan includePlan) error {
	if master.Sources == nil {
		master.Sources = make(map[string]model.Source)
	}
	if master.Workloads == nil {
		master.Workloads = make(map[string]model.Workload)
	}
	qualifiedSources, err := qualifySources(child.Sources, include, plan)
	if err != nil {
		return err
	}
	qualifiedWorkloads, err := qualifyWorkloads(child.Workloads, include, plan)
	if err != nil {
		return err
	}
	if collision(master.Sources, qualifiedSources) || collision(master.Workloads, qualifiedWorkloads) || duplicateSourceIdentity(master.Sources, qualifiedSources) {
		return failure(fault.Invalid, "composition_conflict")
	}
	for name, source := range qualifiedSources {
		master.Sources[name] = source
	}
	for name, workload := range qualifiedWorkloads {
		master.Workloads[name] = workload
	}
	return nil
}

func qualifySources(sources map[string]model.Source, include model.Include, _ includePlan) (map[string]model.Source, error) {
	result := make(map[string]model.Source, len(sources))
	for name, source := range sources {
		qualified, err := qualify(include.Namespace, name)
		if err != nil {
			return nil, err
		}
		result[qualified] = source
	}
	return result, nil
}

func qualifyWorkloads(workloads map[string]model.Workload, include model.Include, plan includePlan) (map[string]model.Workload, error) {
	result := make(map[string]model.Workload, len(workloads))
	for name, workload := range workloads {
		qualified, err := qualify(include.Namespace, name)
		if err != nil {
			return nil, err
		}
		for index := range workload.Bindings {
			source, err := qualifyBindingSource(workload.Bindings[index].Source, include, plan)
			if err != nil {
				return nil, err
			}
			workload.Bindings[index].Source = source
		}
		result[qualified] = workload
	}
	return result, nil
}

func qualifyBindingSource(source string, include model.Include, plan includePlan) (string, error) {
	if strings.Contains(source, "__") {
		parts := strings.SplitN(source, "__", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !plan.permits(parts[0], parts[1], include) {
			return "", failure(fault.Denied, "include_cross_reference")
		}
		return source, nil
	}
	return qualify(include.Namespace, source)
}

func qualify(namespace, name string) (string, error) {
	value := namespace + "__" + name
	if !validQualifiedName(value) {
		return "", failure(fault.Invalid, "include_qualified_name")
	}
	return value, nil
}

func validQualifiedName(value string) bool { return len(value) <= 64 && validIdentifier(value) }

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, c := range value {
		if !validIdentifierCharacter(c) || index == 0 && !validIdentifierStart(c) {
			return false
		}
	}
	return true
}

func validIdentifierCharacter(value rune) bool {
	return asciiLowercase(value) || asciiDigit(value) || value == '_' || value == '-'
}

func validIdentifierStart(value rune) bool { return asciiLowercase(value) || asciiDigit(value) }

func asciiLowercase(value rune) bool { return value >= 'a' && value <= 'z' }

func asciiDigit(value rune) bool { return value >= '0' && value <= '9' }

func collision[T any](existing, additions map[string]T) bool {
	for name := range additions {
		if _, found := existing[name]; found {
			return true
		}
	}
	return false
}

func duplicateSourceIdentity(existing, additions map[string]model.Source) bool {
	combined := make(map[string]model.Source, len(existing)+len(additions))
	for name, source := range existing {
		combined[name] = source
	}
	for name, source := range additions {
		combined[name] = source
	}
	return hasDuplicateSourceIdentity(combined)
}

func hasDuplicateSourceIdentity(sources map[string]model.Source) bool {
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		identity := sourceIdentity(source)
		if _, found := seen[identity]; found {
			return true
		}
		seen[identity] = struct{}{}
	}
	return false
}

type sourceDeclarationIdentity struct {
	Type            string
	Endpoint        string
	Account         string
	Region          string
	Project         string
	Environment     string
	Namespace       string
	Mount           string
	Path            string
	Resources       []string
	Identity        string
	TokenFile       string
	Profile         string
	SOPS            string
	AgeIdentityFile string
	VaultAllowlist  []string
	VaultKeys       []string
}

func sourceIdentity(source model.Source) string {
	identity := sourceDeclarationIdentity{
		Type: source.Type, Endpoint: source.Endpoint, Account: source.Account, Region: source.Region, Project: source.Project,
		Environment: source.Environment, Namespace: source.Namespace, Mount: source.Mount, Path: source.Path,
		Resources: sortedStrings(source.Resources), Identity: source.Identity, TokenFile: source.TokenFile, Profile: source.Profile,
		SOPS: source.SOPS, AgeIdentityFile: source.AgeIdentityFile, VaultAllowlist: sortedStrings(source.VaultAllowlist), VaultKeys: sortedStrings(source.VaultKeys),
	}
	data, _ := json.Marshal(identity)
	return string(data)
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func fragmentSources(child model.Policy, namespace string) []string {
	result := make([]string, 0, len(child.Sources))
	for name := range child.Sources {
		qualified, err := qualify(namespace, name)
		if err == nil {
			result = append(result, qualified)
		}
	}
	return result
}
