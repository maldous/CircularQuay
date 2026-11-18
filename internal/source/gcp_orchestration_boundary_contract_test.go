package source

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGCPOrchestrationBoundaryContractAdmissionPrecedesFactory(t *testing.T) {
	for _, test := range []struct {
		name    string
		source  model.Source
		binding model.Binding
		kind    fault.Kind
	}{
		{"resource", gcpOrchestrationSource(), model.Binding{Resource: "projects/other/secrets/fixture"}, fault.Denied},
		{"identity", model.Source{Project: "synthetic", Identity: "developer"}, gcpOrchestrationBinding(), fault.Unsupported},
	} {
		t.Run(test.name, func(t *testing.T) {
			factory := func(context.Context, model.Source) (gcpSecretAccess, error) {
				t.Fatal("refused resource or identity reached client construction")
				return gcpSecretAccess{}, nil
			}
			value, err := readGCPWithFactory(context.Background(), test.source, test.binding, factory)
			assertRejectedProviderValue(t, value, err, test.kind, "PRIVATE_DATA")
		})
	}
}

func TestGCPOrchestrationBoundaryContractConstructionFailureIsSanitized(t *testing.T) {
	calls := 0
	connect := func(ctx context.Context, options ...option.ClientOption) (*secretmanager.Client, error) {
		calls++
		if ctx.Err() != nil || len(options) != 2 {
			t.Fatal("constructor lost the request context or explicit options")
		}
		return nil, status.Error(codes.Unavailable, "PRIVATE_DATA")
	}
	factory := func(ctx context.Context, source model.Source) (gcpSecretAccess, error) {
		return newGCPSecretAccessWithConstructor(ctx, source, connect)
	}
	value, err := readGCPWithFactory(context.Background(), gcpOrchestrationSource(), gcpOrchestrationBinding(), factory)
	assertRejectedProviderValue(t, value, err, fault.Unavailable, "PRIVATE_DATA")
	if calls != 1 {
		t.Fatalf("constructor failure was retried: %d", calls)
	}
}

func TestGCPOrchestrationBoundaryContractAccessFailureClosesOnce(t *testing.T) {
	for _, test := range []struct {
		name string
		code codes.Code
		kind fault.Kind
	}{
		{"permission", codes.PermissionDenied, fault.Denied},
		{"missing", codes.NotFound, fault.Missing},
		{"deadline", codes.DeadlineExceeded, fault.Unavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			runGCPOrchestrationFailure(t, test.code, test.kind)
		})
	}
}

func runGCPOrchestrationFailure(t *testing.T, code codes.Code, kind fault.Kind) {
	t.Helper()
	calls, closes := 0, 0
	factory := func(ctx context.Context, source model.Source) (gcpSecretAccess, error) {
		assertGCPOrchestrationSource(t, ctx, source)
		return gcpSecretAccess{
			access: func(ctx context.Context, request *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
				calls++
				assertGCPOrchestrationAccess(t, ctx, request)
				if closes != 0 {
					t.Fatal("client closed before its request completed")
				}
				return nil, status.Error(code, "PRIVATE_DATA")
			},
			close: func() error {
				closes++
				return nil
			},
		}, nil
	}
	value, err := readGCPWithFactory(context.Background(), gcpOrchestrationSource(), gcpOrchestrationBinding(), factory)
	assertRejectedProviderValue(t, value, err, kind, "PRIVATE_DATA")
	if calls != 1 || closes != 1 {
		t.Fatalf("failed access lifetime = calls%d closes%d", calls, closes)
	}
}

