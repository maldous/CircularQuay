package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
)

type proofInputCurveWrapper struct{ elliptic.Curve }

func TestProofInputContractUnsupportedCurveEncodingCannotBecomeP256(t *testing.T) {
	// The wrapper retains the P-256 name but is not one of the supported standard
	// curve identities accepted by pinned ECDSA.Bytes. No raw X/Y writes or
	// malformed coordinate construction is needed to exercise this refusal.
	key := &ecdsa.PublicKey{Curve: proofInputCurveWrapper{elliptic.P256()}}
	if data, err := key.Bytes(); err == nil || data != nil {
		t.Fatal("unsupported curve control unexpectedly acquired an encoding")
	}
	if validES256Key(key) || algorithmKeyMatches("ES256", key) {
		t.Fatal("unsupported curve encoding became admitted proof authority")
	}
	data, err := hex.DecodeString("046b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c2964fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5")
	if err != nil {
		t.Fatal(err)
	}
	valid, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), data)
	if err != nil || !validES256Key(valid) || !algorithmKeyMatches("ES256", valid) {
		t.Fatal("supported parsed P-256 control did not retain admission")
	}
}

func TestProofInputContractMalformedClaimsReturnWholeZero(t *testing.T) {
	parsed, err := proofClaims("%")
	if !errors.Is(err, ErrInvalid) || !reflect.DeepEqual(parsed, proof{}) {
		t.Fatal("malformed base64 claims produced partial proof authority")
	}
}
