package policy

import (
	"reflect"
	"testing"

	"github.com/maldous/CircularQuay/internal/model"
)

func TestAuthorizationExactIdentityOperationAndDisclosure(t *testing.T) {
	policy := validPolicy()
	policy.Grants = []model.Grant{
		{Subject: "https://issuer.example.test|alice", Client: "desktop", Workload: "demo", Operations: []string{"list", "status", "read"}, Keys: []string{"DEMO_MESSAGE"}},
		{Subject: "local:operator", Workload: "demo", Operations: []string{"run"}},
	}
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		subject, client, workload, operation, key string
		want                                      bool
	}{
		{"https://issuer.example.test|alice", "desktop", "demo", "read", "DEMO_MESSAGE", true},
		{"https://issuer.example.test|bob", "desktop", "demo", "read", "DEMO_MESSAGE", false},
		{"https://issuer.example.test|alice", "agent", "demo", "read", "DEMO_MESSAGE", false},
		{"https://issuer.example.test|alice", "desktop", "other", "read", "DEMO_MESSAGE", false},
		{"https://issuer.example.test|alice", "desktop", "demo", "read", "", false},
		{"https://issuer.example.test|alice", "desktop", "demo", "read", "OTHER_KEY", false},
		{"https://issuer.example.test|alice", "desktop", "demo", "run", "", false},
		{"local:operator", "", "demo", "run", "", true},
		{"local:operator", "desktop", "demo", "run", "", false},
		{"local:operator", "", "demo", "read", "DEMO_MESSAGE", false},
		{"*", "", "demo", "run", "", false},
		{"local:operator", "", "demo", "rotate", "", false},
	}
	for _, test := range tests {
		if got := Authorize(policy, test.subject, test.client, test.workload, test.operation, test.key); got != test.want {
			t.Errorf("authorization mismatch for operation %q", test.operation)
		}
	}
	if got := VisibleWorkloads(policy, "https://issuer.example.test|alice", "desktop"); !reflect.DeepEqual(got, []string{"demo"}) {
		t.Fatal("discovery did not honor grant")
	}
	if got := VisibleWorkloads(policy, "local:operator", ""); len(got) != 0 {
		t.Fatal("invocation grant leaked discovery")
	}
}

func TestGrantValidation(t *testing.T) {
	for _, grant := range []model.Grant{
		{Subject: "*", Workload: "demo", Operations: []string{"read"}, Keys: []string{"DEMO_MESSAGE"}},
		{Subject: "alice", Client: "*", Workload: "demo", Operations: []string{"status"}},
		{Subject: "alice", Workload: "missing", Operations: []string{"status"}},
		{Subject: "alice", Workload: "demo"},
		{Subject: "alice", Workload: "demo", Operations: []string{"root"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"status", "status"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"read"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"read"}, Keys: []string{"MISSING"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"read"}, Keys: []string{"DEMO_MESSAGE", "DEMO_MESSAGE"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"run"}, Keys: []string{"DEMO_MESSAGE"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"cancel"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"get_result"}},
		{Subject: "alice", Workload: "demo", Operations: []string{"read_secret"}, Keys: []string{"DEMO_MESSAGE"}},
	} {
		policy := validPolicy()
		policy.Grants = []model.Grant{grant}
		if Validate(policy) == nil {
			t.Fatal("unsafe grant was accepted")
		}
	}
}

func TestMixedRunAndReadGrantScopesOnlyReadsByKey(t *testing.T) {
	policy := validPolicy()
	workload := policy.Workloads["demo"]
	workload.Bindings = append(workload.Bindings, model.Binding{Name: "OTHER_MESSAGE", Source: "fixture", Field: "other", AllowOffline: true})
	policy.Workloads["demo"] = workload
	policy.Grants = []model.Grant{{Subject: "alice", Workload: "demo", Operations: []string{"run", "read"}, Keys: []string{"DEMO_MESSAGE"}}}
	if err := Validate(policy); err != nil {
		t.Fatal(err)
	}
	if !Authorize(policy, "alice", "", "demo", "run", "") {
		t.Fatal("complete workload run grant was denied")
	}
	if !Authorize(policy, "alice", "", "demo", "read", "DEMO_MESSAGE") {
		t.Fatal("declared read key was denied")
	}
	if Authorize(policy, "alice", "", "demo", "read", "OTHER_MESSAGE") {
		t.Fatal("read key grant widened to another binding")
	}
}

func FuzzAuthorizationDoesNotBroadenClient(f *testing.F) {
	f.Add("alice", "desktop", "DEMO_MESSAGE")
	f.Add("bob", "agent", "OTHER_KEY")
	f.Fuzz(func(t *testing.T, subject, client, key string) {
		policy := validPolicy()
		policy.Grants = []model.Grant{{Subject: "alice", Client: "desktop", Workload: "demo", Operations: []string{"read"}, Keys: []string{"DEMO_MESSAGE"}}}
		if Authorize(policy, subject, client, "demo", "read", key) && (subject != "alice" || client != "desktop" || key != "DEMO_MESSAGE") {
			t.Fatal("authorization widened")
		}
	})
}
