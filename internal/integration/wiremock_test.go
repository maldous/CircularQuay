//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

const (
	wireMockVersion       = "3.13.2"
	wireMockSHA256        = "d097b19bd483c5038479b13a5c71e9faf8f2f5106584f0c120a7770ab0bdb367"
	wireMockJARURL        = "https://repo.maven.apache.org/maven2/org/wiremock/wiremock-standalone/3.13.2/wiremock-standalone-3.13.2.jar"
	wireMockSHAURL        = wireMockJARURL + ".sha256"
	requestLimit          = 1 << 20
	wireMockStartAttempts = 3
)

type toolManifest struct {
	Version  int `json:"version"`
	WireMock struct {
		Version string `json:"version"`
		SHA256  string `json:"sha256"`
		JAR     string `json:"jar"`
		Source  string `json:"source"`
	} `json:"wiremock"`
}

type wireMock struct {
	base    string
	client  *http.Client
	cmd     *exec.Cmd
	dir     string
	stopped bool
	exited  chan struct{}
}

func startWireMock(t *testing.T) *wireMock {
	t.Helper()
	dir := privateDirectory(t)
	jar := wireMockJar(t, dir)
	password := "wiremock-test-only-password"
	keyStore := makeKeyStore(t, dir, password)
	client := wireMockClient(t, keyStore, password)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	var startupError error
	// Java cannot inherit the Go listener. Retry a raced port assignment only
	// after the previous process has been reaped, within one startup budget.
	for attempt := 0; attempt < wireMockStartAttempts; attempt++ {
		w := startWireMockAttempt(t, dir, jar, keyStore, password, client)
		t.Cleanup(func() { w.stop(t) })
		startupError = w.waitReady(ctx)
		if startupError == nil {
			return w
		}
		w.stop(t)
		if ctx.Err() != nil {
			break
		}
	}
	t.Fatalf("WireMock HTTPS startup failed after bounded attempts: %v", startupError)
	return nil
}

func startWireMockAttempt(t *testing.T, dir, jar, keyStore, password string, client *http.Client) *wireMock {
	t.Helper()
	port := loopbackPort(t)
	w := &wireMock{base: fmt.Sprintf("https://127.0.0.1:%d", port), dir: dir, exited: make(chan struct{})}
	w.client = client
	w.cmd = wireMockCommand(t, dir, jar, keyStore, password, port)
	if err := w.cmd.Start(); err != nil {
		t.Fatalf("start WireMock: %v", err)
	}
	// One goroutine owns Wait for the entire process lifetime. Shutdown and
	// forced termination observe the same completion rather than racing Wait.
	go func() {
		_ = w.cmd.Wait()
		close(w.exited)
	}()
	return w
}

func wireMockJar(t *testing.T, dir string) string {
	t.Helper()
	manifest := readManifest(t)
	requireWireMockManifest(t, manifest)
	path := configuredJarPath(t)
	requireRegularJar(t, path)
	return copyVerifiedJar(t, dir, path, manifest.WireMock.SHA256)
}

func requireWireMockManifest(t *testing.T, manifest toolManifest) {
	t.Helper()
	if manifest.Version != 1 || manifest.WireMock.Version != wireMockVersion || manifest.WireMock.SHA256 != wireMockSHA256 || manifest.WireMock.JAR != wireMockJARURL || manifest.WireMock.Source != wireMockSHAURL {
		t.Fatal("tools/integration-tools.json does not contain the approved WireMock selection")
	}
}

func configuredJarPath(t *testing.T) string {
	t.Helper()
	path := os.Getenv("CQUAY_WIREMOCK_JAR")
	if path == "" || !filepath.IsAbs(path) {
		t.Fatal("integration tag requires an absolute CQUAY_WIREMOCK_JAR selected from tools/integration-tools.json")
	}
	return path
}

func requireRegularJar(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 256<<20 {
		t.Fatal("configured WireMock JAR must be a bounded regular file")
	}
}

