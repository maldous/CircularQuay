# Synthetic examples

`mcp-local.json` is an additional fabricated local MCP policy. Replace its placeholder `local:uid:1000` with `cquay doctor`'s kernel-derived `local_subject` (a SID on Windows). Its grant permits only discovery/status and explicit disclosure of the fabricated `DEMO_MESSAGE`; it does not permit executing actions. Protect the resulting policy and audit directory.

`linux-deployment.json` is a structurally and semantically valid, fully
declared Linux reference for a fabricated SOPS/age hybrid workload, systemd
credential projection, signed ciphertext repository, and optional hosted MCP
discovery/status grant with named identity scope mappings. Its
paths, recipient, service, subject, and HTTPS names are placeholders, not
provisioned resources or credentials. It deliberately has no executable or
`run` grant: add a reviewed absolute executable and its actual SHA-256 only
when enabling actions. Read [policy configuration](../docs/configuration.md)
before adapting it. Enroll the real signer and immutable initial commit before
using the repository; the fabricated commit is an explicit replacement value.

These values are deliberately fabricated. No example grants access to an actual provider or stores an operational key.

From the repository root:

```sh
cquay --config examples/demo.json config validate
cquay --config examples/demo.json plan
cquay --config examples/demo.json run --workload demo -- /usr/bin/printf '%s\n' 'Synthetic workload completed.'
```

The command uses only a synthetic fixture and does not print delivered values.
On Windows choose an absolute native executable; do not assume `/usr/bin/printf`,
Bash, or a language runtime exists.

Paths in a policy are resolved relative to that policy file. Copying a policy does not copy or grant its referenced identities. Source credentials and recipient private keys belong in separately protected stores.
