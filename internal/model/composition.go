package model

import (
	"encoding/json"
	"time"
)

// Composition declares a master-owned, pinned graph of data-only policy
// fragments. Source order never grants permission to overwrite another owner.
type Composition struct {
	Includes  []Include    `json:"includes" yaml:"includes"`
	Conflicts string       `json:"conflicts" yaml:"conflicts"`
	Overrides []PolicyEdit `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Locks     []PolicyEdit `json:"locks,omitempty" yaml:"locks,omitempty"`
}

// Include binds one department-owned declaration to a master-authorized
// location, immutable digest, namespace and reference directory. Exactly one
// of File or Repository is selected; children cannot define further locations.
type Include struct {
	ID             string      `json:"id" yaml:"id"`
	Namespace      string      `json:"namespace" yaml:"namespace"`
	SHA256         string      `json:"sha256" yaml:"sha256"`
	File           string      `json:"file,omitempty" yaml:"file,omitempty"`
	Repository     *Repository `json:"repository,omitempty" yaml:"repository,omitempty"`
	Path           string      `json:"path,omitempty" yaml:"path,omitempty"`
	Commit         string      `json:"commit,omitempty" yaml:"commit,omitempty"`
	BaseDirectory  string      `json:"base_directory" yaml:"base_directory"`
	DependsOn      []string    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Visibility     string      `json:"visibility" yaml:"visibility"`
	Refresh        string      `json:"refresh,omitempty" yaml:"refresh,omitempty"`
	CacheExpiresAt *time.Time  `json:"cache_expires_at,omitempty" yaml:"cache_expires_at,omitempty"`
}

// PolicyEdit is a master-owned RFC 6901 field override or final equality lock.
// It contains configuration declarations, never provider credential values.
type PolicyEdit struct {
	Pointer string          `json:"pointer" yaml:"pointer"`
	Value   json.RawMessage `json:"value" yaml:"value"`
}

// ConfigurationGuard retains resolver-only path and private-domain boundaries
// after the master document has been compiled into an ordinary runtime Policy.
// It is deliberately absent from JSON/YAML and from diagnostic output.
type ConfigurationGuard struct {
	Inputs        []string
	ReadOnlyRoots []string
	LocalSources  []string
	SourceDomains map[string]string
	LocalRoots    map[string]string
}
