package source

import (
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/maldous/CircularQuay/internal/model"
)

// The pinned workload constructor registers its assertion callback without
// invoking it. This control exercises the real default factory and tenant
// wrapper only, never GetToken, provider access or operational credentials.
func TestAzureDefaultConstructorContractWorkloadPreservesOwnedAssertion(t *testing.T) {
	path := privateCloudTokenFile(t)
	source := model.Source{
		Account:   "synthetic-tenant",
		Identity:  "workload",
		Profile:   "synthetic-client",
		TokenFile: path,
	}
	credential, err := azureCredential(source)
	if err != nil {
		t.Fatal("explicit synthetic workload construction failed", err)
	}
	bound, ok := credential.(tenantCredential)
	if !ok || bound.tenant != source.Account {
		t.Fatal("default workload construction lost its tenant binding")
	}
	workload, ok := bound.credential.(*azidentity.WorkloadIdentityCredential)
	if !ok || workload == nil {
		t.Fatal("default workload construction did not retain the pinned SDK credential")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("owned assertion could not be checked after construction", err)
	}
	if string(contents) != "synthetic-token" {
		t.Fatal("default workload construction changed the owned assertion")
	}
}
