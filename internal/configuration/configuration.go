// Package configuration resolves one protected master policy and its
// master-authorized, pinned data-only fragments into one validated runtime
// policy. It never reads provider values or executes workloads.
package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"github.com/maldous/CircularQuay/internal/policy"
	"github.com/maldous/CircularQuay/internal/repository"
)

const (
	maximumIncludes       = 16
	maximumDepth          = 8
	maximumResolvedBytes  = 8 << 20
	maximumFragmentBytes  = 1 << 20
	maximumNamespaceBytes = 24
)

// Provenance identifies an accepted shared fragment without revealing its
// filesystem path, repository URL, raw bytes, credential references, or values.
// Local fragments never contribute provenance metadata.
type Provenance struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
	SHA256    string `json:"sha256"`
}

// Result is a complete, validated configuration snapshot. Callers must retain
// this snapshot for one operation instead of resolving fragments repeatedly.
type Result struct {
	Policy     model.Policy `json:"-"`
	provenance []Provenance
}

// Provenance returns a copy of the bounded accepted-fragment metadata.
func (r Result) Provenance() []Provenance { return append([]Provenance(nil), r.provenance...) }

type resolver struct {
	readFile func(string, bool) ([]byte, error)
	fetch    func(context.Context, model.Repository, string) (string, error)
	current  func(context.Context, model.Repository) (string, error)
	now      func() time.Time
}

var nativeResolver = resolver{
	readFile: platform.ReadProtected,
	fetch:    repository.FetchPinned,
	current:  repository.CurrentContext,
	now:      time.Now,
}

// Load reads a protected master policy and resolves its complete configuration
// graph. It performs no provider reads, shell execution, or ambient discovery.
func Load(ctx context.Context, path string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, failure(fault.Unavailable, "cancelled")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Result{}, failure(fault.Invalid, "master_location")
	}
	data, err := nativeResolver.readFile(absolute, false)
	if err != nil {
		return Result{}, failure(fault.Trust, "master_file")
	}
	return parse(ctx, data, absolute, nativeResolver)
}

// Parse resolves a master document supplied by its caller. The caller owns the
// master bytes; all included local files remain protected-file reads.
func Parse(ctx context.Context, data []byte, path string) (Result, error) {
	return parse(ctx, data, path, nativeResolver)
}

// ReadRoot strictly decodes and path-binds one master declaration without
// resolving includes, reading referenced files, or contacting repositories.
// Callers can use it for a root-owned local admission check before allowing a
// configuration graph to request repository access. Its result is not a
// runnable policy until Load or Parse returns a complete Result.
func ReadRoot(data []byte, path string) (model.Policy, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "master_location")
	}
	policy, err := policy.DecodeDocument(data, absolute)
	if err != nil {
		return model.Policy{}, failure(fault.Invalid, "master_document")
	}
	return policy, nil
}

func parse(ctx context.Context, data []byte, path string, r resolver) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, failure(fault.Unavailable, "cancelled")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Result{}, failure(fault.Invalid, "master_location")
	}
	master, err := ReadRoot(data, absolute)
	if err != nil {
		return Result{}, err
	}
	if master.Composition == nil {
		return resolveSingle(ctx, data, absolute)
	}
	return r.resolveMaster(ctx, master, absolute, len(data))
}

func resolveSingle(ctx context.Context, data []byte, path string) (Result, error) {
	resolved, err := policy.Parse(data, path)
	if err != nil {
		return Result{}, err
	}
	guard := &model.ConfigurationGuard{Inputs: []string{path}, SourceDomains: make(map[string]string)}
	for source := range resolved.Sources {
		guard.SourceDomains[source] = "root"
	}
	resolved.ConfigurationGuard = guard
	if err = policy.ValidateResolvedReferences(resolved, path); err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, failure(fault.Unavailable, "cancelled")
	}
	return Result{Policy: resolved}, nil
}

