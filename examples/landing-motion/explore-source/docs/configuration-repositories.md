# Configuration repositories and CI

A shared GitHub repository is a straightforward starting point for a developer
or a CI job. Its public configuration describes providers, resource references,
workloads, and policy; secret values and bootstrap credentials stay outside it.
Give `cq` a repository, tree, or file URL and select the configuration:

```sh
cq bootstrap \
  --config-source https://github.com/example-org/cq-config/tree/FULL_COMMIT_SHA/github \
  --config-path cq.json \
  --config-ref FULL_COMMIT_SHA \
  --config-sha256 CONFIGURATION_SHA256 \
  --output ./cq.json
```

Replace the example organization, repository, path and pins with reviewed values.
`FULL_COMMIT_SHA` is the full 40-character lowercase commit;
`CONFIGURATION_SHA256` is the 64-character lowercase file digest obtained through
a trusted maintainer channel. The URL revision must agree with the commit pin.
A tree URL selects a directory; `--config-path` selects the file within it. A file URL can
select the file directly. The current CLI requires both pins for every remote
bootstrap, including interactive use; resolve a branch or tag through your
trusted review process before invoking it. The v1 enrollment contract also
requires repository/signature trust and revalidation of updates. Public
readability does not grant permission to change local trust, retrieve secrets, execute a workload, or install a service.

## Bootstrap command prerequisites

Use the selected executable's supported commands and enrollment profile. The
pinned retrieval command above materializes a new protected policy file; it
does not establish provider login or start a service. With no remote source,
bootstrap validates the selected local policy. `flow run` requires one supported
CI runner and a local policy or pinned remote configuration; runner markers
are context, not authentication. An unavailable discovery profile fails closed.

The remaining guide defines the complete v1 enrollment and identity contract.
Use [qualification](qualification.md) to confirm the selected artifact and its
supported profiles before relying on a discovery or authentication flow.

## Single-file or master-and-includes configuration

A self-contained configuration file remains fully supported. Alternatively,
a master explicitly enrolls department-owned or local source files and resolves
them into the same coherent typed policy. See the
[canonical composition contract](configuration-governance.md); splitting files
changes authorship/storage, not authorization or the required final model.
These paragraphs define v1 behavior, not additional flags or JSON keys to feed
to a runtime that has not implemented the selected composition profile.

The master declares trusted sources, immutable version/digest pins, permitted
namespaces and ownership, source order and typed conflict/override rules. Pins
bind exact bytes but do not enroll an untrusted publisher; source trust and
signing/enrollment policy are independent checks. A department file cannot
expand its namespace, grant ceiling, trusted sources, fetch credentials or
include permissions. Any transitive include requires master-authorized trust
and bounds; it cannot bootstrap its own authority from downloaded content.

Source order is explicit and repeatable, but never means unrestricted
last-writer-wins. Enterprise locks and denials survive every source and local
override. Defaults fill only unconstrained fields; conflicting objects/scalars
and multiple memberships follow the canonical typed combination rules or fail
validation. Reject cycles, repeated/aliased sources, duplicate qualified object
identities, namespace escape and unresolved references. Do not silently ignore a
conflicting department or personal source to produce a partial environment.

Bound file count, graph depth, individual and total bytes, parse/resolve time,
network requests, redirects and concurrency. Resolve declared relative paths
against the owning source's approved base/root, not the current directory or
an unrelated source, and revalidate protected file identity/pins at use. Deny
path traversal, link escape, ambient shell/.env interpolation and downloaded
content selecting privileged local files. Fetch all required authenticated
inputs into a protected candidate, validate the complete graph and effective
policy, then atomically publish one generation. No partial include becomes
active when another fails. Retain safe source/revision and field-level decision
provenance for authorized explain, without credential values or secret hashes.

## Corporate GitHub and personal local configuration

