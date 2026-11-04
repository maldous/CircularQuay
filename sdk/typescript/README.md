# CircularQuay TypeScript SDK

`@maldous/cquay-sdk` is the v1 typed client boundary for CircularQuay. It has
two explicit adapters: a caller-owned connected MCP client session for the
four current MCP tools, and a verified local `cquay` executable for the full
implemented command interface. It does not discover credentials, load shell
profiles, create a connection, install a binary, retry work, cache results or
start a service.

The package is versioned for the v1 release train (`1.0.0-dev.0` in this
source tree). Registry ownership and publication are release operations; they
are not established by this package manifest.

## MCP client

Install the official MCP TypeScript SDK at the pinned peer version, create and
connect its client using the transport you selected, then adapt that session:

```ts
import { CallToolResultSchema } from "@modelcontextprotocol/sdk/types.js";
import { CquayClient, officialMCPAdapter } from "@maldous/cquay-sdk";

const cquay = new CquayClient(officialMCPAdapter(session, CallToolResultSchema));
const workloads = await cquay.listWorkloads({ signal: abortController.signal });
const status = await cquay.getStatus(workloads[0]!);
```

`discoverTools`, `listWorkloads`, `getStatus`, and `runWorkload` return only
bounded, validated protocol data. `read_secret` is deliberately separate:

```ts
const disclosure = cquay.disclosure();
const secret = await disclosure.readSecret("api", "SERVICE_TOKEN");
useSecret(secret.reveal()); // plaintext is available only here
```

`SecretDisclosure` redacts `toString`, `toJSON`, and Node inspection. A value
already disclosed to a client or model cannot be recalled or zeroized by this
SDK. Provider versions are retained as opaque UTF-8 metadata, so provider
paths, RFC3339 forms, and other provider-defined version formats remain valid.
Present expired secret deadlines fail with the stable `expired` category;
malformed deadlines fail as `protocol` data.

## Local CLI client

An operator supplies an absolute executable path and its expected SHA-256.
The SDK checks the regular file and digest at construction and immediately
before each start. Configuration and state paths are required and must be
absolute. The child receives only the explicit `environment` object; the SDK
does not inherit the parent environment. Use it only for values that the
operator intends to give the local process.

```ts
import { LocalCLIClient } from "@maldous/cquay-sdk";

const cli = await LocalCLIClient.create({
  executablePath: "/opt/cquay/cquay",
  expectedSHA256: "<64 lower-case hex characters>",
  configPath: "/srv/cquay/cquay.json",
  stateDirectory: "/srv/cquay/state",
  environment: { LANG: "C" }
});

await cli.version(); // requires a compatible v1 executable
await cli.configSet("/workloads/api/environment", { APP_MODE: "test" });
await cli.doctor();
await cli.plan();
```

The local profile provides `bootstrap`, configuration CRUD (including `configInit`), `doctor`, `plan`,
`status`, `run`, `snapshot`, `apply`, `sync`, `flowRun`, `version`, and
`providers`, and explicit `updateCheck`. The SDK never checks for updates
unless the caller invokes `updateCheck`. JSON configuration values are written to standard input rather
than command-line arguments. `run` and `flowRun` inherit child output by
design, so they are not captured by the SDK. Every local process is started
without a shell, observes cancellation, is waited for, and reports only a
stable error category. Their returned `LocalResult.exitCode` records a normal
completed workload or flow exit, including a non-zero application exit.

`configSet` and `configReplace` accept portable I-JSON values: finite numbers
and integers in JavaScript's safe integer range. Use the native CLI's explicit
file input when an operator needs a wider JSON number representation.

Use the package's build, typecheck, and test commands in a controlled
development environment. Release qualification and published package evidence
are recorded in the repository's qualification documentation.
