"""Owned-input documentation helpers and full synthetic snapshot contracts.

The real module's two import-time Git observations and explicit document-list
read are intercepted. No checked-in corpus is read, no subprocess is launched,
and no checked-in Markdown/build/browser or refreshed product output is selected.
Full snapshot cases explicitly enable the real declared MarkdownIt dependency
only over temporary authored documents/assets. No real corpus or Git is read.
The real import uses the existing SDK-lock MarkdownIt4.2.0 dependency; these
oracles do not qualify the browser renderer pinned separately to3.0.0.
"""

from __future__ import annotations

import hashlib
import importlib.util
import json
import runpy
import socket
import sys
import subprocess
import tempfile
import unittest
from collections import Counter
from pathlib import Path
from unittest import mock

from markdown_it import MarkdownIt

SCRIPT = Path(__file__).parents[3] / "examples/landing-motion/build-explore.py"
SPEC = importlib.util.spec_from_file_location("build_explore_contract", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
EXPLORE = importlib.util.module_from_spec(SPEC)
IMPORT_INPUTS = {
    "documents": ["README.md", "docs/guide.md", "docs/missing.md"],
    "assets": ["assets/icon.svg"],
    "repositoryContextRef": "synthetic-context",
}


def import_observation(arguments, *, cwd, text):
    """Replace only the two actual import observations, never a general runner."""
    if cwd != SCRIPT.resolve().parent.parent.parent or text is not True:
        raise AssertionError("unexpected import observation context")
    if arguments == ["git", "rev-parse", "HEAD"]:
        return "1" * 40 + "\n"
    if arguments == ["git", "branch", "--show-current"]:
        return "synthetic-context\n"
    raise AssertionError("unexpected import command")


def import_document_list(path, *arguments, **keywords):
    if path != SCRIPT.resolve().parent / "explore-documents.json" or arguments or keywords:
        raise AssertionError("unexpected import document read")
    return json.dumps(IMPORT_INPUTS)


with mock.patch.object(subprocess, "check_output", side_effect=import_observation) as observed, \
        mock.patch.object(Path, "read_text", autospec=True, side_effect=import_document_list) as read:
    SPEC.loader.exec_module(EXPLORE)
    assert observed.call_count == 2
    read.assert_called_once()
assert EXPLORE.MarkdownIt is MarkdownIt


class BuildExploreOwnedContracts(unittest.TestCase):
    def setUp(self) -> None:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.repository = self.root / "repository"
        self.landing = self.root / "landing"
        self.source = self.landing / "explore-source"
        self.content = self.landing / "explore-content"
        self.repository.mkdir()
        self.landing.mkdir()
        patches = mock.patch.multiple(
            EXPLORE, ROOT=self.landing, REPOSITORY_ROOT=self.repository,
            CONTENT=self.content, ASSETS=self.content / "assets", SOURCE=self.source,
            SOURCE_MANIFEST=self.source / "manifest.json", PRODUCT_INPUTS=IMPORT_INPUTS,
            PRODUCT_DOCUMENTS=frozenset(IMPORT_INPUTS["documents"]),
            PRODUCT_ASSETS=tuple(IMPORT_INPUTS["assets"]), CURRENT_HEAD="1" * 40,
            BASE_REVISION="", REPOSITORY_CONTEXT_REF="synthetic-context",
        )
        self.enterContext(patches)
        # These tests select helpers and pre-render refusals only. A renderer
        # call would be a scope error, not a fake successful parser result.
        self.enterContext(mock.patch.object(EXPLORE, "MarkdownIt", side_effect=AssertionError("renderer not selected")))
        self.enterContext(mock.patch.object(EXPLORE.subprocess, "check_output", side_effect=AssertionError("no process selected")))

    def write_document(self, name: str, data: bytes) -> Path:
        path = self.repository / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        return path

    def seed_snapshot(self) -> dict:
        self.write_document("README.md", b"# Synthetic overview\n")
        self.write_document("docs/guide.md", b"# Synthetic guide\n")
        self.write_document("assets/icon.svg", b"synthetic opaque asset")
        self.write_document("not-allowlisted.md", b"synthetic unselected document")
        EXPLORE.refresh_local_source()
        return json.loads(EXPLORE.SOURCE_MANIFEST.read_text())

    def test_presentation_mapping_counts_preserve_the_input(self) -> None:
        original = "The right secrets, ready for your apps.\nGet the right secrets ready for your apps.\nThe right secrets, ready for your apps."
        rendered, mappings = EXPLORE.presentation_copy(original)
        self.assertEqual("Get the right secrets ready.\n" * 2 + "Get the right secrets ready.", rendered)
        self.assertEqual([2, 1], [item["replacements"] for item in mappings])
        self.assertEqual([old for old, _ in EXPLORE.PRESENTATION_COPY], [item["from"] for item in mappings])
        self.assertEqual([new for _, new in EXPLORE.PRESENTATION_COPY], [item["to"] for item in mappings])
        self.assertTrue(original.startswith("The right secrets,"))
        self.assertEqual(("unchanged", []), EXPLORE.presentation_copy("unchanged"))

    def test_snapshot_reads_only_the_explicit_owned_inputs_and_binds_hashes(self) -> None:
        manifest = self.seed_snapshot()
        self.assertEqual(["README.md", "docs/guide.md"], EXPLORE.local_paths())
        self.assertEqual("working-tree-dirty", manifest["sourceState"])
        self.assertEqual("1" * 40, manifest["baseRevision"])
        self.assertEqual("synthetic-context", manifest["repositoryContextRef"])
        self.assertEqual(EXPLORE.SNAPSHOT_DATE, manifest["snapshotDate"])
        self.assertEqual(["README.md", "docs/guide.md", "assets/icon.svg"], [item["path"] for item in manifest["files"]])
        for item in manifest["files"]:
            with self.subTest(path=item["path"]):
                before = (self.repository / item["path"]).read_bytes()
                self.assertEqual(before, EXPLORE.local_blob(item["path"]))
                self.assertEqual(before, EXPLORE.frozen_blob(item["path"]))
                self.assertEqual(hashlib.sha256(before).hexdigest(), item["sha256"])
                self.assertEqual(len(before), item["bytes"])
        self.assertFalse((self.source / "not-allowlisted.md").exists())
        self.assertEqual(["README.md", "docs/guide.md"], EXPLORE.frozen_paths())

    def test_snapshot_missing_asset_is_a_real_refusal_not_atomic_rollback(self) -> None:
        self.write_document("README.md", b"# Synthetic overview\n")
        with self.assertRaises(FileNotFoundError):
            EXPLORE.refresh_local_source()
        self.assertEqual(b"# Synthetic overview\n", (self.source / "README.md").read_bytes())
        self.assertFalse(EXPLORE.SOURCE_MANIFEST.exists())

    def test_frozen_hash_mismatch_stops_before_renderer_and_output(self) -> None:
        self.seed_snapshot()
        (self.source / "docs/guide.md").write_bytes(b"synthetic changed bytes")
        with self.assertRaisesRegex(SystemExit, r"^frozen source hash mismatch: docs/guide\.md$"):
            EXPLORE.build()
        EXPLORE.MarkdownIt.assert_not_called()
        self.assertFalse(self.content.exists())
        self.assertFalse((self.landing / "explore-manifest.js").exists())

    def test_document_ids_categories_and_reader_order_are_stable(self) -> None:
        self.assertEqual("readme", EXPLORE.doc_id("README.md"))
        self.assertEqual("docs-a-guide", EXPLORE.doc_id("docs/A Guide.md"))
        cases = {
            "README.md": "Overview", "RELEASE.md": "Releases", "SECURITY.md": "Security",
            "docs/operators/test.md": "Operations", "docs/integrations/test.md": "Integrations",
            "docs/hosted-identity.md": "Integrations", "sdk/test.md": "Guides",
            "docs/getting-started.md": "Guides", "docs/other.md": "Reference",
        }
        for path, category in cases.items():
            with self.subTest(path=path):
                self.assertEqual(category, EXPLORE.category(path))
        paths = ["docs/z.md", "RELEASE.md", "docs/a.md", "README.md", "sdk/x.md", "docs/README.md"]
        sources = {path: "# " + title for path, title in zip(paths, ["Alpha", "Release", "Zulu", "Readme", "Guide", "Index"])}
        self.assertEqual(["README.md", "docs/README.md", "sdk/x.md", "docs/z.md", "docs/a.md", "RELEASE.md"], EXPLORE.reader_order(paths, sources))
        self.assertEqual("Title", EXPLORE.title_from_source("not a heading\n# Title\n", "fallback"))
        self.assertEqual("fallback", EXPLORE.title_from_source("#not-atx\nbody", "fallback"))

    def test_heading_slugs_closing_markers_fences_and_parts_preserve_source(self) -> None:
        seen = Counter()
        title = "**Same** [label](target) `code` <b>value</b>"
        self.assertEqual("same-label-code-value", EXPLORE.github_slug(title, seen))
        self.assertEqual("same-label-code-value-1", EXPLORE.github_slug(title, seen))
        self.assertEqual("section", EXPLORE.github_slug("!!!", seen))
        self.assertEqual("ångström-one", EXPLORE.github_slug("Ångström One", seen))
        self.assertEqual("ångströmone", EXPLORE.github_slug("Ångström_One", seen))
        self.assertEqual("Closing", EXPLORE.heading_text(" Closing ### "))
        self.assertEqual("Not # closed text", EXPLORE.heading_text("Not # closed text"))
        source = "# Title\n```md\n## Not a heading\n```\n" + "word " * 2501 + "\n## Next\nbody\n## Next\nlast\n"
        headings, parts = EXPLORE.headings_and_parts(source, "docs/guide.md")
        self.assertEqual(["title", "next", "next-1"], [item["id"] for item in headings])
        self.assertEqual([1, 2, 2], [item["level"] for item in headings])
        self.assertEqual(["part-1", "part-2"], [item["id"] for item in parts])
        self.assertEqual(["Title", "Next"], [item["title"] for item in parts])
        self.assertEqual(source, "".join(item["source"] for item in parts))
        self.assertEqual([], EXPLORE.headings_and_parts("", "docs/empty.md")[1])
        self.assertEqual("Overview", EXPLORE.headings_and_parts("plain body", "docs/plain.md")[1][0]["title"])

    def test_urls_assets_and_markdown_classification_do_not_fetch(self) -> None:
        self.assertEqual("docs/a b.md", EXPLORE.resolve_path("docs/nested/item.md", "../a%20b.md"))
        self.assertTrue(EXPLORE.is_markdown("UPPER.MD"))
        self.assertFalse(EXPLORE.is_markdown("guide.txt"))
        for value in ("https://example.invalid/a", "HTTP://example.invalid/a", "mailto:user@example.invalid"):
            with self.subTest(value=value):
                self.assertEqual(value, EXPLORE.safe_external(value))
        for value in ("javascript:alert(1)", "data:text/plain,a", "file:///synthetic", "//example.invalid/a", "/relative"):
            with self.subTest(value=value):
                self.assertEqual("#", EXPLORE.safe_external(value))
        self.assertEqual("https://github.com/maldous/CircularQuay/blob/synthetic-context/docs/a%20b.md", EXPLORE.repository_context_url("docs/a b.md", False))
        self.assertEqual("https://github.com/maldous/CircularQuay/tree/synthetic-context/docs", EXPLORE.repository_context_url("docs", True))
        digest = hashlib.sha256(b"owned").hexdigest()
        self.assertEqual(digest + ".svg", EXPLORE.asset_name("assets/ICON.SVG", b"owned"))
        self.assertEqual(digest + ".bin", EXPLORE.asset_name("assets/file.unsupportedlong", b"owned"))

    def test_safe_html_removes_executable_content_and_rewrites_owned_assets(self) -> None:
        external = []
        asset = mock.Mock(return_value="/explore-content/assets/owned.svg")
        parser = EXPLORE.SafeHtml("docs/guide.md", asset, external)
        parser.feed('<script>bad<style>nested</style></script><p onclick="bad">safe &amp; &#65;</p><a href="javascript:bad" onclick="bad">link</a><img src="local.svg" alt="owned"><img src="https://image.example.invalid/a" alt="remote"><img alt="only"><source srcset="local.svg 1x"><source srcset="https://image.example.invalid/b 1x"><section class="explore-details">details</section>')
        parser.close()
        rendered = "".join(parser.out)
        self.assertEqual('<p>safe &amp; &#65;</p><a href="#">link</a><img src="/explore-content/assets/owned.svg" alt="owned"><a href="https://image.example.invalid/a">remote</a>only<source srcset="/explore-content/assets/owned.svg"><source><section class="explore-details">details</section>', rendered)
        self.assertEqual([mock.call("local.svg"), mock.call("local.svg")], asset.call_args_list)
        self.assertEqual([{"document": "docs/guide.md", "uri": "https://image.example.invalid/a", "alt": "remote"}, {"document": "docs/guide.md", "uri": "https://image.example.invalid/b 1x", "alt": "source image"}], external)

    def test_safe_html_missing_local_image_is_recorded_without_a_network_effect(self) -> None:
        external = []
        asset = mock.Mock(return_value=None)
        parser = EXPLORE.SafeHtml("README.md", asset, external)
        parser.feed('<unknown>text</unknown><img src="missing.svg" alt="missing"><picture class="other"><source srcset="missing.svg 1x"></picture>')
        parser.close()
        self.assertEqual('text<a href="#">missing</a><picture><source></picture>', "".join(parser.out))
        self.assertEqual([{"document": "README.md", "uri": "missing.svg", "alt": "missing"}], external)
        self.assertEqual(2, asset.call_count)

    def test_details_flattening_keeps_malformed_source_and_nested_copy_literal(self) -> None:
        self.assertEqual('<section class="explore-details"><h3>Title</h3>Body</section>after', EXPLORE.flatten_details('<details open><summary>Title</summary>Body</details>after'))
        for source in ("plain", "<details", "<details>not summary</details>", "<details><summary>unfinished", "<details><summary>Title</summary>no end"):
            with self.subTest(source=source):
                self.assertEqual(source, EXPLORE.flatten_details(source))

    def test_remote_images_become_text_or_links_without_fetching(self) -> None:
        external = []
        source = '<!--owned--><a href="https://target.example.invalid"><img src="https://image.example.invalid/a" alt="A &amp; B"/></a><img src="http://image.example.invalid/b"><img src="/explore-content/assets/owned.svg" alt="local"><p>&amp;&#65;</p>'
        rendered = EXPLORE.replace_remote_images(source, "README.md", external)
        self.assertEqual('<!--owned--><a href="https://target.example.invalid">A &amp; B</a><a href="http://image.example.invalid/b">external image</a><img src="/explore-content/assets/owned.svg" alt="local"><p>&amp;&#65;</p>', rendered)
        self.assertEqual([{"document": "README.md", "uri": "https://image.example.invalid/a", "alt": "A & B"}, {"document": "README.md", "uri": "http://image.example.invalid/b", "alt": "external image"}], external)
        self.assertEqual('<a href="https://target.example.invalid">text', EXPLORE.replace_remote_images('<a href="https://target.example.invalid">text', "README.md", []))

    def test_final_sanitizer_preserves_only_declared_attributes_and_nesting(self) -> None:
        source = '<p onclick="bad">safe <a href="javascript:bad">text</a><a href="https://example.invalid/?a=1&amp;b=2" title="owned" onclick="bad">ok</a><img src="https://image.example.invalid" alt="remote"><img src="/explore-content/assets/owned.svg" alt="owned"><source srcset="/explore-content/assets/owned.svg"><code class="language-go" style="bad">x</code><section class="explore-details">body</section><svg>opaque</svg></p>'
        self.assertEqual('<p>safe text<a href="https://example.invalid/?a=1&amp;b=2" title="owned">ok</a><img alt="remote"><img src="/explore-content/assets/owned.svg" alt="owned"><source srcset="/explore-content/assets/owned.svg"><code class="language-go">x</code><section class="explore-details">body</section>opaque</p>', EXPLORE.sanitize_rendered_html(source))
        self.assertEqual('<p>a<br>b</p>', EXPLORE.sanitize_rendered_html('<p>a<br/>b</p>'))
        self.assertIsNone(EXPLORE.RenderedHtmlSanitizer.valid_href("#raw"))
        self.assertEqual("/explore#doc=owned", EXPLORE.RenderedHtmlSanitizer.valid_href("/explore#doc=owned"))
        self.assertEqual("/explore-content/assets/owned.svg", EXPLORE.RenderedHtmlSanitizer.valid_href("/explore-content/assets/owned.svg"))


class BuildExploreFullSnapshotContracts(unittest.TestCase):
    """Run actual unchanged renderer only against a bounded authored snapshot."""

    write_document = BuildExploreOwnedContracts.write_document

    def setUp(self) -> None:
        BuildExploreOwnedContracts.setUp(self)
        self.enterContext(mock.patch.object(EXPLORE, "MarkdownIt", MarkdownIt))
        self.enterContext(mock.patch.object(subprocess, "Popen", side_effect=AssertionError("no child process")))
        self.enterContext(mock.patch.object(socket, "create_connection", side_effect=AssertionError("no network")))
        self.enterContext(mock.patch.object(socket, "getaddrinfo", side_effect=AssertionError("no DNS")))
        self.enterContext(mock.patch.object(socket.socket, "connect", side_effect=AssertionError("no socket")))

    def rich_snapshot(self) -> dict:
        self.readme = (
            "# The right secrets, ready for your apps.\n\n"
            "[guide](docs/guide.md#second) [self](#the-right-secrets-ready-for-your-apps) "
            "[remote](https://example.invalid/?a=1&b=2) [mail](mailto:example@example.invalid)\n\n"
            "[asset](assets/icon.svg#owned) [directory](docs/) "
            "[unknown](missing.md#missing) [folder](unknown/)\n\n"
            "![local](assets/icon.svg) ![unlisted](assets/unlisted.svg) "
            "![missing](assets/missing.svg) ![empty]() "
            "![remote](https://image.example.invalid/owned.svg)\n\n"
            "<picture><source srcset=\"assets/icon.svg 1x\"><img src=\"assets/icon.svg\" alt=\"raw\"></picture>\n\n"
            "<img src=\"assets/missing.svg\" alt=\"missing raw\">\n\n"
            "<details><summary>Owned details</summary>Details &amp; content</details>\n\n"
            "<script>forbidden-script</script>\n\n"
            "~~deleted~~ **bold** `code`\n\n"
            "| Column | Value |\n| --- | --- |\n| owned | 1 |\n"
        ).encode()
        self.guide = (
            "# Guide\n\n[home](../README.md) [first](#guide) [missing anchor](#absent)\n\n"
            + "owned " * 2501
            + "\n\n## Second\n\n[second](#second) [missing asset](../assets/missing.svg)\n"
        ).encode()
        self.asset_bytes = b"owned opaque asset, never decoded or executed"
        self.write_document("README.md", self.readme)
        self.write_document("docs/guide.md", self.guide)
        self.write_document("assets/icon.svg", self.asset_bytes)
        self.enterContext(mock.patch.object(EXPLORE, "PRODUCT_ASSETS", ("assets/icon.svg", "assets/missing.svg")))
        # Refresh only existing files. A declared-but-unavailable asset is an
        # explicit render-time refusal case, never a fabricated retained input.
        with mock.patch.object(EXPLORE, "PRODUCT_ASSETS", ("assets/icon.svg",)):
            EXPLORE.refresh_local_source()
        return json.loads(EXPLORE.SOURCE_MANIFEST.read_bytes())

    def module(self, name: str) -> dict:
        data = (self.content / name).read_text()
        self.assertTrue(data.startswith("export default "))
        self.assertTrue(data.endswith(";\n"))
        return json.loads(data[len("export default "):-2])

    def output_bytes_and_times(self) -> dict:
        roots = [self.content, self.landing / "explore-manifest.js"]
        result = {}
        for root in roots:
            paths = root.rglob("*") if root.is_dir() else [root]
            for path in paths:
                if path.is_file():
                    result[path.relative_to(self.landing).as_posix()] = (path.read_bytes(), path.stat().st_mtime_ns)
        return result

    def test_actual_build_preserves_every_owned_source_and_public_file_hash(self) -> None:
        frozen = self.rich_snapshot()
        source_before = {p.relative_to(self.source): p.read_bytes() for p in self.source.rglob("*") if p.is_file()}
        EXPLORE.build()
        provenance = json.loads((self.content / "manifest.json").read_bytes())
        self.assertEqual(2, provenance["documentCount"])
        self.assertEqual(["README.md", "docs/guide.md"], [r["path"] for r in provenance["sourcePaths"]])
        self.assertEqual([hashlib.sha256(self.readme).hexdigest(), hashlib.sha256(self.guide).hexdigest()],
                         [r["sha256"] for r in provenance["sourcePaths"]])
        self.assertEqual([1, 2], [r["parts"] for r in provenance["sourcePaths"]])
        self.assertEqual(frozen["baseRevision"], provenance["baseRevision"])
        self.assertEqual("working-tree-dirty", provenance["sourceState"])
        self.assertEqual(hashlib.sha256(EXPLORE.SOURCE_MANIFEST.read_bytes()).hexdigest(), provenance["sourceFreezeManifestSha256"])
        self.assertEqual(source_before, {p.relative_to(self.source): p.read_bytes() for p in self.source.rglob("*") if p.is_file()})
        self.assertEqual(self.readme, (self.repository / "README.md").read_bytes())
        self.assertEqual(self.guide, (self.repository / "docs/guide.md").read_bytes())
        self.assertEqual(1, len(provenance["assets"]))
        asset = provenance["assets"][0]
        self.assertEqual("assets/icon.svg", asset["sourcePath"])
        self.assertEqual(hashlib.sha256(self.asset_bytes).hexdigest(), asset["sha256"])
        self.assertEqual(self.asset_bytes, (self.content / asset["file"]).read_bytes())
        for item in provenance["publicFiles"]:
            with self.subTest(path=item["path"]):
                data = (self.landing / item["path"]).read_bytes()
                self.assertEqual(len(data), item["bytes"])
                self.assertEqual(hashlib.sha256(data).hexdigest(), item["sha256"])
        self.assertEqual(sorted(item["path"] for item in provenance["publicFiles"]), provenance["generatedFiles"])
        self.assertNotIn("explore-content/manifest.json", provenance["generatedFiles"])
        self.assertNotIn(str(self.root), json.dumps(provenance))
        mappings = provenance["presentationCopy"]["files"]
        self.assertEqual([{"path": "README.md", "mappings": [{"from": "The right secrets, ready for your apps.",
                          "to": "Get the right secrets ready.", "replacements": 1}]}], mappings)

    def test_real_markdown_routes_parts_assets_context_and_sanitizes_raw_html(self) -> None:
        self.rich_snapshot()
        EXPLORE.build()
        readme = self.module("readme.js")
        guide = self.module("docs-guide.js")
        self.assertEqual("Get the right secrets ready.", readme["title"])
        self.assertEqual(["part-1", "part-2"], [p["id"] for p in guide["parts"]])
        html = "".join(p["html"] for p in readme["parts"])
        guide_html = "".join(p["html"] for p in guide["parts"])
        self.assertIn('/explore#doc=docs-guide&amp;part=part-2&amp;anchor=second', html)
        self.assertIn('/explore#doc=readme&amp;part=part-1&amp;anchor=the-right-secrets-ready-for-your-apps', html)
        self.assertIn('/explore#doc=docs-guide&amp;part=part-1&amp;anchor=absent', guide_html)
        self.assertIn('id="second"', guide_html)
        self.assertIn('/blob/synthetic-context/assets/icon.svg#owned', html)
        self.assertIn('/tree/synthetic-context/docs', html)
        self.assertIn('/blob/synthetic-context/missing.md#missing', html)
        self.assertIn('/tree/synthetic-context/unknown', html)
        self.assertIn('title="' + EXPLORE.REPOSITORY_REFERENCE_NOTE + '"', html)
        self.assertIn('href="https://example.invalid/?a=1&amp;b=2"', html)
        self.assertIn('href="mailto:example@example.invalid"', html)
        self.assertIn('href="https://image.example.invalid/owned.svg"', html)
        self.assertNotIn('src="https://', html)
        self.assertNotIn("forbidden-script", html)
        self.assertNotIn("<details", html)
        self.assertIn('<section class="explore-details"><h3>Owned details</h3>', html)
        self.assertIn("<table>", html)
        self.assertIn("deleted", html)
        digest = hashlib.sha256(self.asset_bytes).hexdigest()
        self.assertIn('/explore-content/assets/' + digest + '.svg', html)
        self.assertNotIn('src="assets/', html)
        provenance = json.loads((self.content / "manifest.json").read_bytes())
        self.assertGreater(provenance["rawHtmlSanitized"]["README.md"], 0)
        self.assertTrue(any(r["uri"] == "https://image.example.invalid/owned.svg" for r in provenance["externalImagesRenderedAsLinks"]))
        self.assertEqual({("README.md", "missing.md#missing"), ("README.md", "unknown/"),
                          ("docs/guide.md", "../assets/missing.svg")},
                         {(r["document"], r["target"]) for r in provenance["unresolvedRelativeLinks"]})

    def test_check_is_readonly_then_refuses_each_changed_missing_and_extra_output(self) -> None:
        self.rich_snapshot()
        EXPLORE.build()
        before = self.output_bytes_and_times()
        EXPLORE.build(check=True)
        self.assertEqual(before, self.output_bytes_and_times())
        cases = [(self.content / "readme.js", b"changed"),
                 (self.landing / "explore-manifest.js", None),
                 (self.content / "stale.js", b"stale"),
                 (self.content / "assets" / "stale.svg", b"stale")]
        for path, changed in cases:
            with self.subTest(path=path.name):
                original = path.read_bytes() if path.exists() else None
                path.parent.mkdir(parents=True, exist_ok=True)
                if changed is None:
                    path.unlink()
                else:
                    path.write_bytes(changed)
                changed_snapshot = self.output_bytes_and_times()
                with self.assertRaisesRegex(SystemExit, "^explore snapshot is stale: ") as refusal:
                    EXPLORE.build(check=True)
                self.assertIn(path.relative_to(self.landing).as_posix(), str(refusal.exception))
                self.assertEqual(changed_snapshot, self.output_bytes_and_times())
                if original is None:
                    path.unlink()
                else:
                    path.write_bytes(original)
        EXPLORE.build(check=True)

    def test_rebuild_removes_only_stale_owned_js_and_assets_and_preserves_other_files(self) -> None:
        self.rich_snapshot()
        EXPLORE.build()
        stale = [self.content / "stale.js", self.content / "assets" / "stale.svg"]
        protected = [self.content / "keep.txt", self.content / "assets" / "directory" / "keep.txt"]
        for path in stale + protected:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"owned marker")
        EXPLORE.build()
        self.assertTrue(all(not path.exists() for path in stale))
        for path in protected:
            self.assertEqual(b"owned marker", path.read_bytes())
        EXPLORE.build(check=True)

    def test_refresh_build_recaptures_only_declared_owned_documents(self) -> None:
        self.rich_snapshot()
        self.write_document("README.md", b"# New owned heading\n\nBody\n")
        self.write_document("private-not-selected.md", b"not public snapshot input")
        with mock.patch.object(EXPLORE, "PRODUCT_ASSETS", ("assets/icon.svg",)):
            EXPLORE.build(refresh_local=True)
        self.assertEqual(b"# New owned heading\n\nBody\n", (self.source / "README.md").read_bytes())
        self.assertFalse((self.source / "private-not-selected.md").exists())
        self.assertEqual("New owned heading", self.module("readme.js")["title"])

    def test_frozen_manifest_context_fallback_and_empty_document_are_explicit(self) -> None:
        self.write_document("README.md", b"plain body\n")
        self.write_document("docs/guide.md", b"")
        self.write_document("assets/icon.svg", b"owned")
        EXPLORE.refresh_local_source()
        manifest = json.loads(EXPLORE.SOURCE_MANIFEST.read_bytes())
        del manifest["repositoryContextRef"]
        EXPLORE.SOURCE_MANIFEST.write_text(json.dumps(manifest))
        EXPLORE.build()
        self.assertEqual("README", self.module("readme.js")["title"])
        self.assertEqual([], self.module("docs-guide.js")["parts"])
        output = (self.landing / "explore-manifest.js").read_text()
        self.assertTrue(output.startswith("export const documentation="))
        data = json.loads(output[len("export const documentation="):-2])
        self.assertEqual("feature/initial-implementation", data["repositoryContextRef"])
        self.assertEqual(["readme", "docs-guide"], [doc["id"] for doc in data["documents"]])
        self.assertEqual("./explore-source/manifest.json", data["sourceFreeze"])

    def test_missing_declared_asset_is_not_hashed_or_silently_fabricated(self) -> None:
        self.rich_snapshot()
        EXPLORE.build()
        data = json.loads((self.content / "manifest.json").read_bytes())
        self.assertEqual(["assets/icon.svg"], [r["sourcePath"] for r in data["assets"]])
        self.assertFalse((self.source / "assets/missing.svg").exists())
        self.assertNotIn("missing.svg", json.dumps(data["assets"]))
        self.assertTrue(any(r["uri"] == "assets/missing.svg" for r in data["externalImagesRenderedAsLinks"]))

    def test_raw_html_present_directory_and_missing_asset_refusals_use_owned_paths(self) -> None:
        self.write_document("README.md", b'# Owned\n\n<img src="assets/dir.svg" alt="directory">\n\n![dir](assets/dir.svg)\n')
        self.write_document("docs/guide.md", b"# Guide\n")
        self.write_document("assets/icon.svg", b"owned")
        EXPLORE.refresh_local_source()
        (self.source / "assets/dir.svg").mkdir()
        self.enterContext(mock.patch.object(EXPLORE, "PRODUCT_ASSETS", ("assets/dir.svg",)))
        EXPLORE.build()
        self.assertEqual([], json.loads((self.content / "manifest.json").read_bytes())["assets"])
        self.assertNotIn('src="assets/dir.svg"', self.module("readme.js")["parts"][0]["html"])

    def test_asset_pin_mismatch_and_invalid_document_bytes_stop_before_any_output(self) -> None:
        self.rich_snapshot()
        original = (self.source / "assets/icon.svg").read_bytes()
        (self.source / "assets/icon.svg").write_bytes(b"changed owned asset")
        with self.assertRaisesRegex(SystemExit, "^frozen source hash mismatch: assets/icon\\.svg$"):
            EXPLORE.build()
        self.assertFalse(self.content.exists())
        (self.source / "assets/icon.svg").write_bytes(original)
        manifest = json.loads(EXPLORE.SOURCE_MANIFEST.read_bytes())
        invalid = b"\xff"
        (self.source / "README.md").write_bytes(invalid)
        for item in manifest["files"]:
            if item["path"] == "README.md":
                item["sha256"] = hashlib.sha256(invalid).hexdigest()
                item["bytes"] = len(invalid)
        EXPLORE.SOURCE_MANIFEST.write_text(json.dumps(manifest))
        with self.assertRaises(UnicodeDecodeError):
            EXPLORE.build()
        self.assertFalse(self.content.exists())

    def test_empty_selected_snapshot_emits_honest_empty_manifest_without_private_discovery(self) -> None:
        self.source.mkdir()
        manifest = {"baseRevision": "3" * 40, "files": [], "repositoryContextRef": "owned-empty"}
        EXPLORE.SOURCE_MANIFEST.write_text(json.dumps(manifest))
        (self.source / "private-not-selected.md").write_bytes(b"not selected")
        EXPLORE.build()
        provenance = json.loads((self.content / "manifest.json").read_bytes())
        self.assertEqual(0, provenance["documentCount"])
        self.assertEqual([], provenance["sourcePaths"])
        self.assertEqual([], provenance["assets"])
        self.assertEqual([], provenance["externalImagesRenderedAsLinks"])
        self.assertEqual([], provenance["unresolvedRelativeLinks"])
        self.assertFalse((self.content / "private-not-selected.js").exists())
        EXPLORE.build(check=True)

    def test_actual_cli_copy_detached_import_and_refresh_check_stay_owned(self) -> None:
        repository = self.root / "cli-repository"
        landing = repository / "examples/landing-motion"
        landing.mkdir(parents=True)
        script = landing / "build-explore.py"
        script.write_bytes(SCRIPT.read_bytes())
        inputs = {"documents": ["README.md"], "assets": [], "repositoryContextRef": "owned context"}
        (landing / "explore-documents.json").write_text(json.dumps(inputs))
        (repository / "README.md").write_text("# Owned CLI\n\nBody\n")
        def metadata(arguments, *, cwd, text):
            self.assertEqual(repository, cwd)
            self.assertIs(text, True)
            if arguments == ["git", "rev-parse", "HEAD"]:
                return "2" * 40 + "\n"
            if arguments == ["git", "branch", "--show-current"]:
                return "\n"
            raise AssertionError("undeclared CLI subprocess")
        with mock.patch.object(subprocess, "check_output", side_effect=metadata) as observation, \
                mock.patch.object(sys, "argv", [str(script), "--refresh-local-source"]):
            namespace = runpy.run_path(str(script), run_name="__main__")
        self.assertEqual("2" * 40, namespace["CURRENT_BRANCH"])
        self.assertEqual(2, observation.call_count)
        generated = landing / "explore-content/readme.js"
        before = (generated.read_bytes(), generated.stat().st_mtime_ns)
        with mock.patch.object(subprocess, "check_output", side_effect=metadata), \
                mock.patch.object(sys, "argv", [str(script), "--check"]):
            runpy.run_path(str(script), run_name="__main__")
        self.assertEqual(before, (generated.read_bytes(), generated.stat().st_mtime_ns))
        self.assertIn(b"Owned CLI", before[0])

    def test_sanitizers_retain_safe_text_entities_and_refuse_nested_executable_content(self) -> None:
        external = []
        parser = EXPLORE.SafeHtml("README.md", lambda _: None, external)
        parser.feed('<script><style>hidden&amp;&#65;</style>hidden</script><iframe>hidden</iframe><p>A&amp;&#65;<br/></p><source><picture></picture><section class="other">safe</section>')
        parser.close()
        self.assertEqual('<p>A&amp;&#65;<br></p><source><picture></picture><section>safe</section>', "".join(parser.out))
        self.assertEqual([], external)
        self.assertEqual('<h2 id="owned">Title</h2><code>x</code><source>',
                         EXPLORE.sanitize_rendered_html('<h2 id="owned" bad="x">Title</h2><code class="other">x</code><source srcset="https://example.invalid/x">'))
        self.assertEqual('<p>x</p>', EXPLORE.sanitize_rendered_html('<p>x</wrong></p>'))

    def test_remote_image_attribute_and_anchor_lifetimes_are_preserved(self) -> None:
        images = []
        rendered = EXPLORE.replace_remote_images('<a href="https://example.invalid"><span>owned</span>&amp;&#65;<!--note--><img src="https://image.invalid/x" alt="x"></a><img src="/local"><img/>', "README.md", images)
        self.assertEqual('<a href="https://example.invalid"><span>owned</span>&amp;&#65;<!--note-->x</a><img src="/local"><img/>', rendered)
        self.assertEqual([{"document": "README.md", "uri": "https://image.invalid/x", "alt": "x"}], images)

    def test_heading_fences_parts_and_details_preserve_boundaries_and_literal_copy(self) -> None:
        source = "intro\n~~~text\n# hidden\n~~~\n" + "owned " * 2501 + "\n## Next ##\nlast\n"
        headings, parts = EXPLORE.headings_and_parts(source, "README.md")
        self.assertEqual([{"id": "next", "title": "Next", "level": 2}], headings)
        self.assertEqual(["Overview", "Next"], [p["title"] for p in parts])
        self.assertEqual(source, "".join(p["source"] for p in parts))
        self.assertEqual('before<section class="explore-details"><h3>A</h3>B</section>between<section class="explore-details"><h3>C</h3>D</section>after',
                         EXPLORE.flatten_details('before<DETAILS><SUMMARY>A</SUMMARY>B</DETAILS>between<details><summary>C</summary>D</details>after'))
        self.assertEqual('<details>prefix<summary>A</summary>B</details>', EXPLORE.flatten_details('<details>prefix<summary>A</summary>B</details>'))



if __name__ == "__main__":
    unittest.main()