func (r resolver) resolveMaster(ctx context.Context, master model.Policy, masterPath string, masterBytes int) (Result, error) {
	if !validMaster(master.Composition) {
		return Result{}, failure(fault.Invalid, "composition")
	}
	plan, err := newPlan(master.Composition.Includes)
	if err != nil {
		return Result{}, err
	}
	if masterBytes > maximumFragmentBytes {
		return Result{}, failure(fault.Invalid, "master_size")
	}
	compiled := copyPolicy(withoutComposition(master))
	guard := &model.ConfigurationGuard{Inputs: []string{masterPath}, SourceDomains: make(map[string]string), LocalRoots: make(map[string]string)}
	for source := range compiled.Sources {
		guard.SourceDomains[source] = "root"
	}
	provenance := make([]Provenance, 0, len(plan.includes))
	total := masterBytes
	for _, include := range plan.includes {
		candidate, err := r.resolveInclude(ctx, include)
		if err != nil {
			return Result{}, err
		}
		total += len(candidate.data)
		if total > maximumResolvedBytes {
			return Result{}, failure(fault.Invalid, "composition_size")
		}
		if err = mergeFragment(&compiled, candidate.policy, include, plan); err != nil {
			return Result{}, err
		}
		provenance = recordFragmentGuard(guard, provenance, include, candidate)
	}
	resolved, err := applyEdits(compiled, master.Composition, masterPath)
	if err != nil {
		return Result{}, err
	}
	if hasDuplicateSourceIdentity(resolved.Sources) {
		return Result{}, failure(fault.Invalid, "composition_conflict")
	}
	resolved.ConfigurationGuard = guard
	if err = policy.ValidateResolvedReferences(resolved, masterPath); err != nil {
		return Result{}, err
	}
	if err = ctx.Err(); err != nil {
		return Result{}, failure(fault.Unavailable, "cancelled")
	}
	if err = plan.cacheFresh(r.now); err != nil {
		return Result{}, err
	}
	return Result{Policy: resolved, provenance: provenance}, nil
}

type includeCandidate struct {
	data   []byte
	path   string
	roots  []string
	policy model.Policy
}

func (r resolver) resolveInclude(ctx context.Context, include model.Include) (includeCandidate, error) {
	if err := ctx.Err(); err != nil {
		return includeCandidate{}, failure(fault.Unavailable, "cancelled")
	}
	data, path, roots, err := r.readInclude(ctx, include)
	if err != nil {
		return includeCandidate{}, err
	}
	child, err := decodeFragment(data, include)
	if err != nil {
		return includeCandidate{}, err
	}
	return includeCandidate{data: data, path: path, roots: roots, policy: child}, nil
}

func recordFragmentGuard(guard *model.ConfigurationGuard, provenance []Provenance, include model.Include, candidate includeCandidate) []Provenance {
	guard.Inputs = appendGuardInputs(guard.Inputs, candidate.path, include.Repository)
	guard.ReadOnlyRoots = append(guard.ReadOnlyRoots, candidate.roots...)
	sources := fragmentSources(candidate.policy, include.Namespace)
	for _, source := range sources {
		guard.SourceDomains[source] = include.ID
		if include.Visibility == "local" {
			guard.LocalRoots[source] = include.BaseDirectory
		}
	}
	if include.Visibility == "local" {
		guard.LocalSources = append(guard.LocalSources, sources...)
		return provenance
	}
	return append(provenance, Provenance{ID: include.ID, Namespace: include.Namespace, SHA256: include.SHA256})
}

func withoutComposition(policy model.Policy) model.Policy {
	policy.Composition = nil
	policy.ConfigurationGuard = nil
	return policy
}

func failure(kind fault.Kind, operation string) error {
	return fault.New(kind, "configuration_"+operation)
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func matchesDigest(data []byte, expected string) bool {
	digest := sha256.Sum256(data)
	return validDigest(expected) && hex.EncodeToString(digest[:]) == expected
}

func within(base, child string) bool {
	base = comparablePath(base)
	child = comparablePath(child)
	return child == base || strings.HasPrefix(child, base+string(filepath.Separator))
}

func samePath(first, second string) bool { return comparablePath(first) == comparablePath(second) }

func comparablePath(value string) string {
	value = filepath.Clean(value)
	if runtime.GOOS == "windows" {
		return strings.ToLower(value)
	}
	return value
}

func copyPolicy(policy model.Policy) model.Policy {
	policy.Sources = copySources(policy.Sources)
	policy.Workloads = copyWorkloads(policy.Workloads)
	policy.Grants = append([]model.Grant(nil), policy.Grants...)
	return policy
}

func copySources(values map[string]model.Source) map[string]model.Source {
	result := make(map[string]model.Source, len(values))
	for name, value := range values {
		result[name] = value
	}
	return result
}

func copyWorkloads(values map[string]model.Workload) map[string]model.Workload {
	result := make(map[string]model.Workload, len(values))
	for name, value := range values {
		result[name] = value
	}
	return result
}

func appendGuardInputs(inputs []string, child string, repository *model.Repository) []string {
	inputs = append(inputs, child)
	if repository == nil {
		return inputs
	}
	return append(inputs, repository.AllowedSignersFile, repository.TokenFile)
}
