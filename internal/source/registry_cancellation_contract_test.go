package source

import (
	"context"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestRegistryCancellationBeforeDispatch(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer stop()
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{{"cancelled", cancelled}, {"expired deadline", expired}} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			r := NewRegistry()
			r.Register("synthetic", ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
				called = true
				return model.Value{Data: "synthetic-value"}, nil
			}))
			value, err := r.Read(tc.ctx, model.Source{Type: "synthetic", Resources: []string{"item"}}, model.Binding{Resource: "item"})
			if called || value != (model.Value{}) || !fault.Is(err, fault.Denied) {
				t.Fatalf("cancelled read dispatched or released a result: called=%v, category=%s", called, fault.Category(err))
			}
		})
	}
}

func TestRegistryCancelledEnvironmentDoesNotReleaseValue(t *testing.T) {
	t.Setenv("TEST_SYNTHETIC_CANCELLED_VALUE", "synthetic-value")
	r := NewRegistry()
	s := model.Source{Type: "env", Resources: []string{"TEST_SYNTHETIC_CANCELLED_VALUE"}}
	b := model.Binding{Resource: "TEST_SYNTHETIC_CANCELLED_VALUE"}
	if value, err := r.Read(context.Background(), s, b); err != nil || value.Data != "synthetic-value" {
		t.Fatal("synthetic environment control was not readable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	value, err := r.Read(ctx, s, b)
	if value != (model.Value{}) || !fault.Is(err, fault.Denied) {
		t.Fatal("cancelled environment read did not deny value release")
	}
}

func TestRegistryCancellationDuringReadDiscardsValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := NewRegistry()
	r.Register("synthetic", ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) {
		cancel()
		return model.Value{Data: "synthetic-value", Version: "synthetic-version"}, nil
	}))
	value, err := r.Read(ctx, model.Source{Type: "synthetic", Resources: []string{"item"}}, model.Binding{Resource: "item"})
	if value != (model.Value{}) || !fault.Is(err, fault.Denied) {
		t.Fatal("cancelled in-flight read released a provider result")
	}
}

func TestRegistryUncancelledReadPreservesResult(t *testing.T) {
	want := model.Value{Data: "synthetic-value", Version: "synthetic-version"}
	r := NewRegistry()
	r.Register("synthetic", ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) { return want, nil }))
	value, err := r.Read(context.Background(), model.Source{Type: "synthetic", Resources: []string{"item"}}, model.Binding{Resource: "item"})
	if err != nil || value != want {
		t.Fatal("uncancelled read changed a provider result")
	}
}

func TestRegistryUncancelledErrorPreserved(t *testing.T) {
	want := fault.New(fault.Missing, "synthetic-missing")
	r := NewRegistry()
	r.Register("synthetic", ReadFunc(func(context.Context, model.Source, model.Binding) (model.Value, error) { return model.Value{}, want }))
	value, err := r.Read(context.Background(), model.Source{Type: "synthetic", Resources: []string{"item"}}, model.Binding{Resource: "item"})
	if value != (model.Value{}) || err != want {
		t.Fatal("uncancelled read changed a provider error")
	}
}
