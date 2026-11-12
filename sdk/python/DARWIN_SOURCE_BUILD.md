# Darwin Intel cryptography source-build preparation

This is a proposed input set and native qualification plan for the existing,
approved `cryptography==50.0.2` source exception on Darwin `x86_64`. It is not an
adopted build closure, a successful native build, or a replacement for the six
current Darwin runtime/static locks. Keep the existing SDK checks, Intel support,
strict audits and integration owner. The core `cq-sdk` package still has no
runtime dependencies; this work concerns its optional MCP qualification closure.

## Inputs and current limits

| Input | Repository path | Meaning |
| --- | --- | --- |
| Exact allowed sdist | [`darwin-source-policy.json`](darwin-source-policy.json) | Only cryptography 50.0.2, its public PyPI URL and SHA-256 |
| Direct Python build pins | [`requirements-darwin-build.txt`](requirements-darwin-build.txt) | Maturin 1.16.0, CFFI 2.1.1 and setuptools 84.2.1 |
| Five build closures | `locks/build-darwin-py311.txt` through `locks/build-darwin-py315.txt` | Four wheel-only, hash-pinned packages per interpreter, including pycparser 3.12 |
| Python input provenance | [`darwin-build-inputs.json`](darwin-build-inputs.json) | Lock/file hashes, wheel URLs, upstream metadata hashes and source epoch |
| Native input proposal | [`darwin-native-toolchain.json`](darwin-native-toolchain.json) | Rust, Apple and OpenSSL identities, settings and incomplete acquisition state |

The five Python closures were resolved for Intel Darwin using cross-target
metadata. Their file/hash consistency and audits do not prove native execution.
The upstream sdist's `Cargo.toml` declares Rust **1.83.0 as its minimum**; the
selected compiler proposal is **Rust 1.99.1**, with an official archive hash.
`Cargo.lock` has 32 checksummed registry packages and no Git dependencies.

OpenSSL 3.5.9 has the expected SHA-256
`603f5602e2eef00d77fbd429d34dcd5822bb301757a1bc9cdb24c670f1eb859a`,
supplied by the operator in a checksum line naming `openssl-3.5.9.tar.gz`.
A separate download of the official source archive (53,279,637 bytes) matched
that digest. This verifies the acquired source bytes against the supplied
expectation; it is not a native acquisition, build or qualification receipt.
The earlier restricted checksum-link click was not retried. The native executor
must verify its own acquired archive before extraction or execution.

The Xcode 16.4/build 16F6 and SDK 15.5 values come from published runner-image
metadata, not the actual native executor. Before execution, obtain the actual
image/acquisition provenance, verify its Apple signing/trust evidence, and match
native readback to the proposal. Each CPython 3.11–3.15 input also needs an exact
patch version, native architecture and acquisition/provenance receipt. Minor
version wheel tags alone are not Python toolchain pins.

The proposed `MACOSX_DEPLOYMENT_TARGET=10.15` matches the strictest minimum in
the selected Python build wheels (CFFI). It does not prove that cryptography,
Rust, CPython or the completed SDK supports execution on macOS 10.15. Validate
the resulting Mach-O minimums and the actual supported OS range; do not silently
raise a supported deployment minimum to make a build pass.

## Native preparation after the missing inputs are verified

Run only on the admitted native Intel macOS executor, in the current owner’s
task checkout. This document does not dispatch a job or change a workflow.
Use task-local directories; do not install Rust, Python or OpenSSL into system
paths, change global `xcode-select`, or use Apple's system OpenSSL/LibreSSL.
Read the source policy, both manifests and current project instructions first.

The following commands are a template for each selected CPython input. Supply
`CQ_NATIVE_PYTHON` as its absolute, acquisition-verified native executable and
`CQ_ARCHIVES` as the protected directory containing the exact verified archives.
The admitted native owner must record those input identities, not infer them
from these variable names. Run with a reviewed environment without credentials.

