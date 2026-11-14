package cq

import (
	"context"
	"strings"
	"unicode/utf8"

	"golang.org/x/mod/semver"

	"github.com/maldous/CircularQuay/internal/jsondocument"
)

// RuntimeVersion is the bounded public metadata reported by the linked
// CQ core. It contains no policy, provider, or credential information.
type RuntimeVersion struct {
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Go       string `json:"go"`
	Maturity string `json:"maturity"`
}

// Version reads and validates the linked CQ core's version metadata.
// It performs no subprocess start, provider access, policy load, prompt, or
// workload startup.
func (c *LocalClient) Version(ctx context.Context) (RuntimeVersion, error) {
	result, err := c.jsonCommand(ctx, "version", nil, "version")
	if err != nil {
		return RuntimeVersion{}, err
	}
	var wire runtimeVersionWire
	if jsondocument.Decode(result.JSON, &wire) != nil || wire.Version == nil || wire.Commit == nil || wire.Go == nil || wire.Maturity == nil {
		return RuntimeVersion{}, &Error{Category: CategoryProtocol}
	}
	version := RuntimeVersion{Version: *wire.Version, Commit: *wire.Commit, Go: *wire.Go, Maturity: *wire.Maturity}
	major, valid := runtimeVersionMajor(version.Version)
	if !valid {
		return RuntimeVersion{}, &Error{Category: CategoryProtocol}
	}
	if major != "1" {
		return RuntimeVersion{}, &Error{Category: CategoryUnsupported}
	}
	return version, nil
}

type runtimeVersionWire struct {
	Version  *string `json:"version"`
	Commit   *string `json:"commit"`
	Go       *string `json:"go"`
	Maturity *string `json:"maturity"`
}

// AssertRuntimeCompatibility verifies that the linked core reports compatible
// v1 metadata. It has the same no-subprocess and no-provider-access boundary as
// Version.
func (c *LocalClient) AssertRuntimeCompatibility(ctx context.Context) (RuntimeVersion, error) {
	return c.Version(ctx)
}

func runtimeVersionMajor(value string) (string, bool) {
	if len(value) > maxTextLength || !utf8.ValidString(value) {
		return "", false
	}
	canonical := value
	if !strings.HasPrefix(canonical, "v") {
		canonical = "v" + canonical
	}
	if !semver.IsValid(canonical) || !completeRuntimeSemVer(canonical) {
		return "", false
	}
	return strings.TrimPrefix(semver.Major(canonical), "v"), true
}

func completeRuntimeSemVer(value string) bool {
	withoutBuild := strings.TrimSuffix(value, semver.Build(value))
	return semver.Canonical(value) == withoutBuild
}
