# CircularQuay implementation and review status

| Area | Implemented behavior | Local verification |
| --- | --- | --- |
| Focused hero | Eight primary nodes, central Managed policies, six optional capabilities; “Get the right secrets ready.” | Impeccable visual review approved |
| Compact landing | Single-line motto, direct first-featured Explore link, continuously animated eight-node diagram, and no horizontal diagram/story divider | Current-browser delta review pending |
| Diagram finish | All routes and overlays are 2px; hub base is borderless; 18px text padding | Current-browser delta review pending |
| Responsive shell | Mobile icon/name/purpose face grid with 18px row gutter; stable shared header gutter/padding; language before theme ≤760px | Current-browser delta review pending |
| Loader and keys | Full-viewport centred logo loader; neutral-grey breath and clockwise logo-band arc; reduced still; Escape closes search dropdown while the diagram retains Escape pause | Current-browser delta review pending |
| Explore | Six groups, 55 documents, 85 parts, implemented Help distribution/autocomplete, current cq/cq-sdk guides | 611 closed reader links; 59-file frozen source |
| Shared shell | Consistent header, themes and discreet bottom footer; compact Explore menu for larger layouts; outlined text Help control | Current-browser delta review pending |
| Login | Thirty searchable provider choices and local account-action feedback | Browser interaction checks; no real sign-in or email persistence |
| Service boundary | Eighteen disabled integration points with fail-closed consent/capability gates | Five service tests pass |

The local source freeze is based on `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`, with SHA-256 `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Frozen Markdown and assets are hash-recorded and reproducible after later Git commits. `--refresh-local-source` explicitly replaces the snapshot.

The frontend is available for local review. Authentication/provisioning, configured vendor adapters, approved legal/CMP content, production domains and device/assistive-technology qualification remain deployment requirements. Design assessment and engineering provenance are kept in `CANDIDATE-REVIEW.md` and `review.json`.
