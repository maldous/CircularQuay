# ADR 0001: Go core and evidence-bound assurance

Status: accepted design. Actual implementation and qualification are recorded in [the qualification matrix](../qualification.md).

## Decision

Use a single Go module and a small pure policy core with separately testable provider/platform/MCP adapters. Use upstream SOPS and native SDKs rather than implement cryptography. Ship native artifacts without requiring a language runtime on ordinary target hosts.

## Alternatives

Rust provides stronger ownership/data-race guarantees in safe code and Kani/Verus verification options. It now has official major-cloud SDKs; it is not rejected for lack of SDKs. Go fits the existing SOPS/OpenBao ecosystem, official MCP implementation and maintained SonarQube Community analysis with less integration ownership. Revisit this decision if code-level formal proof becomes the dominant requirement.

C# NativeAOT, bundled Python/TypeScript and proof-oriented languages remain possible but enlarge this project's platform/runtime/integration qualification surface. The official TypeScript and Python SDKs are explicit client boundaries alongside the Go core; they do not replace native distribution or justify another runtime language merely to claim verification.

## Boundaries

Go has an embedded runtime and garbage collection. SDK/string copies prevent guaranteed memory erasure. Keep first-party unsafe/cgo out of the portable policy core; review unavoidable OS interfaces separately. TLS/OS/provider correctness is not proved by language safety.

Use TLA+/TLC for bounded state-machine invariants plus conformance tests. A finite model is not end-to-end program verification. Gobra, Dafny, Kani or Verus would need a separately justified scope and trusted-base review.

## Consequences

Pin the actual build/audit toolchain. Separate native release, race/instrumented and Android build lanes. Test real artifacts and devices, not only cross-compilation. A clean Sonar result is reproducible profile evidence, not machine certification.
