package assurance

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// This independent contract limit must not grow with the parser implementation.
const coverageBoundaryByteLimit = 16 << 20

func TestCoverageBoundaryContractExactByteLimitPreservesDeduplicatedSummary(t *testing.T) {
	profile := coverageBoundaryProfile(t, coverageBoundaryByteLimit)
	result, err := ParseCoverage(strings.NewReader(profile))
	want := Coverage{Covered: 1, Total: 1, Percent: 100, Files: 1}
	if err != nil || result != want {
		t.Fatal("complete exact-limit profile was refused or its denominator changed")
	}
}

func TestCoverageBoundaryContractValidOneByteOverflowReturnsNoSummary(t *testing.T) {
	// All rows remain valid: the old truncation removed only the last newline.
	profile := coverageBoundaryProfile(t, coverageBoundaryByteLimit+1)
	result, err := ParseCoverage(strings.NewReader(profile))
	assertCoverageBoundaryRefusal(t, result, err)
	if err.Error() != "coverage profile size limit exceeded" {
		t.Fatal("valid overflow was not identified as an incomplete bounded profile")
	}
}

func TestCoverageBoundaryContractCoveredPrefixCannotHideUncoveredTail(t *testing.T) {
	prefix := coverageBoundaryProfile(t, coverageBoundaryByteLimit)
	tail := modulePrefix + "owned.go:2.1,2.2 1 0\n"
	tailReader := strings.NewReader(tail)
	result, err := ParseCoverage(io.MultiReader(strings.NewReader(prefix), tailReader))
	assertCoverageBoundaryRefusal(t, result, err)
	if len(tail)-tailReader.Len() > 1 {
		t.Fatal("overflow detection consumed more than its single sentinel byte")
	}
}

func TestCoverageBoundaryContractReaderFailureWithholdsParsedPrefix(t *testing.T) {
	profile := "mode: atomic\n" + modulePrefix + "owned.go:1.1,1.2 1 1\n"
	reader := io.MultiReader(strings.NewReader(profile), coverageBoundaryReadFailure{})
	result, err := ParseCoverage(reader)
	assertCoverageBoundaryRefusal(t, result, err)
	if err.Error() != "coverage read failed" {
		t.Fatal("reader failure changed its safe diagnostic or leaked upstream text")
	}
}

func TestCoverageBoundaryContractAggregateStatementLimitIsInclusiveAndAtomic(t *testing.T) {
	var profile strings.Builder
	profile.WriteString("mode: atomic\n")
	for line := 1; line <= 10; line++ {
		fmt.Fprintf(&profile, "%sowned.go:%d.1,%d.2 10000000 1\n", modulePrefix, line, line)
	}
	result, err := ParseCoverage(strings.NewReader(profile.String()))
	want := Coverage{Covered: 100_000_000, Total: 100_000_000, Percent: 100, Files: 1}
	if err != nil || result != want {
		t.Fatal("exact aggregate statement limit was refused or miscounted")
	}
	profile.WriteString(modulePrefix + "owned.go:11.1,11.2 1 0\n")
	result, err = ParseCoverage(strings.NewReader(profile.String()))
	assertCoverageBoundaryRefusal(t, result, err)
	if err.Error() != "coverage size limit exceeded" {
		t.Fatal("aggregate overflow changed its existing failure contract")
	}
}

// Equal-coordinate rows differ only in their positive execution-count width.
// q rows of length L, of which r have length L+1, fill the exact byte boundary
// while contributing only one covered statement to the union denominator.
func coverageBoundaryProfile(t *testing.T, size int) string {
	t.Helper()
	header := "mode: atomic\n"
	short := modulePrefix + "owned.go:1.1,1.2 1 1\n"
	long := modulePrefix + "owned.go:1.1,1.2 1 10\n"
	remaining := size - len(header)
	rows, extra := remaining/len(short), remaining%len(short)
	profile := header + strings.Repeat(short, rows-extra) + strings.Repeat(long, extra)
	if len(profile) != size {
		t.Fatal("synthetic coverage fixture did not reach its exact byte boundary")
	}
	return profile
}

func assertCoverageBoundaryRefusal(t *testing.T, result Coverage, err error) {
	t.Helper()
	if err == nil || result != (Coverage{}) {
		t.Fatal("incomplete coverage returned success or a partial/favorable summary")
	}
}

type coverageBoundaryReadFailure struct{}

func (coverageBoundaryReadFailure) Read([]byte) (int, error) {
	return 0, errors.New("SYNTHETIC_PRIVATE_COVERAGE_READER_FAILURE")
}
