package keyservice

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestServiceReleaseContractInvalidRequestsNeverResolveEnrollment(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		operation Operation
		change    func(*serviceReleaseRequest)
		code      string
	}{
		{"wrap artifact", OperationWrap, func(r *serviceReleaseRequest) { r.artifact.ID = "" }, "keyservice_plaintext"},
		{"wrap empty", OperationWrap, func(r *serviceReleaseRequest) { r.material = nil }, "keyservice_plaintext"},
		{"unwrap artifact", OperationUnwrap, func(r *serviceReleaseRequest) { r.artifact.Version = "" }, "keyservice_ciphertext"},
		{"unwrap empty", OperationUnwrap, func(r *serviceReleaseRequest) { r.material = nil }, "keyservice_ciphertext"},
		{"age artifact", OperationRetrieveAge, func(r *serviceReleaseRequest) { r.artifact.ID = "" }, "keyservice_artifact"},
		{"wrap binding", OperationWrap, func(r *serviceReleaseRequest) { r.binding.Tenant = "" }, "keyservice_binding"},
		{"unwrap binding", OperationUnwrap, func(r *serviceReleaseRequest) { r.binding.Profile = "" }, "keyservice_binding"},
		{"age binding", OperationRetrieveAge, func(r *serviceReleaseRequest) { r.binding.Principal = "" }, "keyservice_binding"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := newServiceReleaseFixture(t, scenario.operation)
			request := newServiceReleaseRequest()
			scenario.change(&request)
			value, err := serviceReleaseInvoke(t, fixture.service(), scenario.operation, request)
			requireServiceReleaseEmpty(t, value, err, fault.Invalid, scenario.code)
			fixture.requireCalls(serviceReleaseCalls{})
		})
	}
}

func TestServiceReleaseContractAbsentOrUnboundEnrollmentNeverReachesProvider(t *testing.T) {
	for _, operation := range []Operation{OperationUnwrap, OperationRetrieveAge} {
		for _, missing := range []bool{false, true} {
			t.Run(string(operation)+"/enrollment", func(t *testing.T) {
				fixture := newServiceReleaseFixture(t, operation)
				fixture.found = !missing
				if !missing {
					fixture.enrollment.Binding.Application = "different-application"
				}
				value, err := serviceReleaseInvoke(t, fixture.service(), operation, newServiceReleaseRequest())
				requireServiceReleaseEmpty(t, value, err, fault.Denied, "keyservice_enrollment")
				fixture.requireCalls(serviceReleaseCalls{active: 1})
			})
		}
	}
}

func TestServiceReleaseContractProviderErrorClearsPartialOwnedMaterial(t *testing.T) {
	for _, operation := range []Operation{OperationUnwrap, OperationRetrieveAge} {
		for _, failure := range []struct {
			name string
			err  error
			kind fault.Kind
		}{
			{"typed", fault.New(fault.Unavailable, "SYNTHETIC_PRIVATE_PROVIDER_DETAIL"), fault.Unavailable},
			{"untyped", errors.New("SYNTHETIC_PRIVATE_PROVIDER_DETAIL"), fault.Apply},
		} {
			t.Run(string(operation)+"/"+failure.name, func(t *testing.T) {
				fixture := newServiceReleaseFixture(t, operation)
				fixture.providerError = failure.err
				value, err := serviceReleaseInvoke(t, fixture.service(), operation, newServiceReleaseRequest())
				requireServiceReleaseEmpty(t, value, err, failure.kind, "keyservice_provider")
				fixture.requireCalls(serviceReleaseCalls{active: 1, authorize: 1, provider: 1})
				fixture.requireCleared()
			})
		}
	}
}

func TestServiceReleaseContractAgeMaterialMustBeNonemptyAndBounded(t *testing.T) {
	for _, material := range [][]byte{nil, bytes.Repeat([]byte{'s'}, maxKeyMaterial+1)} {
		t.Run("material", func(t *testing.T) {
			fixture := newServiceReleaseFixture(t, OperationRetrieveAge)
			fixture.material = material
			value, err := serviceReleaseInvoke(t, fixture.service(), OperationRetrieveAge, newServiceReleaseRequest())
			requireServiceReleaseEmpty(t, value, err, fault.Trust, "keyservice_age_private_key")
			fixture.requireCalls(serviceReleaseCalls{active: 1, authorize: 1, provider: 1})
			fixture.requireCleared()
		})
	}
}

