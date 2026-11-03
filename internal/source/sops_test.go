package source

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// This native helper tests the subprocess contract, not cryptography. Real SOPS
// round trips belong to the separately recorded installed-tool integration lane.
func TestMain(m *testing.M) {
	if providerTestMain() {
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "decrypt" || os.Args[1] == "encrypt") {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, platform.MaxDocumentSize+1))
		if err != nil {
			os.Exit(90)
		}
		var value map[string]json.RawMessage
		if json.Unmarshal(data, &value) != nil {
			os.Exit(91)
		}
		if string(value["test_operation"]) == `"failure"` {
			os.Exit(92)
		}
		if string(value["test_operation"]) == `"overflow"` {
			_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), platform.MaxDocumentSize+10))
			os.Exit(0)
		}
		if os.Getenv("HOME") == "" || os.Getenv("SOPS_AGE_KEY") != "" {
			os.Exit(93)
		}
		if os.Args[1] == "decrypt" {
			if value["test_data"] == nil {
				os.Exit(94)
			}
			_, _ = os.Stdout.Write(value["test_data"])
		} else {
			_, _ = os.Stdout.Write([]byte(`{"sops":{"mac":"synthetic-mac","age":[{"recipient":"age1synthetic"}]},"test_data":` + string(data) + `}`))
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func sopsFixture(t *testing.T, metadata string) (model.Source, string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "fixture.json")
	data := []byte(`{"sops":` + metadata + `,"test_data":{"KEY":"synthetic-only"}}`)
	if err := platform.WriteProtected(file, data); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return model.Source{Type: "sops", Path: file, SOPS: exe}, file
}

func TestSOPSReaderIsolatedAge(t *testing.T) {
	t.Setenv("SOPS_AGE_KEY", "must-not-reach-child")
	config, _ := sopsFixture(t, `{"mac":"fake","age":[{"recipient":"age1synthetic"}]}`)
	value, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
	if err != nil || value.Data != "synthetic-only" {
		t.Fatal("isolated decrypt failed")
	}
	_, err = (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "MISSING"})
	if !fault.Is(err, fault.Missing) {
		t.Fatal("missing field accepted")
	}
}

func TestSOPSRejectsMetadataBeforeToken(t *testing.T) {
	config, _ := sopsFixture(t, `{"mac":"fake","hc_vault":[{"vault_address":"https://evil.invalid","engine_path":"transit","key_name":"other"}]}`)
	config.TokenFile = filepath.Join(t.TempDir(), "nonexistent-token")
	config.VaultAllowlist = []string{"https://approved.invalid"}
	config.VaultKeys = []string{"transit/workload"}
	_, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
	if !fault.Is(err, fault.Trust) {
		t.Fatal("hostile metadata reached identity access")
	}
}

func TestSOPSRejectsRecipientDowngrade(t *testing.T) {
	cases := []struct {
		metadata string
		transit  bool
		kind     fault.Kind
	}{
		{`{"mac":"fake","age":[{}],"hc_vault":[{"vault_address":"https://approved.invalid","engine_path":"transit","key_name":"workload"}]}`, true, fault.Trust},
		{`{"mac":"fake","hc_vault":[{}]}`, false, fault.Trust},
		{`{"mac":"fake","age":[{}],"kms":[{}]}`, false, fault.Unsupported},
		{`{"mac":"fake","key_groups":[{}]}`, true, fault.Unsupported},
		{`{"age":[{}]}`, false, fault.Invalid},
	}
	for _, test := range cases {
		config, _ := sopsFixture(t, test.metadata)
		config.VaultAllowlist = []string{"https://approved.invalid"}
		config.VaultKeys = []string{"transit/workload"}
		if test.transit {
			config.Identity = "transit"
		}
		_, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
		if !fault.Is(err, test.kind) {
			t.Fatal("recipient policy not enforced")
		}
	}
}

