package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/maldous/CircularQuay/internal/platform"
)

func TestTrustStoreHTTPClientUsesDeclaredReplacementRoots(t *testing.T) {
	rootPEM, certificate := syntheticTrustStoreCertificate(t)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.StartTLS()
	t.Cleanup(server.Close)

	path := filepath.Join(canonicalTrustStoreDirectory(t), "authority-roots.pem")
	if err := platform.WriteProtected(path, rootPEM); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(rootPEM)
	client, err := TrustStoreHTTPClient(path, hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Get(server.URL)
	if err != nil || response.StatusCode != http.StatusNoContent {
		t.Fatalf("declared root request = response %v, error %v", response, err)
	}
	_ = response.Body.Close()

	otherPEM, _ := syntheticTrustStoreCertificate(t)
	otherPath := filepath.Join(canonicalTrustStoreDirectory(t), "other-roots.pem")
	if err := platform.WriteProtected(otherPath, otherPEM); err != nil {
		t.Fatal(err)
	}
	otherDigest := sha256.Sum256(otherPEM)
	otherClient, err := TrustStoreHTTPClient(otherPath, hex.EncodeToString(otherDigest[:]))
	if err != nil {
		t.Fatal(err)
	}
	if response, err := otherClient.Get(server.URL); err == nil {
		_ = response.Body.Close()
		t.Fatal("unrelated replacement root verified authority")
	}
}

func TestTrustStoreRejectsDigestAndPEMDrift(t *testing.T) {
	rootPEM, certificate := syntheticTrustStoreCertificate(t)
	path := filepath.Join(canonicalTrustStoreDirectory(t), "authority-roots.pem")
	if err := platform.WriteProtected(path, rootPEM); err != nil {
		t.Fatal(err)
	}
	if _, err := TrustStoreHTTPClient(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("mismatched trust-store digest accepted")
	}
	for _, data := range [][]byte{
		[]byte("not pem"),
		append([]byte("leading\n"), rootPEM...),
		append([]byte("-----BEGIN GARBAGE-----\ncorrupt\n"), rootPEM...),
		append([]byte("-----BEGIN CERTIFICATE-----\ncorrupt\n-----END CERTIFICATE-----\n"), rootPEM...),
		append(append([]byte(nil), rootPEM...), []byte("trailing")...),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("synthetic")}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]}),
	} {
		if _, err := certificatePool(data); err == nil {
			t.Fatal("non-certificate PEM accepted")
		}
	}
}

func canonicalTrustStoreDirectory(t *testing.T) string {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

func syntheticTrustStoreCertificate(t *testing.T) ([]byte, tls.Certificate) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "cquay synthetic root"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "127.0.0.1"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}
}
