package source

import (
	"context"
	"net/http"
	"sync"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/observability"
	"github.com/maldous/CircularQuay/internal/transport"
)

// Capability states an implementation boundary, not a reliability certification.
type Capability struct {
	Type      string `json:"type"`
	Read      bool   `json:"read"`
	Versioned bool   `json:"versioned"`
	Maturity  string `json:"maturity"`
}

// Registry contains only explicitly registered first-party adapters.
type Registry struct {
	mu      sync.RWMutex
	readers map[string]Reader
}

// NewRegistry builds the default registry without performing network calls.
func NewRegistry() *Registry { return NewRegistryWithClient(nil) }

// NewRegistryWithClient permits transport injection for isolated contract tests.
func NewRegistryWithClient(client *http.Client) *Registry {
	c := transport.New()
	if client != nil {
		c.HTTP = client
	}
	r := &Registry{readers: map[string]Reader{}}
	r.Register("json", ReadFunc(readJSON))
	r.Register("env", ReadFunc(readEnvironment))
	r.Register("bitwarden", ReadFunc(readVendor))
	r.Register("onepassword", ReadFunc(readVendor))
	r.Register("openbao", ReadFunc(func(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
		return readVault(ctx, c, s, b)
	}))
	r.Register("vault", ReadFunc(func(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
		return readVault(ctx, c, s, b)
	}))
	for _, name := range []string{"doppler", "infisical", "akeyless"} {
		provider := name
		r.Register(name, ReadFunc(func(ctx context.Context, s model.Source, b model.Binding) (model.Value, error) {
			return readHosted(ctx, c, provider, s, b)
		}))
	}
	registerCloud(r)
	return r
}

// Register enrolls a compiled-in adapter. Configuration cannot load executable plugins.
func (r *Registry) Register(name string, reader Reader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.readers[name] = reader
}

// Read enforces exact scope before any adapter receives a value request.
func (r *Registry) Read(ctx context.Context, s model.Source, b model.Binding) (value model.Value, err error) {
	ctx, complete := observability.Start(ctx, observability.Operation{Phase: observability.PhaseSource, ProviderKind: s.Type})
	defer func() { complete(observability.OutcomeForError(err)) }()
	if !allowed(s, b) {
		return model.Value{}, fault.New(fault.Denied, "source_scope")
	}
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	r.mu.RLock()
	reader := r.readers[s.Type]
	r.mu.RUnlock()
	if reader == nil {
		return model.Value{}, fault.New(fault.Unsupported, "source_type")
	}
	value, err = reader.Read(ctx, s, b)
	return value, err
}

func validateVersionCapability(s model.Source, b model.Binding) error {
	if b.Version != "" && !model.SourceSupportsVersion(s.Type) {
		return fault.New(fault.Invalid, "source_version_unsupported")
	}
	return nil
}

// validateReturnedVersion prevents an adapter from claiming that an upstream
// selector was honored solely because it sent that selector in a request.
func validateReturnedVersion(b model.Binding, actual string) error {
	if b.Version != "" && actual != b.Version {
		return fault.New(fault.Trust, "source_version_mismatch")
	}
	return nil
}

func allowed(s model.Source, b model.Binding) bool {
	if s.Type == "json" || s.Type == "sops" {
		return b.Resource == "" || b.Resource == s.Path
	}
	for _, name := range s.Resources {
		if name == b.Resource && name != "" {
			return true
		}
	}
	return false
}

// Capabilities identifies source implementations and preserves live-test uncertainty.
func Capabilities() []Capability {
	var out []Capability
	for _, name := range []string{"json", "env", "sops", "openbao", "vault", "doppler", "infisical", "akeyless"} {
		out = append(out, Capability{Type: name, Read: true, Versioned: model.SourceSupportsVersion(name), Maturity: "contract-tested; live qualification pending"})
	}
	for _, name := range []string{"aws-sm", "aws-ssm", "azure", "gcp"} {
		out = append(out, Capability{Type: name, Read: true, Versioned: model.SourceSupportsVersion(name), Maturity: "official SDK; scope/unit tests; live qualification pending"})
	}
	for _, name := range []string{"bitwarden", "onepassword"} {
		out = append(out, Capability{Type: name, Read: true, Maturity: "fixed vendor CLI; live qualification pending"})
	}
	return out
}