func TestSOPSTransitAndTokenScope(t *testing.T) {
	config, _ := sopsFixture(t, `{"mac":"fake","hc_vault":[{"vault_address":"https://approved.invalid/","engine_path":"transit","key_name":"workload"}]}`)
	config.VaultAllowlist = []string{"https://approved.invalid"}
	config.VaultKeys = []string{"transit/workload"}
	config.AuditVerified = true
	config.TokenFile = filepath.Join(t.TempDir(), "token")
	if err := platform.WriteProtected(config.TokenFile, []byte("synthetic-token\n")); err != nil {
		t.Fatal(err)
	}
	value, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
	if err != nil || !strings.HasPrefix(value.Version, "sha256:") || !value.AuditVerified {
		t.Fatal("declared Transit source failed")
	}
}

func TestSOPSOutputFailureAndBounds(t *testing.T) {
	for _, test := range []struct {
		op   string
		kind fault.Kind
	}{{"failure", fault.Denied}, {"overflow", fault.Invalid}} {
		config, file := sopsFixture(t, `{"mac":"fake","age":[{}]}`)
		body := `{"sops":{"mac":"fake","age":[{}]},"test_operation":"` + test.op + `"}`
		if err := platform.WriteProtected(file, []byte(body)); err != nil {
			t.Fatal(err)
		}
		_, err := (SOPSReader{}).Read(context.Background(), config, model.Binding{Name: "KEY"})
		if !fault.Is(err, test.kind) {
			t.Fatal("subprocess failure classification")
		}
	}
}

func TestSnapshotMockRoundTrip(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	config := model.Snapshot{Path: filepath.Join(t.TempDir(), "snapshot.json"), Recipients: []string{"age1synthetic"}, SOPS: exe}
	want := map[string]string{"KEY": "synthetic\nUnicode-✓-$()"}
	if err := EncryptSnapshot(context.Background(), config, want); err != nil {
		t.Fatal(err)
	}
	got, err := DecryptSnapshot(context.Background(), config)
	if err != nil || got["KEY"] != want["KEY"] {
		t.Fatal("snapshot contract roundtrip failed")
	}
	config.Recipients = nil
	if err := EncryptSnapshot(context.Background(), config, want); !fault.Is(err, fault.Invalid) {
		t.Fatal("empty recipients accepted")
	}
	config.Recipients = []string{"age1bad,recipient"}
	if err := EncryptSnapshot(context.Background(), config, want); !fault.Is(err, fault.Invalid) {
		t.Fatal("invalid recipient accepted")
	}
}

func TestSOPSEnvironmentValidation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "secret")
	if err := platform.WriteProtected(file, []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	env, err := sopsEnvironment(t.TempDir(), file, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(env, "\n"), "SOPS_AGE_KEY_FILE=") {
		t.Fatal("explicit identity omitted")
	}
	if _, err := sopsEnvironment(t.TempDir(), "", file, nil); !fault.Is(err, fault.Trust) {
		t.Fatal("unscoped token accepted")
	}
	if _, err := sopsEnvironment(t.TempDir(), "", file, []string{"http://evil.invalid"}); !fault.Is(err, fault.Trust) {
		t.Fatal("untrusted endpoint accepted")
	}
	if err := platform.WriteProtected(file, []byte("one\ntwo")); err != nil {
		t.Fatal(err)
	}
	if _, err := sopsEnvironment(t.TempDir(), "", file, []string{"https://approved.invalid"}); !fault.Is(err, fault.Invalid) {
		t.Fatal("multiline token accepted")
	}
}

func TestDecodeStringsRejectsCoercion(t *testing.T) {
	for _, data := range []string{`null`, `[]`, `{"KEY":1}`, `{"KEY":"bad\u0000value"}`, `{"":"value"}`} {
		if _, err := decodeStrings([]byte(data)); !fault.Is(err, fault.Invalid) {
			t.Fatal("unsupported string map accepted")
		}
	}
}

func TestSOPSExecutableRequired(t *testing.T) {
	_, err := runSOPS(context.Background(), "sops", nil, nil, "", "", nil)
	if !fault.Is(err, fault.Unsupported) {
		t.Fatal("PATH executable accepted")
	}
	_, err = runSOPS(context.Background(), filepath.Join(t.TempDir(), "missing"), nil, nil, "", "", nil)
	if !fault.Is(err, fault.Unsupported) {
		t.Fatal("missing executable accepted")
	}
}

