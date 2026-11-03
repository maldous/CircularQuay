# Qualification matrix

This matrix distinguishes implementation from independent or live qualification. The source is in development; no production qualification, certification, availability SLA, or universal provider/client compatibility is claimed.

| Capability | Implementation | Operational evidence |
| --- | --- | --- |
| Strict policy and complete generations | In development; consult package tests | Independent review pending. |
| Local literal configuration | Synthetic example provided | Installed-artifact run pending. |
| SOPS/age and Transit | In development | Selected-version round trip, denial and native-key custody pending. |
| Cloud/SaaS readers | Connector-specific; inspect `cquay providers` | Live-provider qualification pending; mocks do not qualify accounts. |
| Native publication | Connector-specific | Target entitlement/identity and deployment tests pending. |
| Linux/systemd | Implementation/templates in development | Cold reboot/offline recovery pending on an operator-owned host. |
| Windows | Implementation/templates in development | Actual SID/ACL/DPAPI, no-login reboot and restore pending. |
| Termux | Recipe/implementation in development | Real-device post-unlock, DNS/TLS and background tests pending. |
| Containers | Tooling/templates in development | Image-history, runtime and recreation verification pending. |
| Local MCP | In development | Client/version and disclosure tests pending. |
| Hosted MCP | Experimental | Issuer/resource/client conformance and cross-user tests pending. |
| Sonar and full assurance | Harness in development | No independent completed gate claimed here. |
| Public distribution | Source publication in progress | No release, module-index, OCI or MCP Registry publication claimed. |

Record each qualification receipt with a source commit, artifact digest, platform/provider/client version, setup, test outcome and sanitized evidence link. Do not mark a row tested based on a plan, cross-compilation, generated manifest, or absence of a complaint.

Existing operational machines are outside this implementation's deployment scope. Operators install from documented artifacts and provide their own installation feedback.
