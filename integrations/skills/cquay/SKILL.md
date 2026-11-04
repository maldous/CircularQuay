---
name: cquay
description: Configure, inspect and run CircularQuay workloads through its CLI, official SDKs or optional MCP connection. Use for bootstrap, policy CRUD, CI flow, telemetry and scoped lifecycle administration; raw disclosure requires its own authorization.
license: Apache-2.0
metadata:
  compatibility: Requires the selected CircularQuay executable or official SDK/MCP adapter, protected policy and state, and dependencies for enabled features.
---

# cquay

`cquay`, pronounced **“see-key”**, connects existing providers to approved apps.
Use the installed release's help and capability contract. Product guides define
the complete v1 end state; release/qualification records establish the exact
available and tested artifact. A skill, plugin, SDK or agent name grants nothing.

## Select the boundary

- Use the CLI for operator-selected local policy, bootstrap and workload actions.
- Use an official Go, TypeScript or Python SDK when an application owns the
  integration. Keep its credentials, cancellation and process/session lifecycle
  explicit. Python/TypeScript local adapters require a separately authenticated,
  digest-selected native executable; package installation is not binary trust.
- Use optional MCP only through the configured stdio OS identity or authenticated
  HTTP resource. Discover its actual catalog; do not invent administrative tools
  or assume every client/server release exposes the same operations.

For task-specific commands and prerequisites, read
[operating workflows](references/operations.md). Read
[security boundaries](references/boundaries.md) before delivery, delegation,
raw disclosure, custody changes or incident recovery.

The current repository MCP catalog contains `list_workloads`, `get_status`,
`run_workload` and `read_secret`, subject to authorization filtering. Remote
configuration CRUD, managed-key grant administration and agent delegation are
v1 requirements; they are not tools in that catalog. Use local CLI/SDK policy
editing only under its separate operator authority. Current remote bootstrap
requires an explicit GitHub source, full commit and configuration digest; do
not attempt automatic discovery or sign-in based solely on the product design.

## Inspect before acting

Use `version`, `providers`, protected `config validate`, `doctor`, `plan` and
scoped `status` as appropriate. These inspect declarations or metadata; do not
retrieve a secret to validate a policy. Configuration reads may expose literal
policy content: minimize output and keep it in the authorized review context.
Unknown settings, unsupported profiles and unavailable capabilities fail closed.

Bootstrap uses local policy first unless an explicit override selects another
source. Authenticate and verify enrolled discovery; a DNS hint, CI variable or
public Git URL is not trust. Use immutable revision/digest pins for unattended
first enrollment. Never read shell profiles or unrelated credential caches.

## Apply the authorized task

Policy CRUD uses validated JSON, RFC 6901 pointers and whole-document SHA-256
preconditions. Re-read after a conflict rather than overwriting another writer.
Changes to grants, tenant mappings, trust, export, snapshots or native activation
need the task's corresponding authority; workload permission does not imply it.

Invoke only exact declared workloads/actions and constrained inputs. Do not
substitute an arbitrary shell, provider URL, executable or mutable script.
Deliver through the runtime and return safe status or allowlisted results.
Caller-controlled code with injected secrets is disclosure-capable.

CLI, UI/API and MCP administration use the same scoped authorization ceiling.
Delegation names grantor, recipient, workload/operation, timing and audit reason;
it cannot widen the grantor's authority. Use only the selected interface's
supported operations. Provider credentials, age private keys and refresh tokens
are bootstrap material, never ordinary model-visible management fields.

## Diagnose without disclosure

Configure telemetry through protected `observability` declarations and explicit
collector trust/credential references. Values, value hashes, tokens, raw errors,
arguments and provider bodies do not belong in logs, metrics, traces or hooks.
Required audit has a separate durability contract. Use synthetic fixtures for
failure, cancellation, overload and redaction checks.

Release awareness is notice-and-continue. `update check --json` is the explicit
inspection route; SDK/MCP/CI sessions never prompt or install updates. A failed
advisory check cannot waive an explicit local minimum version. Verify a new
artifact and obtain installation authority separately.

## Governed lifecycle administration

Read [lifecycle task boundaries](references/lifecycle.md) for canonical
hierarchy/defaults/locks, SCIM, workload OIDC/SPIFFE, posture/Conditional Access,
external policy engines, scanner gates and provider-native remediation. These
v1 requirements do not create tools or fields in the selected release. Use only
its verified capabilities under common validation/authorization/approval/audit.
GitHub/GitGuardian/issuer permissions and protected references remain separate;
no account connection, purchase, scan transmission or live mutation is implied.
Incident closure does not prove revocation or rotation. Warning/exception policy
cannot override issuer denial; a known exposure permits use only through a
separately approved finite exact-version exception, presented with a warning.
GitHub metadata requests require `hide_secret=true` and rejection of unhidden
secret responses. Return allowlisted metadata, never raw alerts or
replacement keys; never infer GitGuardian identifiers from a local secret hash.
OTel export is not immutable audit. Fix forward through approved migrations
without resetting locks, exposure, revocation or replay state.

## Raw disclosure and recovery

`read_secret` deliberately returns plaintext to the client/model context and
may enter its retention systems. It requires explicit disclosure authority for
one exact alias and a current server grant. No bulk export, bootstrap-key read
or alternate-account/cache workaround follows from a denial.

Revocation stops future authorized delivery; it cannot recall plaintext, copied
age private keys or released data keys. Offline expiry depends on trusted local
enforcement and cannot defeat a hostile endpoint administrator. Rotate exposed
application credentials at their issuer under the actual incident scope.
Preserve audit, replay and fallback state. Installation, production changes,
credential rotation, registry publication and history rewriting remain separate
actions with their own owner-authorized scope.
