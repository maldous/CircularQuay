# Configuration repositories and CI

A shared GitHub repository is a straightforward starting point for a developer
or a CI job. Its public configuration describes providers, resource references,
workloads, and policy; secret values and bootstrap credentials stay outside it.
Give `cquay` a repository, tree, or file URL and select the configuration:

```sh
cquay bootstrap \
  --config-source https://github.com/example-org/cquay-config/tree/main/github \
  --config-path cquay.json \
  --output ./cquay.json
```

Replace the example organization, repository, and path with your own. A tree URL
selects a directory; `--config-path` selects the file within it. A file URL can
select the file directly. `--config-ref FULL_COMMIT_SHA` pins an exact revision.
When a branch or tag is supplied, `cquay` resolves and records its commit and
configuration digest before use. Updates are revalidated against the enrolled
repository and signing policy. Public readability does not grant permission to
change local trust, retrieve secrets, execute a workload, or install a service.

### Use the same configuration in GitHub Actions

With a verified `cquay` binary on the runner's PATH, a job can select its
configuration through the environment and run a declared workload:

```yaml
permissions:
  contents: read
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      CQUAY_CONFIG_SOURCE: https://github.com/example-org/cquay-config/tree/${{ vars.CQUAY_CONFIG_REF }}
      CQUAY_CONFIG_PATH: github/cquay.json
      CQUAY_CONFIG_REF: ${{ vars.CQUAY_CONFIG_REF }}
      CQUAY_CONFIG_SHA256: ${{ vars.CQUAY_CONFIG_SHA256 }}
    steps:
      - name: Run the configured build
        run: cquay flow run --workload ci-build --non-interactive
```

Set the repository or environment variables `CQUAY_CONFIG_REF` and
`CQUAY_CONFIG_SHA256` to the reviewed full commit SHA and configuration-file
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
or from an earlier step that writes `CQUAY_CONFIG_SOURCE` to `$GITHUB_ENV`.
GitHub [makes that file's entries available to subsequent steps](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#setting-an-environment-variable). For an
unattended first enrollment, both the full commit and configuration digest
are required. `cquay bootstrap --non-interactive --output ./cquay.json` materializes the
selected configuration for later commands without putting secret values in it.

Without an explicit source, `cquay` checks local configuration, then uses
administrator-enrolled discovery. An authenticated configuration service or
GitHub App can associate verified job claims with an approved configuration
reference. A configured repository convention can supply the same reference.
This enrollment is explicit: there is no universal GitHub configuration-service
endpoint, and a public repository or runner environment is not its own trust
anchor. DNS, device provisioning, and cloud metadata can supply verified hints;
they cannot authorize access by themselves.

[GitLab CI](https://docs.gitlab.com/ci/secrets/id_token_authentication/) and
[Azure Pipelines](https://learn.microsoft.com/en-us/azure/devops/pipelines/library/service-endpoints?view=azure-devops) use the same `cquay flow run` command and
configuration settings with their respective job identities and environment
mechanisms. Provider adapters detect the runner context, preserve explicit
overrides, and support extensible discovery. Publishing values to later steps
is an explicit policy-controlled operation through the CI provider's supported
mechanism; the pipeline and subsequent steps then have access to those values.

Explicit command options take precedence over `CQUAY_` configuration settings,
then local configuration, then enrolled discovery. With no override, an
existing local configuration is used before discovery. Provenance diagnostics
show which source won without printing credentials. Overrides cannot widen an
identity grant or bypass the enrolled trust policy. `cquay bootstrap` offers
a text interface and browser or displayed-URL authentication when interaction
is needed, and starts only services authorized by the selected enrollment.

[Documentation](README.md) · [CircularQuay](../README.md)