func TestServiceReleaseContractFreshLookupErrorWithholdsProviderResult(t *testing.T) {
	for _, operation := range []Operation{OperationWrap, OperationUnwrap, OperationRetrieveAge} {
		t.Run(string(operation), func(t *testing.T) {
			fixture := newServiceReleaseFixture(t, operation)
			fixture.lookupErrorAt = 2
			value, err := serviceReleaseInvoke(t, fixture.service(), operation, newServiceReleaseRequest())
			requireServiceReleaseEmpty(t, value, err, fault.Apply, "keyservice_provider")
			fixture.requireCalls(serviceReleaseCalls{active: 2, authorize: 1, provider: 1})
			fixture.requireCleared()
		})
	}
}

func TestServiceReleaseContractFreshAuthorizationDenialWithholdsProviderResult(t *testing.T) {
	for _, operation := range []Operation{OperationWrap, OperationUnwrap, OperationRetrieveAge} {
		t.Run(string(operation), func(t *testing.T) {
			fixture := newServiceReleaseFixture(t, operation)
			fixture.authorizationErrorAt = 2
			value, err := serviceReleaseInvoke(t, fixture.service(), operation, newServiceReleaseRequest())
			requireServiceReleaseEmpty(t, value, err, fault.Expired, "keyservice_authorization")
			fixture.requireCalls(serviceReleaseCalls{active: 2, authorize: 2, provider: 1})
			fixture.requireCleared()
		})
	}
}

func TestServiceReleaseContractSuccessReturnsOnlyAnIndependentAuthorizedCopy(t *testing.T) {
	for _, operation := range []Operation{OperationWrap, OperationUnwrap, OperationRetrieveAge} {
		t.Run(string(operation), func(t *testing.T) {
			fixture := newServiceReleaseFixture(t, operation)
			request := newServiceReleaseRequest()
			before := bytes.Clone(request.material)
			value, err := serviceReleaseInvoke(t, fixture.service(), operation, request)
			if err != nil || string(value) != "SYNTHETIC_PROVIDER_MATERIAL" || !bytes.Equal(before, request.material) {
				t.Fatal("authorized result lost its exact value or changed caller input")
			}
			fixture.requireCalls(serviceReleaseCalls{active: 2, authorize: 2, provider: 1})
			fixture.requireCleared()
			value[0] = 'X'
			fixture.requireCleared()
		})
	}
}

type serviceReleaseRequest struct {
	binding  ProfileBinding
	artifact Artifact
	material []byte
}

func newServiceReleaseRequest() serviceReleaseRequest {
	return serviceReleaseRequest{binding: fixtureBinding(), artifact: fixtureArtifact(), material: []byte("SYNTHETIC_CALLER_INPUT")}
}

func serviceReleaseInvoke(t *testing.T, service *Service, operation Operation, request serviceReleaseRequest) ([]byte, error) {
	t.Helper()
	before := bytes.Clone(request.material)
	var value []byte
	var err error
	switch operation {
	case OperationWrap:
		value, err = service.Wrap(context.Background(), request.binding, request.artifact, request.material)
	case OperationUnwrap:
		value, err = service.Unwrap(context.Background(), request.binding, request.artifact, request.material)
	case OperationRetrieveAge:
		value, err = service.RetrieveAgePrivateKey(context.Background(), request.binding, request.artifact)
	default:
		t.Fatal("unsupported synthetic service operation")
	}
	if !bytes.Equal(before, request.material) {
		t.Fatal("service changed caller-owned material")
	}
	return value, err
}

type serviceReleaseCalls struct {
	active    int
	authorize int
	provider  int
}

type serviceReleaseFixture struct {
	t                    *testing.T
	operation            Operation
	enrollment           Enrollment
	found                bool
	material             []byte
	requestMaterial      []byte
	calls                serviceReleaseCalls
	providerError        error
	lookupErrorAt        int
	authorizationErrorAt int
}

func newServiceReleaseFixture(t *testing.T, operation Operation) *serviceReleaseFixture {
	t.Helper()
	return &serviceReleaseFixture{
		t: t, operation: operation, enrollment: fixtureEnrollment(), found: true,
		material: []byte("SYNTHETIC_PROVIDER_MATERIAL"),
	}
}

func (f *serviceReleaseFixture) service() *Service {
	f.t.Helper()
	service, err := NewService(Dependencies{Authorizer: f, Store: f, Provisioner: f, KMS: f, Age: f})
	if err != nil {
		f.t.Fatal("explicit synthetic service dependencies were rejected")
	}
	return service
}

