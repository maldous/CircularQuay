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

The Xcode 16.4/build 16F6 and SDK 15.5 proposal is now accompanied by completed
native Apple and Rust/Cargo receipts retained by the native owner. Reuse those
receipts and bind each build to its actual tool identities. The exact CPython
pins below come from staged native package evidence; they do not certify an
installed framework or execution. Minor-version wheel tags alone are not
Python toolchain pins.

The five reviewed inputs are **3.11.9, 3.12.10, 3.13.17, 3.14.10 and 3.15.2**.
Their installer, framework/interpreter, header and bundled-pip digests are in
[`darwin-cpython-inputs.json`](darwin-cpython-inputs.json), bound to
[run 43153887957, attempt 1](https://github.com/maldous/CircularQuay/actions/runs/43153887957).
All five signed payloads passed staged provenance. The three archive jobs
passed their archive hashes; both cached-framework jobs stopped before Python
or pip execution at root-protection preflight. Use fresh official installation
for those inputs rather than rechecking or changing the cached frameworks.

The proposed `MACOSX_DEPLOYMENT_TARGET=10.15` matches the strictest minimum in
the selected Python build wheels (CFFI). It does not prove that cryptography,
Rust, CPython or the completed SDK supports execution on macOS 10.15. Validate
the resulting Mach-O minimums and the actual supported OS range; do not silently
raise a supported deployment minimum to make a build pass.

## Native preparation after the missing inputs are verified

Run only on the admitted native Intel macOS executor, in the current owner’s
task checkout. This document does not dispatch a job or change a workflow.
Use task-local directories for build inputs, build outputs and receipts. Do not
install Rust or OpenSSL into system paths, change global `xcode-select`, or use
Apple's system OpenSSL/LibreSSL.

For this qualification batch, the exact CPython input record permits the
official framework installer on a fresh disposable Intel macOS GitHub-hosted
job dedicated to one recorded patch version. Before installation, verify the
exact acquired archive/package hashes, Apple notarization and PSF installer
identity against [`darwin-cpython-inputs.json`](darwin-cpython-inputs.json).
Select only the required framework component from that verified installer.
This route permits its system-framework installation on that disposable runner
only; it does not apply to Foo, shared hosts or long-lived executors. Do not
repair arbitrary host permissions or execute an unidentified cached framework.
Do not run wrapper scripts that upgrade pip or install unrelated packages.
Record the runner image, package digest/signer, installed receipt, protected
paths, signed binary hashes, native version/architecture, and the recorded pip
frontend hash/version. Retain the receipts and discard the runner after the job.
This input receipt is not a wheel-build or consumer-lock qualification.

Read the source policy, manifests and current project instructions first.
Reuse the retained native owner's completed Apple Xcode/SDK and Rust/Cargo
receipts. The initial archive flags in the native manifest do not supersede
those newer receipts or require repeating the completed provenance batch.

Initialize one private batch root before preparing its framework driver:

```sh
set -euo pipefail
umask 077
export CQ_REPO="$(git rev-parse --show-toplevel)"
export CQ_BUILD_ROOT="$(mktemp -d "${RUNNER_TEMP:?}/cq-darwin-build.XXXXXX")"
```

### Fresh framework and task-local pip operating path

Use one fresh disposable job per exact patch version in the input record. For
archive inputs, verify the recorded outer archive hash and signed installer
payload hash. For direct package inputs, verify the recorded installer hash.
Use the already established PSF signer/notarization and payload checks; do not
repeat the completed five-input discovery batch or execute archive wrappers.
The component's provenance derives from its verified official installer. Select
only the recorded `package_id` framework component, without documentation,
launcher, certificate, pip-upgrade or unrelated package components.

Install that component through the normal macOS installer on this disposable
job. Verify the resulting installer receipt, framework and interpreter hashes,
protected paths, exact patch version and Intel architecture against the record.
Set `CQ_FRAMEWORK_PYTHON` to that verified framework interpreter. Keep it at its
signed absolute framework location; do not relocate or rewrite Mach-O linkage.

Create the task-local driver environment with
`"$CQ_FRAMEWORK_PYTHON" -I -m venv --without-pip "$CQ_BUILD_ROOT/driver"`.
Set `CQ_NATIVE_PYTHON="$CQ_BUILD_ROOT/driver/bin/python"`. Bootstrap only the
recorded bundled pip wheel into that environment, with its SHA-256 checked
before use, no index and no dependencies. This avoids an implicit `ensurepip`
bootstrap adding other packages or selecting a different frontend. Run this
with `CQ_PYTHON_PATCH` selecting the exact reviewed input:

```sh
: "${CQ_PYTHON_PATCH:?exact reviewed patch version required}"
export CQ_PYTHON_PATCH
cq_bootstrap_recorded_pip() {
"$1" -I - <<'PY'
import hashlib, json, os, runpy, sys
from pathlib import Path
sdk = Path(os.environ['CQ_REPO']) / 'sdk/python'
record = json.loads((sdk / 'darwin-cpython-inputs.json').read_text())
selected = next(x for x in record['inputs'] if x['python'] == os.environ['CQ_PYTHON_PATCH'])
assert sys.version.split()[0] == selected['python']
import ensurepip
wheel = Path(ensurepip.__file__).parent / '_bundled' / selected['bundled_pip']['name']
assert hashlib.sha256(wheel.read_bytes()).hexdigest() == selected['bundled_pip']['sha256']
sys.path.insert(0, str(wheel))
sys.argv = ['pip', '--isolated', '--disable-pip-version-check', 'install',
            '--no-index', '--no-deps', str(wheel)]
runpy.run_module('pip', run_name='__main__')
PY
}
cq_bootstrap_recorded_pip "$CQ_NATIVE_PYTHON"
```

Record the resulting task-local pip version and installation location. Use the
existing per-minor hash-locked build closure from there. The selected framework
is a verified runtime input; all Rust/OpenSSL installations, environments,
build caches, outputs and receipts remain task-local. No framework installation
or package execution is performed by this source update.

The following commands are a template for each selected CPython input. Supply
`CQ_NATIVE_PYTHON` as its absolute, acquisition-verified native executable and
`CQ_ARCHIVES` as the protected directory containing the exact verified archives.
The admitted native owner must record those input identities, not infer them
from these variable names. Run with a reviewed environment without credentials.

```sh
set -euo pipefail
umask 077
: "${CQ_REPO:?selected checkout required}"
: "${CQ_BUILD_ROOT:?initialized private batch root required}"
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
"$CQ_NATIVE_PYTHON" -m venv --without-pip "$CQ_CASE/python"
export CQ_BUILD_PYTHON="$CQ_CASE/python/bin/python"
cq_bootstrap_recorded_pip "$CQ_BUILD_PYTHON"
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

Native inputs, builds and qualification must be bound to their original
receipts before reviewed consumer integration. The qualified repeat-build
record below supplies the later native output evidence; input preparation alone
does not supply that evidence. Earlier
inaccessible hosted artifacts are not required inputs to this plan and must not
be recovered through a retry or alternate extraction route. Preparation and
metadata checks are not native runtime results.

## Qualified native wheel consumer intake

`darwin-qualified-wheels.json` records only the five qualified repeat-build
outputs from run 43167081230 (3.14.10 uses attempt 2). It binds the original
native receipt, exact wheel and archive hashes, source policy and workflow.
The receipt records execution on Intel macOS 15.7.9; the 10.15 wheel tag does
not establish execution on 10.15 or a free-threaded interpreter.

The five original archives and redistribution notices are retained on the
[`artifacts/darwin-cryptography-50.0.2` branch](https://github.com/maldous/CircularQuay/tree/f8b06fa61ff41039c78d29379b8754be4bb5863d).
The manifest binds each ZIP to the exact Git commit and blob as well as the
archive and wheel SHA-256 hashes. Hosted CI stages the wheel over the public
commit-addressed route with read-only permissions:

```sh
python .github/scripts/sdk_python_darwin_wheel.py \
  --python "$EXACT_PYTHON_PATCH" --output "$FRESH_TASK_WHEEL_DIRECTORY"
```

Intake verifies the SHA-256 of the commit-addressed archive before reading the
expected wheel.
It verifies both clean-build copies before the wheel is eligible for resolution.
Resolve the **complete** closure using the
exact local wheel as an explicit resolver input, ordinary binary-only PyPI
packages, and the corresponding requirements file. Retain the actual pip JSON
report. Use the admitted exact interpreter and a fresh resolver environment;
this resolves dependencies as well as the explicitly selected local wheel:

```sh
"$NATIVE_PYTHON" -m pip --isolated --disable-pip-version-check install \
  --dry-run --ignore-installed --only-binary=:all: \
  --index-url https://pypi.org/simple --report "$REPORT" \
  -r "$REQUIREMENTS" "$QUALIFIED_WHEEL"
```

Generate its lock with the existing helper and these additional flags:

```sh
python .github/scripts/sdk_python_lock.py \
  --report "$REPORT" --requirements "$REQUIREMENTS" \
  --output "$PROPOSED_LOCK" --receipt "$LOCK_RECEIPT" --revision "$SOURCE_HEAD" \
  --darwin-archive "$RETAINED_ARTIFACT_ZIP" --darwin-wheel "$QUALIFIED_WHEEL" \
  --darwin-python "$EXACT_PYTHON_PATCH" --darwin-custody "$REVIEWED_IMMUTABLE_ID"
```

The custody value records the already-reviewed object/version identity; passing
an arbitrary value does not approve a destination. It must exactly match the
non-null binding in the reviewed qualified-wheel record. The helper hashes both
inputs, verifies the two wheel copies at the exact clean-build archive paths, and
admits only that local cryptography 50.0.2 wheel for the exact native
Darwin/x86_64/Python-patch report. Other packages still require public PyPI
wheel URLs and SHA-256 hashes. The receipt retains custody and native evidence,
without treating the transient local wheel URI as artifact identity.

Prepare runtime-darwin-py311 through py315 and ci-darwin-py314 separately. The
static closure reuses the qualified 3.14.10 wheel. Bind each complete resolver
report/lock digest and intake record in the existing runtime/static lock
provenance. Populate a reviewed wheelhouse with all locked binary dependencies,
then verify fresh offline installation and audit each entire closure:

```sh
"$FRESH_CONSUMER_PYTHON" -m pip --isolated install \
  --no-index --find-links "$REVIEWED_WHEELHOUSE" \
  --require-hashes --only-binary=:all: -r "$PROPOSED_LOCK"
"$AUDIT_PYTHON" -m pip_audit --strict --disable-pip --require-hashes -r "$PROPOSED_LOCK"
```

The audit interpreter uses the existing reviewed audit toolchain. Run these
and the applicable existing SDK/installed native-client checks. Preserve all
thresholds and denial/cleanup checks. No replacement consumer lock, workflow
enablement, audit pass or installed-SDK result is supplied by this intake
preparation itself.
