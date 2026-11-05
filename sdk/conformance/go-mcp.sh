#!/usr/bin/env bash
# Runs the official Go MCP client/session conformance fixture. The caller must
# build cq, create a temporary synthetic policy and state directory, and
# grant its doctor --identity-only subject list/status/read on demo before use.
set -euo pipefail

: "${CQ_EXECUTABLE:?set an absolute repository-built cq path}"
: "${CQ_CONFIG:?set an absolute synthetic policy path}"
: "${CQ_STATE_DIRECTORY:?set an absolute synthetic state path}"

exec env CQ_GO_MCP_CONFORMANCE=1 go test -tags=integration ./sdk/go -run '^TestOfficialMCPClientSessionAgainstSyntheticCq$' -count=1
