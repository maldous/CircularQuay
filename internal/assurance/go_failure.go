package assurance

import (
	"bufio"
	"bytes"
	"context"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxGoFailurePackages  = 16
	maxGoFailureLocations = 32
	maxGoFailureLine      = 1_000_000
)

var (
	goFailurePackageLine  = regexp.MustCompile(`^(?:#\s+|FAIL\s+)(github\.com/maldous/CircularQuay(?:/[A-Za-z0-9._/-]+)?)(?:\s+(?:\[build failed\]|[0-9]+(?:\.[0-9]+)?s))?$`)
	goFailureLocationLine = regexp.MustCompile(`^([A-Za-z0-9._/-]+\.go):([1-9][0-9]{0,6})(?::[1-9][0-9]{0,4})?:`)
)

type goFailureCapture struct {
	packages  []string
	locations []GoFailureLocation
	unknown   bool
	overflow  bool
}

// goSourceCheckSpec binds a Go command to the manifest-derived source scope
// whose failures may be summarized in the version-one report.
type goSourceCheckSpec struct {
	name        string
	binary      string
	arguments   []string
	directory   string
	environment []string
	scope       goSourceScope
	source      *SourceIdentity
}

// executeGoSourceCheck retains only source-bound failure metadata for the two
// Go commands whose differing audit environment needs attribution.
func executeGoSourceCheck(ctx context.Context, command commandFunc, specification goSourceCheckSpec) Check {
	start := time.Now()
	result := command(ctx, specification.binary, specification.arguments, specification.directory, specification.environment)
	check := classifyCommand(specification.name, true, result, time.Since(start))
	if check.Status == "failed" {
		check.Failure = summarizeGoFailure(result.goFailure, specification.scope, specification.source)
	}
	return check
}

func parseGoFailureCapture(arguments []string, output []byte, overflow bool) goFailureCapture {
	if !goFailureCommand(arguments) {
		return goFailureCapture{}
	}
	capture := goFailureCapture{overflow: overflow}
	if capture.overflow {
		return capture
	}
	return scanGoFailureOutput(capture, output)
}

func scanGoFailureOutput(capture goFailureCapture, output []byte) goFailureCapture {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Buffer(make([]byte, 256), 64<<10)
	for scanner.Scan() {
		capture = parseGoFailureLine(capture, scanner.Text())
		if capture.overflow {
			return capture
		}
	}
	if scanner.Err() != nil {
		capture.unknown = true
	}
	return capture
}

func goFailureCommand(arguments []string) bool {
	return len(arguments) > 0 && (arguments[0] == "test" || arguments[0] == "vet")
}

// parseGoFailureLine gives package records precedence over locations and stops
// retaining candidates as soon as either metadata cap is exceeded.
func parseGoFailureLine(capture goFailureCapture, line string) goFailureCapture {
	if match := goFailurePackageLine.FindStringSubmatch(line); match != nil {
		return appendGoFailurePackage(capture, match[1])
	}
	location, matched, valid := goFailureLocation(line)
	if !matched {
		return capture
	}
	if !valid {
		capture.unknown = true
		return capture
	}
	return appendGoFailureLocation(capture, location)
}

func appendGoFailurePackage(capture goFailureCapture, candidate string) goFailureCapture {
	if len(capture.packages) >= maxGoFailurePackages {
		capture.overflow = true
		return capture
	}
	capture.packages = append(capture.packages, candidate)
	return capture
}

func goFailureLocation(line string) (GoFailureLocation, bool, bool) {
	match := goFailureLocationLine.FindStringSubmatch(line)
	if match == nil {
		return GoFailureLocation{}, false, false
	}
	lineNumber, err := strconv.Atoi(match[2])
	if err != nil || lineNumber > maxGoFailureLine {
		return GoFailureLocation{}, true, false
	}
	return GoFailureLocation{Path: match[1], Line: lineNumber}, true, true
}