A developer selects a master that names an authenticated corporate GitHub
source and a protected personal local source. Selecting that master does not
permit the developer to redefine the employer's trust root or corporate grant
ceiling; corporate participation remains under its enrolled authority. Standalone
personal single-file use remains supported without enrolling corporate access.
The corporate revision, expected
file digest and signing/source trust come through the employer's enrolled
channel. The local source is explicitly enrolled and pinned/reviewed under its
own protected-file policy. Personal configuration declares protected local
credential references; actual personal values and decryption keys remain in
separately authorized local custody. Neither being logged into GitHub nor being
able to read a public repository establishes corporate workload access.

The master assigns separate corporate and personal namespaces, declares their
source order and conflict rules, and preserves locked enterprise access and
configuration requirements. Personal preferences apply only where authorized;
a personal file cannot replace a corporate binding, loosen a required gate,
choose a production context or supply a fallback identity for a corporate deny.
Multiple groups resolve under the same deterministic policy as a single file.

One resolved environment still selects values by exact workload/domain grants.
Do not export both namespaces wholesale to every child process. A workload
consuming from both domains needs the corresponding explicit authorities and
must satisfy all applicable locks. Corporate and personal provider identities,
decryption keys, approval, cache and incident/revocation state stay partitioned;
a key from one domain cannot unlock or authorize another by discovery or merge.

Resolve **all** configuration locally, including corporate sources. Read
through the client's VPN/network and credentials; no configuration content or
source credentials may reach a hosted backend/proxy/analytics/log, even for
corporate configuration. A browser honors CORS/file permissions; use explicit
native CLI/SDK/client integration rather than a server fetch proxy. Save only
locally and preserve all supported configuration options. Resolve personal material locally. Do not upload personal secret values,
local source content, keys or protected references to a corporate repository,
service, CI artifact, telemetry destination or support channel. Explain and
provenance expose only metadata authorized for their viewer/domain; personal
paths, object names or aggregate fingerprints must not leak through a corporate
explain request or exporter. A remote service cannot read a caller's local file
by implication: select an explicitly qualified local resolver boundary or
return unavailable/unsupported without requesting an upload or omitting a
required personal input. CLI/API/SDK/MCP and managed-service paths enforce the
same resolved-policy rules, not identical filesystem reach.

### Refresh, stale inputs and optional offline cache

The master explicitly selects update discovery, approved trust roots/signers,
version/digest review and refresh cadence for each source. Never follow a moving
branch, changed local file, new signer or redirect as silently trusted content.
A changed pin or trust mapping needs its corresponding approved revision. An
unavailable source, pin mismatch, expired authority or unresolved conflict cannot
relax corporate policy or infer production access. New activation requires the
complete admitted graph; continued use of an existing generation follows its
separately declared validity/refresh policy and current revocation checks.

Offline caching is optional and explicit per source/domain and resolved
generation. Declare whether it is permitted, encryption/custody, maximum age,
expiry, refresh failure behavior and which operations may consume it. Cached
inputs bind all source versions/digests and the effective policy/authority
generation; no mixing a new department file with an unreviewed old master.
Personal cache stays local in its own domain. Preserve locks and known
revocation/exposure state. A qualifying availability failure permits cache use
only inside the approved finite policy; trust/digest failure, denial, expiry or
missing required inputs never authorize fallback. A strict fresh-source policy
denies when freshness cannot be proved. Offline use cannot promise knowledge
of corporate changes learned only after disconnection or recall copied values.

### Hosted initialization respects composed policy

