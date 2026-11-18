package policy

import (
	"path/filepath"

	"github.com/maldous/CircularQuay/internal/model"
)

const maximumTargetSelectorBytes = 4096

// validateTargetContract rejects declarations that an implemented target adapter
// would otherwise ignore. A target has exactly one delivery authority shape.
func validateTargetContract(target model.Target) error {
	if !validTargetText(target) {
		return invalid("target_fields")
	}
	switch target.Type {
	case "file":
		return validateFileTarget(target)
	case "systemd":
		return validateSystemdTarget(target)
	case "windows":
		return validateWindowsTarget(target)
	case "cloudflare":
		return validateCloudflareTarget(target)
	case "netlify":
		return validateNetlifyTarget(target)
	case "windmill":
		return validateWindmillTarget(target)
	default:
		return invalid("target_type")
	}
}

func validTargetText(target model.Target) bool {
	return targetPathText(target.Path) && targetPathText(target.TokenFile) &&
		targetEndpointText(target.Endpoint) && targetSelector(target.Account, false) && targetSelector(target.Project, false) &&
		targetSelector(target.Environment, false) && targetSelector(target.Resource, false) &&
		targetSelector(target.Service, false)
}

func targetPathText(value string) bool {
	return len(value) <= maximumTargetSelectorBytes && safeText(value)
}

func targetEndpointText(value string) bool {
	return len(value) <= maximumTargetSelectorBytes && safeText(value)
}

func targetSelector(value string, required bool) bool {
	if value == "" {
		return !required
	}
	return len(value) <= maximumTargetSelectorBytes && safeSelector(value)
}

func nativeTargetPath(target model.Target) bool {
	return target.Path != "" && filepath.IsAbs(target.Path)
}

func noRemoteFields(target model.Target) bool {
	return target.Endpoint == "" && target.Account == "" && target.Project == "" &&
		target.Environment == "" && target.TokenFile == ""
}

func validateFileTarget(target model.Target) error {
	if !nativeTargetPath(target) || !noRemoteFields(target) || target.Resource != "" ||
		target.Service != "" || target.Activation != "" {
		return invalid("file_target")
	}
	return nil
}

func validateSystemdTarget(target model.Target) error {
	if !nativeTargetPath(target) || !noRemoteFields(target) || target.Resource != "" ||
		!validNativeActivation(target, true) {
		return invalid("systemd_target")
	}
	return nil
}

func validateWindowsTarget(target model.Target) error {
	if !nativeTargetPath(target) || !noRemoteFields(target) || target.Activation == "reload" ||
		!validNativeActivation(target, false) {
		return invalid("windows_target")
	}
	return nil
}

func validNativeActivation(target model.Target, allowReload bool) bool {
	if target.Activation == "" {
		return target.Service == ""
	}
	if target.Activation == "none" {
		return target.Service == ""
	}
	if target.Activation != "restart" && (!allowReload || target.Activation != "reload") {
		return false
	}
	return target.Service != ""
}

func validateCloudflareTarget(target model.Target) error {
	if !targetSelector(target.Account, true) || !targetSelector(target.Resource, true) ||
		target.TokenFile == "" || !filepath.IsAbs(target.TokenFile) || target.Project != "" || target.Environment != "" ||
		target.Service != "" || target.Activation != "" {
		return invalid("cloudflare_target")
	}
	return nil
}

func validateNetlifyTarget(target model.Target) error {
	if !targetSelector(target.Account, true) || !targetSelector(target.Project, true) ||
		!targetSelector(target.Environment, true) || target.TokenFile == "" || !filepath.IsAbs(target.TokenFile) ||
		target.Resource != "" || target.Service != "" || target.Activation != "" {
		return invalid("netlify_target")
	}
	return nil
}

func validateWindmillTarget(target model.Target) error {
	if !validWindmillRequiredFields(target) || windmillHasIgnoredFields(target) {
		return invalid("windmill_target")
	}
	return nil
}

func validWindmillRequiredFields(target model.Target) bool {
	return targetSelector(target.Project, true) && targetSelector(target.Resource, true) &&
		target.Endpoint != "" && target.TokenFile != "" && filepath.IsAbs(target.TokenFile)
}

func windmillHasIgnoredFields(target model.Target) bool {
	return target.Path != "" || target.Account != "" || target.Environment != "" ||
		target.Service != "" || target.Activation != ""
}
