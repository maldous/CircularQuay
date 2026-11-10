package observability

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestOTelCoverageHeadersAndCollectorURLBoundaries(t *testing.T) {
	if headers := collectorHeaders(""); len(headers) != 0 {
		t.Fatalf("empty token headers = %#v", headers)
	}
	if headers := collectorHeaders("synthetic-token"); headers["Authorization"] != "Bearer synthetic-token" || len(headers) != 1 {
		t.Fatalf("token headers = %#v", headers)
	}
	urls, err := parseCollectorURLs("https://collector.example")
	if err != nil {
		t.Fatalf("root collector = %v", err)
	}
	if urls != (collectorURLs{traces: "https://collector.example/v1/traces", metrics: "https://collector.example/v1/metrics", logs: "https://collector.example/v1/logs"}) {
		t.Fatalf("collector URLs = %#v", urls)
	}
	for _, endpoint := range []string{"", "https://user@collector.example", "https://collector.example/#fragment"} {
		if _, err := parseCollectorURLs(endpoint); err == nil {
			t.Fatalf("unsafe collector endpoint accepted: %q", endpoint)
		}
	}
}

func TestOTelCoverageCollectorClientAndDisabledBackendLifecycle(t *testing.T) {
	assertCollectorClientDefaultsAndRedirectPolicy(t)
	assertDisabledBackendLifecycle(t)
}

func assertCollectorClientDefaultsAndRedirectPolicy(t *testing.T) {
	t.Helper()

	client, health, err := collectorClient(model.Observability{TimeoutSeconds: 7}, nil)
	if err != nil || health == nil || client.Timeout != 7*time.Second || client.Transport == nil || client.CheckRedirect == nil {
		t.Fatalf("default collector client = %#v, health=%#v, err=%v", client, health, err)
	}
	request, err := http.NewRequest(http.MethodGet, "https://redirect.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.CheckRedirect(request, nil) == nil {
		t.Fatal("collector client followed a redirect")
	}
}

func assertDisabledBackendLifecycle(t *testing.T) {
	t.Helper()

	diagnostic := &diagnostic{level: "off"}
	backend, err := newOTelBackend(context.Background(), model.Observability{Endpoint: "https://collector.example"}, "", "", diagnostic)
	if err != nil {
		t.Fatalf("disabled OTel backend = %v", err)
	}
	if backend.tracer != nil || backend.metrics != nil || backend.logs != nil || backend.exportState == nil {
		t.Fatalf("disabled backend configured an unexpected signal: %#v", backend)
	}
	if err := backend.shutdown(context.Background()); err != nil {
		t.Fatalf("disabled backend shutdown = %v", err)
	}
	shutdownConfigured(nil)
}

func TestOTelCoverageExportHealthCountsWithoutAffectingDiagnostics(t *testing.T) {
	health := &exportHealth{}
	health.failed()
	health.failed()
	if got := health.count(); got != 2 {
		t.Fatalf("export failures = %d, want 2", got)
	}
	var absent *exportHealth
	absent.failed()
	if absent.count() != 0 {
		t.Fatalf("nil export health count = %d", absent.count())
	}

	diagnostic := &diagnostic{level: "off"}
	withDiagnostic := &exportHealth{diagnostic: diagnostic}
	withDiagnostic.failed()
	withDiagnostic.failed()
	if withDiagnostic.count() != 2 || diagnostic.health() != 0 {
		t.Fatalf("disabled diagnostic health = failures %d writes %d", withDiagnostic.count(), diagnostic.health())
	}
}

func TestOTelCoverageResourceOmitsUnsetOptionalFields(t *testing.T) {
	resource, err := telemetryResource(model.Observability{ServiceName: "cq"}, "")
	if err != nil {
		t.Fatal(err)
	}
	attributes := resource.Attributes()
	if len(attributes) != 1 || string(attributes[0].Key) != "service.name" || attributes[0].Value.AsString() != "cq" {
		t.Fatalf("resource attributes = %#v", attributes)
	}
}
