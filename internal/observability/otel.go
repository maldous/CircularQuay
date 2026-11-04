package observability

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxOTLPRequestBytes  = 1 << 20
	maxOTLPResponseBytes = 1 << 20
	instrumentationScope = "github.com/maldous/CircularQuay"
	operationEventName   = "cquay.operation"
	outcomeAttributeName = "cquay.outcome"
)

type otelBackend struct {
	tracer trace.Tracer
	logger otellog.Logger

	operationTotal    otelmetric.Int64Counter
	operationDuration otelmetric.Float64Histogram

	traces      *sdktrace.TracerProvider
	metrics     *sdkmetric.MeterProvider
	logs        *sdklog.LoggerProvider
	diag        *diagnostic
	exportState *exportHealth

	shutdownOnce sync.Once
	shutdownErr  error
	shutdownFail atomic.Bool
}

func newOTelBackend(ctx context.Context, config model.Observability, token, version string, diagnostic *diagnostic) (*otelBackend, error) {
	client, health, err := collectorClient(config, diagnostic)
	if err != nil {
		return nil, err
	}
	resource, err := telemetryResource(config, version)
	if err != nil {
		return nil, err
	}
	urls, err := parseCollectorURLs(config.Endpoint)
	if err != nil {
		return nil, err
	}
	headers := collectorHeaders(token)
	result := &otelBackend{diag: diagnostic, exportState: health}
	if enabled(config.Traces) {
		if err = result.configureTraces(ctx, config, resource, client, headers, urls.traces); err != nil {
			return nil, err
		}
	}
	if enabled(config.Metrics) {
		if err = result.configureMetrics(ctx, config, resource, client, headers, urls.metrics); err != nil {
			shutdownConfigured(result)
			return nil, err
		}
	}
	if enabled(config.Logs) {
		if err = result.configureLogs(ctx, config, resource, client, headers, urls.logs); err != nil {
			shutdownConfigured(result)
			return nil, err
		}
	}
	return result, nil
}

type collectorURLs struct{ traces, metrics, logs string }

