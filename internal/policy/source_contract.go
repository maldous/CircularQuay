package policy

import (
	"path/filepath"

	"github.com/maldous/CircularQuay/internal/model"
)

type sourceField uint32

const (
	fieldEndpoint sourceField = 1 << iota
	fieldAccount
	fieldRegion
	fieldProject
	fieldEnvironment
	fieldNamespace
	fieldMount
	fieldPath
	fieldResources
	fieldIdentity
	fieldTokenFile
	fieldProfile
	fieldSOPS
	fieldAgeIdentityFile
	fieldVaultAllowlist
	fieldVaultKeys
)

type sourceContract struct {
	fields sourceField
	check  func(model.Source) error
}

var sourceContracts = map[string]sourceContract{
	"json":        {fieldPath, requireSourcePath},
	"env":         {fieldPath | fieldResources | fieldIdentity, requireEmptyIdentity},
	"sops":        {fieldPath | fieldIdentity | fieldTokenFile | fieldSOPS | fieldAgeIdentityFile | fieldVaultAllowlist | fieldVaultKeys, validateSOPSContract},
	"openbao":     {fieldEndpoint | fieldNamespace | fieldMount | fieldResources | fieldIdentity | fieldTokenFile, requireVaultContract},
	"vault":       {fieldEndpoint | fieldNamespace | fieldMount | fieldResources | fieldIdentity | fieldTokenFile, requireVaultContract},
	"doppler":     {fieldEndpoint | fieldProject | fieldEnvironment | fieldResources | fieldIdentity | fieldTokenFile, requireDopplerContract},
	"infisical":   {fieldEndpoint | fieldProject | fieldEnvironment | fieldPath | fieldResources | fieldIdentity | fieldTokenFile, requireInfisicalContract},
	"akeyless":    {fieldEndpoint | fieldResources | fieldIdentity | fieldTokenFile, requireAkeylessContract},
	"bitwarden":   {fieldProject | fieldPath | fieldResources | fieldIdentity | fieldTokenFile, requireVendorContract},
	"onepassword": {fieldProject | fieldPath | fieldResources | fieldIdentity | fieldTokenFile, requireVendorContract},
	"aws-sm":      {fieldAccount | fieldRegion | fieldResources | fieldIdentity, requireAWSContract},
	"aws-ssm":     {fieldAccount | fieldRegion | fieldResources | fieldIdentity, requireAWSContract},
	"azure":       {fieldEndpoint | fieldAccount | fieldResources | fieldIdentity | fieldTokenFile | fieldProfile, requireAzureContract},
	"gcp":         {fieldAccount | fieldProject | fieldResources | fieldIdentity, requireGCPContract},
}

// validateSourceContract rejects provider settings which no compiled adapter
// consumes. It keeps a policy from falsely asserting an identity or boundary
// that the subsequent reader would ignore or reject only after resolution.
func validateSourceContract(source model.Source) error {
	contract, found := sourceContracts[source.Type]
	if !found || !sourceContractText(source) || !sourceFieldsAllowed(source, contract.fields) {
		return invalid("source_contract")
	}
	return contract.check(source)
}

func sourceFieldsAllowed(source model.Source, allowed sourceField) bool {
	return sourcePresentFields(source)&^allowed == 0
}

func sourcePresentFields(source model.Source) sourceField {
	var fields sourceField
	for _, entry := range []struct {
		field sourceField
		set   bool
	}{
		{fieldEndpoint, source.Endpoint != ""}, {fieldAccount, source.Account != ""}, {fieldRegion, source.Region != ""},
		{fieldProject, source.Project != ""}, {fieldEnvironment, source.Environment != ""}, {fieldNamespace, source.Namespace != ""},
		{fieldMount, source.Mount != ""}, {fieldPath, source.Path != ""}, {fieldResources, len(source.Resources) != 0},
		{fieldIdentity, source.Identity != ""}, {fieldTokenFile, source.TokenFile != ""}, {fieldProfile, source.Profile != ""},
		{fieldSOPS, source.SOPS != ""}, {fieldAgeIdentityFile, source.AgeIdentityFile != ""},
		{fieldVaultAllowlist, len(source.VaultAllowlist) != 0}, {fieldVaultKeys, len(source.VaultKeys) != 0},
	} {
		if entry.set {
			fields |= entry.field
		}
	}
	return fields
}

func sourceContractText(source model.Source) bool {
	for _, value := range []string{source.Account, source.Region, source.Project, source.Environment, source.Namespace, source.Mount, source.Profile} {
		if !safeText(value) {
			return false
		}
	}
	return true
}

func requireSourcePath(source model.Source) error {
	if source.Path == "" {
		return invalid("source_contract")
	}
	return nil
}

func requireEmptyIdentity(source model.Source) error {
	if source.Identity != "" && source.Identity != "none" {
		return invalid("source_contract")
	}
	return nil
}

func validateSOPSContract(source model.Source) error {
	if source.Path == "" || !filepath.IsAbs(source.SOPS) {
		return invalid("source_contract")
	}
	if source.TokenFile != "" {
		return validateSOPSTokenContract(source)
	}
	if (source.Identity != "" && source.Identity != "none") || source.AgeIdentityFile == "" {
		return invalid("source_contract")
	}
	return nil
}

func validateSOPSTokenContract(source model.Source) error {
	if source.Identity != "token_file" || source.AgeIdentityFile != "" || len(source.VaultAllowlist) == 0 || len(source.VaultKeys) == 0 {
		return invalid("source_contract")
	}
	return nil
}

func requireTokenContract(source model.Source, scope bool) error {
	if !scope || source.Identity != "token_file" || source.TokenFile == "" {
		return invalid("source_contract")
	}
	return nil
}

func requireVendorContract(source model.Source) error {
	if !filepath.IsAbs(source.Path) {
		return invalid("source_contract")
	}
	return requireTokenContract(source, source.Project != "")
}

func requireVaultContract(source model.Source) error {
	return requireTokenContract(source, source.Mount != "")
}

func requireDopplerContract(source model.Source) error {
	return requireTokenContract(source, source.Project != "" && source.Environment != "")
}

func requireInfisicalContract(source model.Source) error {
	return requireTokenContract(source, source.Project != "" && source.Environment != "")
}

func requireAkeylessContract(source model.Source) error { return requireTokenContract(source, true) }

func requireAWSContract(source model.Source) error {
	if source.Account == "" || source.Region == "" || (source.Identity != "ec2" && source.Identity != "ecs") {
		return invalid("source_contract")
	}
	return nil
}

func requireAzureContract(source model.Source) error {
	if source.Account == "" || source.Profile == "" {
		return invalid("source_contract")
	}
	if source.Identity == "workload" && source.TokenFile != "" {
		return nil
	}
	if source.Identity == "managed" && source.TokenFile == "" {
		return nil
	}
	return invalid("source_contract")
}

func requireGCPContract(source model.Source) error {
	if source.Account == "" || source.Account == "default" || source.Project == "" || source.Identity != "metadata" {
		return invalid("source_contract")
	}
	return nil
}
