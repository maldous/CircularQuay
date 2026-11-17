package cq

import (
	"runtime/debug"
	"strings"
	"testing"
)

// These serial tests replace the package-local metadata callback only. No
// embedded build information, environment, filesystem or runner is consulted.
func TestBuildIdentityContractUnavailableMetadataDoesNotRetainPriorIdentity(t *testing.T) {
	valid := &debug.BuildInfo{
		Main:     debug.Module{Path: circularQuayModule, Version: "v1.4.2"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0"}},
	}
	for _, tc := range []struct {
		name string
		info *debug.BuildInfo
		ok   bool
	}{
		{"absent", nil, false},
		{"nil-despite-success", nil, true},
		{"unavailable-despite-info", valid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertBuildIdentityUnavailableMetadataCase(t, valid, tc.info, tc.ok)
		})
	}
}

func assertBuildIdentityUnavailableMetadataCase(t *testing.T, valid, info *debug.BuildInfo, ok bool) {
	t.Helper()
	calls := 0
	withBuildIdentityReader(t, func() (*debug.BuildInfo, bool) {
		calls++
		if calls == 1 {
			return valid, true
		}
		return info, ok
	})
	if got := sdkBuildIdentity(); got != (buildIdentity{version: "1.4.2", commit: "abcdef0"}) {
		t.Fatal("fabricated valid metadata did not establish its routing control")
	}
	if got := sdkBuildIdentity(); got != (buildIdentity{version: developmentVersion}) {
		t.Fatal("unavailable metadata retained an earlier version or revision")
	}
	if calls != 2 {
		t.Fatal("build identity cached metadata or read it more than once per selection")
	}
}

func TestBuildIdentityContractMainModuleHasItsOwnTaggedVersion(t *testing.T) {
	for _, version := range []string{"v1.4.2", "v1.4.2-rc.3+build.7"} {
		t.Run(version, func(t *testing.T) {
			info := &debug.BuildInfo{
				Main: debug.Module{Path: circularQuayModule, Version: version},
				Deps: []*debug.Module{{Path: "example.invalid/unrelated-module", Version: "v1.99.99"}},
			}
			withBuildIdentityReader(t, func() (*debug.BuildInfo, bool) { return info, true })
			want := buildIdentity{version: strings.TrimPrefix(version, "v")}
			if got := sdkBuildIdentity(); got != want {
				t.Fatal("the SDK's tagged main module lost its own version-routing identity")
			}
		})
	}
}

func TestBuildIdentityContractSparseDependenciesStillSelectSDK(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "example.invalid/host", Version: "v1.99.99"},
		Deps: []*debug.Module{
			nil,
			{Path: "example.invalid/unrelated-module", Version: "v1.88.88"},
			{Path: circularQuayModule, Version: "v1.4.2"},
		},
	}
	withBuildIdentityReader(t, func() (*debug.BuildInfo, bool) { return info, true })
	if got := sdkBuildIdentity(); got != (buildIdentity{version: "1.4.2"}) {
		t.Fatal("nil or unrelated dependency displaced the explicitly identified SDK module")
	}
}

func TestBuildIdentityContractRevisionRequiresExactKeyAndBoundedHex(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  string
		text string
		want string
	}{
		{"minimum", "vcs.revision", "aBcD012", "aBcD012"},
		{"maximum", "vcs.revision", strings.Repeat("Ab01", 16), strings.Repeat("Ab01", 16)},
		{"empty", "vcs.revision", "", ""},
		{"short", "vcs.revision", "abcdef", ""},
		{"long", "vcs.revision", strings.Repeat("a", 65), ""},
		{"nonhex", "vcs.revision", "abcdefg", ""},
		{"leading-space", "vcs.revision", " abcdef0", ""},
		{"trailing-newline", "vcs.revision", "abcdef0\n", ""},
		{"case-changed-key", "VCS.REVISION", "abcdef0", ""},
		{"other-revision-key", "dependency.revision", "abcdef0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := &debug.BuildInfo{
				Main: debug.Module{Path: circularQuayModule, Version: "v1.4.2"},
				Settings: []debug.BuildSetting{
					{Key: "vcs", Value: "git"},
					{Key: "vcs.modified", Value: "true"},
					{Key: tc.key, Value: tc.text},
				},
			}
			withBuildIdentityReader(t, func() (*debug.BuildInfo, bool) { return info, true })
			if got := sdkBuildIdentity(); got != (buildIdentity{version: "1.4.2", commit: tc.want}) {
				t.Fatal("revision metadata bypassed its exact key or bounded hexadecimal contract")
			}
		})
	}
}

func withBuildIdentityReader(t *testing.T, reader func() (*debug.BuildInfo, bool)) {
	t.Helper()
	previous := readBuildInfo
	t.Cleanup(func() { readBuildInfo = previous })
	readBuildInfo = reader
}