func parseCollectorURLs(endpoint string) (collectorURLs, error) {
	base, err := transport.Endpoint(endpoint)
	if err != nil || base.Path != "" && base.Path != "/" {
		return collectorURLs{}, fault.New(fault.Invalid, "observability endpoint")
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return collectorURLs{
		traces:  endpoint + "/v1/traces",
		metrics: endpoint + "/v1/metrics",
		logs:    endpoint + "/v1/logs",
	}, nil
}

func collectorClient(config model.Observability, diagnostic *diagnostic) (*http.Client, *exportHealth, error) {
	health := &exportHealth{diagnostic: diagnostic}
	if config.TrustStoreFile != "" {
		client, err := transport.TrustStoreHTTPClientNoProxy(config.TrustStoreFile, config.TrustStoreSHA256)
		if err != nil {
			return nil, nil, err
		}
		client.Timeout = signalTimeout(config)
		return client, health, nil
	}
	configured := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client := transport.NewHTTPClient(configured)
	client.Timeout = signalTimeout(config)
	return client, health, nil
}

type exportHealth struct {
	failures   atomic.Uint64
	reported   atomic.Bool
	diagnostic *diagnostic
}

func (h *exportHealth) failed() {
	if h == nil {
		return
	}
	h.failures.Add(1)
	if h.diagnostic != nil && h.reported.CompareAndSwap(false, true) {
		h.diagnostic.record(context.Background(), diagnosticRecord{Time: time.Now().UTC(), Event: "telemetry_export", Outcome: OutcomeUnavailable})
	}
}
func (h *exportHealth) count() uint64 {
	if h == nil {
		return 0
	}
	return h.failures.Load()
}

func telemetryExportError() error { return errors.New("telemetry export") }

type safeTraceExporter struct {
	next   sdktrace.SpanExporter
	health *exportHealth
}

func (e safeTraceExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if err := e.next.ExportSpans(ctx, spans); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func (e safeTraceExporter) Shutdown(ctx context.Context) error {
	if err := e.next.Shutdown(ctx); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

type safeMetricExporter struct {
	next   sdkmetric.Exporter
	health *exportHealth
}

func (e safeMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return e.next.Temporality(kind)
}

func (e safeMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return e.next.Aggregation(kind)
}

func (e safeMetricExporter) Export(ctx context.Context, metrics *metricdata.ResourceMetrics) error {
	if err := e.next.Export(ctx, metrics); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func (e safeMetricExporter) ForceFlush(ctx context.Context) error {
	if err := e.next.ForceFlush(ctx); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func (e safeMetricExporter) Shutdown(ctx context.Context) error {
	if err := e.next.Shutdown(ctx); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

type safeLogExporter struct {
	next   sdklog.Exporter
	health *exportHealth
}

func (e safeLogExporter) Export(ctx context.Context, records []sdklog.Record) error {
	if err := e.next.Export(ctx, records); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func (e safeLogExporter) ForceFlush(ctx context.Context) error {
	if err := e.next.ForceFlush(ctx); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func (e safeLogExporter) Shutdown(ctx context.Context) error {
	if err := e.next.Shutdown(ctx); err != nil {
		e.health.failed()
		return telemetryExportError()
	}
	return nil
}

func collectorHeaders(token string) map[string]string {
	if token == "" {
		return map[string]string{}
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

func telemetryResource(config model.Observability, version string) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{attribute.String("service.name", config.ServiceName)}
	if version != "" {
		attributes = append(attributes, attribute.String("service.version", version))
	}
	if config.Environment != "" {
		attributes = append(attributes, attribute.String("deployment.environment.name", config.Environment))
	}
	if config.ResourceAttributes != nil {
		if config.ResourceAttributes.ServiceNamespace != "" {
			attributes = append(attributes, attribute.String("service.namespace", config.ResourceAttributes.ServiceNamespace))
		}
		if config.ResourceAttributes.ServiceInstanceID != "" {
			attributes = append(attributes, attribute.String("service.instance.id", config.ResourceAttributes.ServiceInstanceID))
		}
	}
	return resource.NewWithAttributes("", attributes...), nil
}

func (b *otelBackend) configureTraces(ctx context.Context, config model.Observability, resource *resource.Resource, client *http.Client, headers map[string]string, endpoint string) error {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHTTPClient(client),
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithCompression(otlptracehttp.NoCompression),
		otlptracehttp.WithMaxRequestSize(maxOTLPRequestBytes),
		otlptracehttp.WithMaxResponseSize(maxOTLPResponseBytes),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{Enabled: false}),
	)
	if err != nil {
		return fault.New(fault.Apply, "observability traces")
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(*config.Traces.SampleRatio)),
		sdktrace.WithBatcher(safeTraceExporter{next: exporter, health: b.exportState},
			sdktrace.WithMaxQueueSize(config.QueueSize),
			sdktrace.WithMaxExportBatchSize(config.BatchSize),
			sdktrace.WithBatchTimeout(time.Duration(config.ExportIntervalSeconds)*time.Second),
			sdktrace.WithExportTimeout(signalTimeout(config)),
		),
	)
	b.traces = provider
	b.tracer = provider.Tracer(instrumentationScope)
	return nil
}

func (b *otelBackend) configureMetrics(ctx context.Context, config model.Observability, resource *resource.Resource, client *http.Client, headers map[string]string, endpoint string) error {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(endpoint),
		otlpmetrichttp.WithHTTPClient(client),
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression),
		otlpmetrichttp.WithMaxRequestSize(maxOTLPRequestBytes),
		otlpmetrichttp.WithMaxResponseSize(maxOTLPResponseBytes),
		otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{Enabled: false}),
	)
	if err != nil {
		return fault.New(fault.Apply, "observability metrics")
	}
	reader := sdkmetric.NewPeriodicReader(safeMetricExporter{next: exporter, health: b.exportState},
		sdkmetric.WithInterval(time.Duration(config.Metrics.IntervalSeconds)*time.Second),
		sdkmetric.WithTimeout(signalTimeout(config)),
		sdkmetric.WithMaxExportBatchSize(config.BatchSize),
	)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithResource(resource), sdkmetric.WithReader(reader))
	meter := provider.Meter(instrumentationScope)
	if err = b.configureOperationMetrics(meter); err != nil {
		_ = provider.Shutdown(context.Background())
		return fault.New(fault.Apply, "observability metrics")
	}
	b.metrics = provider
	b.configureRuntimeMetrics(meter)
	return nil
}

func (b *otelBackend) configureOperationMetrics(meter otelmetric.Meter) error {
	var err error
	b.operationTotal, err = meter.Int64Counter("cquay.operation.total")
	if err != nil {
		return err
	}
	b.operationDuration, err = meter.Float64Histogram("cquay.operation.duration", otelmetric.WithUnit("s"))
	return err
}

func (b *otelBackend) configureRuntimeMetrics(meter otelmetric.Meter) {
	goroutines, firstErr := meter.Int64ObservableGauge("cquay.runtime.goroutines")
	heap, secondErr := meter.Int64ObservableGauge("cquay.runtime.heap.bytes")
	if firstErr != nil || secondErr != nil {
		return
	}
	_, _ = meter.RegisterCallback(func(_ context.Context, observer otelmetric.Observer) error {
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		observer.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))
		observer.ObserveInt64(heap, int64(memory.HeapAlloc))
		return nil
	}, goroutines, heap)
}

