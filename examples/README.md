# Synthetic examples

`mcp-local.json` is an additional fabricated local MCP policy. Replace its placeholder `local:uid:1000` with `cquay doctor`'s kernel-derived `local_subject` (a SID on Windows). Its grant permits only discovery/status and explicit disclosure of the fabricated `DEMO_MESSAGE`; it does not permit executing actions. Protect the resulting policy and audit directory.

These values are deliberately fabricated. No example grants access to an actual provider or stores an operational key.

From the repository root:

```sh
cquay --config examples/demo.json plan
cquay --config examples/demo.json run --workload demo -- /usr/bin/env
```

The environment display is for this synthetic demo only. Use a real application rather than `env` when handling credentials. On Windows choose an absolute native executable; do not assume `/usr/bin/env`, Bash, or a language runtime exists.

Paths in a policy are resolved relative to that policy file. Copying a policy does not copy or grant its referenced identities. Source credentials and recipient private keys belong in separately protected stores.