// TestSOPSRealAgeRoundTrip is opt-in because release authenticity must be
// verified before a downloaded helper is executed. It uses only temporary,
// synthetic identities, and never prints the identity or decrypted document.
func TestSOPSRealAgeRoundTrip(t *testing.T) {
	sops, ageKeygen := os.Getenv("CQUAY_TEST_SOPS"), os.Getenv("CQUAY_TEST_AGE_KEYGEN")
	if sops == "" || ageKeygen == "" {
		t.Skip("verified upstream SOPS and age-keygen executables not configured")
	}
	if err := platform.ValidateExecutable(sops); err != nil {
		t.Fatal("untrusted integration SOPS helper")
	}
	if err := platform.ValidateExecutable(ageKeygen); err != nil {
		t.Fatal("untrusted integration age helper")
	}
	dir := t.TempDir()
	identity := filepath.Join(dir, "identity.txt")
	makeIdentity := func(path string) string {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), ageKeygen, "-o", path)
		cmd.Env = []string{"HOME=" + dir}
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
		if err := cmd.Run(); err != nil {
			t.Fatal("synthetic identity creation failed")
		}
		cmd = exec.CommandContext(context.Background(), ageKeygen, "-y", path)
		cmd.Env = []string{"HOME=" + dir}
		cmd.Stderr = io.Discard
		pub, err := cmd.Output()
		if err != nil {
			t.Fatal("synthetic public identity derivation failed")
		}
		return strings.TrimSpace(string(pub))
	}
	recipient := makeIdentity(identity)
	config := model.Snapshot{Path: filepath.Join(dir, "snapshot.json"), SOPS: sops, Recipients: []string{recipient}, IdentityFile: identity}
	want := map[string]string{"KEY": "synthetic-real-crypto-✓-$()\nsecond line", "OTHER": "independently-scoped"}
	if err := EncryptSnapshot(context.Background(), config, want); err != nil {
		t.Fatal("upstream encryption failed")
	}
	ciphertext, err := platform.ReadProtected(config.Path, true)
	if err != nil || bytes.Contains(ciphertext, []byte(want["KEY"])) || bytes.Contains(ciphertext, []byte(want["OTHER"])) {
		t.Fatal("plaintext detected in ciphertext or unreadable encrypted output")
	}
	got, err := DecryptSnapshot(context.Background(), config)
	if err != nil || len(got) != len(want) || got["KEY"] != want["KEY"] || got["OTHER"] != want["OTHER"] {
		t.Fatal("upstream age roundtrip failed")
	}
	reader := model.Source{Type: "sops", Path: config.Path, SOPS: sops, AgeIdentityFile: identity}
	selected, err := (SOPSReader{}).Read(context.Background(), reader, model.Binding{Name: "APP_KEY", Field: "KEY"})
	if err != nil || selected.Data != want["KEY"] || !strings.HasPrefix(selected.Version, "sha256:") {
		t.Fatal("real field selection or ciphertext version failed")
	}
	wrong := filepath.Join(dir, "wrong-identity.txt")
	makeIdentity(wrong)
	config.IdentityFile = wrong
	if _, err := DecryptSnapshot(context.Background(), config); !fault.Is(err, fault.Denied) {
		t.Fatal("unrelated recipient decrypted encrypted snapshot")
	}
	config.IdentityFile = identity
	data := []byte(`{"sops":{"mac":"fake","age":[{}],"hc_vault":[{"vault_address":"https://evil.invalid"}]}}`)
	if err := platform.WriteProtected(config.Path, data); err != nil {
		t.Fatal("write synthetic hostile metadata fixture")
	}
	if _, err := DecryptSnapshot(context.Background(), config); !fault.Is(err, fault.Trust) {
		t.Fatal("offline reader allowed an online key recipient")
	}
}