func (b *otelBackend) configureLogs(ctx context.Context, config model.Observability, resource *resource.Resource, client *http.Client, headers map[string]string, endpoint string) error {
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(endpoint),
		otlploghttp.WithHTTPClient(client),
		otlploghttp.WithHeaders(headers),
		otlploghttp.WithCompression(otlploghttp.NoCompression),
		otlploghttp.WithMaxRequestSize(maxOTLPRequestBytes),
		otlploghttp.WithMaxResponseSize(maxOTLPResponseBytes),
		otlploghttp.WithRetry(otlploghttp.RetryConfig{Enabled: false}),
	)
	if err != nil {
		return fault.New(fault.Apply, "observability logs")
	}
	processor := sdklog.NewBatchProcessor(safeLogExporter{next: exporter, health: b.exportState},
		sdklog.WithMaxQueueSize(config.QueueSize),
		sdklog.WithExportMaxBatchSize(config.BatchSize),
		sdklog.WithExportInterval(time.Duration(config.ExportIntervalSeconds)*time.Second),
		sdklog.WithExportTimeout(signalTimeout(config)),
	)
	provider := sdklog.NewLoggerProvider(sdklog.WithResource(resource), sdklog.WithProcessor(processor))
	b.logs = provider
	b.logger = provider.Logger(instrumentationScope)
	return nil
}

func (b *otelBackend) begin(ctx context.Context, operation Operation) (context.Context, span) {
	if b.tracer == nil {
		return ctx, localSpan{backend: b, operation: operation, correlation: correlation(ctx), started: time.Now()}
	}
	ctx = trace.ContextWithSpanContext(ctx, trace.SpanContext{})
	ctx, current := b.tracer.Start(ctx, "cquay."+string(operation.Phase), trace.WithAttributes(operationAttributes(operation)...))
	return ctx, otelSpan{backend: b, span: current, operation: operation, correlation: correlation(ctx), started: time.Now()}
}

func (b *otelBackend) event(ctx context.Context, event Event) {
	b.record(ctx, event)
}

