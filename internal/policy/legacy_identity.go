package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/maldous/CircularQuay/internal/model"
)

// MCPLegacyFingerprint binds a verified legacy HTTP principal to its one
// configured issuer, key set and audience. Named-authority configurations use
// MCPAuthorityFingerprint instead. An incomplete legacy contract has no
// fingerprint and must not authenticate an HTTP principal.
func MCPLegacyFingerprint(config model.MCP) string {
	if len(config.Authorities) != 0 || config.Issuer == "" || config.JWKSURL == "" || config.Audience == "" {
		return ""
	}
	data, err := json.Marshal(struct {
		Issuer   string
		JWKSURL  string
		Audience string
	}{config.Issuer, config.JWKSURL, config.Audience})
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