func copyVerifiedJar(t *testing.T, dir, path, expected string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open configured WireMock JAR: %v", err)
	}
	defer file.Close()
	copyPath := filepath.Join(dir, "wiremock-standalone.jar")
	copyFile, err := os.OpenFile(copyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatalf("create verified WireMock copy: %v", err)
	}
	defer copyFile.Close()
	hash := sha256.New()
	count, err := io.CopyBuffer(io.MultiWriter(hash, copyFile), io.LimitReader(file, (256<<20)+1), make([]byte, 32<<10))
	if err != nil || count > 256<<20 {
		t.Fatal("configured WireMock JAR exceeds verification bound")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		t.Fatal("configured WireMock JAR digest does not match tools/integration-tools.json")
	}
	return copyPath
}

func readManifest(t *testing.T) toolManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "tools", "integration-tools.json"))
	if err != nil {
		t.Fatalf("read integration tools manifest: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest toolManifest
	if err = decoder.Decode(&manifest); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		t.Fatal("invalid integration tools manifest")
	}
	return manifest
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func privateDirectory(t *testing.T) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatalf("resolve integration temporary directory: %v", err)
	}
	parent, err := os.MkdirTemp(base, "cquay-wiremock-")
	if err != nil {
		t.Fatalf("create private integration directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	// The absent child lets native protection establish its DACL at creation;
	// chmod on an already-created temporary directory cannot do that on Windows.
	dir := filepath.Join(parent, "instance")
	if err = platform.EnsurePrivateDirectory(dir); err != nil {
		t.Fatalf("secure integration directory: %v", err)
	}
	return dir
}

func makeKeyStore(t *testing.T, dir, password string) string {
	t.Helper()
	path := filepath.Join(dir, "wiremock.p12")
	args := []string{"-genkeypair", "-alias", "wiremock", "-keyalg", "RSA", "-keysize", "2048", "-storetype", "PKCS12", "-keystore", path, "-storepass", password, "-keypass", password, "-dname", "CN=127.0.0.1", "-ext", "SAN=IP:127.0.0.1,DNS:localhost", "-validity", "1", "-noprompt"}
	runKeytool(t, dir, args...)
	return path
}

func runKeytool(t *testing.T, dir string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	arguments := append([]string{"-J-Duser.home=" + dir, "-J-Djava.io.tmpdir=" + dir}, args...)
	cmd := exec.CommandContext(ctx, "keytool", arguments...)
	cmd.Dir, cmd.Env = dir, privateEnvironment(dir)
	cmd.WaitDelay = time.Second
	output := &limitedOutput{limit: 4096}
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Run(); err != nil {
		t.Fatalf("keytool failed: %v: %s", err, output.String())
	}
}

func privateEnvironment(home string) []string {
	environment := []string{"HOME=" + home, "USERPROFILE=" + home, "APPDATA=" + home, "LOCALAPPDATA=" + home,
		"TMPDIR=" + home, "TMP=" + home, "TEMP=" + home, "PATH=" + os.Getenv("PATH")}
	for _, key := range []string{"SystemRoot", "WINDIR", "ComSpec", "PATHEXT"} {
		if value := os.Getenv(key); value != "" {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func loopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func wireMockClient(t *testing.T, keyStore, password string) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(exportCertificate(t, filepath.Dir(keyStore), keyStore, password)) {
		t.Fatal("load WireMock test CA")
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}
	return &http.Client{Timeout: 5 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func exportCertificate(t *testing.T, dir, keyStore, password string) []byte {
	t.Helper()
	path := filepath.Join(dir, "wiremock.pem")
	runKeytool(t, dir, "-exportcert", "-rfc", "-alias", "wiremock", "-keystore", keyStore, "-storepass", password, "-file", path)
	certificate, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read WireMock certificate: %v", err)
	}
	return certificate
}

func wireMockCommand(t *testing.T, dir, jar, keyStore, password string, port int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("java", "-Duser.home="+dir, "-Djava.io.tmpdir="+dir, "-jar", jar, "--bind-address", "127.0.0.1", "--port", "0", "--https-port", fmt.Sprint(port), "--https-keystore", keyStore, "--keystore-type", "PKCS12", "--keystore-password", password, "--key-manager-password", password, "--disable-banner")
	cmd.Dir, cmd.Env = dir, privateEnvironment(dir)
	cmd.Stdout, cmd.Stderr = boundedWriter{}, boundedWriter{}
	return cmd
}

func (w *wireMock) waitReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if w.healthy(ctx) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.exited:
			return errors.New("WireMock exited before its HTTPS health check succeeded")
		case <-ticker.C:
		}
	}
}

func (w *wireMock) healthy(ctx context.Context) bool {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, w.base+"/__admin/health", nil)
	if err != nil {
		return false
	}
	response, err := w.client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK
}

