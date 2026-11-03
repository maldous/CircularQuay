package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/platform"
)

// SOPSReader delegates cryptography to a protected upstream SOPS executable.
type SOPSReader struct{}

const (
	sopsInputTypeFlag    = "--input-type"
	sopsOutputTypeFlag   = "--output-type"
	sopsFilenameFlag     = "--filename-override"
	sopsJSONFilename     = "cquay.sops.json"
	sopsJSONDocumentType = "json"
)

// Read resolves one string field, after inspecting online key metadata before a
// token is supplied. Transit sources cannot silently use an offline recipient.
func (SOPSReader) Read(ctx context.Context, config model.Source, binding model.Binding) (model.Value, error) {
	path := sopsPath(config, binding)
	data, err := platform.ReadProtected(path, false)
	if err != nil {
		return model.Value{}, err
	}
	transit := config.Identity == "transit" || config.TokenFile != ""
	if err := validateSOPSMetadata(data, config, transit); err != nil {
		return model.Value{}, err
	}
	version := sopsVersion(data)
	if binding.Version != "" && binding.Version != version {
		return model.Value{}, fault.New(fault.Trust, "SOPS ciphertext version mismatch")
	}
	plaintext, err := runSOPS(ctx, config.SOPS, sopsDecryptArgs(), data, config.AgeIdentityFile, config.TokenFile, config.VaultAllowlist)
	if err != nil {
		return model.Value{}, err
	}
	values, err := decodeStrings(plaintext)
	if err != nil {
		return model.Value{}, err
	}
	field := selectedField(binding)
	value, ok := values[field]
	if !ok {
		return model.Value{}, fault.New(fault.Missing, "SOPS selected field")
	}
	return model.Value{Data: value, Version: version, AuditVerified: config.AuditVerified}, nil
}

func sopsPath(config model.Source, binding model.Binding) string {
	if config.Path != "" {
		return config.Path
	}
	return binding.Resource
}