func (f *serviceReleaseFixture) Active(_ context.Context, binding ProfileBinding) (Enrollment, bool, error) {
	f.t.Helper()
	f.calls.active++
	if binding != fixtureBinding() {
		f.t.Fatal("lookup received an unbound profile")
	}
	if f.calls.active == f.lookupErrorAt {
		return f.enrollment, true, errors.New("SYNTHETIC_PRIVATE_LOOKUP_DETAIL")
	}
	return f.enrollment, f.found, nil
}

func (f *serviceReleaseFixture) Authorize(_ context.Context, binding ProfileBinding, artifact Artifact, operation Operation, enrollment Enrollment) error {
	f.t.Helper()
	f.calls.authorize++
	if binding != fixtureBinding() || artifact != fixtureArtifact() || operation != f.operation || enrollment != f.enrollment {
		f.t.Fatal("authorization lost its exact operation, artifact or enrollment")
	}
	if f.calls.authorize == f.authorizationErrorAt {
		return fault.New(fault.Expired, "SYNTHETIC_PRIVATE_AUTHORIZATION_DETAIL")
	}
	return nil
}

func (f *serviceReleaseFixture) Wrap(_ context.Context, request WrapRequest) (WrapResult, error) {
	f.t.Helper()
	if f.operation != OperationWrap {
		f.t.Fatal("operation selected Wrap instead of its enrolled primitive")
	}
	f.recordProvider(request.Association, request.Context, request.Plaintext)
	return WrapResult{Ciphertext: f.material, Receipt: receipt(request.Association, request.Context)}, f.providerError
}

func (f *serviceReleaseFixture) Unwrap(_ context.Context, request UnwrapRequest) (UnwrapResult, error) {
	f.t.Helper()
	if f.operation != OperationUnwrap {
		f.t.Fatal("operation selected Unwrap instead of its enrolled primitive")
	}
	f.recordProvider(request.Association, request.Context, request.Ciphertext)
	return UnwrapResult{Plaintext: f.material, Receipt: receipt(request.Association, request.Context)}, f.providerError
}

func (f *serviceReleaseFixture) RetrievePrivateKey(_ context.Context, request AgePrivateKeyRequest) ([]byte, error) {
	f.t.Helper()
	f.calls.provider++
	if f.operation != OperationRetrieveAge || request.Association != f.enrollment.Age || request.Context != associationContext(f.enrollment, fixtureArtifact()) {
		f.t.Fatal("age retrieval lost its exact association or artifact context")
	}
	return f.material, f.providerError
}

func (f *serviceReleaseFixture) recordProvider(association KMSAssociation, binding AssociationContext, material []byte) {
	f.t.Helper()
	f.calls.provider++
	if association != f.enrollment.KMS || binding != associationContext(f.enrollment, fixtureArtifact()) || string(material) != "SYNTHETIC_CALLER_INPUT" {
		f.t.Fatal("provider received an unbound request or changed caller material")
	}
	f.requestMaterial = material
}

func (f *serviceReleaseFixture) Provision(context.Context, EnrollmentRequest) (EnrollmentResult, error) {
	f.t.Helper()
	f.t.Fatal("release operation attempted enrollment provisioning")
	return EnrollmentResult{}, errors.New("unexpected synthetic provisioning")
}

func (f *serviceReleaseFixture) Enable(context.Context, EnrollmentRequest, Enrollment) (EnrollmentDisposition, error) {
	f.t.Helper()
	f.t.Fatal("release operation attempted enrollment mutation")
	return EnrollmentReconcile, errors.New("unexpected synthetic mutation")
}

func (f *serviceReleaseFixture) requireCalls(want serviceReleaseCalls) {
	f.t.Helper()
	if f.calls != want {
		f.t.Fatal("service crossed an unexpected release boundary")
	}
}

func (f *serviceReleaseFixture) requireCleared() {
	f.t.Helper()
	if !zeroed(f.material) || !zeroed(f.requestMaterial) {
		f.t.Fatal("service retained owned provider or request material")
	}
}

func requireServiceReleaseEmpty(t *testing.T, value []byte, err error, kind fault.Kind, operation string) {
	t.Helper()
	var safe *fault.Error
	if !errors.As(err, &safe) || safe.Kind != kind || safe.Operation != operation {
		t.Fatal("service refusal lost its exact safe category or operation")
	}
	if value != nil || errors.Unwrap(err) != nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE") {
		t.Fatal("service refusal exposed partial material or private diagnostics")
	}
}
