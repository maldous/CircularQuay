package source

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// These constructor/selection contracts never retrieve a token. Explicit fake
// factories model SDK construction failure without ambient identity discovery.
func TestAzureCredentialConstructionContractWorkloadAdmissionAndLifetime(t *testing.T) {
	for _, phase := range []string{"file", "authority", "constructor", "success"} {
		t.Run(phase, func(t *testing.T) { assertAzureWorkloadConstruction(t, phase) })
	}
}

func assertAzureWorkloadConstruction(t *testing.T, phase string) {
	t.Helper()
	source := model.Source{Account: "synthetic-tenant", Profile: "synthetic-client", TokenFile: privateCloudTokenFile(t)}
	probe := &azureCredentialConstructionProbe{t: t, phase: phase, source: source}
	credential, err := azureWorkloadCredentialWithOperations(source, probe.validate, probe.getenv, probe.workload)
	probe.assertResult(credential, err, azureCredentialConstructionCalls(phase))
}

type azureCredentialConstructionProbe struct {
	t      *testing.T
	phase  string
	source model.Source
	calls  []string
}

func (p *azureCredentialConstructionProbe) validate(path string) error {
	p.t.Helper()
	p.calls = append(p.calls, "file")
	if path != p.source.TokenFile {
		p.t.Fatal("workload identity selected an unrelated assertion file")
	}
	if p.phase == "file" {
		return fault.New(fault.Trust, "owned assertion refusal")
	}
	return platform.ValidateProtectedPath(path)
}

func (p *azureCredentialConstructionProbe) getenv(name string) string {
	p.t.Helper()
	p.calls = append(p.calls, "authority")
	if name != "AZURE_AUTHORITY_HOST" {
		p.t.Fatal("workload constructor read an undeclared ambient selector")
	}
	if p.phase == "authority" {
		return "https://synthetic-override.example.test"
	}
	return ""
}

func (p *azureCredentialConstructionProbe) workload(options *azidentity.WorkloadIdentityCredentialOptions) (azcore.TokenCredential, error) {
	p.t.Helper()
	p.calls = append(p.calls, "constructor")
	if options.TenantID != p.source.Account || options.ClientID != p.source.Profile || options.TokenFilePath != p.source.TokenFile {
		p.t.Fatal("workload constructor lost explicit tenant, client or file")
	}
	assertAzureCredentialConstructionHTTP(p.t, options.ClientOptions.Transport, 30*time.Second)
	if p.phase == "constructor" {
		return nil, errors.New("PRIVATE_CREDENTIAL_CONSTRUCTOR")
	}
	return p, nil
}

func (p *azureCredentialConstructionProbe) GetToken(context.Context, azpolicy.TokenRequestOptions) (azcore.AccessToken, error) {
	p.t.Error("construction attempted token retrieval")
	return azcore.AccessToken{}, fault.New(fault.Denied, "unreached token")
}

func azureCredentialConstructionCalls(phase string) []string {
	if phase == "file" {
		return []string{"file"}
	}
	if phase == "authority" {
		return []string{"file", "authority"}
	}
	return []string{"file", "authority", "constructor"}
}

func (p *azureCredentialConstructionProbe) assertResult(credential azcore.TokenCredential, err error, calls []string) {
	p.t.Helper()
	if !reflect.DeepEqual(p.calls, calls) {
		p.t.Fatal("identity construction changed its admission order")
	}
	if p.phase == "success" {
		bound, ok := credential.(tenantCredential)
		if err != nil || !ok || bound.tenant != p.source.Account || bound.credential != p {
			p.t.Fatal("credential construction lost its exact tenant binding", err)
		}
		return
	}
	p.assertFailure(credential, err)
}

func (p *azureCredentialConstructionProbe) assertFailure(credential azcore.TokenCredential, err error) {
	p.t.Helper()
	if credential != nil {
		p.t.Fatal("failed constructor returned a partial credential")
	}
	if p.phase == "file" {
		requireProviderInputError(p.t, err, fault.Trust, "owned assertion refusal")
		return
	}
	if p.phase == "authority" {
		requireProviderInputError(p.t, err, fault.Trust, "azure_authority_override")
		return
	}
	requireProviderInputError(p.t, err, fault.Denied, "azure_identity")
}

