package assurance

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/maldous/CircularQuay/internal/jsondocument"
	"github.com/maldous/CircularQuay/internal/platform"
)

const (
	toolManifestVersion  = 1
	maxToolManifestBytes = 64 << 10
	maxToolBinaryBytes   = 128 << 20
	maxToolPathBytes     = 4096
)

var toolIDs = []string{"git", "go", "gofmt", "staticcheck", "gosec", "govulncheck", "gitleaks", "actionlint", "sonar-scanner"}

// ToolManifestSchema is the v1 structural contract for an operator-supplied
// tool manifest. Loading also performs native path and binary-digest checks.
//
//go:embed tool_manifest.schema.json
var ToolManifestSchema []byte

// ToolManifest is an operator declaration. Its metadata is not evidence that
// a tool was executed, that a probe succeeded, or that a publisher is trusted.
type ToolManifest struct {
	Version  int          `json:"version"`
	Platform string       `json:"platform"`
	Tools    []ToolEntry  `json:"tools"`
	Helpers  []ToolHelper `json:"helpers,omitempty"`
}

type toolManifestPresence struct {
	Version  int                 `json:"version"`
	Platform string              `json:"platform"`
	Tools    []toolEntryPresence `json:"tools"`
	Helpers  json.RawMessage     `json:"helpers,omitempty"`
}

type toolEntryPresence struct {
	ID    string          `json:"id"`
	Rules json.RawMessage `json:"rules,omitempty"`
}

// ToolEntry binds one fixed tool identifier to an external executable.
type ToolEntry struct {
	ID                  string     `json:"id"`
	Path                string     `json:"path"`
	SHA256              string     `json:"sha256"`
	Version             string     `json:"version"`
	VersionOutputSHA256 string     `json:"version_output_sha256"`
	Rules               *ToolRules `json:"rules,omitempty"`
}

// ToolRules declares analyzer-rule metadata expected by a later probe. It is
// neither evidence of an executed probe nor a binding for tool dependencies
// such as a scanner runtime, compiler, GOROOT, or external rules files.
// Execution-layer qualification must independently bind those dependencies.
type ToolRules struct {
	Kind              string     `json:"kind"`
	RulesetSHA256     string     `json:"ruleset_sha256"`
	Input             *ToolInput `json:"input,omitempty"`
	ServerVersion     string     `json:"server_version,omitempty"`
	ActiveRulesSHA256 string     `json:"active_rules_sha256,omitempty"`
	ProfileSHA256     string     `json:"profile_sha256,omitempty"`
	GateSHA256        string     `json:"gate_sha256,omitempty"`
}

// ToolInput names one protected rule file or snapshot tree. Its bytes are
// bound during manifest loading; it never supplies an executable command.
type ToolInput struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ToolLocations identifies source and private audit output paths that tools
// and the manifest must not occupy.
type ToolLocations struct {
	Source string
	Output string
}

// VerifiedToolManifest contains the operator-declared metadata plus binary
// SHA-256 values verified from protected external files. It is not a tool
// probe, publisher-provenance attestation, or host-isolation guarantee.
type VerifiedToolManifest struct {
	ManifestSHA256 string           `json:"manifest_sha256"`
	Tools          []VerifiedTool   `json:"tools"`
	Helpers        []VerifiedHelper `json:"helpers,omitempty"`
}

// VerifiedTool records a verified executable identity and unverified declared
// version-output/rules metadata for later execution-layer qualification.
type VerifiedTool struct {
	ID                          string     `json:"id"`
	Path                        string     `json:"path"`
	SHA256                      string     `json:"sha256"`
	DeclaredVersion             string     `json:"declared_version"`
	DeclaredVersionOutputSHA256 string     `json:"declared_version_output_sha256"`
	DeclaredRules               *ToolRules `json:"declared_rules,omitempty"`
}

// ParseToolManifest strictly decodes and validates the v1 declaration without
// opening paths or running tools. It accepts portable POSIX or drive-absolute
// path syntax; LoadToolManifest subsequently requires a local host path.
func ParseToolManifest(data []byte) (ToolManifest, error) {
	if len(data) == 0 || len(data) > maxToolManifestBytes {
		return ToolManifest{}, errors.New("invalid tool manifest")
	}
	var manifest ToolManifest
	var presence toolManifestPresence
	if jsondocument.Decode(data, &manifest) != nil || json.Unmarshal(data, &presence) != nil || !validManifest(manifest, presence) {
		return ToolManifest{}, errors.New("invalid tool manifest")
	}
	return manifest, nil
}