The [first-run flow](management-and-discovery.md#first-run-and-minimal-managed-registration)
uses the complete selected single-file or master/include configuration before
identity authentication and minimal hosted key service initialization. CLI or
configuration can explicitly disable hosted use; resolve either opt-out and
enterprise locks before any hosted-service network call, not after discovery
has already contacted that service. A locked conflict fails before network.
Corporate/personal composition uploads neither local files nor personal
credentials merely to enroll a client. Self-hosted/offline choices do not fall
back to a public hosted registration, and no manual selection infers a
production grant or payment consent.

## One precedence and trust model

Entry-point/source selection precedence is distinct from the master's typed
include-resolution rules. Selecting another entry file cannot widen a corporate
grant, replace its enrolled trust root or bypass a lock for a corporate operation.

Command options take precedence over `CQ_` configuration settings, local
configuration and enrolled discovery. Report safe provenance for the winning
source without printing credentials or downloaded secret values. Local config
is used before discovery when no override selects another source. An override
can narrow context or choose an enrolled profile; it cannot widen a signed grant.

DNS/DHCP, device provisioning, cloud-init and cloud metadata supply bounded
hints under the chosen enrollment. Authenticate the referenced service/artifact,
verify its issuer/repository and digest policy, and reject a substituted tenant,
resource or authority before retrieving a value. Interactive bootstrap may use
browser or displayed-URL sign-in; unattended CI uses its explicit workload
identity and never waits for a human prompt.

The same provenance reaches CLI, official SDK and optional agent entry points.
Only the enrolled user-service scope may be started. Service installation,
provider login and CI later-step export remain separately authorized actions.

### Use the same configuration in GitHub Actions

With a verified `cq` binary on the runner's PATH, a job can select its
configuration through the environment and run a declared workload:

```yaml
permissions:
  contents: read
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      CQ_CONFIG_SOURCE: https://github.com/example-org/cq-config/tree/${{ vars.CQ_CONFIG_REF }}
      CQ_CONFIG_PATH: github/cq.json
      CQ_CONFIG_REF: ${{ vars.CQ_CONFIG_REF }}
      CQ_CONFIG_SHA256: ${{ vars.CQ_CONFIG_SHA256 }}
    steps:
      - name: Run the configured build
        run: cq flow run --workload ci-build --non-interactive
```

Set the repository or environment variables `CQ_CONFIG_REF` and
`CQ_CONFIG_SHA256` to the reviewed full commit SHA and configuration-file
SHA-256, obtained through a trusted maintainer channel. The URL's commit must
match the pin. Missing pins or a URL/commit mismatch stop retrieval; a
configuration digest mismatch prevents use. Public readability alone cannot
authorize a configuration.

The configuration declares `ci-build` and its approved command. The job's
workload identity is verified against the configured issuer, audience,
repository, ref, and environment before any protected value is delivered.
[`id-token: write`](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-cloud-providers) permits requesting an identity token; access still requires
the receiving authority's explicit policy. An optional identity integration
uses that token; standalone configurations use their declared provider trust.

The source URL can also come from a GitHub repository or environment variable,
or from an earlier step that writes `CQ_CONFIG_SOURCE` to `$GITHUB_ENV`.
GitHub [makes that file's entries available to subsequent steps](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#setting-an-environment-variable). For an
unattended first enrollment, both the full commit and configuration digest
are required. `cq bootstrap --non-interactive --output ./cq.json` materializes the
selected configuration for later commands without putting secret values in it.

Without an explicit source, `cq` checks local configuration, then uses
administrator-enrolled discovery. An authenticated configuration service or
GitHub App can associate verified job claims with an approved configuration
reference. A configured repository convention can supply the same reference.
This enrollment is explicit: there is no universal GitHub configuration-service
endpoint, and a public repository or runner environment is not its own trust
anchor. DNS, device provisioning, and cloud metadata can supply verified hints;
they cannot authorize access by themselves.

[GitLab CI](https://docs.gitlab.com/ci/secrets/id_token_authentication/) and
[Azure Pipelines](https://learn.microsoft.com/en-us/azure/devops/pipelines/library/service-endpoints?view=azure-devops) use the same `cq flow run` command and
configuration settings with their respective job identities and environment
mechanisms. Provider adapters detect the runner context, preserve explicit
overrides, and support extensible discovery. Publishing values to later steps
is an explicit policy-controlled operation through the CI provider's supported
mechanism; the pipeline and subsequent steps then have access to those values.

Explicit command options take precedence over `CQ_` configuration settings,
then local configuration, then enrolled discovery. With no override, an
existing local configuration is used before discovery. Provenance diagnostics
show which source won without printing credentials. Overrides cannot widen an
identity grant or bypass the enrolled trust policy. `cq bootstrap` offers
a text interface and browser or displayed-URL authentication when interaction
is needed, and starts only services authorized by the selected enrollment.

[Documentation](README.md) · [CircularQuay](../README.md)