func TestAzureCredentialConstructionContractManagedMetadataOverrideIsClosed(t *testing.T) {
	for index := range azureCredentialConstructionMetadataNames() {
		t.Run(azureCredentialConstructionMetadataNames()[index], func(t *testing.T) {
			assertAzureManagedOverride(t, index)
		})
	}
}

func azureCredentialConstructionMetadataNames() []string {
	return []string{"IDENTITY_ENDPOINT", "IDENTITY_HEADER", "MSI_ENDPOINT", "MSI_SECRET", "IMDS_ENDPOINT", "AZURE_POD_IDENTITY_AUTHORITY_HOST"}
}

func assertAzureManagedOverride(t *testing.T, selected int) {
	t.Helper()
	names := azureCredentialConstructionMetadataNames()
	var calls []string
	override := func() bool {
		return azureMetadataOverrideWithLookup(func(name string) string {
			calls = append(calls, name)
			if name == names[selected] {
				return "SYNTHETIC_OVERRIDE_SELECTOR"
			}
			return ""
		})
	}
	construct := func(*azidentity.ManagedIdentityCredentialOptions) (azcore.TokenCredential, error) {
		t.Fatal("metadata override reached a credential constructor")
		return nil, nil
	}
	credential, err := azureManagedCredentialWithOperations(model.Source{Account: "synthetic-tenant", Profile: "synthetic-client"}, override, construct)
	requireProviderInputError(t, err, fault.Trust, "azure_metadata_override")
	if credential != nil || !reflect.DeepEqual(calls, names[:selected+1]) {
		t.Fatal("metadata override broadened selectors, continued lookup or returned a credential")
	}
}

func TestAzureCredentialConstructionContractManagedExplicitClientAndConstructorResult(t *testing.T) {
	for _, phase := range []string{"constructor", "success"} {
		t.Run(phase, func(t *testing.T) { assertAzureManagedConstruction(t, phase) })
	}
}

func assertAzureManagedConstruction(t *testing.T, phase string) {
	t.Helper()
	source := model.Source{Account: "synthetic-tenant", Profile: "synthetic-client"}
	probe := &azureCredentialConstructionProbe{t: t, phase: phase, source: source}
	var names []string
	override := func() bool {
		probe.calls = append(probe.calls, "metadata")
		return azureMetadataOverrideWithLookup(func(name string) string { names = append(names, name); return "" })
	}
	credential, err := azureManagedCredentialWithOperations(source, override, probe.managed)
	if !reflect.DeepEqual(names, azureCredentialConstructionMetadataNames()) {
		t.Fatal("managed constructor changed its closed metadata selector list")
	}
	probe.assertResult(credential, err, []string{"metadata", "constructor"})
}

func (p *azureCredentialConstructionProbe) managed(options *azidentity.ManagedIdentityCredentialOptions) (azcore.TokenCredential, error) {
	p.t.Helper()
	p.calls = append(p.calls, "constructor")
	if !reflect.DeepEqual(options.ID, azidentity.ClientID(p.source.Profile)) {
		p.t.Fatal("managed constructor lost explicit client ID")
	}
	assertAzureCredentialConstructionHTTP(p.t, options.ClientOptions.Transport, 5*time.Second)
	if p.phase == "constructor" {
		return nil, errors.New("PRIVATE_CREDENTIAL_CONSTRUCTOR")
	}
	return p, nil
}

func assertAzureCredentialConstructionHTTP(t *testing.T, sender azpolicy.Transporter, timeout time.Duration) {
	t.Helper()
	client, ok := sender.(*http.Client)
	if !ok || client.Timeout != timeout || client.Transport == nil || client.CheckRedirect == nil {
		t.Fatal("credential constructor lost bounded production HTTP options")
	}
	request := httptest.NewRequest(http.MethodGet, "https://unrelated.example.test/token", nil)
	if err := client.CheckRedirect(request, nil); err == nil {
		t.Fatal("credential constructor permitted an authority redirect")
	}
}