```sh
set -euo pipefail
umask 077
export CQ_REPO="$(git rev-parse --show-toplevel)"
export CQ_BUILD_ROOT="$(mktemp -d "${RUNNER_TEMP:?}/cq-darwin-build.XXXXXX")"
: "${CQ_NATIVE_PYTHON:?absolute verified native CPython required}"
: "${CQ_ARCHIVES:?protected verified input directory required}"
export DEVELOPER_DIR=/Applications/Xcode_16.4.app/Contents/Developer
test "$(uname -s)" = Darwin
test "$(uname -m)" = x86_64
sw_vers
/usr/bin/xcodebuild -version
/usr/bin/xcrun clang --version
/usr/bin/xcrun --sdk macosx --show-sdk-version
/usr/bin/codesign --verify --deep --strict /Applications/Xcode_16.4.app
"$CQ_NATIVE_PYTHON" -I -c 'import platform, sys; assert sys.platform == "darwin"; assert platform.machine() == "x86_64"; print(sys.executable, sys.version)'
"$CQ_NATIVE_PYTHON" -m pip --version
export SDKROOT="$(/usr/bin/xcrun --sdk macosx --show-sdk-path)"
export MACOSX_DEPLOYMENT_TARGET=10.15
export SOURCE_DATE_EPOCH=1790782204
```

Require the observed Xcode/build/SDK and exact Python identity to match the
reviewed input receipt before continuing. Retain the image version and tool
readback, including the acquired Python's pip frontend identity. A signature
check alone does not establish acquisition provenance.
`SOURCE_DATE_EPOCH` is the whole-second sdist upload timestamp recorded in
`darwin-build-inputs.json`; verify equality with both manifests rather than
substituting the current time.

Verify all archive bytes before extraction or executing an installer. This
guard checks the recorded archive digests before extraction or execution:

```sh
"$CQ_NATIVE_PYTHON" -I - <<'PY'
import hashlib, json, os, re
from pathlib import Path
repo, archives = Path(os.environ['CQ_REPO']), Path(os.environ['CQ_ARCHIVES'])
sdk = repo / 'sdk/python'
source = json.loads((sdk / 'darwin-source-policy.json').read_text())
native = json.loads((sdk / 'darwin-native-toolchain.json').read_text())
inputs = json.loads((sdk / 'darwin-build-inputs.json').read_text())
assert inputs['source_date_epoch'] == native['build_settings']['source_date_epoch'] == int(os.environ['SOURCE_DATE_EPOCH'])
for entry in (source, native['rust'], native['openssl']):
    digest = entry['sha256']
    if not isinstance(digest, str) or re.fullmatch(r'[0-9a-f]{64}', digest) is None:
        raise SystemExit('Unverified archive digest; acquisition/build remains held')
    archive = archives / entry['url'].rsplit('/', 1)[1]
    assert hashlib.sha256(archive.read_bytes()).hexdigest() == digest, archive.name
for relative, digest in [(inputs['source_policy'], inputs['source_policy_sha256']),
                         (inputs['direct_requirements'], inputs['direct_requirements_sha256'])]:
    assert hashlib.sha256((repo / relative).read_bytes()).hexdigest() == digest
for closure in inputs['closures']:
    assert hashlib.sha256((repo / closure['lock']).read_bytes()).hexdigest() == closure['lock_sha256']
PY
```

Extract only those verified releases into private task directories with safe
archive-member handling. Install the standalone Rust archive using its
`install.sh --prefix="$CQ_BUILD_ROOT/rust"`; keep its installer log and actual
`rustc --version --verbose`/`cargo --version` readback. Put only that prefix's
`bin` ahead of the reviewed native tools on `PATH`, and use a task-local
`CARGO_HOME`. Do not run an unpinned `rustup update` or package-manager upgrade.

### Acquire the Python and Cargo closure once

For each minor, set `CQ_MINOR` to `311`, `312`, `313`, `314` or `315`, and check
that it matches `CQ_NATIVE_PYTHON`. Use a separate wheelhouse for each minor.
During this admitted acquisition phase, download only the locked wheels, then
install from that local wheelhouse with network discovery disabled:

