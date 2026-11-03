package resolve

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func checkCancellation(ctx context.Context) error {
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "resolution canceled")
	}
	return nil
}

// Provider revisions enter metadata receipts. Bound them independently of secret
// values and reject control characters before persisting or emitting diagnostics.
func validVersion(version string) bool {
	if len(version) > 1024 || !utf8.ValidString(version) {
		return false
	}
	return !strings.ContainsFunc(version, unicode.IsControl)
}

func validateGeneration(w model.Workload, g model.Generation, now time.Time) error {
	if !completeGeneration(w, g, now) {
		return fault.New(fault.Trust, "snapshot generation")
	}
	for i, b := range w.Bindings {
		if err := validateSnapshotBinding(b, g.Values, g.Receipts[i], now); err != nil {
			return err
		}
	}
	return nil
}

func completeGeneration(workload model.Workload, generation model.Generation, now time.Time) bool {
	return len(generation.Values) == len(workload.Bindings) && len(generation.Receipts) == len(workload.Bindings) && !generation.RetrievedAt.IsZero() && !generation.RetrievedAt.After(now)
}

func validateSnapshotBinding(binding model.Binding, values map[string]model.Value, receipt model.Receipt, now time.Time) error {
	value, found := values[binding.Name]
	if !found || !binding.AllowOffline || value.Leased || value.Renewable {
		return fault.New(fault.Denied, "snapshot scope")
	}
	if receipt.Source != binding.Source || receipt.Resource != binding.Resource || receipt.Version != value.Version {
		return fault.New(fault.Trust, "snapshot provenance")
	}
	return validateValue(value, now, false)
}
