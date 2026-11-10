// Package assurance runs reproducible source checks without exposing subprocess output.
package assurance

import (
	"bufio"
	"errors"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const modulePrefix = "github.com/maldous/CircularQuay/"

var coverageLine = regexp.MustCompile(`^(.+):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)$`)

type coverageBlock struct {
	statements int64
	covered    bool
}

// Coverage is statement/block coverage, not branch coverage or proof of correctness.
type Coverage struct {
	Covered int64   `json:"covered_statements"`
	Total   int64   `json:"total_statements"`
	Percent float64 `json:"statement_percent"`
	Files   int     `json:"files"`
}

// ParseCoverage validates an untrusted Go coverprofile and summarizes owned code.
func ParseCoverage(r io.Reader) (Coverage, error) {
	scanner := bufio.NewScanner(io.LimitReader(r, 16<<20))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	if !scanner.Scan() || !validCoverageMode(scanner.Text()) {
		return Coverage{}, errors.New("invalid coverage header")
	}
	files := make(map[string]bool)
	blocks := make(map[string]coverageBlock)
	var result Coverage
	for scanner.Scan() {
		name, key, block, err := parseCoverageBlock(scanner.Text())
		if err != nil {
			return Coverage{}, err
		}
		if err = result.mergeBlock(blocks, key, block); err != nil {
			return Coverage{}, err
		}
		files[name] = true
	}
	if scanner.Err() != nil {
		return Coverage{}, errors.New("coverage read failed")
	}
	if result.Total == 0 {
		return Coverage{}, errors.New("coverage contains no executable statements")
	}
	result.Files = len(files)
	result.Percent = 100 * float64(result.Covered) / float64(result.Total)
	return result, nil
}

func parseCoverageBlock(line string) (name, key string, block coverageBlock, err error) {
	parts := coverageLine.FindStringSubmatch(line)
	if parts == nil {
		return "", "", block, errors.New("invalid coverage block")
	}
	name, err = ownedCoveragePath(parts[1])
	if err != nil {
		return "", "", block, err
	}
	if err = validateBlock(parts); err != nil {
		return "", "", block, err
	}
	statements, err := strconv.ParseInt(parts[6], 10, 64)
	if err != nil || statements > 10_000_000 {
		return "", "", block, errors.New("invalid statement count")
	}
	executions, err := strconv.ParseUint(parts[7], 10, 64)
	if err != nil {
		return "", "", block, errors.New("invalid execution count")
	}
	return name, strings.Join(parts[1:6], ":"), coverageBlock{statements: statements, covered: executions > 0}, nil
}

// Whole-module coverage repeats blocks across test binaries. Coordinates must
// agree on NumStmt. Union execution while counting each statement only once.
func (result *Coverage) mergeBlock(blocks map[string]coverageBlock, key string, block coverageBlock) error {
	if previous, exists := blocks[key]; exists {
		if previous.statements != block.statements {
			return errors.New("inconsistent coverage block")
		}
		if block.covered && !previous.covered {
			result.Covered += block.statements
			blocks[key] = block
		}
		return nil
	}
	blocks[key] = block
	result.Total += block.statements
	if result.Total > 100_000_000 {
		return errors.New("coverage size limit exceeded")
	}
	if block.covered {
		result.Covered += block.statements
	}
	return nil
}

func validCoverageMode(header string) bool {
	return header == "mode: set" || header == "mode: count" || header == "mode: atomic"
}

func ownedCoveragePath(raw string) (string, error) {
	if !strings.HasPrefix(raw, modulePrefix) {
		return "", errors.New("coverage contains an unowned source")
	}
	name := strings.TrimPrefix(raw, modulePrefix)
	if strings.ContainsAny(name, "\\\x00") || path.Clean(name) != name || strings.HasPrefix(name, "../") || path.IsAbs(name) || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		return "", errors.New("coverage contains an invalid path")
	}
	return name, nil
}

func validateBlock(parts []string) error {
	coordinates := make([]uint64, 4)
	for index := range coordinates {
		number, err := strconv.ParseUint(parts[index+2], 10, 32)
		if err != nil || number == 0 {
			return errors.New("invalid coverage coordinates")
		}
		coordinates[index] = number
	}
	if coordinates[0] > coordinates[2] || coordinates[0] == coordinates[2] && coordinates[1] > coordinates[3] {
		return errors.New("reversed coverage coordinates")
	}
	return nil
}
