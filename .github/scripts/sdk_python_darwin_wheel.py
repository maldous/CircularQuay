"""Stage a verified Intel Darwin qualification wheel from its immutable Git object."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import urllib.request
import zipfile
from pathlib import Path
from typing import Any

from sdk_python_lock import ValidationError, darwin_intake


MANIFEST = Path(__file__).resolve().parents[2] / "sdk/python/darwin-qualified-wheels.json"
CUSTODY = re.compile(
    r"git\+https://github\.com/maldous/CircularQuay\.git@"
    r"([a-f0-9]{40}):(archives/sha256/([a-f0-9]{64})\.zip)"
)
MAX_ARCHIVE = 16 * 1024 * 1024


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Restrict retrieval to the exact public repository object URL."""

    def redirect_request(self, *args: Any, **kwargs: Any) -> None:
        return None


def stage(version: str, output: Path) -> dict[str, Any]:
    """Read the pinned Git object and verify SHA-256 before staging the wheel."""
    manifest = json.loads(MANIFEST.read_bytes())
    choices = [
        row for row in manifest["qualified_repeat_builds"] if row["python"] == version
    ]
    if len(choices) != 1:
        raise ValidationError("CPython patch has no qualified wheel")
    selected = choices[0]
    locator = CUSTODY.fullmatch(selected.get("custody") or "")
    if locator is None:
        raise ValidationError("wheel has no valid Git custody binding")
    commit, path, expected_sha256 = locator.groups()
    if expected_sha256 != selected["artifact_sha256"]:
        raise ValidationError("Git object path does not match the native receipt")

    url = f"https://raw.githubusercontent.com/maldous/CircularQuay/{commit}/{path}"
    opener = urllib.request.build_opener(NoRedirect)
    with opener.open(url, timeout=60) as response:
        archive_bytes = response.read(MAX_ARCHIVE + 1)
    if len(archive_bytes) > MAX_ARCHIVE:
        raise ValidationError("qualified archive exceeds the retrieval limit")
    if hashlib.sha256(archive_bytes).hexdigest() != expected_sha256:
        raise ValidationError("retrieved archive SHA-256 differs from the native receipt")

    if not re.fullmatch(r"[a-f0-9]{40}", selected["custody_git_blob"]):
        raise ValidationError("missing Git blob provenance")

    filename = selected["wheel"]
    if not re.fullmatch(
        r"cryptography-50\.0\.2-cp3[0-9]+-abi3(?:\.abi3t)?-macosx_10_15_x86_64\.whl",
        filename,
    ):
        raise ValidationError("manifest wheel filename is outside the qualification scope")
    output.mkdir(mode=0o700)
    archive = output / "archive.zip"
    archive.write_bytes(archive_bytes)
    wheel = output / filename
    with zipfile.ZipFile(archive) as bundle:
        member = bundle.getinfo(f"build/one/dist/{filename}")
        if member.file_size > MAX_ARCHIVE:
            raise ValidationError("qualified wheel exceeds the size limit")
        wheel.write_bytes(bundle.read(member))
    intake = darwin_intake(archive, wheel, version, locator.group())
    intake.pop("wheel_uri")
    (output / "intake.json").write_text(
        json.dumps(intake, indent=2) + "\n", encoding="utf-8"
    )
    return intake


def main() -> int:
    """Stage one patch-specific archive into a fresh task directory."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--python", required=True)
    parser.add_argument("--output", type=Path, required=True)
    options = parser.parse_args()
    try:
        print(stage(options.python, options.output)["wheel"])
    except (KeyError, OSError, ValidationError, zipfile.BadZipFile) as error:
        parser.exit(1, f"Darwin wheel intake failed: {error}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
