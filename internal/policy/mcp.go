package policy

import (
	"net"
	"net/url"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

func validateMCP(config model.MCP) error {
	if config.Issuer == "" && config.Audience == "" && config.JWKSURL == "" {
		return validateMCPLists(config)
	}
	if !validMCPAuthority(config) {
		return invalid("mcp_authority")
	}
	if !mcpKeySetMatchesIssuer(config) {
		return invalid("mcp_key_origin")
	}
	return validateMCPLists(config)
}

func validMCPAuthority(config model.MCP) bool {
	return secureEndpoint(config.Issuer, false) && secureEndpoint(config.Audience, true) && secureEndpoint(config.JWKSURL, false)
}

// mcpKeySetMatchesIssuer keeps key retrieval inside the issuer origin. The
// runtime applies the same constraint before constructing an OIDC verifier.
func mcpKeySetMatchesIssuer(config model.MCP) bool {
	issuer, issuerErr := url.Parse(config.Issuer)
	keys, keysErr := url.Parse(config.JWKSURL)
	return issuerErr == nil && keysErr == nil && issuer.Scheme == keys.Scheme && issuer.Host == keys.Host
}

func validateMCPLists(config model.MCP) error {
	for _, origin := range config.Origins {
		if !secureEndpoint(origin, true) {
			return invalid("mcp_origin")
		}
	}
	for _, host := range config.Hosts {
		if host == "" || !safeText(host) || strings.ContainsAny(host, "/*?@#") {
			return invalid("mcp_host")
		}
	}
	if config.Address != "" {
		if _, _, err := net.SplitHostPort(config.Address); err != nil {
			return invalid("mcp_address")
		}
	}
	return nil
}