func sopsVersion(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func selectedField(binding model.Binding) string {
	if binding.Field != "" {
		return binding.Field
	}
	return binding.Name
}

// EncryptSnapshot writes a static value map using only the declared age recipients.
// Eligibility and generation completeness are enforced by the policy/resolver.
func EncryptSnapshot(ctx context.Context, config model.Snapshot, values map[string]string) error {
	if len(config.Recipients) == 0 {
		return fault.New(fault.Invalid, "snapshot recipients required")
	}
	for _, recipient := range config.Recipients {
		if !strings.HasPrefix(recipient, "age1") || strings.ContainsAny(recipient, ",\n\r\x00 ") {
			return fault.New(fault.Invalid, "snapshot age recipient")
		}
	}
	plaintext, err := json.Marshal(values)
	if err != nil {
		return fault.New(fault.Invalid, "snapshot values")
	}
	if _, err = decodeStrings(plaintext); err != nil {
		return err
	}
	args := append(sopsEncryptArgs(), "--age", strings.Join(config.Recipients, ","))
	encrypted, err := runSOPS(ctx, config.SOPS, args, plaintext, "", "", nil)
	if err != nil {
		return err
	}
	if err := validateSOPSMetadata(encrypted, model.Source{}, false); err != nil {
		return err
	}
	return platform.WriteProtected(config.Path, encrypted)
}

// DecryptSnapshot reads an age-only snapshot, with no implicit token/key discovery.
func DecryptSnapshot(ctx context.Context, config model.Snapshot) (map[string]string, error) {
	data, err := platform.ReadProtected(config.Path, true)
	if err != nil {
		return nil, err
	}
	if err := validateSOPSMetadata(data, model.Source{}, false); err != nil {
		return nil, err
	}
	plaintext, err := runSOPS(ctx, config.SOPS, sopsDecryptArgs(), data, config.IdentityFile, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeStrings(plaintext)
}

func sopsDecryptArgs() []string {
	return []string{"decrypt", sopsInputTypeFlag, sopsJSONDocumentType, sopsOutputTypeFlag, sopsJSONDocumentType, sopsFilenameFlag, sopsJSONFilename}
}

func sopsEncryptArgs() []string {
	return []string{"encrypt", sopsInputTypeFlag, sopsJSONDocumentType, sopsOutputTypeFlag, sopsJSONDocumentType, sopsFilenameFlag, sopsJSONFilename}
}

type sopsMetadata struct {
	Age    []json.RawMessage `json:"age"`
	PGP    []json.RawMessage `json:"pgp"`
	KMS    []json.RawMessage `json:"kms"`
	GCP    []json.RawMessage `json:"gcp_kms"`
	Azure  []json.RawMessage `json:"azure_kv"`
	Huawei []json.RawMessage `json:"huawei_kms"`
	Groups []json.RawMessage `json:"key_groups"`
	Vault  []struct {
		Address string `json:"vault_address"`
		Engine  string `json:"engine_path"`
		Key     string `json:"key_name"`
	} `json:"hc_vault"`
	MAC string `json:"mac"`
}

func validateSOPSMetadata(data []byte, config model.Source, transit bool) error {
	var document struct {
		SOPS sopsMetadata `json:"sops"`
	}
	if err := json.Unmarshal(data, &document); err != nil || document.SOPS.MAC == "" {
		return fault.New(fault.Invalid, "SOPS metadata required")
	}
	m := document.SOPS
	if hasUnsupportedSOPSRecipients(m) {
		return fault.New(fault.Unsupported, "SOPS recipient backend")
	}
	if !transit {
		return validateAgeOnlySOPSRecipients(m)
	}
	return validateTransitSOPSRecipients(m, config)
}

func hasUnsupportedSOPSRecipients(metadata sopsMetadata) bool {
	return len(metadata.Groups)+len(metadata.PGP)+len(metadata.KMS)+len(metadata.GCP)+len(metadata.Azure)+len(metadata.Huawei) != 0
}

func validateAgeOnlySOPSRecipients(metadata sopsMetadata) error {
	if len(metadata.Vault) != 0 || len(metadata.Age) == 0 {
		return fault.New(fault.Trust, "age-only recipient policy")
	}
	return nil
}

func validateTransitSOPSRecipients(metadata sopsMetadata, config model.Source) error {
	if len(metadata.Age) != 0 || len(metadata.Vault) == 0 || len(config.VaultAllowlist) == 0 || len(config.VaultKeys) == 0 {
		return fault.New(fault.Trust, "Transit-only recipient policy")
	}
	for _, key := range metadata.Vault {
		if !approvedVaultAddress(key.Address, config.VaultAllowlist) || !contains(config.VaultKeys, key.Engine+"/"+key.Key) {
			return fault.New(fault.Trust, "unapproved Transit recipient")
		}
	}
	return nil
}

func approvedVaultAddress(address string, allowlist []string) bool {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	for _, allowed := range allowlist {
		if strings.TrimSuffix(address, "/") == strings.TrimSuffix(allowed, "/") {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func decodeStrings(data []byte) (map[string]string, error) {
	var values map[string]string
	if len(data) > platform.MaxDocumentSize || json.Unmarshal(data, &values) != nil || values == nil {
		return nil, fault.New(fault.Invalid, "string document required")
	}
	for key, value := range values {
		if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, '\x00') {
			return nil, fault.New(fault.Invalid, "invalid snapshot string")
		}
	}
	return values, nil
}

type cappedBuffer struct {
	bytes.Buffer
	overflow bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > platform.MaxDocumentSize {
		b.overflow = true
		return len(data), nil
	}
	return b.Buffer.Write(data)
}

func runSOPS(ctx context.Context, executable string, args []string, input []byte, ageFile, tokenFile string, allowlist []string) ([]byte, error) {
	if !filepath.IsAbs(executable) {
		return nil, fault.New(fault.Unsupported, "absolute SOPS executable required")
	}
	info, err := os.Stat(executable)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fault.New(fault.Unsupported, "SOPS executable unavailable")
	}
	if err := platform.ValidateExecutable(executable); err != nil {
		return nil, err
	}
	home, err := os.MkdirTemp("", "cquay-sops-*")
	if err != nil {
		return nil, fault.New(fault.Apply, "isolate SOPS identity discovery")
	}
	defer os.RemoveAll(home)
	env, err := sopsEnvironment(home, ageFile, tokenFile, allowlist)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env, cmd.Stdin, cmd.Dir = env, bytes.NewReader(input), home
	var output cappedBuffer
	cmd.Stdout, cmd.Stderr = &output, io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fault.New(fault.Denied, "SOPS operation failed")
	}
	if output.overflow {
		return nil, fault.New(fault.Invalid, "SOPS output limit")
	}
	return output.Bytes(), nil
}

func sopsEnvironment(home, ageFile, tokenFile string, allowlist []string) ([]string, error) {
	env := []string{"HOME=" + home, "USERPROFILE=" + home, "SOPS_HC_VAULT_ALLOWLIST=none"}
	for _, key := range []string{"SystemRoot", "WINDIR", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	if ageFile != "" {
		if _, err := platform.ReadProtected(ageFile, true); err != nil {
			return nil, err
		}
		env = append(env, "SOPS_AGE_KEY_FILE="+ageFile)
	}
	if tokenFile != "" {
		token, err := transitToken(tokenFile, allowlist)
		if err != nil {
			return nil, err
		}
		env[2] = "SOPS_HC_VAULT_ALLOWLIST=" + strings.Join(allowlist, ",")
		env = append(env, "VAULT_TOKEN="+token)
	}
	return env, nil
}

func transitToken(tokenFile string, allowlist []string) (string, error) {
	if len(allowlist) == 0 {
		return "", fault.New(fault.Trust, "Transit endpoint required")
	}
	for _, endpoint := range allowlist {
		if !approvedVaultAddress(endpoint, []string{endpoint}) {
			return "", fault.New(fault.Trust, "invalid Transit endpoint")
		}
	}
	data, err := platform.ReadProtected(tokenFile, true)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" || strings.ContainsAny(token, "\r\n\x00") {
		return "", fault.New(fault.Invalid, "invalid Transit token file")
	}
	return token, nil
}
