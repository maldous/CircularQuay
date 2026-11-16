package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestStartupUpdateNoticeStateFailureDoesNotBlockPlan(t *testing.T) {
	app, legacy, original := advisoryLegacyStateFixture(t)
	config, requests := advisoryFeedFixture(t)
	if !shouldCheckUpdates("plan", config, true) {
		t.Fatal("interactive plan did not select the startup notice")
	}
	if err := app.checkUpdateNotice(context.Background(), config); err != nil {
		t.Fatalf("optional startup notice blocked plan: %v", err)
	}
	if app.runner.Err.(*bytes.Buffer).String() != "CircularQuay could not check for updates; continuing.\n" {
		t.Fatal("startup notice did not return the fixed safe diagnostic")
	}
	requireAdvisoryPlan(t, app)
	requireUnchangedAdvisoryState(t, legacy, original)
	if requests.Load() != 0 {
		t.Fatal("failed advisory state admission contacted the release feed")
	}
}

func TestStartupNoticeDoesNotRelaxExplicitUpdateOrWorkloadState(t *testing.T) {
	app, legacy, original := advisoryLegacyStateFixture(t)
	config, requests := advisoryFeedFixture(t)
	policy := model.Policy{Updates: config}
	if _, err := app.updateCheckCommand(context.Background(), nil, policy); !fault.Is(err, fault.Trust) {
		t.Fatalf("explicit update check lost the state trust denial: %v", err)
	}
	if _, err := app.engine(policy); !fault.Is(err, fault.Trust) {
		t.Fatalf("workload engine lost the state trust denial: %v", err)
	}
	if app.runner.Err.(*bytes.Buffer).Len() != 0 {
		t.Fatal("explicit state denial became an advisory continuation")
	}
	requireUnchangedAdvisoryState(t, legacy, original)
	if requests.Load() != 0 {
		t.Fatal("state-denied explicit operation contacted the release feed")
	}
}

func advisoryLegacyStateFixture(t *testing.T) (application, string, []byte) {
	t.Helper()
	configDirectory := syntheticUserConfigDirectory(t)
	legacy := filepath.Join(configDirectory, "circularquay", "state")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"workload":"synthetic","fallback_fence":true}`)
	if err := os.WriteFile(filepath.Join(legacy, "synthetic.json"), original, 0o600); err != nil {
		t.Fatal(err)
	}
	app := application{runner: Runner{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}, buildVersion: "1.0.0"}
	if _, err := app.store(); !fault.Is(err, fault.Trust) {
		t.Fatalf("synthetic legacy state did not establish the existing fence: %v", err)
	}
	return app, legacy, original
}

func advisoryFeedFixture(t *testing.T) (*model.Updates, *atomic.Int32) {
	t.Helper()
	requests := &atomic.Int32{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)
	trust := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	digest := sha256.Sum256(trust)
	trustPath := filepath.Join(privateCLIDirectory(t), "feed-ca.pem")
	if err := os.WriteFile(trustPath, trust, 0o600); err != nil {
		t.Fatal(err)
	}
	return &model.Updates{
		Check: "interactive", Channel: "stable", IntervalHours: 1, TimeoutSeconds: 1,
		FeedEndpoint: server.URL, TrustStoreFile: trustPath, TrustStoreSHA256: hex.EncodeToString(digest[:]),
	}, requests
}

func requireAdvisoryPlan(t *testing.T, app application) {
	t.Helper()
	code, err := app.plan(model.Policy{Workloads: map[string]model.Workload{"synthetic": {Mode: "offline"}}})
	if err != nil || code != 0 {
		t.Fatalf("read-only plan did not continue: code %d, error %v", code, err)
	}
	var result struct {
		Valid         bool                  `json:"valid"`
		ValueRequests int                   `json:"value_requests"`
		Workloads     []struct{ ID string } `json:"workloads"`
	}
	if err := json.Unmarshal(app.runner.Out.(*bytes.Buffer).Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Valid || result.ValueRequests != 0 || len(result.Workloads) != 1 {
		t.Fatal("plan did not remain a valid, value-free inspection")
	}
	if result.Workloads[0].ID != "synthetic" {
		t.Fatal("plan lost its selected workload")
	}
}

func requireUnchangedAdvisoryState(t *testing.T, legacy string, original []byte) {
	t.Helper()
	actual, err := os.ReadFile(filepath.Join(legacy, "synthetic.json"))
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatal("legacy state changed during update inspection")
	}
	entries, err := os.ReadDir(legacy)
	if err != nil || len(entries) != 1 {
		t.Fatal("update inspection changed legacy state membership")
	}
	requireMissingPath(t, filepath.Join(filepath.Dir(filepath.Dir(legacy)), "cq", "state"))
}