// LoadToolManifest reads a protected manifest and binds every fixed tool ID to
// an external, protected executable whose bytes match the declared SHA-256.
func LoadToolManifest(path string, locations ToolLocations) (VerifiedToolManifest, error) {
	manifestPath, data, err := readToolManifest(path)
	if err != nil {
		return VerifiedToolManifest{}, err
	}
	manifest, err := ParseToolManifest(data)
	if err != nil {
		return VerifiedToolManifest{}, err
	}
	if manifest.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		return VerifiedToolManifest{}, errors.New("tool manifest platform mismatch")
	}
	source, output, err := canonicalLocations(locations)
	if err != nil || containedBy(manifestPath, source) || containedBy(manifestPath, output) {
		return VerifiedToolManifest{}, errors.New("invalid tool manifest location")
	}
	tools, err := verifyTools(manifest.Tools, source, output)
	if err != nil {
		return VerifiedToolManifest{}, err
	}
	helpers, err := verifyToolHelpers(manifest.Helpers, tools, source, output)
	if err != nil {
		return VerifiedToolManifest{}, err
	}
	if err = bindToolInputs(manifest.Tools, source, output); err != nil {
		return VerifiedToolManifest{}, err
	}
	return VerifiedToolManifest{ManifestSHA256: digest(data), Tools: tools, Helpers: helpers}, nil
}

func validManifest(manifest ToolManifest, presence toolManifestPresence) bool {
	if !validManifestShape(manifest, presence) {
		return false
	}
	seen := make(map[string]bool, len(manifest.Tools))
	for index, tool := range manifest.Tools {
		if !slices.Contains(toolIDs, tool.ID) || seen[strings.ToLower(tool.ID)] || !validToolEntry(tool, presence.Tools[index]) {
			return false
		}
		seen[strings.ToLower(tool.ID)] = true
	}
	return len(seen) == len(manifest.Tools)
}

func validManifestShape(manifest ToolManifest, presence toolManifestPresence) bool {
	return manifest.Version == toolManifestVersion && manifest.Platform != "" &&
		len(manifest.Tools) > 0 && len(manifest.Tools) <= len(toolIDs) &&
		len(manifest.Tools) == len(presence.Tools) && validToolHelpers(manifest.Helpers, presence.Helpers)
}

func validToolEntry(tool ToolEntry, presence toolEntryPresence) bool {
	if !validPortableToolPath(tool.Path) || !validDigest(tool.SHA256) || !validDigest(tool.VersionOutputSHA256) || !safeMetadata(tool.Version) {
		return false
	}
	if !requiresRules(tool.ID) {
		return tool.Rules == nil && len(presence.Rules) == 0
	}
	return tool.Rules != nil && validRules(tool, presence.Rules)
}

func requiresRules(id string) bool {
	return id == "staticcheck" || id == "gosec" || id == "govulncheck" || id == "gitleaks" || id == "actionlint" || id == "sonar-scanner"
}

func validRules(tool ToolEntry, raw json.RawMessage) bool {
	rules := *tool.Rules
	if !validDigest(rules.RulesetSHA256) || rules.Kind != ruleKind(tool.ID) {
		return false
	}
	switch rules.Kind {
	case "embedded":
		return validEmbeddedRules(tool, rules, raw)
	case "file", "vulndb_snapshot":
		return validInputRules(rules, raw)
	default:
		return validSonarRules(rules, raw)
	}
}

func validEmbeddedRules(tool ToolEntry, rules ToolRules, raw json.RawMessage) bool {
	return rules.Input == nil && rules.RulesetSHA256 == tool.SHA256 && !hasRulesInput(raw) && !hasSonarRuleMetadata(raw)
}

func validInputRules(rules ToolRules, raw json.RawMessage) bool {
	return hasRulesInput(raw) && validToolInput(rules.Input, rules.RulesetSHA256) && !hasSonarRuleMetadata(raw)
}

func validSonarRules(rules ToolRules, raw json.RawMessage) bool {
	return rules.Input == nil && !hasRulesInput(raw) && safeMetadata(rules.ServerVersion) && validDigest(rules.ActiveRulesSHA256) && validDigest(rules.ProfileSHA256) && validDigest(rules.GateSHA256)
}

func ruleKind(id string) string {
	switch id {
	case "staticcheck", "gosec", "actionlint":
		return "embedded"
	case "gitleaks":
		return "file"
	case "govulncheck":
		return "vulndb_snapshot"
	default:
		return "sonar_remote"
	}
}

func validToolInput(input *ToolInput, digest string) bool {
	return input != nil && validPortableToolPath(input.Path) && validDigest(input.SHA256) && input.SHA256 == digest
}

func hasSonarRuleMetadata(raw json.RawMessage) bool {
	return hasRawRulesField(raw, "server_version") || hasRawRulesField(raw, "active_rules_sha256") || hasRawRulesField(raw, "profile_sha256") || hasRawRulesField(raw, "gate_sha256")
}

func hasRulesInput(raw json.RawMessage) bool {
	return hasRawRulesField(raw, "input")
}

func hasRawRulesField(raw json.RawMessage, key string) bool {
	var presence map[string]json.RawMessage
	if json.Unmarshal(raw, &presence) != nil {
		return true
	}
	_, found := presence[key]
	return found
}

func safeMetadata(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) == "" {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return false
		}
	}
	return true
}

func validPortableToolPath(path string) bool {
	if len(path) == 0 || len(path) > maxToolPathBytes || hasControlCharacter(path) {
		return false
	}
	if strings.HasPrefix(path, "/") {
		return !strings.HasPrefix(path, "//")
	}
	return len(path) > 3 && isDriveLetter(path[0]) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
}

func isDriveLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func hasControlCharacter(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func readToolManifest(path string) (string, []byte, error) {
	if !localAbsoluteToolPath(path) {
		return "", nil, errors.New("invalid tool manifest location")
	}
	file, err := platform.OpenProtected(path, true)
	if err != nil {
		return "", nil, errors.New("invalid tool manifest protection")
	}
	defer file.Close()
	canonical, err := canonicalRegularPath(path)
	if err != nil {
		return "", nil, errors.New("invalid tool manifest location")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxToolManifestBytes+1))
	if err != nil || len(data) > maxToolManifestBytes {
		return "", nil, errors.New("invalid tool manifest")
	}
	return canonical, data, nil
}

func canonicalLocations(locations ToolLocations) (string, string, error) {
	source, err := canonicalDirectory(locations.Source)
	if err != nil {
		return "", "", err
	}
	output, err := canonicalDirectory(locations.Output)
	if err != nil {
		return "", "", err
	}
	return source, output, nil
}

func canonicalDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("relative location")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("invalid location")
	}
	return filepath.EvalSymlinks(path)
}

func canonicalRegularPath(path string) (string, error) {
	if !localAbsoluteToolPath(path) {
		return "", errors.New("relative path")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("invalid path")
	}
	return filepath.EvalSymlinks(path)
}

func localAbsoluteToolPath(path string) bool {
	if !validPortableToolPath(path) {
		return false
	}
	if runtime.GOOS == "windows" {
		return len(path) > 3 && isDriveLetter(path[0]) && path[1] == ':' && (path[2] == '\\' || path[2] == '/')
	}
	return filepath.IsAbs(path) && !strings.HasPrefix(path, "//")
}

func verifyTools(entries []ToolEntry, source, output string) ([]VerifiedTool, error) {
	seen := map[string]bool{}
	tools := make([]VerifiedTool, 0, len(entries))
	for _, entry := range entries {
		tool, err := verifyTool(entry, source, output, seen)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func verifyTool(entry ToolEntry, source, output string, seen map[string]bool) (VerifiedTool, error) {
	path, actual, err := verifyExecutableIdentity(entry.Path, entry.SHA256, source, output, seen)
	if err != nil {
		return VerifiedTool{}, err
	}
	return VerifiedTool{ID: entry.ID, Path: path, SHA256: actual, DeclaredVersion: entry.Version, DeclaredVersionOutputSHA256: entry.VersionOutputSHA256, DeclaredRules: entry.Rules}, nil
}

func verifyExecutableIdentity(declaredPath, expected, source, output string, seen map[string]bool) (string, string, error) {
	if !localAbsoluteToolPath(declaredPath) {
		return "", "", errors.New("invalid tool path")
	}
	if err := platform.ValidateFileMetadata(declaredPath, false); err != nil {
		return "", "", errors.New("invalid tool protection")
	}
	path, err := canonicalRegularPath(declaredPath)
	if err != nil || containedBy(path, source) || containedBy(path, output) || seen[strings.ToLower(path)] {
		return "", "", errors.New("invalid tool path")
	}
	if err = platform.ValidateExecutable(path); err != nil {
		return "", "", errors.New("invalid tool protection")
	}
	actual, err := executableSHA256(path)
	if err != nil || actual != expected {
		return "", "", errors.New("tool digest mismatch")
	}
	seen[strings.ToLower(path)] = true
	return path, actual, nil
}

func executableSHA256(path string) (string, error) {
	file, opened, err := openToolFile(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash, read, err := boundedToolHash(file)
	if err != nil {
		return "", err
	}
	if err = unchangedToolFile(path, opened, file, read); err != nil {
		return "", err
	}
	return hash, nil
}

func openToolFile(path string) (*os.File, os.FileInfo, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	file, err := root.Open(filepath.Base(path))
	if err != nil {
		return nil, nil, err
	}
	opened, err := file.Stat()
	named, nameErr := os.Lstat(path)
	if err != nil || nameErr != nil || !opened.Mode().IsRegular() || !os.SameFile(opened, named) || opened.Size() > maxToolBinaryBytes {
		_ = file.Close()
		return nil, nil, errors.New("tool changed")
	}
	return file, opened, nil
}

func boundedToolHash(file *os.File) (string, int64, error) {
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(file, maxToolBinaryBytes+1))
	if err != nil {
		return "", 0, err
	}
	if read > maxToolBinaryBytes {
		return "", 0, errors.New("tool too large")
	}
	return hex.EncodeToString(hash.Sum(nil)), read, nil
}

func unchangedToolFile(path string, opened os.FileInfo, file *os.File, read int64) error {
	after, statErr := file.Stat()
	named, nameErr := os.Lstat(path)
	if statErr != nil || nameErr != nil || read != opened.Size() || !os.SameFile(opened, after) || !os.SameFile(opened, named) || after.Size() != opened.Size() || after.Mode() != opened.Mode() {
		return errors.New("tool changed")
	}
	return nil
}

func containedBy(path, directory string) bool {
	relative, err := filepath.Rel(directory, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
