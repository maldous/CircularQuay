package source

import (
	"context"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
)

// These contracts enter the real wrappers only with identity inputs that must
// refuse before credential construction, STS or metadata access. They do not
// exercise a successful identity path or construct a provider SDK client.
func TestCloudAdmissionContractAWSSecretRejectsMalformedECSLocator(t *testing.T) {
	cloudAdmissionECSCases(t, "aws-sm", syntheticAWSSecretARN, ReadFunc(readAWSSecret))
}

func TestCloudAdmissionContractAWSParameterRejectsMalformedECSLocator(t *testing.T) {
	cloudAdmissionECSCases(t, "aws-ssm", syntheticAWSSSMARN, ReadFunc(readAWSParameter))
}

func cloudAdmissionECSCases(t *testing.T, sourceType, resource string, direct ReadFunc) {
	t.Helper()
	for _, tc := range []struct{ name, locator string }{
		{"relative", "SYNTHETIC_PRIVATE_LOCATOR"},
		{"query", "/v2/credentials/synthetic?SYNTHETIC_PRIVATE_LOCATOR"},
		{"fragment", "/v2/credentials/synthetic#SYNTHETIC_PRIVATE_LOCATOR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Only this controlled variable is read on the rejected ECS path.
			// Setenv restores its prior state when this subtest completes.
			t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", tc.locator)
			source := model.Source{
				Type:          sourceType,
				Account:       "123456789012",
				Region:        "ap-southeast-2",
				Identity:      "ecs",
				Resources:     []string{resource},
				AuditVerified: true,
			}
			binding := model.Binding{Resource: resource, Version: "3", Field: "selected"}
			cloudAdmissionWrappersRefuse(t, source, binding, direct, fault.Invalid, "ecs_identity")
		})
	}
}

func TestCloudAdmissionContractAzureRejectsMetadataOverrideBeforeProvider(t *testing.T) {
	// The first checked override is controlled and nonempty. The guard returns
	// before inspecting any other environment key or creating managed identity.
	t.Setenv("IDENTITY_ENDPOINT", "https://SYNTHETIC_PRIVATE_OVERRIDE.invalid/metadata")
	source := model.Source{
		Type:          "azure",
		Endpoint:      syntheticAzureEndpoint,
		Account:       "11111111-1111-4111-8111-111111111111",
		Profile:       "22222222-2222-4222-8222-222222222222",
		Identity:      "managed",
		Resources:     []string{"fixture"},
		AuditVerified: true,
	}
	binding := model.Binding{Resource: "fixture", Version: "expected", Field: "selected"}
	cloudAdmissionWrappersRefuse(t, source, binding, ReadFunc(readAzure), fault.Trust, "azure_metadata_override")
}

func cloudAdmissionWrappersRefuse(t *testing.T, source model.Source, binding model.Binding, direct ReadFunc, kind fault.Kind, operation string) {
	t.Helper()
	value, err := direct.Read(context.Background(), source, binding)
	requireCloudValueRefusal(t, value, err, kind, operation)

	// The declared resource and version capability admit the same request to
	// the real registered adapter; neither the Registry nor the wrapper may
	// replace its sanitized refusal or retain data or provenance metadata.
	value, err = NewRegistry().Read(context.Background(), source, binding)
	requireCloudValueRefusal(t, value, err, kind, operation)
}
