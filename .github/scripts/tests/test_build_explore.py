"""Owned-input documentation helpers, not a rendered snapshot qualification.

The real module's two import-time Git observations and explicit document-list
read are intercepted. No checked-in corpus is read, no subprocess is launched,
and no Markdown parser/build/browser or refreshed product output is selected.
The real import uses the existing SDK-lock MarkdownIt4.2.0 dependency; these
oracles do not qualify the browser renderer pinned separately to3.0.0.
"""

from __future__ import annotations

import hashlib
import importlib.util
import json
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


if __name__ == "__main__":
    unittest.main()
