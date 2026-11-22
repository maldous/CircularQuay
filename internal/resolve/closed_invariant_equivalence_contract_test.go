package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestClosedInvariantResolveFingerprintKeepsExactClosedProjection(t *testing.T) {
	workload := model.Workload{Mode: "offline", Bindings: []model.Binding{{Name: "TOKEN", Source: "data", AllowOffline: true}}, Environment: map[string]string{"SYNTHETIC": "λ\x00value"}}
	policy := model.Policy{Version: 1, Workloads: map[string]model.Workload{"worker": workload}, Sources: map[string]model.Source{"data": {Type: "json", Path: "owned-values"}}}
	engine := &Engine{Policy: policy}
	encoded, err := json.Marshal(struct {
		Version  int
		Workload model.Workload
		Sources  map[string]model.Source
	}{policy.Version, workload, policy.Sources})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	if engine.fingerprint("worker") != hex.EncodeToString(digest[:]) {
		t.Fatal("snapshot fingerprint changed exact JSON projection")
	}
	workload.Mode = "hybrid"
	policy.Workloads["worker"] = workload
	if engine.fingerprint("worker") == hex.EncodeToString(digest[:]) {
		t.Fatal("fingerprint failed to bind changed workload mode")
	}
}

func TestClosedInvariantResolveSnapshotKeepsRealTimeEncodingFailure(t *testing.T) {
	engine := &Engine{Policy: model.Policy{Workloads: map[string]model.Workload{"worker": {}}}}
	// Generation is a caller-owned typed API input, not policy fingerprint
	// data. Its time.Time marshaler can genuinely refuse an out-of-range year.
	generation := model.Generation{Workload: "worker", RetrievedAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}
	got, err := engine.encodeSnapshot(generation)
	if got != "" || !fault.Is(err, fault.Apply) || err.Error() != "activation_failed: snapshot encode" {
		t.Fatal("closed fingerprint simplification removed a real generation encoding refusal")
	}
}
