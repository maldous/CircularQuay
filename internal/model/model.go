// Package model defines CircularQuay's versioned, non-secret configuration contract.
package model

import "time"

// Policy contains declarations, never credential values.
type Policy struct {
	Version    int                 `json:"version" yaml:"version"`
	Sources    map[string]Source   `json:"sources" yaml:"sources"`
	Workloads  map[string]Workload `json:"workloads" yaml:"workloads"`
	Grants     []Grant             `json:"grants,omitempty" yaml:"grants,omitempty"`
	MCP        MCP                 `json:"mcp,omitempty" yaml:"mcp,omitempty"`
	Repository Repository          `json:"repository,omitempty" yaml:"repository,omitempty"`
}

// Repository declares an authenticated, signed ciphertext release source.
// Its signing trust is local policy; downloaded content cannot change it.
type Repository struct {
	URL                string `json:"url,omitempty" yaml:"url,omitempty"`
	Directory          string `json:"directory,omitempty" yaml:"directory,omitempty"`
	Ref                string `json:"ref,omitempty" yaml:"ref,omitempty"`
	AllowedSignersFile string `json:"allowed_signers_file,omitempty" yaml:"allowed_signers_file,omitempty"`
	InitialCommit      string `json:"initial_commit,omitempty" yaml:"initial_commit,omitempty"`
	TokenFile          string `json:"token_file,omitempty" yaml:"token_file,omitempty"`
}

// Source fixes a provider, its identity selector, and permitted resource scope.
type Source struct {
	Type            string   `json:"type" yaml:"type"`
	Endpoint        string   `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Account         string   `json:"account,omitempty" yaml:"account,omitempty"`
	Region          string   `json:"region,omitempty" yaml:"region,omitempty"`
	Project         string   `json:"project,omitempty" yaml:"project,omitempty"`
	Environment     string   `json:"environment,omitempty" yaml:"environment,omitempty"`
	Namespace       string   `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Mount           string   `json:"mount,omitempty" yaml:"mount,omitempty"`
	Path            string   `json:"path,omitempty" yaml:"path,omitempty"`
	Resources       []string `json:"resources,omitempty" yaml:"resources,omitempty"`
	Identity        string   `json:"identity,omitempty" yaml:"identity,omitempty"`
	TokenFile       string   `json:"token_file,omitempty" yaml:"token_file,omitempty"`
	Profile         string   `json:"profile,omitempty" yaml:"profile,omitempty"`
	SOPS            string   `json:"sops,omitempty" yaml:"sops,omitempty"`
	AgeIdentityFile string   `json:"age_identity_file,omitempty" yaml:"age_identity_file,omitempty"`
	VaultAllowlist  []string `json:"vault_allowlist,omitempty" yaml:"vault_allowlist,omitempty"`
	VaultKeys       []string `json:"vault_keys,omitempty" yaml:"vault_keys,omitempty"`
	AuditVerified   bool     `json:"audit_verified,omitempty" yaml:"audit_verified,omitempty"`
}

// Binding selects one exact value. Field selection does not narrow upstream grants.
type Binding struct {
	Name         string `json:"name" yaml:"name"`
	Source       string `json:"source" yaml:"source"`
	Resource     string `json:"resource,omitempty" yaml:"resource,omitempty"`
	Field        string `json:"field,omitempty" yaml:"field,omitempty"`
	Version      string `json:"version,omitempty" yaml:"version,omitempty"`
	AllowOffline bool   `json:"allow_offline,omitempty" yaml:"allow_offline,omitempty"`
	AllowPublish bool   `json:"allow_publish,omitempty" yaml:"allow_publish,omitempty"`
}

// Workload groups values with the same trust and lifecycle boundary.
type Workload struct {
	Mode          string            `json:"mode" yaml:"mode"`
	Bindings      []Binding         `json:"bindings" yaml:"bindings"`
	Environment   map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
	Command       []string          `json:"command,omitempty" yaml:"command,omitempty"`
	CommandSHA256 string            `json:"command_sha256,omitempty" yaml:"command_sha256,omitempty"`
	UntrustedCode bool              `json:"untrusted_code,omitempty" yaml:"untrusted_code,omitempty"`
	Directory     string            `json:"directory,omitempty" yaml:"directory,omitempty"`
	Snapshot      Snapshot          `json:"snapshot,omitempty" yaml:"snapshot,omitempty"`
	Targets       map[string]Target `json:"targets,omitempty" yaml:"targets,omitempty"`
	AuditRequired bool              `json:"audit_required,omitempty" yaml:"audit_required,omitempty"`
}

// Snapshot declares explicitly authorized offline access.
type Snapshot struct {
	Path         string   `json:"path,omitempty" yaml:"path,omitempty"`
	Recipients   []string `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	IdentityFile string   `json:"identity_file,omitempty" yaml:"identity_file,omitempty"`
	SOPS         string   `json:"sops,omitempty" yaml:"sops,omitempty"`
}

// Target declares a one-way delivery destination.
type Target struct {
	Type           string `json:"type" yaml:"type"`
	Path           string `json:"path,omitempty" yaml:"path,omitempty"`
	Endpoint       string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Account        string `json:"account,omitempty" yaml:"account,omitempty"`
	Project        string `json:"project,omitempty" yaml:"project,omitempty"`
	Environment    string `json:"environment,omitempty" yaml:"environment,omitempty"`
	Resource       string `json:"resource,omitempty" yaml:"resource,omitempty"`
	TokenFile      string `json:"token_file,omitempty" yaml:"token_file,omitempty"`
	AcceptSnapshot bool   `json:"accept_snapshot,omitempty" yaml:"accept_snapshot,omitempty"`
	Service        string `json:"service,omitempty" yaml:"service,omitempty"`
	Activation     string `json:"activation,omitempty" yaml:"activation,omitempty"`
}

// Grant is enforced before fetching values or invoking actions.
type Grant struct {
	Subject    string   `json:"subject" yaml:"subject"`
	Client     string   `json:"client,omitempty" yaml:"client,omitempty"`
	Workload   string   `json:"workload" yaml:"workload"`
	Operations []string `json:"operations" yaml:"operations"`
	Keys       []string `json:"keys,omitempty" yaml:"keys,omitempty"`
}

// MCP configures a protected resource; it is not an authorization server.
type MCP struct {
	Issuer    string   `json:"issuer,omitempty" yaml:"issuer,omitempty"`
	Audience  string   `json:"audience,omitempty" yaml:"audience,omitempty"`
	JWKSURL   string   `json:"jwks_url,omitempty" yaml:"jwks_url,omitempty"`
	Address   string   `json:"address,omitempty" yaml:"address,omitempty"`
	Origins   []string `json:"origins,omitempty" yaml:"origins,omitempty"`
	Hosts     []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	AuditPath string   `json:"audit_path,omitempty" yaml:"audit_path,omitempty"`
}

// Value preserves lifecycle metadata. It must never be serialized in diagnostics.
type Value struct {
	Data          string
	Version       string
	ExpiresAt     time.Time
	Leased        bool
	Renewable     bool
	AuditVerified bool
}

// Receipt contains provenance only, never values or value hashes.
type Receipt struct {
	Source   string `json:"source"`
	Resource string `json:"resource,omitempty"`
	Version  string `json:"version,omitempty"`
}

// Generation is complete before any activation. Values are intentionally private to encoding.
type Generation struct {
	Workload    string
	Values      map[string]Value
	Receipts    []Receipt
	RetrievedAt time.Time
	Offline     bool
}
