package keyservice

import (
	"context"
	"errors"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
)

func TestKeyServiceBoundaryContractRequiresEveryExplicitCollaborator(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		remove func(*Dependencies)
	}{
		{"authorizer", func(d *Dependencies) { d.Authorizer = nil }},
		{"store", func(d *Dependencies) { d.Store = nil }},
		{"provisioner", func(d *Dependencies) { d.Provisioner = nil }},
		{"KMS", func(d *Dependencies) { d.KMS = nil }},
		{"age", func(d *Dependencies) { d.Age = nil }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fake := &keyServiceBoundaryFixture{t: t}
			dependencies := fake.dependencies()
			scenario.remove(&dependencies)
			service, err := NewService(dependencies)
			assertKeyServiceBoundaryFault(t, err, fault.Invalid, "keyservice_dependencies")
			if service != nil || fake.calls != (keyServiceBoundaryCalls{}) {
				t.Fatal("missing collaborator produced a service or invoked a dependency")
			}
		})
	}
}

type keyServiceBoundaryAdmissionCase struct {
	name, contextMode string
	mutate            func(*EnrollmentRequest)
	denyAt            int
	kind              fault.Kind
	operation         string
	calls             keyServiceBoundaryCalls
}

func TestKeyServiceBoundaryContractEnrollmentAdmissionBeforeCollaborators(t *testing.T) {
	for _, scenario := range []keyServiceBoundaryAdmissionCase{
		{name: "nil context", contextMode: "nil", kind: fault.Denied, operation: "keyservice_cancelled"},
		{name: "cancelled context", contextMode: "cancelled", kind: fault.Denied, operation: "keyservice_cancelled"},
		{name: "binding", mutate: func(r *EnrollmentRequest) { r.Binding.Tenant = "" }, kind: fault.Invalid, operation: "keyservice_enrollment"},
		{name: "operation identity", mutate: func(r *EnrollmentRequest) { r.OperationID = "" }, kind: fault.Invalid, operation: "keyservice_enrollment"},
		{name: "expected generation", mutate: func(r *EnrollmentRequest) { r.ExpectedGeneration = "invalid\ngeneration" }, kind: fault.Invalid, operation: "keyservice_enrollment"},
		{name: "typed authorization refusal", denyAt: 1, kind: fault.Expired, operation: "keyservice_authorization", calls: keyServiceBoundaryCalls{authorize: 1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			assertKeyServiceBoundaryAdmission(t, scenario)
		})
	}
}

func assertKeyServiceBoundaryAdmission(t *testing.T, scenario keyServiceBoundaryAdmissionCase) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := &keyServiceBoundaryFixture{t: t, denyAt: scenario.denyAt}
	service := fake.service()
	request := fixtureRequest()
	if scenario.mutate != nil {
		scenario.mutate(&request)
	}
	result, err := service.Enroll(keyServiceBoundaryContext(ctx, cancel, scenario.contextMode), request)
	assertKeyServiceBoundaryFault(t, err, scenario.kind, scenario.operation)
	if result != (EnrollmentResult{}) || fake.calls != scenario.calls {
		t.Fatal("enrollment admission returned partial metadata or reached a later collaborator")
	}
}

func keyServiceBoundaryContext(ctx context.Context, cancel context.CancelFunc, mode string) context.Context {
	switch mode {
	case "nil":
		return nil
	case "cancelled":
		cancel()
	}
	return ctx
}

type keyServiceBoundaryStageCase struct {
	name      string
	configure func(*keyServiceBoundaryFixture, context.CancelFunc)
	kind      fault.Kind
	operation string
	calls     keyServiceBoundaryCalls
}