```sh
export CQ_MINOR="$("$CQ_NATIVE_PYTHON" -I -c 'import sys; print(f"{sys.version_info.major}{sys.version_info.minor}")')"
case "$CQ_MINOR" in 311|312|313|314|315) ;; *) exit 1 ;; esac
export CQ_BUILD_LOCK="$CQ_REPO/sdk/python/locks/build-darwin-py${CQ_MINOR}.txt"
export CQ_WHEELHOUSE="$CQ_BUILD_ROOT/wheels-$CQ_MINOR"
mkdir -m 0700 "$CQ_WHEELHOUSE"
"$CQ_NATIVE_PYTHON" "$CQ_REPO/.github/scripts/sdk_python_lock.py" \
  --verify-lock "$CQ_BUILD_LOCK" \
  --requirements "$CQ_REPO/sdk/python/requirements-darwin-build.txt"
"$CQ_NATIVE_PYTHON" -m pip --isolated download --no-cache-dir \
  --index-url https://pypi.org/simple --require-hashes --only-binary=:all: \
  --dest "$CQ_WHEELHOUSE" -r "$CQ_BUILD_LOCK"
```

Bind the downloaded filenames and hashes to `darwin-build-inputs.json`, retain
the receipt, and reject extra wheels. These are build-only inputs; installing
them does not replace any SDK runtime lock.

In a verified cryptography source copy, verify the manifest's hashes for
`pyproject.toml`, root `Cargo.toml` and `Cargo.lock`, including
`[tool.maturin] locked = true`. With the selected Rust executable, acquire the
locked crate closure into a task-local vendor directory:

```sh
export CARGO_HOME="$CQ_BUILD_ROOT/cargo-acquisition"
mkdir -m 0700 "$CARGO_HOME"
# CQ_CRYPTOGRAPHY_SOURCE is the verified extracted cryptography-50.0.2 root.
cd "$CQ_CRYPTOGRAPHY_SOURCE"
"$CQ_BUILD_ROOT/rust/bin/cargo" vendor --locked --versioned-dirs "$CQ_BUILD_ROOT/vendor" \
  > "$CQ_BUILD_ROOT/cargo-source.toml"
```

Check all 32 registry archives against `Cargo.lock` checksums, retain the
vendored file/checksum inventory and source-replacement configuration, and
confirm no Git dependency appeared. Preserve the exact lock bytes. Do not
resolve newer crates or retry denied acquisition through another transport.
All later builds must use this verified input store offline.

## Two clean native builds per interpreter

Repeat the following in two new task directories with independent source
extractions, Python environments, OpenSSL prefixes, Cargo homes and targets.
Reuse only the verified immutable input store. Never reuse the first build's
compiled outputs or pip cache. Record all commands, environment settings,
source/input hashes and bounded tool/version logs.

For each run, set `CQ_CASE` to its private directory, and set
`CQ_OPENSSL_SOURCE` and `CQ_CRYPTOGRAPHY_SOURCE` to its verified extracted
source roots. Rust remains at the verified task-local prefix. Build OpenSSL
statically with the selected Apple compiler; retain its tests and configuration:

```sh
export PATH="$CQ_BUILD_ROOT/rust/bin:$PATH"
export CC="$(/usr/bin/xcrun --find clang)"
export CARGO_BUILD_JOBS=2
export CARGO_BUILD_TARGET=x86_64-apple-darwin
export OPENSSL_DIR="$CQ_CASE/openssl"
export OPENSSL_STATIC=1
export OPENSSL_NO_VENDOR=1
cd "$CQ_OPENSSL_SOURCE"
./Configure darwin64-x86_64-cc no-shared -fPIC \
  --prefix="$OPENSSL_DIR" --openssldir="$OPENSSL_DIR/ssl"
make -j2
make test
make install_sw
"$OPENSSL_DIR/bin/openssl" version -a
"$CQ_NATIVE_PYTHON" -m venv "$CQ_CASE/python"
export CQ_BUILD_PYTHON="$CQ_CASE/python/bin/python"
"$CQ_BUILD_PYTHON" -m pip --isolated install --no-cache-dir --no-index \
  --find-links "$CQ_WHEELHOUSE" --require-hashes --only-binary=:all: \
  -r "$CQ_BUILD_LOCK"
export CARGO_HOME="$CQ_CASE/cargo"
export CARGO_TARGET_DIR="$CQ_CASE/target"
mkdir -m 0700 "$CARGO_HOME"
cp "$CQ_BUILD_ROOT/cargo-source.toml" "$CARGO_HOME/config.toml"
export CARGO_NET_OFFLINE=true
export PIP_NO_INDEX=1
cd "$CQ_CRYPTOGRAPHY_SOURCE"
cargo metadata --locked --offline --format-version 1 > "$CQ_CASE/cargo-metadata.json"
"$CQ_BUILD_PYTHON" -m pip --isolated wheel --no-cache-dir --no-index \
  --no-build-isolation --no-deps --wheel-dir "$CQ_CASE/dist" .
```