func (w *wireMock) stop(t *testing.T) {
	t.Helper()
	if w.stopped || w.cmd.Process == nil {
		return
	}
	w.stopped = true
	w.client.CloseIdleConnections()
	w.shutdown()
	if w.wait(3 * time.Second) {
		return
	}
	_ = w.cmd.Process.Kill()
	if w.wait(5 * time.Second) {
		return
	}
	t.Errorf("WireMock process did not exit after kill")
}

func (w *wireMock) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, w.base+"/__admin/shutdown", nil)
	if err != nil {
		return
	}
	request.Header.Set("Connection", "close")
	response, err := w.client.Do(request)
	if err == nil {
		response.Body.Close()
	}
}

func (w *wireMock) wait(limit time.Duration) bool {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case <-w.exited:
		return true
	case <-timer.C:
		return false
	}
}

func (w *wireMock) reset(t *testing.T) { w.admin(t, http.MethodPost, "/__admin/reset", nil, nil) }
func (w *wireMock) mapping(t *testing.T, body string) {
	w.admin(t, http.MethodPost, "/__admin/mappings", strings.NewReader(body), nil)
}

func (w *wireMock) count(t *testing.T, requestURL string) int {
	t.Helper()
	var result struct {
		Count int `json:"count"`
	}
	pattern := fmt.Sprintf(`{"method":"GET","url":"%s","headers":{"X-Vault-Token":{"equalTo":"wiremock-test-token"}}}`, requestURL)
	w.admin(t, http.MethodPost, "/__admin/requests/count", strings.NewReader(pattern), &result)
	return result.Count
}

func (w *wireMock) verifyJournal(t *testing.T, want int) {
	t.Helper()
	var all struct {
		Count int `json:"count"`
	}
	w.admin(t, http.MethodPost, "/__admin/requests/count", strings.NewReader(`{}`), &all)
	if all.Count != want {
		t.Fatalf("WireMock total request count = %d, want %d", all.Count, want)
	}
	var unmatched struct {
		Requests []json.RawMessage `json:"requests"`
	}
	w.admin(t, http.MethodGet, "/__admin/requests/unmatched", nil, &unmatched)
	if unmatched.Requests == nil || len(unmatched.Requests) != 0 {
		t.Fatal("WireMock journal contains unmatched requests or lacks request evidence")
	}
}

func (w *wireMock) admin(t *testing.T, method, path string, body io.Reader, out any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, w.base+path, body)
	if err != nil {
		t.Fatalf("build WireMock admin request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := w.client.Do(request)
	if err != nil {
		t.Fatalf("call WireMock admin: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("WireMock admin returned %d", response.StatusCode)
	}
	if out != nil && json.NewDecoder(io.LimitReader(response.Body, requestLimit)).Decode(out) != nil {
		t.Fatal("decode WireMock admin response")
	}
}

type boundedWriter struct{}

func (boundedWriter) Write(data []byte) (int, error) { return len(data), nil }

type limitedOutput struct {
	mu    sync.Mutex
	data  []byte
	limit int
}

func (w *limitedOutput) Write(data []byte) (int, error) {
	length := len(data)
	w.mu.Lock()
	defer w.mu.Unlock()
	if room := w.limit - len(w.data); room > 0 {
		if len(data) > room {
			data = data[:room]
		}
		w.data = append(w.data, data...)
	}
	return length, nil
}
func (w *limitedOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.TrimSpace(string(w.data))
}
