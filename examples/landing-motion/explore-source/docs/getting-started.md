# Getting started

Build from source with Go 1.27.3 using the steps below, or select a verified
native artifact from the [GitHub Releases page](https://github.com/maldous/CircularQuay/releases).
Download verification, release-candidate selection, installation and removal
follow the [release and operator guide](../RELEASE.md#download-and-install-a-qualified-native-archive).
The example creates only a local synthetic fixture; it does not contact a
provider, read credentials or start a service. Actual published artifacts and
qualification are recorded in [the qualification matrix](qualification.md).

## Linux and macOS

```sh
umask 077
cd -P "$(mktemp -d)"
release_ref='<verified-release-tag>' # Select an existing authenticated release.
git clone --branch "$release_ref" https://github.com/maldous/CircularQuay.git cq
cd cq
go version # requires Go 1.27.3
go build -o ./cq ./cmd/cq

./cq init --output cquay.json
./cq --config cquay.json doctor
./cq --config cquay.json plan
./cq --config cquay.json run --workload demo -- /usr/bin/printf '%s\n' 'Synthetic workload completed.'
```

## Windows PowerShell

```powershell
$ReleaseRef = '<verified-release-tag>' # Select an existing authenticated release.
git clone --branch $ReleaseRef https://github.com/maldous/CircularQuay.git cq
Set-Location cq
go version # requires Go 1.27.3
go build -o .\cq.exe .\cmd\cq

.\cq.exe init --output cquay.json
.\cq.exe --config cquay.json doctor
.\cq.exe --config cquay.json plan
.\cq.exe --config cquay.json run --workload demo -- C:\Windows\System32\cmd.exe /d /c 'echo Synthetic workload completed.'
```

The final command runs an explicitly named, trusted operating-system program
with a synthetic value. Do not use environment-dumping commands when a policy
contains credentials. A declared workload command has access to values that
policy delivers to it.

For policy editing, conditional updates, doctor behavior, and all command
flags, see the [command reference](commands.md).

## First-run product flow

The [first-run contract](management-and-discovery.md#first-run-and-minimal-managed-registration)
checks an existing selected policy, then trusted discovery, then manual selection
when no admissible source is found. It authenticates through the resolved
identity and initializes the hosted key service unless specifically disabled
via command line or configuration. Resolve both opt-out surfaces, precedence
and enterprise locks before any hosted-service network call. Self-hosted/offline
profiles remain supported; an invalid policy or conflicting lock is repaired,
not discarded to obtain a permissive hosted default.

The default service is minimal identifier-to-age-private-key registration and
authorized retrieval with key-release windows, not a remote policy/configuration
orchestration subscription. No personal configuration, keys or secrets upload
by default. Free allowance and an upgrade opportunity do not imply an automatic
charge; event allowances, thresholds, prices and vendor remain undecided. The service can
stop future key retrieval, but age does not expire a copied key or prevent it
decrypting matching old ciphertext.

[Profile utilization](managed-key-service.md#profile-utilization-and-fair-metering)
counts hosted accesses, not keys/profiles: one daily fetch is one use, twenty
independent host fetches are twenty, local key reuse is zero. Automatic retries
of one logical access reconcile a single event; failed/denied pre-release calls
are not customer retrieval uses. No subscription is purchased automatically.

The synthetic commands above describe the selected executable's documented
route. They do not prove the full first-run/service contract is implemented or
supply new enrollment/opt-out flags; use the [command reference](commands.md)
and [qualification](qualification.md) for actual capability evidence.

## Connect your application or pipeline

Use [bootstrap and discovery](configuration-repositories.md) to select trusted
local or repository configuration, and `cq flow run --workload ID
--non-interactive` for an enrolled CI workload. Explicit overrides select the
context; they cannot widen authorization. Discovery and job environment markers
do not authenticate a source, user or provider identity on their own.

Applications choose the [official SDK](sdks.md) local adapter or an optional MCP
session. Start with synthetic values, validate the policy and check a denial
before connecting an operational provider. Add [observability and release
awareness](operational-configuration.md) through protected configuration, then
verify safe output with your selected collector. A release notice continues
without reading input; installation is a separate verified operator action.

## Choose an availability policy

| Mode | Intended use | Boundary |
| --- | --- | --- |
| Offline | Provisioned static values from local, approved artifacts | No central per-read audit or immediate remote revocation. |
| Online | Fresh provider retrieval or unwrap at each launch | Startup depends on the provider; auditing depends on the actual backend configuration. |
| Hybrid | Online retrieval with an explicitly approved encrypted fallback | Only qualifying availability failures permit fallback. Denial, TLS failure, malformed data, and expiry do not. |

A deployed snapshot is a delivery workflow for an offline or hybrid workload,
not a fourth availability mode. One-way publication into a platform's native
store gives the destination's permissions, rotation, and audit behavior to the
copy.

To use online, hybrid, snapshot, or publication operations, start with the
[complete Linux reference](../examples/linux-deployment.json) and the
[configuration reference](configuration.md). Copy it to an operator-owned
path and replace only its fabricated resource selectors and protected file
paths; keep actual credentials in the selected provider or protected identity
file. After that policy and its prerequisites are valid, use its declared
workload ID with the normal lifecycle commands:

```sh
CONFIG=/absolute/operator-owned/cquay.json
cq --config "$CONFIG" plan
cq --config "$CONFIG" run --workload reference-api -- /absolute/path/to/your/workload
cq --config "$CONFIG" snapshot --workload reference-api
cq --config "$CONFIG" sync --once --workload reference-api
```

The reference itself names nonexistent synthetic paths and cannot be run until
an operator supplies those external prerequisites. `snapshot` needs a completed
offline or hybrid snapshot declaration; `sync` needs at least one declared
target and only publishes values explicitly permitted by each binding.

Already delivered values cannot be recalled from process memory, old ciphertext, or an authorized client. Removing a recipient is not a substitute for rotating a compromised credential at its issuer.

## Try a credential-free workload

The `examples/` directory contains fabricated values only. From the repository root, with a built `cq` executable:

```sh
cq --config examples/demo.json plan
cq --config examples/demo.json run --workload demo -- /usr/bin/env
```

The second command deliberately demonstrates the child's environment using **synthetic data**. Do not use environment-dumping commands with real credentials. Normal workload commands should consume their assigned values without printing them.

On Windows, use an absolute native executable rather than assuming Bash exists. The CLI does not implicitly evaluate a shell command string. A program named by a workload still has the authority to read values delivered to it.

Native archives and verified offline bundles are the consumer installation route. Only enabled features need their dependencies. The normal CLI does not require an installed Go, Python, Node, Java, or Docker runtime. Source builds require the pinned Go toolchain; the independent audit environment has additional prerequisites.

[Documentation](README.md) · [CQ](../README.md)