func TestKeyServiceBoundaryContractProvisionRefusalDoesNotEnable(t *testing.T) {
	for _, scenario := range []keyServiceBoundaryStageCase{
		{name: "metadata plus provider error", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.provisionError = fault.New(fault.Unavailable, "SYNTHETIC_PRIVATE_PROVISION_DETAIL")
		}, kind: fault.Unavailable, operation: "keyservice_provider", calls: keyServiceBoundaryCalls{authorize: 1, provision: 1}},
		{name: "cancel during provision", configure: func(f *keyServiceBoundaryFixture, cancel context.CancelFunc) {
			f.cancelProvision = cancel
		}, kind: fault.Denied, operation: "keyservice_cancelled", calls: keyServiceBoundaryCalls{authorize: 1, provision: 1}},
		{name: "second authorization refused", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.denyAt = 2
		}, kind: fault.Expired, operation: "keyservice_authorization", calls: keyServiceBoundaryCalls{authorize: 2, provision: 1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			assertKeyServiceBoundaryReconcile(t, scenario)
		})
	}
}

func TestKeyServiceBoundaryContractEnableAcknowledgementNeedsConfirmation(t *testing.T) {
	for _, scenario := range []keyServiceBoundaryStageCase{
		{name: "cancel during acknowledgement", configure: func(f *keyServiceBoundaryFixture, cancel context.CancelFunc) {
			f.cancelEnable = cancel
		}, kind: fault.Denied, operation: "keyservice_cancelled", calls: keyServiceBoundaryCalls{authorize: 2, provision: 1, enable: 1}},
		{name: "confirmation read error", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.activeError = fault.New(fault.Unavailable, "SYNTHETIC_PRIVATE_LOOKUP_DETAIL")
		}, kind: fault.Unavailable, operation: "keyservice_provider", calls: keyServiceBoundaryCalls{authorize: 2, provision: 1, enable: 1, active: 1}},
		{name: "confirmation missing", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.activeMissing = true
		}, kind: fault.Denied, operation: "keyservice_enrollment_confirmation", calls: keyServiceBoundaryCalls{authorize: 2, provision: 1, enable: 1, active: 1}},
		{name: "confirmation changed", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.activeChanged = true
		}, kind: fault.Denied, operation: "keyservice_enrollment_confirmation", calls: keyServiceBoundaryCalls{authorize: 2, provision: 1, enable: 1, active: 1}},
		{name: "final authorization refused", configure: func(f *keyServiceBoundaryFixture, _ context.CancelFunc) {
			f.denyAt = 3
		}, kind: fault.Expired, operation: "keyservice_authorization", calls: keyServiceBoundaryCalls{authorize: 3, provision: 1, enable: 1, active: 1}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			assertKeyServiceBoundaryReconcile(t, scenario)
		})
	}
}

func assertKeyServiceBoundaryReconcile(t *testing.T, scenario keyServiceBoundaryStageCase) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := &keyServiceBoundaryFixture{t: t}
	scenario.configure(fake, cancel)
	result, err := fake.service().Enroll(ctx, fixtureRequest())
	assertKeyServiceBoundaryFault(t, err, scenario.kind, scenario.operation)
	if result != (EnrollmentResult{Disposition: EnrollmentReconcile}) || fake.calls != scenario.calls {
		t.Fatal("uncertain activation exposed enrollment metadata, reported enabled or reached a later stage")
	}
}

func assertKeyServiceBoundaryFault(t *testing.T, err error, kind fault.Kind, operation string) {
	t.Helper()
	actual, ok := err.(*fault.Error)
	if !ok {
		t.Fatal("expected a direct sanitized fault")
	}
	if *actual != (fault.Error{Kind: kind, Operation: operation}) {
		t.Fatal("refusal changed category or exposed collaborator diagnostics")
	}
	if err.Error() != string(kind)+": "+operation || errors.Unwrap(err) != nil {
		t.Fatal("refusal retained a raw diagnostic or an upstream cause")
	}
}

type keyServiceBoundaryCalls struct {
	authorize, provision, enable, active int
	wrap, unwrap, age                    int
}

// The fixture exchanges only fabricated public/opaque association metadata.
// Enable merely acknowledges in memory: it is not durable enrollment, principal
// authentication, an audit grant, provider provisioning or deployment evidence.
type keyServiceBoundaryFixture struct {
	t               *testing.T
	calls           keyServiceBoundaryCalls
	denyAt          int
	provisionError  error
	activeError     error
	activeMissing   bool
	activeChanged   bool
	cancelProvision context.CancelFunc
	cancelEnable    context.CancelFunc
}

