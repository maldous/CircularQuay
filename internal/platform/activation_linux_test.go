//go:build linux && !android

package platform

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

func TestActivateNativeCommandContract(t *testing.T) {
	for _, test := range []struct {
		name, activation, verb string
	}{
		{name: "restart", activation: "restart", verb: "try-restart"},
		{name: "reload", activation: "reload", verb: "reload"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called, err := runActivationContract(t, test.activation, test.verb)
			if err != nil || !called {
				t.Fatalf("activateNativeWithRunner = %v, called %t", err, called)
			}
		})
	}
}

func runActivationContract(t *testing.T, activation, verb string) (bool, error) {
	t.Helper()
	called := false
	err := activateNativeWithRunner(context.Background(), model.Target{Service: "demo.service", Activation: activation}, func(ctx context.Context, command systemCommand) error {
		called = true
		assertSystemctlCommand(t, ctx, command, verb)
		return nil
	})
	return called, err
}

func assertSystemctlCommand(t *testing.T, ctx context.Context, command systemCommand, verb string) {
	t.Helper()
	if command.path != "/usr/bin/systemctl" || !reflect.DeepEqual(command.args, []string{"--no-ask-password", verb, "--", "demo.service"}) || len(command.env) != 0 || command.stdin != nil {
		t.Fatalf("command = %#v", command)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("missing deadline")
	}
}

func TestActivateNativeRejectsInvalidServiceBeforeRunner(t *testing.T) {
	for _, target := range []model.Target{{Service: "demo.socket", Activation: "restart"}, {Service: "demo.service", Activation: "none"}} {
		called := false
		err := activateNativeWithRunner(context.Background(), target, func(context.Context, systemCommand) error {
			called = true
			return nil
		})
		if !fault.Is(err, fault.Invalid) || called {
			t.Fatalf("invalid target = %#v, err %v, called %t", target, err, called)
		}
	}
}

func TestActivateNativeMapsRunnerFailure(t *testing.T) {
	err := activateNativeWithRunner(context.Background(), model.Target{Service: "demo.service", Activation: "restart"}, func(context.Context, systemCommand) error { return errors.New("synthetic failure") })
	if !fault.Is(err, fault.Apply) {
		t.Fatalf("runner failure = %v", err)
	}
}

func TestActivateNativeDeadlineIsBounded(t *testing.T) {
	start := time.Now()
	err := activateNativeWithRunner(context.Background(), model.Target{Service: "demo.service", Activation: "restart"}, func(ctx context.Context, _ systemCommand) error {
		deadline, ok := ctx.Deadline()
		if !ok || deadline.Before(start.Add(29*time.Second)) || deadline.After(start.Add(31*time.Second)) {
			t.Fatalf("deadline = %v, ok %t", deadline, ok)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
