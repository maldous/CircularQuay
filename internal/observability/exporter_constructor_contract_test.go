package observability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type exporterConstructorTransport struct{ calls atomic.Int64 }

func (transport *exporterConstructorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	return nil, errors.New("owned transport refuses every request")
}

func TestExporterConstructorContractCanceledTraceUsesRealSDKRefusal(t *testing.T) {
	config := resourcePipelineConfig()
	transport := &exporterConstructorTransport{}
	backend := &otelBackend{exportState: &exportHealth{}}
	cleanupResourceBackend(t, backend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := backend.configureTraces(ctx, config, telemetryResource(config, "fixture"), &http.Client{Transport: transport}, nil, "https://collector.invalid/v1/traces")
	assertExporterConstructorFailure(t, err, "observability traces")
	if backend.traces != nil || backend.tracer != nil || transport.calls.Load() != 0 {
		t.Fatal("canceled SDK trace construction published a provider or made a request")
	}
}

func TestExporterConstructorContractActualSDKBadHostRefusesBeforeTransport(t *testing.T) {
	for _, signal := range []string{"metrics", "logs"} {
		t.Run(signal, func(t *testing.T) { exporterConstructorBadHost(t, signal) })
	}
}

func exporterConstructorBadHost(t *testing.T, signal string) {
	t.Helper()
	ctx := resourcePipelineContext(t)
	config := resourcePipelineConfig()
	transport := &exporterConstructorTransport{}
	backend := &otelBackend{exportState: &exportHealth{}}
	cleanupResourceBackend(t, backend)
	client := &http.Client{Transport: transport}
	resource := telemetryResource(config, "fixture")
	calls := 0
	var err error
	// These are real errors from the pinned SDK's public option API, not a fake
	// constructor returning an invented failure. The private adapter control is
	// deliberately malformed; public New still validates policy URLs earlier.
	if signal == "metrics" {
		create := func(ctx context.Context, options ...otlpmetrichttp.Option) (sdkmetric.Exporter, error) {
			calls++
			return otlpmetrichttp.New(ctx, append(options, otlpmetrichttp.WithEndpoint("["))...)
		}
		err = backend.configureMetricsWithExporter(ctx, config, resource, client, nil, "https://collector.invalid/v1/metrics", create)
	} else {
		create := func(ctx context.Context, options ...otlploghttp.Option) (sdklog.Exporter, error) {
			calls++
			return otlploghttp.New(ctx, append(options, otlploghttp.WithEndpoint("["))...)
		}
		err = backend.configureLogsWithExporter(ctx, config, resource, client, nil, "https://collector.invalid/v1/logs", create)
	}
	assertExporterConstructorFailure(t, err, "observability "+signal)
	if calls != 1 || transport.calls.Load() != 0 || backend.metrics != nil || backend.logs != nil {
		t.Fatal("failed real SDK constructor published a provider or reached transport")
	}
}

func assertExporterConstructorFailure(t *testing.T, err error, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != fault.Apply || safe.Operation != operation {
		t.Fatal("constructor failure lost its exact sanitized kind and operation")
	}
}

type exporterConstructorAggregation struct {
	sdkmetric.Exporter
	closed atomic.Int64
}

func (exporter *exporterConstructorAggregation) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	if kind == sdkmetric.InstrumentKindHistogram {
		// A valid aggregation that is incompatible with a histogram reaches
		// the real SDK compatibility error, unlike invalid bounds which the
		// SDK deliberately replaces with a default aggregation.
		return sdkmetric.AggregationLastValue{}
	}
	return exporter.Exporter.Aggregation(kind)
}

func (exporter *exporterConstructorAggregation) Shutdown(ctx context.Context) error {
	exporter.closed.Add(1)
	return exporter.Exporter.Shutdown(ctx)
}

