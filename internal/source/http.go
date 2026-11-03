package source

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

func readVault(ctx context.Context, c transport.Client, s model.Source, b model.Binding) (model.Value, error) {
	if s.Mount == "" {
		return model.Value{}, fault.New(fault.Invalid, "vault_mount")
	}
	u, err := transport.URL(s.Endpoint, "v1", s.Mount, "data", b.Resource)
	if err != nil {
		return model.Value{}, err
	}
	if b.Version != "" {
		version, e := strconv.ParseUint(b.Version, 10, 64)
		if e != nil || version == 0 {
			return model.Value{}, fault.New(fault.Invalid, "source_version")
		}
		q := u.Query()
		q.Set("version", b.Version)
		u.RawQuery = q.Encode()
	}
	token, err := transport.ReadCredentialFile(s.TokenFile)
	if err != nil {
		return model.Value{}, err
	}
	headers := map[string]string{"X-Vault-Token": token}
	if s.Namespace != "" {
		headers["X-Vault-Namespace"] = s.Namespace
	}
	var result struct {
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
	if err = c.DoJSON(ctx, http.MethodGet, u, headers, nil, &result); err != nil {
		return model.Value{}, err
	}
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
	v := model.Value{Data: value, Version: strconv.Itoa(result.Data.Metadata.Version), Leased: result.LeaseID != "", Renewable: result.Renewable, AuditVerified: s.AuditVerified}
	if v.Leased && result.LeaseDuration > 0 {
		v.ExpiresAt = time.Now().UTC().Add(time.Duration(result.LeaseDuration) * time.Second)
	}
	return v, nil
}

func readHosted(ctx context.Context, c transport.Client, provider string, s model.Source, b model.Binding) (model.Value, error) {
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
	if s.Project == "" || s.Environment == "" {
		return model.Value{}, fault.New(fault.Invalid, "infisical_scope")
	}
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://us.infisical.com"
	}
	u, err := transport.URL(endpoint, "api", "v4", "secrets", b.Resource)
	if err != nil {
		return model.Value{}, err
	}
	q := u.Query()
	q.Set("projectId", s.Project)
	q.Set("environment", s.Environment)
	q.Set("secretPath", s.Path)
	q.Set("expandSecretReferences", "false")
	q.Set("includeImports", "false")
	if b.Version != "" {
		if _, err = strconv.ParseUint(b.Version, 10, 64); err != nil {
			return model.Value{}, fault.New(fault.Invalid, "source_version")
		}
		q.Set("version", b.Version)
	}
	u.RawQuery = q.Encode()
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
	if result.Secret.Hidden {
		return model.Value{}, fault.New(fault.Denied, "source_value_hidden")
	}
	if result.Secret.SecretValue == nil {
		return model.Value{}, fault.New(fault.Missing, "source_value")
	}
	data, err := selectValue(*result.Secret.SecretValue, b.Field)
	if err != nil {
		return model.Value{}, err
	}
	return model.Value{Data: data, Version: strconv.Itoa(result.Secret.Version), AuditVerified: s.AuditVerified}, nil
}

func readAkeyless(ctx context.Context, c transport.Client, s model.Source, b model.Binding, token string) (model.Value, error) {
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://api.akeyless.io"
	}
	u, err := transport.URL(endpoint, "get-secret-value")
	if err != nil {
		return model.Value{}, err
	}
	body := map[string]any{"names": []string{b.Resource}, "token": token}
	if b.Version != "" {
		version, e := strconv.ParseUint(b.Version, 10, 31)
		if e != nil || version == 0 {
			return model.Value{}, fault.New(fault.Invalid, "source_version")
		}
		body["version"] = version
	}
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