func (f *keyServiceBoundaryFixture) dependencies() Dependencies {
	return Dependencies{Authorizer: f, Store: f, Provisioner: f, KMS: f, Age: f}
}

func (f *keyServiceBoundaryFixture) service() *Service {
	f.t.Helper()
	service, err := NewService(f.dependencies())
	if err != nil {
		f.t.Fatal(err)
	}
	return service
}

func (f *keyServiceBoundaryFixture) Authorize(_ context.Context, binding ProfileBinding, artifact Artifact, operation Operation, enrollment Enrollment) error {
	f.calls.authorize++
	if binding != fixtureBinding() || artifact != (Artifact{}) || operation != OperationEnroll {
		f.t.Fatal("authorization received unexpected synthetic scope")
	}
	f.assertAuthorizationEnrollment(enrollment)
	if f.calls.authorize == f.denyAt {
		return fault.New(fault.Expired, "SYNTHETIC_PRIVATE_AUTHORIZATION_DETAIL")
	}
	return nil
}

func (f *keyServiceBoundaryFixture) assertAuthorizationEnrollment(enrollment Enrollment) {
	f.t.Helper()
	want := Enrollment{}
	if f.calls.authorize > 1 {
		want = fixtureEnrollment()
	}
	if enrollment != want {
		f.t.Fatal("authorization did not receive the exact expected association metadata")
	}
}

func (f *keyServiceBoundaryFixture) Provision(_ context.Context, request EnrollmentRequest) (EnrollmentResult, error) {
	f.calls.provision++
	if request != fixtureRequest() {
		f.t.Fatal("provisioning received an unexpected synthetic request")
	}
	if f.cancelProvision != nil {
		f.cancelProvision()
	}
	return EnrollmentResult{Disposition: EnrollmentEnabled, Enrollment: fixtureEnrollment()}, f.provisionError
}

func (f *keyServiceBoundaryFixture) Enable(_ context.Context, request EnrollmentRequest, enrollment Enrollment) (EnrollmentDisposition, error) {
	f.calls.enable++
	if request != fixtureRequest() || enrollment != fixtureEnrollment() {
		f.t.Fatal("enable acknowledgement received unexpected association metadata")
	}
	if f.cancelEnable != nil {
		f.cancelEnable()
	}
	return EnrollmentEnabled, nil
}

func (f *keyServiceBoundaryFixture) Active(_ context.Context, binding ProfileBinding) (Enrollment, bool, error) {
	f.calls.active++
	if binding != fixtureBinding() {
		f.t.Fatal("confirmation lookup received an unexpected synthetic binding")
	}
	enrollment := fixtureEnrollment()
	if f.activeChanged {
		enrollment.Generation = "synthetic-next-generation"
	}
	return enrollment, !f.activeMissing, f.activeError
}

func (f *keyServiceBoundaryFixture) Wrap(context.Context, WrapRequest) (WrapResult, error) {
	f.calls.wrap++
	f.t.Error("enrollment unexpectedly reached the forbidden KMS fixture")
	return WrapResult{}, fault.New(fault.Unsupported, "synthetic_forbidden_adapter")
}

func (f *keyServiceBoundaryFixture) Unwrap(context.Context, UnwrapRequest) (UnwrapResult, error) {
	f.calls.unwrap++
	f.t.Error("enrollment unexpectedly reached the forbidden KMS fixture")
	return UnwrapResult{}, fault.New(fault.Unsupported, "synthetic_forbidden_adapter")
}

func (f *keyServiceBoundaryFixture) RetrievePrivateKey(context.Context, AgePrivateKeyRequest) ([]byte, error) {
	f.calls.age++
	f.t.Error("enrollment unexpectedly reached the forbidden age fixture")
	return nil, fault.New(fault.Unsupported, "synthetic_forbidden_adapter")
}
