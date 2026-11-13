#!/usr/bin/env python3
"""Build the static CircularQuay documentation explorer from a frozen local snapshot."""

from __future__ import annotations

import argparse
import hashlib
import json
import posixpath
import re
import subprocess
from collections import Counter
from html import escape, unescape
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import quote, unquote, urlparse

from markdown_it import MarkdownIt


# This explorer deliberately snapshots the current local public-documentation
# worktree. BASE_REVISION is repository context only: generated source links
# must never claim that dirty local source is published at that revision.
CURRENT_HEAD = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=Path(__file__).resolve().parent.parent.parent, text=True).strip()
CURRENT_BRANCH = subprocess.check_output(["git", "branch", "--show-current"], cwd=Path(__file__).resolve().parent.parent.parent, text=True).strip() or CURRENT_HEAD
# These are reassigned from explore-source/manifest.json for every ordinary
# build. They only observe current Git state during --refresh-local-source.
BASE_REVISION = ""
REPOSITORY_CONTEXT_REF = "feature/initial-implementation"
REPOSITORY_REFERENCE_NOTE = "Repository reference (may differ from this local snapshot)"
REVISION = "working-tree"
REPOSITORY = "https://github.com/maldous/CircularQuay"
SNAPSHOT_DATE = "2026-11-13"
ROOT = Path(__file__).resolve().parent
REPOSITORY_ROOT = ROOT.parent.parent
CONTENT = ROOT / "explore-content"
ASSETS = CONTENT / "assets"
SOURCE = ROOT / "explore-source"
SOURCE_MANIFEST = SOURCE / "manifest.json"
PRODUCT_INPUTS = json.loads((ROOT / "explore-documents.json").read_text())
PRODUCT_DOCUMENTS = frozenset(PRODUCT_INPUTS["documents"])
PRODUCT_ASSETS = tuple(PRODUCT_INPUTS["assets"])
# Presentation-only copy for the landing explorer. Frozen source bytes and
# their recorded hashes remain the source of historical/documentary truth.
PRESENTATION_COPY = (
    ("The right secrets, ready for your apps.", "Get the right secrets ready."),
    ("Get the right secrets ready for your apps.", "Get the right secrets ready."),
)


def presentation_copy(source: str) -> tuple[str, list[dict[str, int | str]]]:
    """Apply only the approved landing slogan while preserving frozen source."""
    rendered = source
    mappings: list[dict[str, int | str]] = []
    for old, new in PRESENTATION_COPY:
        count = rendered.count(old)
        if count:
            rendered = rendered.replace(old, new)
            mappings.append({"from": old, "to": new, "replacements": count})
    return rendered, mappings


def local_paths() -> list[str]:
    # Public inputs are explicit: never discover maintenance or newly added
    # private Markdown through a recursive repository scan.
    paths = {path for path in PRODUCT_DOCUMENTS if (REPOSITORY_ROOT / path).is_file()}
    return sorted(paths, key=lambda value: (value != "README.md", value.lower()))


def local_blob(path: str) -> bytes:
    return (REPOSITORY_ROOT / path).read_bytes()


def frozen_blob(path: str) -> bytes:
    return (SOURCE / path).read_bytes()