func TestExporterConstructorContractRealSDKInstrumentRefusalShutsDownOwnedProvider(t *testing.T) {
	ctx := resourcePipelineContext(t)
	config := resourcePipelineConfig()
	transport := &exporterConstructorTransport{}
	backend := &otelBackend{exportState: &exportHealth{}}
	cleanupResourceBackend(t, backend)
	var owned *exporterConstructorAggregation
	create := func(ctx context.Context, options ...otlpmetrichttp.Option) (sdkmetric.Exporter, error) {
		exporter, err := otlpmetrichttp.New(ctx, options...)
		if err != nil {
			return nil, err
		}
		owned = &exporterConstructorAggregation{Exporter: exporter}
		return owned, nil
	}
	err := backend.configureMetricsWithExporter(ctx, config, telemetryResource(config, "fixture"), &http.Client{Transport: transport}, nil, "https://collector.invalid/v1/metrics", create)
	assertExporterConstructorFailure(t, err, "observability metrics")
	if owned == nil || owned.closed.Load() != 1 || backend.metrics != nil || backend.operationTotal == nil {
		t.Fatal("instrument refusal lost the valid counter or its exact provider cleanup")
	}
}

type exporterConstructorFixture struct {
	server *httptest.Server
	config model.Observability
	bad    atomic.Bool
}

func newExporterConstructorFixture(t *testing.T) *exporterConstructorFixture {
	t.Helper()
	fixture := &exporterConstructorFixture{config: resourcePipelineConfig()}
	fixture.server = httptest.NewTLSServer(http.HandlerFunc(fixture.serve))
	t.Cleanup(fixture.server.Close)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := platform.EnsurePrivateDirectory(root); err != nil {
		t.Fatal(err)
	}
	fixture.config.Endpoint = fixture.server.URL
	fixture.config.TokenFile = filepath.Join(root, "token")
	fixture.config.TrustStoreFile = filepath.Join(root, "trust.pem")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: fixture.server.Certificate().Raw})
	digest := sha256.Sum256(certificate)
	fixture.config.TrustStoreSHA256 = hex.EncodeToString(digest[:])
	writeExporterConstructorFile(t, fixture.config.TokenFile, []byte("owned-synthetic-token"))
	writeExporterConstructorFile(t, fixture.config.TrustStoreFile, certificate)
	return fixture
}

func writeExporterConstructorFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := platform.WriteProtected(path, data); err != nil {
		t.Fatal(err)
	}
}

func (fixture *exporterConstructorFixture) serve(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer owned-synthetic-token" {
		fixture.bad.Store(true)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	switch request.URL.Path {
	case "/v1/traces", "/v1/metrics", "/v1/logs":
	default:
		fixture.bad.Store(true)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	read, err := io.Copy(io.Discard, io.LimitReader(request.Body, maxOTLPRequestBytes+1))
	if err != nil || read > maxOTLPRequestBytes {
		fixture.bad.Store(true)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	writer.Header().Set("Content-Type", "application/x-protobuf")
	writer.WriteHeader(http.StatusOK)
}

func TestExporterConstructorContractPublicPipelineUsesOnlyOwnedTLSAndShutsDown(t *testing.T) {
	fixture := newExporterConstructorFixture(t)
	ctx := resourcePipelineContext(t)
	runtime, err := New(ctx, &fixture.config, Options{Version: "owned-fixture"})
	if err != nil || runtime == nil {
		t.Fatal("valid owned TLS pipeline control was refused")
	}
	t.Cleanup(func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := runtime.Shutdown(shutdown); err != nil {
			t.Error("owned TLS pipeline shutdown failed")
		}
	})
	backend, ok := runtime.backend.(*otelBackend)
	if !ok || backend.traces == nil || backend.metrics == nil || backend.logs == nil {
		t.Fatal("public pipeline returned without every selected provider")
	}
	if err := runtime.Shutdown(ctx); err != nil || fixture.bad.Load() {
		t.Fatal("owned TLS pipeline lost bounded shutdown or its fixed request contract")
	}
}

func TestExporterConstructorContractCanceledPublicPipelineReturnsNoRuntime(t *testing.T) {
	fixture := newExporterConstructorFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runtime, err := New(ctx, &fixture.config, Options{Version: "owned-fixture"})
	assertExporterConstructorFailure(t, err, "observability traces")
	if runtime != nil || fixture.bad.Load() {
		t.Fatal("canceled public pipeline produced runtime authority")
	}
}
