package policy

import (
	"path/filepath"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

const ageRecipientLength = 62

const ageRecipientData = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

// validateSnapshotDeclaration checks policy shape only. File existence and age
// recipient checksum verification belong to the authenticated SOPS operation,
// not policy loading.
func validateSnapshotDeclaration(workload model.Workload) error {
	snapshot := workload.Snapshot
	if snapshotEmpty(snapshot) {
		return nil
	}
	if snapshot.Path == "" {
		return invalid("snapshot_orphan_option")
	}
	if !snapshotPathsValid(snapshot) || !snapshotRecipientsValid(snapshot.Recipients) {
		return invalid("snapshot_declaration")
	}
	if workload.Mode == "online" || workload.AuditRequired || !allBindingsAllowOffline(workload.Bindings) {
		return invalid("snapshot_eligibility")
	}
	return nil
}

func snapshotEmpty(snapshot model.Snapshot) bool {
	return snapshot.Path == "" && len(snapshot.Recipients) == 0 && snapshot.IdentityFile == "" && snapshot.SOPS == ""
}

func snapshotPathsValid(snapshot model.Snapshot) bool {
	return filepath.IsAbs(snapshot.Path) && filepath.IsAbs(snapshot.IdentityFile) && filepath.IsAbs(snapshot.SOPS) &&
		safeText(snapshot.Path) && safeText(snapshot.IdentityFile) && safeText(snapshot.SOPS)
}

func snapshotRecipientsValid(recipients []string) bool {
	if len(recipients) == 0 || uniqueExact(recipients) != nil {
		return false
	}
	for _, recipient := range recipients {
		if !validAgeRecipient(recipient) {
			return false
		}
	}
	return true
}

func validAgeRecipient(recipient string) bool {
	if len(recipient) != ageRecipientLength || !strings.HasPrefix(recipient, "age1") {
		return false
	}
	for _, character := range recipient[4:] {
		if !strings.ContainsRune(ageRecipientData, character) {
			return false
		}
	}
	return true
}

func allBindingsAllowOffline(bindings []model.Binding) bool {
	for _, binding := range bindings {
		if !binding.AllowOffline {
			return false
		}
	}
	return true
}
