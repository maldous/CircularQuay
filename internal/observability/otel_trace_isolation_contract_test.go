package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestOperationTraceExportIsolatesUntrustedIncomingContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown in-memory tracer: %v", err)
		}
	})
	runtime := &Runtime{backend: &otelBackend{
		tracer: provider.Tracer(instrumentationScope), exportState: &exportHealth{},
	}}
	incoming, remote := untrustedOperationTraceContext(t)
	ctx, finish := Start(WithRuntime(incoming, runtime), Operation{
		Phase: Phase("synthetic private workload"), ProviderKind: "https://private.example/token",
	})
	finish(Outcome("synthetic private failure"))
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("exported operation spans = %d, want 1", len(spans))
	}
	span := spans[0]
	assertIsolatedOperationTrace(t, span, remote)
	if !trace.SpanContextFromContext(ctx).Equal(span.SpanContext) {
		t.Fatal("operation context does not identify its exported span")
	}
	assertClosedOperationTraceAttributes(t, span)
	if runtime.Health() != (Health{}) {
		t.Fatal("successful in-memory trace changed advisory health")
	}
}

func untrustedOperationTraceContext(t *testing.T) (context.Context, trace.SpanContext) {
	t.Helper()
	state, err := trace.ParseTraceState("synthetic=private-context")
	if err != nil {
		t.Fatalf("synthetic trace state: %v", err)
	}
	remote := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1, 2, 3}, SpanID: trace.SpanID{4, 5, 6},
		TraceFlags: trace.FlagsSampled, TraceState: state, Remote: true,
	})
	member, err := baggage.NewMember("synthetic.identity", "private-principal")
	if err != nil {
		t.Fatalf("synthetic baggage member: %v", err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatalf("synthetic baggage: %v", err)
	}
	ctx := baggage.ContextWithBaggage(context.Background(), bag)
	return trace.ContextWithRemoteSpanContext(ctx, remote), remote
}

func assertIsolatedOperationTrace(t *testing.T, span tracetest.SpanStub, remote trace.SpanContext) {
	t.Helper()
	if !span.SpanContext.IsValid() || span.SpanContext.TraceID() == remote.TraceID() || span.SpanContext.IsRemote() {
		t.Fatal("exported operation reused an untrusted trace identity")
	}
	if span.Parent.IsValid() || span.SpanContext.TraceState().Len() != 0 || len(span.Links) != 0 || len(span.Events) != 0 {
		t.Fatal("exported operation inherited untrusted parent metadata")
	}
	if span.Name != "cq.cli" {
		t.Fatal("exported operation name contains an uncontrolled phase")
	}
}

func assertClosedOperationTraceAttributes(t *testing.T, span tracetest.SpanStub) {
	t.Helper()
	want := map[string]string{"cq.phase": "cli", "cq.provider_kind": "none", "cq.outcome": "failed"}
	if len(span.Attributes) != len(want) {
		t.Fatal("exported operation did not retain exactly the closed label set")
	}
	for _, attr := range span.Attributes {
		expected, ok := want[string(attr.Key)]
		if !ok || attr.Value.AsInterface() != expected {
			t.Fatal("exported operation leaked an uncontrolled label or baggage value")
		}
		delete(want, string(attr.Key))
	}
	if len(want) != 0 {
		t.Fatal("exported operation omitted a required label")
	}
}
