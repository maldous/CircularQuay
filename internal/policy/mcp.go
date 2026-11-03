package policy

import (
	"net"
	"strings"

	"github.com/maldous/CircularQuay/internal/model"
)

func validateMCP(config model.MCP) error {
	if config.Issuer == "" && config.Audience == "" && config.JWKSURL == "" {
		return validateMCPLists(config)
	}
	if !secureEndpoint(config.Issuer, false) || !secureEndpoint(config.Audience, true) || !secureEndpoint(config.JWKSURL, false) {
		return invalid("mcp_authority")
	}
	return validateMCPLists(config)
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