func appendGoFailureLocation(capture goFailureCapture, candidate GoFailureLocation) goFailureCapture {
	if len(capture.locations) >= maxGoFailureLocations {
		capture.overflow = true
		return capture
	}
	capture.locations = append(capture.locations, candidate)
	return capture
}

func summarizeGoFailure(capture goFailureCapture, scope goSourceScope, source *SourceIdentity) *GoFailureMetadata {
	if capture.overflow {
		return &GoFailureMetadata{State: "overflow"}
	}
	packages, packagesOK := boundFailurePackages(capture.packages, scope)
	locations, locationsOK := boundFailureLocations(capture.locations, source)
	if len(packages) > maxGoFailurePackages || len(locations) > maxGoFailureLocations {
		return &GoFailureMetadata{State: "overflow"}
	}
	if capture.unknown || !packagesOK || !locationsOK {
		return &GoFailureMetadata{State: "unknown"}
	}
	if len(packages) == 0 && len(locations) == 0 {
		return &GoFailureMetadata{State: "unknown"}
	}
	return &GoFailureMetadata{State: "observed", Packages: packages, Locations: locations}
}

func boundFailurePackages(candidates []string, scope goSourceScope) ([]string, bool) {
	allowed := make(map[string]bool, len(scope.packages))
	for _, value := range scope.packages {
		if value == "." {
			allowed[strings.TrimSuffix(modulePrefix, "/")] = true
			continue
		}
		allowed[modulePrefix+strings.TrimPrefix(value, "./")] = true
	}
	return uniqueBoundedStrings(candidates, allowed)
}

func boundFailureLocations(candidates []GoFailureLocation, source *SourceIdentity) ([]GoFailureLocation, bool) {
	allowed, valid := allowedGoFailurePaths(source)
	if !valid {
		return nil, false
	}
	return boundAllowedFailureLocations(candidates, allowed)
}

func allowedGoFailurePaths(source *SourceIdentity) (map[string]bool, bool) {
	if source == nil {
		return nil, false
	}
	allowed := make(map[string]bool, len(source.Files))
	for _, file := range source.Files {
		if strings.HasSuffix(file.Path, ".go") {
			allowed[file.Path] = true
		}
	}
	return allowed, true
}

func boundAllowedFailureLocations(candidates []GoFailureLocation, allowed map[string]bool) ([]GoFailureLocation, bool) {
	locations := make([]GoFailureLocation, 0, len(candidates))
	seen := map[GoFailureLocation]bool{}
	for _, candidate := range candidates {
		if !allowedFailureLocation(candidate, allowed) {
			return nil, false
		}
		locations = appendUniqueFailureLocation(locations, seen, candidate)
	}
	sort.Slice(locations, func(left, right int) bool {
		if locations[left].Path == locations[right].Path {
			return locations[left].Line < locations[right].Line
		}
		return locations[left].Path < locations[right].Path
	})
	return locations, true
}

func allowedFailureLocation(candidate GoFailureLocation, allowed map[string]bool) bool {
	if candidate.Line < 1 || candidate.Line > maxGoFailureLine {
		return false
	}
	return validFailurePath(candidate.Path) && allowed[candidate.Path]
}

func appendUniqueFailureLocation(locations []GoFailureLocation, seen map[GoFailureLocation]bool, candidate GoFailureLocation) []GoFailureLocation {
	if seen[candidate] {
		return locations
	}
	seen[candidate] = true
	return append(locations, candidate)
}

func uniqueBoundedStrings(candidates []string, allowed map[string]bool) ([]string, bool) {
	values := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if !allowed[candidate] {
			return nil, false
		}
		if !seen[candidate] {
			seen[candidate] = true
			values = append(values, candidate)
		}
	}
	sort.Strings(values)
	return values, true
}

func validFailurePath(value string) bool {
	return !strings.ContainsAny(value, "\\\x00") && !strings.HasPrefix(value, "/") && path.Clean(value) == value && !strings.HasPrefix(value, "../")
}
