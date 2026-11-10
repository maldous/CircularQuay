package source

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

func readVault(ctx context.Context, c transport.Client, s model.Source, b model.Binding) (model.Value, error) {
	u, err := vaultURL(s, b)
	if err != nil {
		return model.Value{}, err
	}
	token, err := transport.ReadCredentialFile(s.TokenFile)
	if err != nil {
		return model.Value{}, err
	}
	headers := map[string]string{"X-Vault-Token": token}
	if s.Namespace != "" {
		headers["X-Vault-Namespace"] = s.Namespace
	}
	var result vaultResult
	if err = c.DoJSON(ctx, http.MethodGet, u, headers, nil, &result); err != nil {
		return model.Value{}, err
	}
	return vaultValue(result, s, b)
}

func vaultURL(s model.Source, b model.Binding) (*url.URL, error) {
	if s.Mount == "" {
		return nil, fault.New(fault.Invalid, "vault_mount")
	}
	u, err := transport.URL(s.Endpoint, "v1", s.Mount, "data", b.Resource)
	if err != nil {
		return nil, err
	}
	if b.Version == "" {
		return u, nil
	}
	version, err := strconv.ParseUint(b.Version, 10, 64)
	if err != nil || version == 0 {
		return nil, fault.New(fault.Invalid, "source_version")
	}
	query := u.Query()
	query.Set("version", b.Version)
	u.RawQuery = query.Encode()
	return u, nil
}

type vaultResult struct {
	Data struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			Version      int    `json:"version"`
			DeletionTime string `json:"deletion_time"`
			Destroyed    bool   `json:"destroyed"`
		} `json:"metadata"`
	} `json:"data"`
	LeaseID       string `json:"lease_id"`
	LeaseDuration int    `json:"lease_duration"`
	Renewable     bool   `json:"renewable"`
}

func vaultValue(result vaultResult, s model.Source, b model.Binding) (model.Value, error) {
	if result.Data.Metadata.Destroyed || result.Data.Metadata.DeletionTime != "" {
		return model.Value{}, fault.New(fault.Missing, "deleted_source_version")
	}
	field := b.Field
	if field == "" {
		field = b.Name
	}
	value, err := stringField(result.Data.Data, field)
	if err != nil {
		return model.Value{}, err
	}
	if err = validateReturnedVersion(b, strconv.Itoa(result.Data.Metadata.Version)); err != nil {
		return model.Value{}, err
	}
	return leasedVaultValue(value, result, s), nil
}

func leasedVaultValue(value string, result vaultResult, s model.Source) model.Value {
	v := model.Value{Data: value, Version: strconv.Itoa(result.Data.Metadata.Version), Leased: result.LeaseID != "", Renewable: result.Renewable, AuditVerified: s.AuditVerified}
	if v.Leased && result.LeaseDuration > 0 {
		v.ExpiresAt = time.Now().UTC().Add(time.Duration(result.LeaseDuration) * time.Second)
	}
	return v
}

func readHosted(ctx context.Context, c transport.Client, provider string, s model.Source, b model.Binding) (model.Value, error) {
	if provider == "infisical" && !validInfisicalPath(s.Path) {
		return model.Value{}, fault.New(fault.Invalid, "infisical_scope")
	}
	token, err := transport.ReadCredentialFile(s.TokenFile)
	if err != nil {
		return model.Value{}, err
	}
	switch provider {
	case "doppler":
		return readDoppler(ctx, c, s, b, token)
	case "infisical":
		return readInfisical(ctx, c, s, b, token)
	case "akeyless":
		return readAkeyless(ctx, c, s, b, token)
	default:
		return model.Value{}, fault.New(fault.Unsupported, "source_type")
	}
}

