package transport

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/platform"
)

const maxTrustStoreCertificates = 128
const maxTrustStoreBytes = 1 << 20

var pemBegin = []byte("-----BEGIN ")

// TrustStoreHTTPClient creates a bounded, redirect-denying client whose root
// store is exactly the protected PEM bundle identified by digest. It never
// mutates global TLS state and does not append system roots.
func TrustStoreHTTPClient(path, expectedSHA256 string) (*http.Client, error) {
	pool, err := trustStorePool(path, expectedSHA256)
	if err != nil {
		return nil, err
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fault.New(fault.Trust, "trust store transport")
	}
	configured := base.Clone()
	configured.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
	return NewHTTPClient(configured), nil
}

// ValidateTrustStore verifies protected PEM bytes and their declared digest
// without changing a process trust store or opening a network connection.
func ValidateTrustStore(path, expectedSHA256 string) error {
	_, err := trustStorePool(path, expectedSHA256)
	return err
}

func trustStorePool(path, expectedSHA256 string) (*x509.CertPool, error) {
	if !validTrustStoreDigest(expectedSHA256) {
		return nil, fault.New(fault.Invalid, "trust store digest")
	}
	data, err := platform.ReadProtected(path, false)
	if err != nil {
		return nil, fault.New(fault.Trust, "trust store file")
	}
	if len(data) == 0 || len(data) > maxTrustStoreBytes {
		return nil, fault.New(fault.Invalid, "trust store size")
	}
	actual := sha256.Sum256(data)
	expected, _ := hex.DecodeString(expectedSHA256)
	if subtle.ConstantTimeCompare(actual[:], expected) != 1 {
		return nil, fault.New(fault.Trust, "trust store digest")
	}
	return certificatePool(data)
}

func validTrustStoreDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f') {
			return false
		}
	}
	return true
}

func certificatePool(data []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	count := 0
	for rest := data; len(bytes.TrimSpace(rest)) != 0; {
		certificate, next, err := certificateBlock(bytes.TrimSpace(rest))
		if err != nil {
			return nil, fault.New(fault.Invalid, "trust store PEM")
		}
		pool.AddCert(certificate)
		count++
		if count > maxTrustStoreCertificates {
			return nil, fault.New(fault.Invalid, "trust store PEM")
		}
		rest = next
	}
	if count == 0 {
		return nil, fault.New(fault.Invalid, "trust store PEM")
	}
	return pool, nil
}

func certificateBlock(data []byte) (*x509.Certificate, []byte, error) {
	block, rest := pem.Decode(data)
	consumed := data[:len(data)-len(rest)]
	if !bytes.HasPrefix(data, pemBegin) || bytes.Count(consumed, pemBegin) != 1 || block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, nil, fault.New(fault.Invalid, "trust store PEM")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !certificate.IsCA {
		return nil, nil, fault.New(fault.Invalid, "trust store PEM")
	}
	return certificate, rest, nil
}
