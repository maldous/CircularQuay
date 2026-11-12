# CircularQuay Intel macOS wheel evidence

This branch retains five original GitHub Actions archives for CircularQuay
Python SDK qualification. The native receipt records the exact feature tree,
workflow, run, attempt, job and artifact identities. Each archive contains
two byte-identical builds of the same cryptography 50.0.2 wheel. Its filename
is the SHA-256 of the original ZIP. Consumers pin a Git commit and verify
the archive and both wheel copies against the receipt.

The retained executions used standard CPython 3.11.9, 3.12.10, 3.13.17, 3.14.10
and 3.15.2 on Intel macOS 15.7.9. The wheel tag records a macOS 10.15 deployment
target. The receipt covers neither execution on macOS 10.15 nor free-threaded
Python. The Python SDK dependency closures have their own qualification.

Each wheel carries the cryptography license texts and an embedded CycloneDX
Rust component inventory. The `notices/` tree preserves the OpenSSL 3.5.9
license and license texts from the exact Cargo registry archives. Each Cargo
source archive was checked against the SHA-256 in `cargo-registry.json`.
`notice-inventory.json` records the retained texts and their digests. The
OpenSSL source location and digest are recorded there as well. Source
provenance and build inputs are in the pinned [CircularQuay source tree](https://github.com/maldous/CircularQuay/tree/941adf3f84a49905d7244bb594fbf517dff9bc32/sdk/python).
The original jobs remain available in [GitHub Actions](https://github.com/maldous/CircularQuay/actions/runs/43167081230).