func readDoppler(ctx context.Context, c transport.Client, s model.Source, b model.Binding, token string) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	if s.Project == "" || s.Environment == "" || b.Version != "" {
		return model.Value{}, fault.New(fault.Invalid, "doppler_scope")
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://api.doppler.com"
	}
	u, err := transport.URL(endpoint, "v3", "configs", "config", "secret")
	if err != nil {
		return model.Value{}, err
	}
	q := u.Query()
	q.Set("project", s.Project)
	q.Set("config", s.Environment)
	q.Set("name", b.Resource)
	u.RawQuery = q.Encode()
	var result struct {
		Value struct {
			Raw *string `json:"raw"`
		} `json:"value"`
	}
	if err = c.DoJSON(ctx, http.MethodGet, u, bearer(token), nil, &result); err != nil {
		return model.Value{}, err
	}
	if result.Value.Raw == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	data, err := selectValue(*result.Value.Raw, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, AuditVerified: s.AuditVerified}, nil
}

func readInfisical(ctx context.Context, c transport.Client, s model.Source, b model.Binding, token string) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	if s.Project == "" || s.Environment == "" || !validInfisicalPath(s.Path) {
		return model.Value{}, fault.New(fault.Invalid, "infisical_scope")
	}
	u, err := infisicalURL(s, b)
	if err != nil {
		return model.Value{}, err
	}
	var result struct {
		Secret struct {
			SecretValue *string `json:"secretValue"`
			Version     int     `json:"version"`
			Hidden      bool    `json:"secretValueHidden"`
		} `json:"secret"`
	}
	if err = c.DoJSON(ctx, http.MethodGet, u, bearer(token), nil, &result); err != nil {
		return model.Value{}, err
	}
	return infisicalValue(result.Secret.SecretValue, result.Secret.Version, result.Secret.Hidden, s, b)
}

func validInfisicalPath(value string) bool {
	return value == "/" || (utf8.ValidString(value) && strings.HasPrefix(value, "/") && value == path.Clean(value) && !strings.ContainsAny(value, "*?[]\\\x00\r\n#") && len(value) <= 4096)
}

func infisicalURL(s model.Source, b model.Binding) (*url.URL, error) {
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://us.infisical.com"
	}
	u, err := transport.URL(endpoint, "api", "v4", "secrets", b.Resource)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("projectId", s.Project)
	q.Set("environment", s.Environment)
	q.Set("secretPath", s.Path)
	q.Set("expandSecretReferences", "false")
	q.Set("includeImports", "false")
	if b.Version != "" {
		if _, err = strconv.ParseUint(b.Version, 10, 64); err != nil {
			return nil, fault.New(fault.Invalid, "source_version")
		}
		q.Set("version", b.Version)
	}
	u.RawQuery = q.Encode()
	return u, nil
}

func infisicalValue(raw *string, version int, hidden bool, s model.Source, b model.Binding) (model.Value, error) {
	if hidden {
		return model.Value{}, fault.New(fault.Denied, "source_value_hidden")
	}
	if raw == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	data, err := selectValue(*raw, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	actual := strconv.Itoa(version)
	if err = validateReturnedVersion(b, actual); err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: actual, AuditVerified: s.AuditVerified}, nil
}

func readAkeyless(ctx context.Context, c transport.Client, s model.Source, b model.Binding, token string) (model.Value, error) {
	if err := validateVersionCapability(s, b); err != nil {
		return model.Value{}, err
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://api.akeyless.io"
	}
	u, err := transport.URL(endpoint, "get-secret-value")
	if err != nil {
		return model.Value{}, err
	}
	body := map[string]any{"names": []string{b.Resource}, "token": token}
	var result map[string]any
	if err = c.DoJSON(ctx, http.MethodPost, u, nil, body, &result); err != nil {
		return model.Value{}, err
	}
	data, err := stringField(result, b.Resource)
	if err != nil {
		return model.Value{}, err
	}
	data, err = selectValue(data, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: b.Version, AuditVerified: s.AuditVerified}, nil
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// CanonicalEndpoint validates public endpoint overrides without accepting URL secrets.
func CanonicalEndpoint(raw string) (*url.URL, error) { return transport.Endpoint(raw) }