func (b *otelBackend) health() Health {
	failures := b.diag.health()
	return Health{DiagnosticWriteFailures: failures, ExportFailures: b.exportState.count(), ShutdownExportFailure: b.shutdownFail.Load()}
}

func (b *otelBackend) record(ctx context.Context, event Event) {
	metricAttributes := append(operationAttributes(event.Operation), attribute.String(outcomeAttributeName, string(event.Outcome)))
	if b.operationTotal != nil {
		b.operationTotal.Add(ctx, 1, otelmetric.WithAttributes(metricAttributes...))
	}
	if b.logger != nil {
		logAttributes := append(metricAttributes, attribute.String("cquay.correlation_id", event.Correlation))
		record := otellog.Record{}
		record.SetTimestamp(event.At)
		record.SetEventName(operationEventName)
		record.SetSeverity(logSeverity(event.Outcome))
		record.SetSeverityText(string(event.Outcome))
		record.SetBody(attribute.StringValue(operationEventName))
		record.AddAttributes(logAttributes...)
		b.logger.Emit(ctx, record)
	}
	if b.diag != nil {
		b.diag.record(ctx, diagnosticRecord{Time: event.At, Event: "operation", Phase: event.Phase, Provider: event.ProviderKind, Outcome: event.Outcome, Correlation: event.Correlation})
	}
}

func operationAttributes(operation Operation) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("cquay.phase", string(operation.Phase)), attribute.String("cquay.provider_kind", operation.ProviderKind)}
}

func logSeverity(outcome Outcome) otellog.Severity {
	if outcome == OutcomeSuccess || outcome == OutcomeFallback {
		return otellog.SeverityInfo
	}
	return otellog.SeverityError
}

type otelSpan struct {
	backend     *otelBackend
	span        trace.Span
	operation   Operation
	correlation string
	started     time.Time
}

func (s otelSpan) end(outcome Outcome) {
	attributes := append(operationAttributes(s.operation), attribute.String(outcomeAttributeName, string(outcome)))
	s.span.SetAttributes(attributes...)
	s.span.End()
	s.backend.finish(context.Background(), s.operation, outcome, s.correlation, s.started)
}

type localSpan struct {
	backend     *otelBackend
	operation   Operation
	correlation string
	started     time.Time
}

func (s localSpan) end(outcome Outcome) {
	s.backend.finish(context.Background(), s.operation, outcome, s.correlation, s.started)
}

func (b *otelBackend) finish(ctx context.Context, operation Operation, outcome Outcome, correlation string, started time.Time) {
	event := Event{Operation: operation, Outcome: outcome, Correlation: correlation, At: time.Now().UTC()}
	b.record(ctx, event)
	if b.operationDuration != nil {
		attributes := append(operationAttributes(operation), attribute.String(outcomeAttributeName, string(outcome)))
		b.operationDuration.Record(ctx, time.Since(started).Seconds(), otelmetric.WithAttributes(attributes...))
	}
}

func (b *otelBackend) shutdown(ctx context.Context) error {
	b.shutdownOnce.Do(func() {
		var errs []error
		if b.traces != nil {
			errs = append(errs, b.traces.ForceFlush(ctx), b.traces.Shutdown(ctx))
		}
		if b.metrics != nil {
			errs = append(errs, b.metrics.ForceFlush(ctx), b.metrics.Shutdown(ctx))
		}
		if b.logs != nil {
			errs = append(errs, b.logs.ForceFlush(ctx), b.logs.Shutdown(ctx))
		}
		b.shutdownErr = errors.Join(errs...)
		if b.shutdownErr != nil && b.diag != nil {
			b.shutdownFail.Store(true)
			b.diag.record(context.Background(), diagnosticRecord{Time: time.Now().UTC(), Event: "telemetry_shutdown", Outcome: OutcomeUnavailable})
		}
	})
	return b.shutdownErr
}

func shutdownConfigured(backend *otelBackend) {
	if backend == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = backend.shutdown(ctx)
}
