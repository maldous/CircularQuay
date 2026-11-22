package mcp

import "github.com/maldous/CircularQuay/internal/model"

// dpopMetadata advertises a global requirement only when every named authority
// requires proofs. A mixed resource still enforces each authority's own policy.
func dpopMetadata(config model.MCP) ([]string, bool) {
	supported := false
	required := len(config.Authorities) > 0
	for _, authority := range config.Authorities {
		if authority.SenderConstraint == nil {
			required = false
		} else {
			supported = true
		}
	}
	if !supported {
		return nil, false
	}
	return []string{"ES256", "RS256"}, required
}
