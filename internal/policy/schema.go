package policy

import _ "embed"

// Schema is the single authoritative policy schema, also exported in releases.
//
//go:embed schema/policy-v1.schema.json
var Schema []byte
