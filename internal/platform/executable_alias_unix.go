//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// Validate each spelling used during executable lookup, including intermediate
// symlink targets. A protected final target alone does not protect a mutable
// alias. Keeping the original path also preserves native argv and script $0.
func trustedExecutableAliases(path string) bool {
	root := string(filepath.Separator)
	if !trustedAncestors(root) {
		return false
	}
	walk := executableAliasWalk{current: root, parts: strings.Split(path, root)}
	for len(walk.parts) != 0 {
		if !walk.advance() {
			return false
		}
	}
	return true
}

type executableAliasWalk struct {
	current string
	parts   []string
	links   int
}

func (walk *executableAliasWalk) advance() bool {
	part := walk.parts[0]
	walk.parts = walk.parts[1:]
	switch part {
	case "", ".":
		return true
	case "..":
		walk.current = filepath.Dir(walk.current)
		return true
	}
	next := filepath.Join(walk.current, part)
	info, err := os.Lstat(next)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return walk.follow(next, info)
	}
	if !trustedEntry(info, false) {
		return false
	}
	walk.current = next
	return true
}

func (walk *executableAliasWalk) follow(path string, info os.FileInfo) bool {
	if !trustedOwner(info) || walk.links == 40 {
		return false
	}
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	if filepath.IsAbs(target) {
		walk.current = string(filepath.Separator)
	}
	walk.parts = append(strings.Split(target, string(filepath.Separator)), walk.parts...)
	walk.links++
	return true
}
