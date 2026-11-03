package policy

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestTargetContractsAcceptImplementedShapes(t *testing.T) {
	nativePath := filepath.Join(string(filepath.Separator), "protected", "target")
	for name, target := range map[string]model.Target{
		"file":       {Type: "file", Path: nativePath, AcceptSnapshot: true},
		"systemd":    {Type: "systemd", Path: nativePath, Service: "demo.service", Activation: "reload", AcceptSnapshot: true},
		"windows":    {Type: "windows", Path: nativePath, Resource: "S-1-5-80-123", AcceptSnapshot: true},
		"cloudflare": {Type: "cloudflare", Account: "account", Resource: "worker", TokenFile: nativePath, AcceptSnapshot: true},
		"netlify":    {Type: "netlify", Account: "account", Project: "site", Environment: "production", TokenFile: nativePath, AcceptSnapshot: true},
		"windmill":   {Type: "windmill", Endpoint: "https://windmill.example", Project: "workspace", Resource: "f/folder/config", TokenFile: nativePath, AcceptSnapshot: true},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateTargetContract(target); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTargetContractsRejectIgnoredOrIncompleteFields(t *testing.T) {
	nativePath := filepath.Join(string(filepath.Separator), "protected", "target")
	cases := map[string]model.Target{
		"file_remote_token":              {Type: "file", Path: nativePath, TokenFile: nativePath},
		"file_activation":                {Type: "file", Path: nativePath, Activation: "none"},
		"systemd_remote_account":         {Type: "systemd", Path: nativePath, Account: "account"},
		"systemd_service_without_action": {Type: "systemd", Path: nativePath, Service: "demo.service"},
		"windows_reload":                 {Type: "windows", Path: nativePath, Service: "demo", Activation: "reload"},
		"windows_service_without_action": {Type: "windows", Path: nativePath, Service: "demo"},
		"cloudflare_missing_token":       {Type: "cloudflare", Account: "account", Resource: "worker"},
		"cloudflare_project":             {Type: "cloudflare", Account: "account", Resource: "worker", TokenFile: nativePath, Project: "unused"},
		"netlify_resource":               {Type: "netlify", Account: "account", Project: "site", Environment: "production", TokenFile: nativePath, Resource: "unused"},
		"windmill_path":                  {Type: "windmill", Endpoint: "https://windmill.example", Project: "workspace", Resource: "f/folder/config", TokenFile: nativePath, Path: "unused"},
		"windmill_account":               {Type: "windmill", Endpoint: "https://windmill.example", Project: "workspace", Resource: "f/folder/config", TokenFile: nativePath, Account: "unused"},
		"windmill_missing_endpoint":      {Type: "windmill", Project: "workspace", Resource: "f/folder/config", TokenFile: nativePath},
		"selector_traversal":             {Type: "cloudflare", Account: "account", Resource: "../worker", TokenFile: nativePath},
		"selector_oversize":              {Type: "cloudflare", Account: strings.Repeat("a", maximumTargetSelectorBytes+1), Resource: "worker", TokenFile: nativePath},
	}
	for name, target := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateTargetContract(target); err == nil {
				t.Fatal("invalid target contract accepted")
			}
		})
	}
}
