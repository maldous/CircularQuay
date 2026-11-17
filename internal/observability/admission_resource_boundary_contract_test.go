package observability

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

func TestTelemetryAdmissionResourceIdentityAndSignalRefusals(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*model.Observability)
		operation string
	}{
		{"diagnostic vocabulary", func(c *model.Observability) { c.DiagnosticLevel = "private-level" }, "observability identity"},
		{"service control character", func(c *model.Observability) { c.ServiceName = "private\nservice" }, "observability identity"},
		{"namespace control character", func(c *model.Observability) { c.ResourceAttributes = &model.ResourceAttributes{ServiceNamespace: "private\x00namespace"} }, "observability resource"},
		{"no enabled signal", func(c *model.Observability) { c.Metrics = nil }, "observability signals"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := telemetryUpperBoundsConfig(t)
			test.configure(config)
			got, err := New(context.Background(), config, Options{})
			assertTelemetryBoundsFailure(t, got, err, fault.Invalid, test.operation)
		})
	}
	if enabled("unknown signal") || safeResourceText(strings.Repeat("x", 129)) || oneOf("private", "off", "error") {
		t.Fatal("closed telemetry configuration vocabulary accepted an unsupported value")
	}
}

func TestTelemetryAdmissionResourceTokenBoundaries(t *testing.T) {
	for _, test := range []struct {
		name  string
		data  string
		value string
		kind  fault.Kind
	}{
		{"trimmed synthetic token", "  synthetic-token\n", "synthetic-token", ""},
		{"empty file", "", "", fault.Trust},
		{"whitespace only", " \n\t", "", fault.Invalid},
		{"embedded control character", "synthetic\x00token", "", fault.Invalid},
		{"over bound", strings.Repeat("x", maximumTokenBytes+1), "", fault.Trust},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := telemetryAdmissionToken(t, test.data)
			got, err := readToken(path)
			if got != test.value {
				t.Fatal("token boundary returned unexpected data")
			}
			assertTelemetryAdmissionTokenResult(t, err, test.kind)
		})
	}
	if got, err := readToken(""); err != nil || got != "" {
		t.Fatal("absent optional token changed its no-read contract")
	}
}

func telemetryAdmissionToken(t *testing.T, data string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "synthetic-token")
	if err := platform.WriteProtected(path, []byte(data)); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertTelemetryAdmissionTokenResult(t *testing.T, err error, kind fault.Kind) {
	t.Helper()
	if kind == "" {
		if err != nil {
			t.Fatal("valid synthetic token was refused")
		}
		return
	}
	assertTelemetryAdmissionFault(t, err, kind, "observability token")
}

func assertTelemetryAdmissionFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation || errors.Unwrap(err) != nil {
		t.Fatal("telemetry refusal did not retain the exact safe typed error")
	}
}

func TestTelemetryAdmissionResourceSetupStopsBeforeExporter(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	config := telemetryUpperBoundsConfig(t)
	config.TokenFile = telemetryAdmissionToken(t, "synthetic-token")
	config.DiagnosticLevel = "debug"
	got, err := New(context.Background(), config, Options{})
	if got != nil || err != errDiagnosticDirectory {
		t.Fatal("missing diagnostic directory reached exporter construction")
	}
	config.DiagnosticLevel = "off"
	config.TrustStoreFile = filepath.Join(t.TempDir(), "missing-trust-store")
	config.TrustStoreSHA256 = strings.Repeat("0", 64)
	got, err = New(context.Background(), config, Options{})
	if got != nil {
		t.Fatal("missing protected trust store returned a telemetry runtime")
	}
	assertTelemetryAdmissionFault(t, err, fault.Trust, "trust store file")
}

func TestTelemetryAdmissionResourceNilRuntimeStaysAdvisory(t *testing.T) {
	ctx := WithRuntime(context.Background(), nil)
	if runtime := From(ctx); runtime == nil || runtime.Health() != (Health{}) {
		t.Fatal("absent runtime did not remain a disabled advisory runtime")
	}
	for _, runtime := range []*Runtime{nil, {}} {
		if runtime.Health() != (Health{}) || runtime.Shutdown(context.Background()) != nil {
			t.Fatal("unconfigured runtime attempted teardown work or reported health")
		}
	}
}

func TestTelemetryAdmissionResourceClientRefusalsKeepCallerUnchanged(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	downstream := &resourceRequestInspector{}
	client := &http.Client{Transport: downstream}
	before := *client
	for _, test := range []struct {
		name      string
		client    *http.Client
		declared  *resource.Resource
		signal    collectorSignal
		operation string
	}{
		{"nil client", nil, resource.Empty(), collectorTrace, "observability transport"},
		{"nil transport", &http.Client{}, resource.Empty(), collectorTrace, "observability transport"},
		{"unknown signal", client, resource.Empty(), collectorSignal(255), "observability transport"},
		{"nil declaration", client, nil, collectorTrace, "observability resource"},
		{"undeclared resource key", client, resource.NewSchemaless(attribute.String("private.key", "synthetic")), collectorTrace, "observability resource"},
		{"nonstring resource value", client, resource.NewSchemaless(attribute.Int("service.name", 7)), collectorTrace, "observability resource"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := collectorResourceClient(test.client, test.declared, test.signal)
			if got != nil || downstream.calls != 0 {
				t.Fatal("invalid collector resource crossed the in-process transport")
			}
			assertTelemetryAdmissionFault(t, err, fault.Invalid, test.operation)
		})
	}
	if !reflect.DeepEqual(*client, before) {
		t.Fatal("resource refusal mutated the caller-owned HTTP client")
	}
}

func TestTelemetryAdmissionResourceNilRequestsRefuseWithoutTransport(t *testing.T) {
	downstream := &resourceRequestInspector{}
	transport := collectorResourceTransport{next: downstream, approved: &resourcepb.Resource{}, signal: collectorTrace}
	for _, request := range []*http.Request{nil, {}} {
		response, err := transport.RoundTrip(request)
		if response != nil || downstream.calls != 0 {
			t.Fatal("absent collector request reached the downstream transport")
		}
		assertResourceTransportFailure(t, err, downstream.calls, false)
	}
}

func TestTelemetryAdmissionResourceWireRefusalsPreserveInputs(t *testing.T) {
	approved := &resourcepb.Resource{}
	before := proto.Clone(approved)
	for _, test := range []struct {
		name     string
		signal   collectorSignal
		body     []byte
		resource *resourcepb.Resource
	}{
		{"missing canonical resource", collectorTrace, nil, nil},
		{"unknown wire signal", collectorSignal(255), nil, approved},
		{"malformed metrics", collectorMetrics, []byte{0xff}, approved},
		{"malformed logs", collectorLogs, []byte{0xff}, approved},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := append([]byte(nil), test.body...)
			got, err := rewriteCollectorResource(test.signal, test.body, test.resource)
			if got != nil || !reflect.DeepEqual(test.body, body) || !proto.Equal(approved, before) {
				t.Fatal("wire refusal returned data or mutated caller inputs")
			}
			assertResourceTransportFailure(t, err, 0, false)
		})
	}
}
