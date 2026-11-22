#!/usr/bin/env bash
# Runs the official Go MCP client/session conformance fixture. The caller must
# build cq, create a temporary synthetic policy and state directory, and
# grant its doctor --identity-only subject list/status/read on demo before use.
set -euo pipefail

: "${CQ_EXECUTABLE:?set an absolute repository-built cq path}"
: "${CQ_CONFIG:?set an absolute synthetic policy path}"
: "${CQ_STATE_DIRECTORY:?set an absolute synthetic state path}"

# Existing bootstrap/manual callers still run the exact original fixture.
# Only the explicit two-variable producer opts into coverage/retention; this
# branch supplies no coverage evidence and never skips the conformance test.
if [[ -z "${CQ_GO_BINARY+x}" && -z "${CQ_GO_MCP_COVERPROFILE+x}" ]]; then
    exec env CQ_GO_MCP_CONFORMANCE=1 go test -tags=integration ./sdk/go -run '^TestOfficialMCPClientSessionAgainstSyntheticCq$' -count=1
fi

# Partial or explicitly empty instrumentation is an error, not a silent fallback.
: "${CQ_GO_MCP_COVERPROFILE:?set a private absolute coverage output path}"
: "${CQ_GO_BINARY:?set the selected absolute pinned Go executable}"
[[ "$CQ_GO_MCP_COVERPROFILE" = /* ]]
[[ "$CQ_GO_BINARY" = /* ]]

# Instrument the already-required fixture once; this is SDK-only supplemental
# evidence, not the untagged Source profile or the whole-product tagged gate.
exec env CQ_GO_MCP_CONFORMANCE=1 "$CQ_GO_BINARY" test -tags=integration ./sdk/go -run '^TestOfficialMCPClientSessionAgainstSyntheticCq$' -count=1 -covermode=atomic -coverpkg=github.com/maldous/CircularQuay/sdk/go -coverprofile="$CQ_GO_MCP_COVERPROFILE"