The wheel build uses the preinstalled hash-locked backend, disables build
isolation and dependency resolution, and uses upstream Maturin's `locked=true`
with offline Cargo. Ensure the native executor's build phase cannot acquire
additional network inputs; `--no-index` alone is not a network sandbox. Do not
allow a missing crate or backend requirement to trigger an online retry.

## Verify outputs before changing consumer locks

For every Python minor and both clean builds:

1. Require one cryptography 50.0.2 wheel with reviewed interpreter/ABI/platform
   tags. Retain its SHA-256, wheel `RECORD`, source-policy proof, input inventory
   and native toolchain receipt. Compare clean-build hashes; investigate a
   mismatch and retain both results rather than declaring reproducibility.
2. Inspect every Mach-O extension using `file`, `lipo -info`, `otool -L` and
   `otool -l`. Require Intel architecture, inspect SDK/deployment load commands
   and ensure no dynamic `libssl`/`libcrypto`, Homebrew, build-directory or
   task-local Rust/OpenSSL path is required at runtime. Do not infer static
   linkage from `OPENSSL_STATIC=1` alone.
3. Install the exact wheel with `--no-index --no-deps` into a fresh task-local
   native environment with the separately verified runtime dependency closure.
   Run `pip check`, isolated imports and cryptographic operations exercising the
   selected OpenSSL backend (key generation, signatures, encryption/decryption
   and serialization). Record `cryptography.__version__` and
   `default_backend().openssl_version_text()`; require the selected versions.
   Qualify the stated OS range on actual supported systems before claiming it.
4. Once the native outputs and complete proposed consumer closures below have
   been reviewed, run the SDK's existing coverage/static/security checks and
   strict dependency audits without exceptions. Build/install its wheel into
   clean native consumers and exercise installed local-client conformance
   against the independently qualified native `cq`. Preserve all existing
   thresholds, denial/cleanup checks and release/distribution controls.

Only after native output review and the clean repeat, prepare proposed changes
to **all six** consumer locks: `locks/runtime-darwin-py311.txt` through
`locks/runtime-darwin-py315.txt`, plus `locks/ci-darwin-py314.txt`. Resolve the
complete target-specific closures using the exact qualified cryptography wheel,
retain reports and update the existing lock provenance. Audit the complete six
proposals, not just cryptography or the four Python build inputs.

The current lock report helper accepts public PyPI wheels; it does not admit a
locally built wheel merely because its hash exists. The integration owner must
separately review the source-exception consumer path, immutable wheel custody,
source/build proof and offline hash-enforced installation before adoption.
Preserve ordinary packages' binary-only/hash gates; do not broaden the source
allowance or make a local wheel look like an upstream PyPI artifact. No consumer
enablement or workflow change is included in this input proposal.

The remaining work is native archive acquisition and byte verification, actual
Apple/Python/Rust verification, locked crate acquisition, native builds and
their qualification, followed by reviewed consumer integration. Earlier
inaccessible hosted artifacts are not required inputs to this plan and must not
be recovered through a retry or alternate extraction route. Preparation and
metadata checks are not native runtime results.