def refresh_local_source() -> None:
    """Capture the authorised local public-doc snapshot; never touch source docs."""
    paths = local_paths()
    asset_paths = PRODUCT_ASSETS
    records = []
    for path in [*paths, *asset_paths]:
        data = local_blob(path)
        destination = SOURCE / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
        records.append({"path": path, "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)})
    SOURCE.mkdir(parents=True, exist_ok=True)
    SOURCE_MANIFEST.write_text(json.dumps({
        "sourceState": "working-tree-dirty", "baseRevision": CURRENT_HEAD,
        "repositoryContextRef": PRODUCT_INPUTS["repositoryContextRef"], "snapshotDate": SNAPSHOT_DATE, "files": records,
    }, indent=2) + "\n")


def frozen_paths() -> list[str]:
    source = json.loads(SOURCE_MANIFEST.read_text())
    return [item["path"] for item in source["files"] if item["path"] in PRODUCT_DOCUMENTS]


def doc_id(path: str) -> str:
    if path == "README.md":
        return "readme"
    return re.sub(r"[^a-z0-9]+", "-", path[:-3].lower()).strip("-")


def category(path: str) -> str:
    if path == "README.md":
        return "Overview"
    if path == "RELEASE.md":
        return "Releases"
    if path in {"SECURITY.md", "docs/assurance.md", "docs/security-profiles.md", "docs/threat-model.md", "docs/credential-boundaries.md", "docs/administration-security.md"}:
        return "Security"
    if path.startswith("docs/operators/"):
        return "Operations"
    if path.startswith("docs/integrations/"):
        return "Integrations"
    if path in {"docs/integrations.md", "docs/enterprise-integration.md", "docs/hosted-identity.md"}:
        return "Integrations"
    if path.startswith("sdk/") or path in {"docs/sdks.md", "docs/getting-started.md", "docs/product-guide.md", "CONTRIBUTING.md", "SUPPORT.md", "CODE_OF_CONDUCT.md"}:
        return "Guides"
    return "Reference"


PINNED_READER_ORDER = [
    "README.md", "docs/README.md", "docs/getting-started.md", "docs/product-guide.md",
    "docs/architecture.md", "docs/comparison.md", "docs/configuration.md", "docs/providers.md",
    "docs/identity-and-expiry.md", "docs/integrations.md", "docs/sdks.md",
]
CATEGORY_READER_ORDER = ["Overview", "Guides", "Integrations", "Security", "Operations", "Reference", "Releases"]


def reader_order(paths: list[str], sources: dict[str, str]) -> list[str]:
    """A stable, task-oriented order for the explorer navigation manifest."""
    pinned = [path for path in PINNED_READER_ORDER if path in paths]
    remainder = [path for path in paths if path not in set(pinned)]
    category_position = {name: index for index, name in enumerate(CATEGORY_READER_ORDER)}
    return pinned + sorted(
        remainder,
        key=lambda path: (category_position[category(path)], title_from_source(sources[path], Path(path).stem).casefold(), path.casefold()),
    )


def github_slug(title: str, seen: Counter[str]) -> str:
    value = re.sub(r"\[([^\]]+)\]\([^)]*\)", r"\1", title)
    value = re.sub(r"[*_`~]", "", value)
    value = re.sub(r"<[^>]*>", "", value).strip().lower()
    value = re.sub(r"[^\w\- ]", "", value, flags=re.UNICODE)
    value = re.sub(r"[ _]+", "-", value).strip("-") or "section"
    suffix = seen[value]
    seen[value] += 1
    return value if suffix == 0 else f"{value}-{suffix}"


# Consume one required separator; heading_text strips any remaining spacing.
# Avoid overlapping whitespace and title repetitions on malformed headings.
HEADING = re.compile(r"^(#{1,6})[ \t](.+)$")


def heading_text(value: str) -> str:
    """Normalize an ATX heading without a backtracking expression."""
    title = value.strip()
    closing_start = title.rfind(" #")
    if closing_start >= 0:
        closing = title[closing_start + 1 :]
        if closing == "#" * len(closing):
            title = title[:closing_start].rstrip()
    return title


def headings_and_parts(source: str, path: str) -> tuple[list[dict], list[dict]]:
    """Split long pages on H2 boundaries without breaking Markdown structures."""
    lines = source.splitlines(keepends=True)
    seen: Counter[str] = Counter()
    headings: list[dict] = []
    boundaries = [0]
    in_fence = False
    words = 0
    for index, line in enumerate(lines):
        if line.lstrip().startswith(("```", "~~~")):
            in_fence = not in_fence
        if not in_fence:
            match = HEADING.match(line.rstrip("\n"))
            if match:
                title = heading_text(match.group(2))
                item = {"id": github_slug(title, seen), "title": title, "level": len(match.group(1)), "line": index}
                headings.append(item)
                if item["level"] == 2 and words > 2500:
                    boundaries.append(index)
                    words = 0
        words += len(re.findall(r"\b\w+\b", line))
    boundaries.append(len(lines))
    boundaries = sorted(set(boundaries))
    parts: list[dict] = []
    for number, (start, end) in enumerate(zip(boundaries, boundaries[1:]), 1):
        chunk = "".join(lines[start:end])
        chunk_headings = [h for h in headings if start <= h["line"] < end]
        title = chunk_headings[0]["title"] if chunk_headings else ("Overview" if number == 1 else f"Part {number}")
        parts.append({"id": f"part-{number}", "title": title, "source": chunk, "headings": [{k: v for k, v in h.items() if k != "line"} for h in chunk_headings]})
    return [{k: v for k, v in h.items() if k != "line"} for h in headings], parts


def is_markdown(path: str) -> bool:
    return path.lower().endswith(".md")


def resolve_path(base: str, target: str) -> str:
    value = posixpath.normpath(posixpath.join(posixpath.dirname(base), unquote(target)))
    while value.startswith("./"):
        value = value[2:]
    return value


def safe_external(href: str) -> str:
    parsed = urlparse(href)
    if parsed.scheme.lower() in {"http", "https", "mailto"}:
        return href
    return "#"


def repository_context_url(path: str, directory: bool) -> str:
    kind = "tree" if directory else "blob"
    return f"{REPOSITORY}/{kind}/{quote(REPOSITORY_CONTEXT_REF, safe='')}/{quote(path)}"


def title_from_source(source: str, fallback: str) -> str:
    for line in source.splitlines():
        if line.startswith("#") and len(line) > 1 and line[1] in " \t":
            return line[2:].strip()
    return fallback


def asset_name(source_path: str, asset: bytes) -> str:
    suffix = Path(source_path).suffix.lower()
    if not re.fullmatch(r"\.[a-z0-9]{1,8}", suffix):
        suffix = ".bin"
    return hashlib.sha256(asset).hexdigest() + suffix


class SafeHtml(HTMLParser):
    """Keep presentational repository HTML while refusing executable/untrusted HTML."""
    allowed = {"p", "br", "strong", "b", "em", "i", "code", "pre", "kbd", "sup", "sub", "a", "picture", "source", "img", "section", "h3"}

    def __init__(self, current: str, local_asset, external_images: list[dict]):
        super().__init__(convert_charrefs=False)
        self.current = current
        self.local_asset = local_asset
        self.external_images = external_images
        self.out: list[str] = []
        self.drop_depth = 0

    def handle_starttag(self, tag, attrs):
        if tag in {"script", "style", "iframe", "object", "svg", "math"}:
            self.drop_depth += 1
            return
        if self.drop_depth or tag not in self.allowed:
            return
        values = dict(attrs)
        if tag == "img":
            alt = values.get("alt", "image")
            src = values.get("src", "")
            if src and not urlparse(src).scheme:
                local = self.local_asset(src)
                if local:
                    self.out.append(f'<img src="{escape(local, quote=True)}" alt="{escape(alt, quote=True)}">')
                    return
            if src:
                self.external_images.append({"document": self.current, "uri": src, "alt": alt})
                self.out.append(f'<a href="{escape(safe_external(src), quote=True)}">{escape(alt)}</a>')
            else:
                self.out.append(escape(alt))
            return
        clean: list[tuple[str, str]] = []
        if tag == "a":
            href = safe_external(values.get("href", ""))
            clean.append(("href", href))
        elif tag == "source":
            srcset = values.get("srcset", "")
            if srcset and not urlparse(srcset.split()[0]).scheme:
                local = self.local_asset(srcset.split()[0])
                if local:
                    clean.append(("srcset", local))
            elif srcset:
                self.external_images.append({"document": self.current, "uri": srcset, "alt": "source image"})
        elif tag in {"picture", "section"} and values.get("class") == "explore-details":
            clean.append(("class", "explore-details"))
        self.out.append("<" + tag + "".join(f' {key}="{escape(value, quote=True)}"' for key, value in clean) + ">")

    def handle_endtag(self, tag):
        if tag in {"script", "style", "iframe", "object", "svg", "math"} and self.drop_depth:
            self.drop_depth -= 1
            return
        if not self.drop_depth and tag in self.allowed and tag not in {"br", "source", "img"}:
            self.out.append(f"</{tag}>")

    def handle_data(self, data):
        if not self.drop_depth:
            self.out.append(escape(data))

    def handle_entityref(self, name):
        if not self.drop_depth:
            self.out.append(f"&{name};")

    def handle_charref(self, name):
        if not self.drop_depth:
            self.out.append(f"&#{name};")


def flatten_details(source: str) -> str:
    """A details block is information, not UI state in the static reader."""
    lowered = source.lower()
    output: list[str] = []
    cursor = 0
    while True:
        details_start = lowered.find("<details", cursor)
        if details_start < 0:
            output.append(source[cursor:])
            return "".join(output)
        details_open_end = lowered.find(">", details_start)
        if details_open_end < 0:
            output.append(source[cursor:])
            return "".join(output)
        summary_start = lowered.find("<summary>", details_open_end + 1)
        if summary_start < 0 or source[details_open_end + 1 : summary_start].strip():
            output.append(source[cursor : details_open_end + 1])
            cursor = details_open_end + 1
            continue
        summary_end = lowered.find("</summary>", summary_start + len("<summary>"))
        details_end = (
            lowered.find("</details>", summary_end + len("</summary>"))
            if summary_end >= 0
            else -1
        )
        if summary_end < 0 or details_end < 0:
            output.append(source[cursor : details_open_end + 1])
            cursor = details_open_end + 1
            continue
        output.append(source[cursor:details_start])
        title = source[summary_start + len("<summary>") : summary_end]
        body = source[summary_end + len("</summary>") : details_end]
        output.append(f'<section class="explore-details"><h3>{title}</h3>{body}</section>')
        cursor = details_end + len("</details>")


def replace_remote_images(rendered: str, current: str, external_images: list[dict]) -> str:
    """Markdown-it may tokenize an HTML badge inside a link as inline HTML.

    Keep its meaningful alternate label and target while preventing a live badge
    request or an accidental current-status claim in this fixed snapshot.
    """
    class RemoteImageRewriter(HTMLParser):
        def __init__(self) -> None:
            super().__init__(convert_charrefs=False)
            self.out: list[str] = []
            self.anchor_open: str | None = None
            self.anchor_content: list[str] = []

        def emit(self, value: str) -> None:
            target = self.anchor_content if self.anchor_open is not None else self.out
            target.append(value)

        def handle_starttag(
            self, tag: str, attrs: list[tuple[str, str | None]]
        ) -> None:
            raw = self.get_starttag_text()
            if tag == "a" and self.anchor_open is None:
                self.anchor_open = raw
                self.anchor_content = []
                return
            if tag != "img":
                self.emit(raw)
                return
            values = dict(attrs)
            uri = values.get("src", "")
            if urlparse(uri).scheme.lower() not in {"http", "https"}:
                self.emit(raw)
                return
            alt = values.get("alt") or "external image"
            external_images.append({"document": current, "uri": uri, "alt": alt})
            if self.anchor_open is not None:
                self.emit(escape(alt))
            else:
                self.emit(f'<a href="{escape(uri, quote=True)}">{escape(alt)}</a>')

        def handle_startendtag(
            self, tag: str, attrs: list[tuple[str, str | None]]
        ) -> None:
            self.handle_starttag(tag, attrs)

        def handle_endtag(self, tag: str) -> None:
            if tag == "a" and self.anchor_open is not None:
                self.out.extend((self.anchor_open, *self.anchor_content, "</a>"))
                self.anchor_open = None
                self.anchor_content = []
                return
            self.emit(f"</{tag}>")

        def handle_data(self, data: str) -> None:
            self.emit(data)

        def handle_entityref(self, name: str) -> None:
            self.emit(f"&{name};")

        def handle_charref(self, name: str) -> None:
            self.emit(f"&#{name};")

        def handle_comment(self, data: str) -> None:
            self.emit(f"<!--{data}-->")

    rewriter = RemoteImageRewriter()
    rewriter.feed(rendered)
    rewriter.close()
    if rewriter.anchor_open is not None:
        rewriter.out.extend((rewriter.anchor_open, *rewriter.anchor_content))
    return "".join(rewriter.out)


class RenderedHtmlSanitizer(HTMLParser):
    """Final attribute allowlist for Markdown-It output and sanitized raw HTML."""
    allowed = {
        "a", "blockquote", "br", "code", "del", "div", "em", "h1", "h2", "h3", "h4", "h5", "h6",
        "hr", "img", "kbd", "li", "ol", "p", "picture", "pre", "section", "source", "strong", "sub", "sup",
        "table", "tbody", "td", "th", "thead", "tr", "ul",
    }
    void = {"br", "hr", "img", "source"}

    def __init__(self):
        super().__init__(convert_charrefs=False)
        self.out: list[str] = []
        self.stack: list[str] = []

    @staticmethod
    def valid_href(value: str) -> str | None:
        if value.startswith("/explore#") or value.startswith("/explore-content/assets/"):
            return value
        return safe_external(value) if safe_external(value) != "#" else None

    def handle_starttag(self, tag, attrs):
        if tag not in self.allowed:
            return
        # HTMLParser preserves named entities in attribute values; normalise them
        # before serialising so query delimiters become `&amp;` exactly once.
        values = {key: unescape(value) if value is not None else "" for key, value in attrs}
        clean: list[tuple[str, str]] = []
        if tag in {"h1", "h2", "h3", "h4", "h5", "h6"} and values.get("id"):
            clean.append(("id", values["id"]))
        if tag == "a" and values.get("href"):
            href = self.valid_href(values["href"])
            if href:
                clean.append(("href", href))
                if values.get("title"):
                    clean.append(("title", values["title"]))
        if tag == "a" and not clean:
            # Retain its readable child text but never emit an inert anchor.
            return
        if tag == "img":
            src = values.get("src", "")
            if src.startswith("/explore-content/assets/"):
                clean.append(("src", src))
            if values.get("alt"):
                clean.append(("alt", values["alt"]))
        if tag == "source" and values.get("srcset", "").startswith("/explore-content/assets/"):
            clean.append(("srcset", values["srcset"]))
        if tag == "code" and values.get("class", "").startswith("language-"):
            clean.append(("class", values["class"]))
        if tag == "section" and values.get("class") == "explore-details":
            clean.append(("class", "explore-details"))
        self.out.append("<" + tag + "".join(f' {key}="{escape(value, quote=True)}"' for key, value in clean) + ">")
        if tag not in self.void:
            self.stack.append(tag)

    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)

    def handle_endtag(self, tag):
        if tag not in self.void and self.stack and tag == self.stack[-1]:
            self.stack.pop()
            self.out.append(f"</{tag}>")

    def handle_data(self, data):
        self.out.append(escape(data))

    def handle_entityref(self, name):
        self.out.append(f"&{name};")

    def handle_charref(self, name):
        self.out.append(f"&#{name};")


def sanitize_rendered_html(rendered: str) -> str:
    parser = RenderedHtmlSanitizer()
    parser.feed(rendered)
    parser.close()
    return "".join(parser.out)


def build(check: bool = False, refresh_local: bool = False) -> None:
    global BASE_REVISION, REPOSITORY_CONTEXT_REF
    if refresh_local:
        refresh_local_source()
    source_freeze_bytes = SOURCE_MANIFEST.read_bytes()
    source_freeze = json.loads(source_freeze_bytes)
    BASE_REVISION = source_freeze["baseRevision"]
    REPOSITORY_CONTEXT_REF = source_freeze.get("repositoryContextRef", "feature/initial-implementation")
    paths = frozen_paths()
    source_bytes = {path: frozen_blob(path) for path in paths}
    expected_hashes = {item["path"]: item["sha256"] for item in source_freeze["files"]
                       if item["path"] in PRODUCT_DOCUMENTS or item["path"] in PRODUCT_ASSETS}
    for path, expected in expected_hashes.items():
        data = frozen_blob(path)
        if expected != hashlib.sha256(data).hexdigest():
            raise SystemExit(f"frozen source hash mismatch: {path}")
    sources: dict[str, str] = {}
    presentation_records = []
    for path, data in source_bytes.items():
        rendered, mappings = presentation_copy(data.decode("utf-8"))
        sources[path] = rendered
        if mappings:
            presentation_records.append({"path": path, "mappings": mappings})
    paths = reader_order(paths, sources)
    docs: list[dict] = []
    by_path: dict[str, dict] = {}
    for path in paths:
        all_headings, parts = headings_and_parts(sources[path], path)
        doc = {
            "id": doc_id(path), "path": path, "title": title_from_source(sources[path], Path(path).stem),
            "category": category(path), "parts": parts, "headings": all_headings,
            "sourceSha256": hashlib.sha256(source_bytes[path]).hexdigest(),
        }
        docs.append(doc)
        by_path[path] = doc

    planned: dict[Path, bytes] = {}
    asset_records: list[dict] = []
    external_images: list[dict] = []
    unresolved_links: list[dict] = []
    raw_html: dict[str, int] = {}
    md = MarkdownIt("commonmark", {"html": True}).enable("table").enable("strikethrough")

    def internal_url(target: str, current: str) -> str | None:
        parsed = urlparse(target)
        if parsed.scheme or parsed.netloc:
            return None
        resolved = resolve_path(current, parsed.path) if parsed.path else current
        if not is_markdown(resolved) or resolved not in by_path:
            return None
        target_doc = by_path[resolved]
        part = target_doc["parts"][0]["id"]
        anchor = ""
        if parsed.fragment:
            anchor = parsed.fragment
            for candidate in target_doc["parts"]:
                if any(h["id"] == anchor for h in candidate["headings"]):
                    part = candidate["id"]
                    break
        value = f"/explore#doc={quote(target_doc['id'])}&part={quote(part)}"
        return value + (f"&anchor={quote(anchor)}" if anchor else "")

    def rewrite_tokens(tokens, current: str) -> None:
        for token in tokens:
            if token.type == "html_block" or token.type == "html_inline":
                raw_html[current] = raw_html.get(current, 0) + 1
            if token.type == "inline":
                rewrite_tokens(token.children or [], current)
            if token.type not in {"link_open", "image"}:
                continue
            href = token.attrGet("href") if token.type == "link_open" else token.attrGet("src")
            if not href:
                continue
            routed = internal_url(href, current)
            if routed:
                token.attrSet("href" if token.type == "link_open" else "src", routed)
                continue
            parsed = urlparse(href)
            if parsed.scheme or parsed.netloc:
                safe = safe_external(href)
                token.attrSet("href" if token.type == "link_open" else "src", safe)
                continue
            local = resolve_path(current, parsed.path)
            if token.type == "image":
                if local not in PRODUCT_ASSETS:
                    token.attrSet("src", "#")
                    continue
                try:
                    data = frozen_blob(local)
                except (FileNotFoundError, IsADirectoryError):
                    token.attrSet("src", "#")
                    continue
                filename = asset_name(local, data)
                planned[ASSETS / filename] = data
                token.attrSet("src", f"/explore-content/assets/{filename}")
                asset_records.append({"sourcePath": local, "sha256": hashlib.sha256(data).hexdigest(), "file": f"assets/{filename}"})
            elif local:
                local_path = SOURCE / local
                if local_path.is_file():
                    token.attrSet("href", repository_context_url(local, False) + (f"#{parsed.fragment}" if parsed.fragment else ""))
                    token.attrSet("title", REPOSITORY_REFERENCE_NOTE)
                elif local_path.is_dir():
                    token.attrSet("href", repository_context_url(local, True) + (f"#{parsed.fragment}" if parsed.fragment else ""))
                    token.attrSet("title", REPOSITORY_REFERENCE_NOTE)
                else:
                    directory = parsed.path.endswith("/")
                    fallback = repository_context_url(local, directory) + (f"#{parsed.fragment}" if parsed.fragment else "")
                    unresolved_links.append({"document": current, "target": href, "fallback": fallback})
                    token.attrSet("href", fallback)
                    token.attrSet("title", REPOSITORY_REFERENCE_NOTE)

    def local_html_asset(reference: str, current: str) -> str | None:
        parsed = urlparse(reference)
        local = resolve_path(current, parsed.path)
        if local not in PRODUCT_ASSETS:
            return None
        try:
            data = frozen_blob(local)
        except (FileNotFoundError, IsADirectoryError):
            return None
        filename = asset_name(local, data)
        planned[ASSETS / filename] = data
        asset_records.append({"sourcePath": local, "sha256": hashlib.sha256(data).hexdigest(), "file": f"assets/{filename}"})
        return f"/explore-content/assets/{filename}"

    def sanitize_html_tokens(tokens, current: str) -> None:
        for token in tokens:
            if token.type in {"html_block", "html_inline"}:
                raw_html[current] = raw_html.get(current, 0) + 1
                parser = SafeHtml(current, lambda ref: local_html_asset(ref, current), external_images)
                parser.feed(token.content)
                parser.close()
                token.content = "".join(parser.out)
            if token.type == "inline":
                sanitize_html_tokens(token.children or [], current)

    for doc in docs:
        rendered_parts = []
        for part in doc["parts"]:
            tokens = md.parse(flatten_details(part["source"]))
            sanitize_html_tokens(tokens, doc["path"])
            rewrite_tokens(tokens, doc["path"])
            # Add GitHub-compatible identifiers to headings before rendering.
            heading_iter = iter(part["headings"])
            pending = next(heading_iter, None)
            for token in tokens:
                if token.type == "heading_open" and pending:
                    token.attrSet("id", pending["id"])
                    pending = next(heading_iter, None)
            rendered = md.renderer.render(tokens, md.options, {})
            rendered = replace_remote_images(rendered, doc["path"], external_images)
            rendered_parts.append({"id": part["id"], "title": part["title"], "html": sanitize_rendered_html(rendered)})
        module = {"id": doc["id"], "path": doc["path"], "title": doc["title"], "parts": rendered_parts}
        planned[CONTENT / f"{doc['id']}.js"] = ("export default " + json.dumps(module, ensure_ascii=False, separators=(",", ":")) + ";\n").encode()

    manifest_docs = [{
        "id": doc["id"], "path": doc["path"], "title": doc["title"], "category": doc["category"],
        "parts": [{"id": part["id"], "title": part["title"], "headings": part["headings"]} for part in doc["parts"]],
        "module": f"./explore-content/{doc['id']}.js",
    } for doc in docs]
    manifest = {
        "repository": REPOSITORY, "revision": REVISION, "baseRevision": BASE_REVISION,
        "sourceState": "working-tree-dirty", "snapshotDate": SNAPSHOT_DATE,
        "repositoryContextRef": REPOSITORY_CONTEXT_REF, "repositoryReferenceNote": REPOSITORY_REFERENCE_NOTE,
        "sourceFreeze": "./explore-source/manifest.json", "sourceFreezeManifestSha256": hashlib.sha256(source_freeze_bytes).hexdigest(),
        "presentationCopy": {"mappings": [{"from": old, "to": new} for old, new in PRESENTATION_COPY], "files": presentation_records},
        "documents": manifest_docs,
    }
    planned[ROOT / "explore-manifest.js"] = ("export const documentation=" + json.dumps(manifest, ensure_ascii=False, separators=(",", ":")) + ";\n").encode()
    public_files = [
        {"path": str(path.relative_to(ROOT)), "sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data)}
        for path, data in sorted(planned.items(), key=lambda item: str(item[0]))
        if path != CONTENT / "manifest.json"
    ]
    provenance = {
        "repository": REPOSITORY, "revision": REVISION, "baseRevision": BASE_REVISION,
        "sourceState": "working-tree-dirty", "snapshotDate": SNAPSHOT_DATE,
        "repositoryContextRef": REPOSITORY_CONTEXT_REF, "repositoryReferenceNote": REPOSITORY_REFERENCE_NOTE,
        "sourceFreezeManifestSha256": hashlib.sha256(source_freeze_bytes).hexdigest(),
        "presentationCopy": {"mappings": [{"from": old, "to": new} for old, new in PRESENTATION_COPY], "files": presentation_records},
        "documentCount": len(docs), "sourcePaths": [{"path": doc["path"], "sha256": doc["sourceSha256"], "parts": len(doc["parts"])} for doc in docs],
        "assets": sorted({json.dumps(item, sort_keys=True): item for item in asset_records}.values(), key=lambda item: item["file"]),
        "rawHtmlSanitized": raw_html, "externalImagesRenderedAsLinks": external_images,
        "unresolvedRelativeLinks": unresolved_links,
        "publicFiles": public_files,
        "generatedFiles": sorted(str(path.relative_to(ROOT)) for path in planned),
    }
    planned[CONTENT / "manifest.json"] = (json.dumps(provenance, indent=2, ensure_ascii=False) + "\n").encode()

    generated_assets = {path for path in planned if path.parent == ASSETS}
    stale_assets = [path for path in ASSETS.glob("*") if path.is_file() and path not in generated_assets]
    stale_documents = [path for path in CONTENT.glob("*.js") if path not in planned]
    if check:
        missing = [str(path.relative_to(ROOT)) for path, data in planned.items() if not path.exists() or path.read_bytes() != data]
        extras = stale_documents + stale_assets
        if missing or extras:
            raise SystemExit("explore snapshot is stale: " + ", ".join(missing + [str(path.relative_to(ROOT)) for path in extras]))
        return
    for path in stale_documents + stale_assets:
        path.unlink()
    for path, data in planned.items():
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--refresh-local-source", action="store_true", help="capture the current authorised local public-doc snapshot")
    options = parser.parse_args()
    build(options.check, options.refresh_local_source)
