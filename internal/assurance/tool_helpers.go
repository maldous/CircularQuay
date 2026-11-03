package assurance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/platform"
)

const sshKeygenHelperID = "ssh-keygen"

// ToolHelper binds a supported native dependency to protected executable bytes.
// It does not claim a version probe, execution, or publisher authentication.
// Only ssh-keygen is supported; arbitrary helper commands are not accepted.
type ToolHelper struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// VerifiedHelper identifies native helper bytes verified during manifest loading.
// It is separate from a probed analyzer and cannot be selected by commandFor.
type VerifiedHelper struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func validToolHelpers(helpers []ToolHelper, raw json.RawMessage) bool {
	if len(raw) == 0 {
		return len(helpers) == 0
	}
	if len(helpers) != 1 {
		return false
	}
	helper := helpers[0]
	return helper.ID == sshKeygenHelperID && validPortableToolPath(helper.Path) && validDigest(helper.SHA256)
}

func verifyToolHelpers(entries []ToolHelper, tools []VerifiedTool, source, output string) ([]VerifiedHelper, error) {
	seen := make(map[string]bool, len(tools)+len(entries))
	for _, tool := range tools {
		seen[strings.ToLower(tool.Path)] = true
	}
	helpers := make([]VerifiedHelper, 0, len(entries))
	for _, entry := range entries {
		if filepath.Base(entry.Path) != canonicalToolName(entry.ID) {
			return nil, errors.New("invalid helper filename")
		}
		path, actual, err := verifyExecutableIdentity(entry.Path, entry.SHA256, source, output, seen)
		if err != nil {
			return nil, err
		}
		helpers = append(helpers, VerifiedHelper{ID: entry.ID, Path: path, SHA256: actual})
	}
	return helpers, nil
}

func (helper VerifiedHelper) binary() helperBinary {
	return helperBinary{id: helper.ID, path: helper.Path, sha256: helper.SHA256}
}

func (helper VerifiedHelper) matches() bool {
	if platform.ValidateFileMetadata(helper.Path, false) != nil || platform.ValidateExecutable(helper.Path) != nil {
		return false
	}
	actual, err := executableSHA256(helper.Path)
	return err == nil && actual == helper.SHA256
}

func (tools *qualifiedTools) copyNativeHelpers(directory string) error {
	for _, helper := range tools.manifest.Helpers {
		target := filepath.Join(directory, canonicalToolName(helper.ID))
		if !helper.matches() || copyHelperBinary(helper.binary(), target) != nil {
			return os.ErrPermission
		}
		tools.helperCopies[helper.ID] = target
	}
	if !tools.helpersMatch() {
		return os.ErrPermission
	}
	return nil
}

// helpersMatch fences both the declared dependency and the private PATH copy.
// These repeated observations detect drift, not atomic use or a hostile host.
func (tools *qualifiedTools) helpersMatch() bool {
	for _, helper := range tools.manifest.Helpers {
		target, prepared := tools.helperCopies[helper.ID]
		if !prepared || !helper.matches() || !existingHelperBinaryMatches(helper.binary(), target) {
			return false
		}
	}
	return true
}

func (tools *qualifiedTools) helperIdentities() []HelperIdentity {
	var identities []HelperIdentity
	for _, helper := range tools.manifest.Helpers {
		_, prepared := tools.helperCopies[helper.ID]
		identities = append(identities, HelperIdentity{VerifiedHelper: helper, Prepared: prepared, IntegrityCurrent: prepared && helper.matches() && existingHelperBinaryMatches(helper.binary(), tools.helperCopies[helper.ID])})
	}
	return identities
}
