package observability

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type constructorRefusalTransport struct{ calls int }

func (transport *constructorRefusalTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls++
	return nil, errors.New("synthetic transport must remain unused")
}

func TestConstructorLifecycleContractSignalRefusalsPrecedeAllocation(t *testing.T) {
	for _, signal := range []string{"traces", "metrics", "logs"} {
		t.Run(signal, func(t *testing.T) { constructorSignalRefusal(t, signal) })
	}
}

func constructorSignalRefusal(t *testing.T, signal string) {
	t.Helper()
	yes := true
	config := model.Observability{}
	switch signal {
	case "traces":
		config.Traces = &model.TracesSignal{Enabled: &yes}
	case "metrics":
		config.Metrics = &model.MetricsSignal{Enabled: &yes}
	default:
		config.Logs = &model.LogsSignal{Enabled: &yes}
	}
	transport := &constructorRefusalTransport{}
	backend := &otelBackend{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := backend.configureSignals(ctx, config, nil, &http.Client{Transport: transport}, nil, collectorURLs{})
	if err == nil || transport.calls != 0 || backend.traces != nil || backend.metrics != nil || backend.logs != nil {
		t.Fatal("missing declared resource reached allocation or collector transport")
	}
}

func TestConstructorLifecycleContractNoSignalsNeedsNoExporter(t *testing.T) {
	backend, err := newOTelBackend(context.Background(), model.Observability{Endpoint: "https://collector.invalid"}, "synthetic", "1.0.0", nil)
	if err != nil || backend == nil {
		t.Fatal("valid disabled-signal constructor failed")
	}
	defer shutdownConfigured(backend)
	if backend.traces != nil || backend.metrics != nil || backend.logs != nil {
		t.Fatal("disabled signals allocated exporter providers")
	}
	shutdownConfigured(nil)
}

func TestConstructorLifecycleContractBadEndpointReturnsNoBackend(t *testing.T) {
	backend, err := newOTelBackend(context.Background(), model.Observability{Endpoint: "https://collector.invalid/private"}, "synthetic", "1.0.0", nil)
	if err == nil || backend != nil {
		t.Fatal("invalid collector endpoint returned a partial backend")
	}
}

type constructorFailureMeter struct {
	otelmetric.Meter
	counters int
	gauges   int
	err      error
}

func (meter *constructorFailureMeter) Int64Counter(name string, _ ...otelmetric.Int64CounterOption) (otelmetric.Int64Counter, error) {
	meter.counters++
	if name != "cq.operation.total" {
		return nil, errors.New("unexpected synthetic counter")
	}
	return nil, meter.err
}

func (meter *constructorFailureMeter) Int64ObservableGauge(name string, _ ...otelmetric.Int64ObservableGaugeOption) (otelmetric.Int64ObservableGauge, error) {
	meter.gauges++
	if name != "cq.runtime.goroutines" && name != "cq.runtime.heap.bytes" {
		return nil, errors.New("unexpected synthetic gauge")
	}
	return nil, meter.err
}

func TestConstructorLifecycleContractMeterFailuresDoNotUsePartialInstruments(t *testing.T) {
	sentinel := errors.New("synthetic meter refusal")
	meter := &constructorFailureMeter{err: sentinel}
	backend := &otelBackend{}
	if err := backend.configureOperationMetrics(meter); !errors.Is(err, sentinel) || meter.counters != 1 || backend.operationTotal != nil || backend.operationDuration != nil {
		t.Fatal("counter refusal did not stop before subsequent instrument creation")
	}
	// The embedded interface is deliberately nil: an erroneous callback
	// registration would fail instead of falling back to a global provider.
	backend.configureRuntimeMetrics(meter)
	if meter.gauges != 2 {
		t.Fatal("runtime gauge creation did not preserve its two fixed admissions")
	}
}
