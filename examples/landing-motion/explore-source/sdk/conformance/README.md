# SDK synthetic conformance fixtures

These fixtures are used only by hosted SDK qualification. Each run creates a
temporary policy and state directory, asks `cq doctor --identity-only` for
the runner's local subject, and writes a narrowly scoped synthetic grant for
that subject. They do not read a developer profile, cloud credential, or
provider secret.

The MCP fixture starts the repository-built `cq mcp serve --transport
stdio` process through the official caller-owned MCP client. It verifies the
authorization-filtered three-tool subset (`list_workloads`, `get_status`, and
`read_secret`) through the public SDK adapter, reads only a synthetic fixture
value, and confirms that status for an ungranted workload is denied. The
synthetic grant intentionally has no run permission.
