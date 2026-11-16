package deliver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

type publicationPreparation struct {
	name            string
	generation      model.Generation
	activeKind      fault.Kind
	activeOperation string
}

func TestPublicationEndedContextPrecedesPreparation(t *testing.T) {
	for _, targetType := range []string{"cloudflare", "netlify", "windmill"} {
		t.Run(targetType, func(t *testing.T) {
			target := publicationCancellationTarget(t, targetType)
			for _, preparation := range []publicationPreparation{
				{name: "missing_credential", generation: generation(), activeKind: fault.Denied, activeOperation: "credential_file"},
				{name: "invalid_generation", generation: model.Generation{}, activeKind: fault.Invalid, activeOperation: "empty_generation"},
			} {
				t.Run(preparation.name, func(t *testing.T) {
					assertPublicationCancellationBeforePreparation(t, target, preparation)
				})
			}
		})
	}
}

func publicationCancellationTarget(t *testing.T, targetType string) model.Target {
	t.Helper()
	tokenFile := filepath.Join(t.TempDir(), "missing-publication-token")
	if _, err := os.Lstat(tokenFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("synthetic token path must be absent")
	}
	return model.Target{
		Type: targetType, Endpoint: "https://publication.invalid", Account: "synthetic-account",
		Project: "synthetic-project", Environment: "synthetic-context", Resource: "synthetic-resource",
		TokenFile: tokenFile, AcceptSnapshot: true,
	}
}

func assertPublicationCancellationBeforePreparation(t *testing.T, target model.Target, preparation publicationPreparation) {
	t.Helper()
	provider := &publicationCancellationTransport{}
	client := &http.Client{Transport: provider}
	assertPublicationFault(t, ApplyWithClient(context.Background(), client, target, preparation.generation), preparation.activeKind, preparation.activeOperation)
	if provider.requests != 0 {
		t.Fatal("preparation failure reached the provider")
	}
	for _, ended := range []struct {
		name   string
		create func() (context.Context, context.CancelFunc)
		cause  error
	}{
		{name: "cancelled", create: cancelledPublicationContext, cause: context.Canceled},
		{name: "deadline_exceeded", create: expiredPublicationContext, cause: context.DeadlineExceeded},
	} {
		t.Run(ended.name, func(t *testing.T) {
			ctx, cancel := ended.create()
			defer cancel()
			if !errors.Is(ctx.Err(), ended.cause) {
				t.Fatal("caller context must already have the selected terminal cause")
			}
			assertPublicationFault(t, ApplyWithClient(ctx, client, target, preparation.generation), fault.Denied, "provider_cancelled")
			if provider.requests != 0 {
				t.Fatal("ended publication reached the provider")
			}
		})
	}
}

func cancelledPublicationContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx, cancel
}

func expiredPublicationContext() (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), time.Unix(1, 0))
}

func assertPublicationFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	var failure *fault.Error
	if !errors.As(err, &failure) {
		t.Fatalf("publication error type = %T; want a structured fault", err)
	}
	if failure.Kind != kind || failure.Operation != operation {
		t.Fatalf("publication fault = (%q, %q); want (%q, %q)", failure.Kind, failure.Operation, kind, operation)
	}
}

type publicationCancellationTransport struct{ requests int }

func (transport *publicationCancellationTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.requests++
	return nil, errors.New("unexpected synthetic publication request")
}
