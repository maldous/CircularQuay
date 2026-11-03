// Package transport implements bounded, credential-safe provider HTTP requests.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

const maxPayload = 4 << 20

// Client holds a pinned-origin HTTP client. Redirects are never followed.
type Client struct{ HTTP *http.Client }

// New preserves the system trust store and session proxy configuration.
func New() Client {
	return Client{HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Endpoint validates a configured provider URL before it sees credentials.
func Endpoint(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return nil, fault.New(fault.Trust, "provider_endpoint")
	}
	if strings.ContainsAny(u.Host, "\x00\r\n") {
		return nil, fault.New(fault.Trust, "provider_endpoint")
	}
	return u, nil
}

// URL appends individually validated path components to a trusted base URL.
func URL(base string, parts ...string) (*url.URL, error) {
	u, err := Endpoint(base)
	if err != nil {
		return nil, err
	}
	p := strings.TrimRight(u.Path, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		for _, segment := range strings.Split(part, "/") {
			if segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, "\x00\r\n\\?#") {
				return nil, fault.New(fault.Invalid, "resource_path")
			}
			p += "/" + segment
		}
	}
	u.Path = p
	u.RawPath = ""
	return u, nil
}

// DoJSON bounds bodies and exposes only safe error categories.
func (c Client) DoJSON(ctx context.Context, method string, u *url.URL, headers map[string]string, body any, out any) error {
	if _, err := EndpointWithoutQuery(u); err != nil {
		return err
	}
	reader, err := requestBody(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return fault.New(fault.Invalid, "request")
	}
	if err := setHeaders(req, headers, body != nil); err != nil {
		return err
	}
	return c.doRequest(req, out)
}

func requestBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fault.New(fault.Invalid, "request_encoding")
	}
	if len(b) > maxPayload {
		return nil, fault.New(fault.Invalid, "request_size")
	}
	return bytes.NewReader(b), nil
}

func setHeaders(req *http.Request, headers map[string]string, hasBody bool) error {
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		if !validHeader(name, value) {
			return fault.New(fault.Invalid, "credential_format")
		}
		req.Header.Set(name, value)
	}
	return nil
}

func validHeader(name, value string) bool {
	if name == "" || strings.ContainsAny(value, "\r\n\x00") {
		return false
	}
	for _, character := range name {
		if !isHeaderToken(character) {
			return false
		}
	}
	return true
}

func isHeaderToken(character rune) bool {
	return character >= '0' && character <= '9' ||
		character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		strings.ContainsRune("!#$%&'*+-.^_`|~", character)
}

func (c Client) doRequest(req *http.Request, out any) error {
	client := c.HTTP
	if client == nil {
		client = New().HTTP
	}
	request, cancel := boundedRequest(req, client.Timeout)
	defer cancel()
	// Endpoint validation binds the configured provider origin to HTTPS with no
	// credentials, selectors, or redirects. RoundTrip performs exactly one
	// request, so this boundary cannot follow a provider-controlled redirect to
	// another origin. Provider endpoints remain an operator-authorized target.
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(request)
	if err != nil {
		return classifyTransport(err)
	}
	if resp == nil || resp.Body == nil {
		return fault.New(fault.Trust, "provider_transport")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Status(resp.StatusCode)
	}
	return decodeResponse(resp.Body, out)
}

func boundedRequest(req *http.Request, timeout time.Duration) (*http.Request, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	return req.Clone(ctx), cancel
}

func decodeResponse(body io.Reader, out any) error {
	if out == nil {
		return discardBounded(body)
	}
	return decodeJSON(body, out)
}

func discardBounded(body io.Reader) error {
	size, err := io.Copy(io.Discard, io.LimitReader(body, maxPayload+1))
	if err != nil {
		return fault.New(fault.Unavailable, "response_read")
	}
	if size > maxPayload {
		return fault.New(fault.Trust, "response_size")
	}
	return nil
}

func decodeJSON(body io.Reader, out any) error {
	data, err := io.ReadAll(io.LimitReader(body, maxPayload+1))
	if err != nil {
		return fault.New(fault.Unavailable, "response_read")
	}
	if len(data) > maxPayload {
		return fault.New(fault.Trust, "response_size")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err = dec.Decode(out); err != nil {
		return fault.New(fault.Trust, "response_encoding")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return fault.New(fault.Trust, "response_encoding")
	}
	return nil
}

// EndpointWithoutQuery validates an already constructed request without discarding selectors.
func EndpointWithoutQuery(u *url.URL) (*url.URL, error) {
	if u == nil {
		return nil, fault.New(fault.Trust, "provider_endpoint")
	}
	copyURL := *u
	copyURL.RawQuery = ""
	return Endpoint(copyURL.String())
}

// Status classifies HTTP outcomes without parsing potentially sensitive response bodies.
func Status(status int) error {
	switch {
	case status == 401 || status == 403:
		return fault.New(fault.Denied, "provider_request")
	case status == 404:
		return fault.New(fault.Missing, "provider_request")
	case status >= 300 && status < 400:
		return fault.New(fault.Trust, "provider_redirect")
	default:
		return fault.New(fault.Trust, "provider_response")
	}
}

func classifyTransport(err error) error {
	var cert x509.CertificateInvalidError
	var unknown x509.UnknownAuthorityError
	var host x509.HostnameError
	var verify *tls.CertificateVerificationError
	if errors.As(err, &cert) || errors.As(err, &unknown) || errors.As(err, &host) || errors.As(err, &verify) {
		return fault.New(fault.Trust, "provider_tls")
	}
	var op *net.OpError
	var dns *net.DNSError
	if errors.As(err, &op) || errors.As(err, &dns) || errors.Is(err, context.DeadlineExceeded) {
		return fault.New(fault.Unavailable, "provider_connection")
	}
	return fault.New(fault.Trust, "provider_transport")
}

// ReadCredentialFile reads a bounded private regular file, never a symlink.
// Native protection includes POSIX permissions and Windows ACL validation.
func ReadCredentialFile(name string) (string, error) {
	if !filepath.IsAbs(name) {
		return "", fault.New(fault.Invalid, "credential_path")
	}
	b, err := platform.ReadProtected(name, true)
	if fault.Is(err, fault.Missing) {
		return "", fault.New(fault.Denied, "credential_file")
	}
	if err != nil || len(b) > 65536 {
		return "", fault.New(fault.Trust, "credential_file")
	}
	token := strings.TrimSpace(string(b))
	if token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return "", fault.New(fault.Trust, "credential_format")
	}
	return token, nil
}
