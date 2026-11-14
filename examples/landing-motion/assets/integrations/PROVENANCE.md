# Integration asset provenance

Observed source study: https://circularquay-motion-study.matthewaldous.chatgpt.site/
Observed: 2026-11-05T05:46:14.812Z

Most local SVGs were downloaded from the official Simple Icons repository
(CC0-1.0). Where that catalogue did not supply a named vector, the catalog uses
the public Devicon v2.16.0 vector (MIT) for PowerShell, Oracle and shared AWS
vendor roles. Each asset is pinned by SHA-256 in `../../integrations.json`; no
icon is loaded at runtime from a third party.

OneDrive and SharePoint use their pre-removal Simple Icons vectors at
`ce334b5bda8d8d054cfde7ce35caf40651078a28`; IBM Cloud uses the pre-removal
vector at `31a228328108720309161728ac48cdd2fb570331`. The source project later
removed Microsoft and IBM marks. Their retained historical status and removal
commits are recorded in the catalog, and they do not assert current CircularQuay
support. MCP uses the official `docs/favicon.svg` from the
modelcontextprotocol/modelcontextprotocol documentation under its documented
CC-BY-4.0 terms. `local-filesystem.svg` is an authored non-brand symbol, not a
provider mark.

`LICENSES.md` retains the required Devicon MIT notice and the exact upstream
Model Context Protocol documentation licensing statement, attribution links and
applicable CC-BY-4.0 licence link for later local reuse or redistribution.

The catalog retains all 113 source-study entries across AUTHENTICATION (30), ENCRYPTION (35) and APPLICATION (48). A logo only illustrates a named ecosystem example. It does not claim a supported, qualified, configured, connected or authorized CircularQuay adapter.

Entries with a local SVG: 113
Distinct retained SVG files: 108
Named-wordmark fallbacks: 0

## Colour provenance

The retained Simple Icons vectors are colour-transformed only after checking
their upstream `hex` values in `data/simple-icons.json` and the slug mapping in
`slugs.md`. The local file has that verified colour as its root `fill`. For each
such item, `integrations.json` records both `upstreamSha256` (the downloaded
upstream SVG) and `sha256` (the retained coloured SVG), plus the exact transform
value. The historical Microsoft and IBM marks use the matching metadata from
their pinned source revisions. Devicon assets retain their source colours; the
official MCP favicon and the authored local-filesystem glyph retain the colours
documented with their individual entries.