func TestGCPOrchestrationBoundaryContractGeneratedSDKAndSelection(t *testing.T) {
	calls, closes := 0, 0
	base := cloudRoundTripper(func(request *http.Request) (*http.Response, error) {
		calls++
		assertGCPOrchestrationHTTP(t, request)
		body := &gcpOrchestrationBody{Reader: strings.NewReader(gcpOrchestrationResponse()), closed: &closes}
		return &http.Response{StatusCode: http.StatusOK, Request: request, Header: http.Header{"Content-Type": {"application/json"}}, Body: body}, nil
	})
	factory := func(ctx context.Context, source model.Source) (gcpSecretAccess, error) {
		assertGCPOrchestrationSource(t, ctx, source)
		connect := func(ctx context.Context, _ ...option.ClientOption) (*secretmanager.Client, error) {
			// This constructor is deliberately REST-only with explicit contained
			// HTTP. It does not exercise production metadata or its gRPC dialer.
			return secretmanager.NewRESTClient(ctx, option.WithHTTPClient(cloudHTTPClient(base)), option.WithEndpoint("https://secretmanager.synthetic"))
		}
		return newGCPSecretAccessWithConstructor(ctx, source, connect)
	}
	value, err := readGCPWithFactory(context.Background(), gcpOrchestrationSource(), gcpOrchestrationBinding(), factory)
	if err != nil || value.Data != "SYNTHETIC_GCP_VALUE" || value.Version != "projects/synthetic/secrets/fixture/versions/7" || !value.AuditVerified {
		t.Fatalf("generated SDK did not retain admitted value: %#v, %v", value, err)
	}
	if calls != 1 || closes != 1 {
		t.Fatalf("generated SDK request/body lifetime = %d/%d", calls, closes)
	}
}

func TestGCPOrchestrationBoundaryContractInvalidResultStillCloses(t *testing.T) {
	closes := 0
	factory := func(context.Context, model.Source) (gcpSecretAccess, error) {
		return gcpSecretAccess{
			access: func(context.Context, *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
				return &secretmanagerpb.AccessSecretVersionResponse{}, nil
			},
			close: func() error {
				closes++
				return nil
			},
		}, nil
	}
	value, err := readGCPWithFactory(context.Background(), gcpOrchestrationSource(), gcpOrchestrationBinding(), factory)
	assertRejectedProviderValue(t, value, err, fault.Missing, "PRIVATE_DATA")
	if closes != 1 {
		t.Fatalf("invalid response leaked client lifetime: %d", closes)
	}
}

func gcpOrchestrationSource() model.Source {
	return model.Source{Project: "synthetic", Account: "service@example.test", Identity: "metadata", AuditVerified: true}
}

func gcpOrchestrationBinding() model.Binding {
	return model.Binding{Resource: "projects/synthetic/secrets/fixture", Version: "7", Field: "KEY"}
}

func assertGCPOrchestrationSource(t *testing.T, ctx context.Context, source model.Source) {
	t.Helper()
	if ctx.Err() != nil || source.Project != "synthetic" || source.Account != "service@example.test" || source.Identity != "metadata" {
		t.Fatal("factory lost admitted source/context")
	}
}

func assertGCPOrchestrationAccess(t *testing.T, ctx context.Context, request *secretmanagerpb.AccessSecretVersionRequest) {
	t.Helper()
	if ctx.Err() != nil || request.Name != "projects/synthetic/secrets/fixture/versions/7" {
		t.Fatal("access lost context or version-qualified resource")
	}
}

func assertGCPOrchestrationHTTP(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.URL.Host != "secretmanager.synthetic" {
		t.Fatalf("request escaped synthetic transport: %s", request.URL)
	}
	if request.URL.Path != "/v1/projects/synthetic/secrets/fixture/versions/7:access" {
		t.Fatal("generated SDK changed version-qualified access route")
	}
}

func gcpOrchestrationResponse() string {
	data := []byte(`{"KEY":"SYNTHETIC_GCP_VALUE"}`)
	return fmt.Sprintf(`{"name":"projects/synthetic/secrets/fixture/versions/7","payload":{"data":"%s","dataCrc32c":"%d"}}`, base64.StdEncoding.EncodeToString(data), validCloudCRC(data))
}

type gcpOrchestrationBody struct {
	*strings.Reader
	closed *int
}

func (body *gcpOrchestrationBody) Close() error {
	*body.closed++
	return nil
}
