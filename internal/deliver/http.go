// Package deliver implements explicit one-way publication to native stores.
package deliver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"github.com/maldous/CircularQuay/internal/transport"
)

const bearerPrefix = "Bearer "

// Apply publishes one complete JSON bundle to one explicitly named native secret.
// It neither deploys application code nor claims runtime read-audit inheritance.
func Apply(ctx context.Context, target model.Target, generation model.Generation) error {
	return ApplyWithClient(ctx, nil, target, generation)
}

// ApplyWithClient permits isolated transport contract tests.
func ApplyWithClient(ctx context.Context, client *http.Client, t model.Target, g model.Generation) error {
	if !t.AcceptSnapshot {
		return fault.New(fault.Denied, "snapshot_publication")
	}
	data, err := bundle(g)
	if err != nil {
		return err
	}
	token, err := transport.ReadCredentialFile(t.TokenFile)
	if err != nil {
		return err
	}
	c := transport.New()
	if client != nil {
		c.HTTP = client
	}
	switch t.Type {
	case "cloudflare":
		return cloudflare(ctx, c, t, token, data)
	case "netlify":
		return netlify(ctx, c, t, token, data)
	case "windmill":
		return windmill(ctx, c, t, token, data)
	default:
		return fault.New(fault.Unsupported, "publication_target")
	}
}

func bundle(g model.Generation) (string, error) {
	if len(g.Values) == 0 {
		return "", fault.New(fault.Invalid, "empty_generation")
	}
	values := make(map[string]string, len(g.Values))
	for name, v := range g.Values {
		if v.Leased || v.Renewable || !v.ExpiresAt.IsZero() {
			return "", fault.New(fault.Denied, "leased_publication")
		}
		if name == "" || strings.ContainsRune(v.Data, 0) {
			return "", fault.New(fault.Invalid, "publication_value")
		}
		values[name] = v.Data
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "", fault.New(fault.Invalid, "publication_encoding")
	}
	return string(b), nil
}

func binding(t model.Target) string {
	if t.Path != "" {
		return t.Path
	}
	return "CQ_CONFIG"
}

func cloudflare(ctx context.Context, c transport.Client, t model.Target, token, data string) error {
	if t.Account == "" || t.Resource == "" {
		return fault.New(fault.Invalid, "cloudflare_target")
	}
	endpoint := t.Endpoint
	if endpoint == "" {
		endpoint = "https://api.cloudflare.com"
	}
	u, err := transport.URL(endpoint, "client", "v4", "accounts", t.Account, "workers", "scripts", t.Resource, "secrets")
	if err != nil {
		return err
	}
	var result struct {
		Success bool `json:"success"`
	}
	body := map[string]any{"name": binding(t), "text": data, "type": "secret_text"}
	if err = c.DoJSON(ctx, http.MethodPut, u, authorizationHeader(token), body, &result); err != nil {
		return err
	}
	if !result.Success {
		return fault.New(fault.Apply, "cloudflare_publication")
	}
	return nil
}

func netlify(ctx context.Context, c transport.Client, t model.Target, token, data string) error {
	if t.Account == "" || t.Project == "" || t.Environment == "" {
		return fault.New(fault.Invalid, "netlify_target")
	}
	endpoint := t.Endpoint
	if endpoint == "" {
		endpoint = "https://api.netlify.com"
	}
	u, err := transport.URL(endpoint, "api", "v1", "accounts", t.Account, "env")
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("site_id", t.Project)
	u.RawQuery = q.Encode()
	body := map[string]any{"key": binding(t), "scopes": []string{"functions"}, "is_secret": true, "values": []any{map[string]any{"context": t.Environment, "value": data}}}
	update, err := transport.URL(endpoint, "api", "v1", "accounts", t.Account, "env", binding(t))
	if err != nil {
		return err
	}
	update.RawQuery = u.RawQuery
	headers := authorizationHeader(token)
	err = c.DoJSON(ctx, http.MethodPut, update, headers, body, nil)
	if !fault.Is(err, fault.Missing) {
		return err
	}
	// This variable is dedicated to one target/context. Never read other values
	// or expand the requested functions scope after a permission or plan failure.
	return c.DoJSON(ctx, http.MethodPost, u, headers, []any{body}, nil)
}

func windmill(ctx context.Context, c transport.Client, t model.Target, token, data string) error {
	if t.Project == "" || t.Resource == "" || t.Endpoint == "" {
		return fault.New(fault.Invalid, "windmill_target")
	}
	headers := authorizationHeader(token)
	u, err := transport.URL(t.Endpoint, "api", "w", t.Project, "variables", "exists", t.Resource)
	if err != nil {
		return err
	}
	var exists bool
	if err = c.DoJSON(ctx, http.MethodGet, u, headers, nil, &exists); err != nil {
		return err
	}
	var path []string
	body := map[string]any{"value": data, "is_secret": true}
	if exists {
		path = []string{"api", "w", t.Project, "variables", "update", t.Resource}
	} else {
		path = []string{"api", "w", t.Project, "variables", "create"}
		body["path"] = t.Resource
	}
	u, err = transport.URL(t.Endpoint, path...)
	if err != nil {
		return err
	}
	return c.DoJSON(ctx, http.MethodPost, u, headers, body, nil)
}

func authorizationHeader(token string) map[string]string {
	return map[string]string{"Authorization": bearerPrefix + token}
}

// ValidateLifecycle can be used before native application delivery.
func ValidateLifecycle(g model.Generation, now time.Time) error {
	for _, v := range g.Values {
		if !v.ExpiresAt.IsZero() && !now.Before(v.ExpiresAt) {
			return fault.New(fault.Expired, "generation")
		}
	}
	return nil
}
