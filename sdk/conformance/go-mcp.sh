#!/usr/bin/env bash
# Runs the official Go MCP client/session conformance fixture. The caller must
# build cquay, create a temporary synthetic policy and state directory, and
# grant its doctor --identity-only subject list/status/read on demo before use.
set -euo pipefail

: "${CQUAY_EXECUTABLE:?set an absolute repository-built cquay path}"
: "${CQUAY_CONFIG:?set an absolute synthetic policy path}"
: "${CQUAY_STATE_DIRECTORY:?set an absolute synthetic state path}"

exec env CQUAY_GO_MCP_CONFORMANCE=1 go test -tags=integration ./sdk/go -run '^TestOfficialMCPClientSessionAgainstSyntheticCquay$' -count=1
