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
	if len(g.Values) != len(w.Bindings) || len(g.Receipts) != len(w.Bindings) || g.RetrievedAt.IsZero() || g.RetrievedAt.After(now) {
		return fault.New(fault.Trust, "snapshot generation")
	}
	for i, b := range w.Bindings {
		v, ok := g.Values[b.Name]
		if !ok || !b.AllowOffline || v.Leased || v.Renewable {
			return fault.New(fault.Denied, "snapshot scope")
		}
		receipt := g.Receipts[i]
		if receipt.Source != b.Source || receipt.Resource != b.Resource || receipt.Version != v.Version {
			return fault.New(fault.Trust, "snapshot provenance")
		}
		if err := validateValue(v, now, false); err != nil {
			return err
		}
	}
	return nil
}
