# CircularQuay implementation and review status

| Area | Implemented behavior | Local verification |
| --- | --- | --- |
| Focused hero | Eight primary nodes, central Managed policies, six optional capabilities; “Get the right secrets ready.” | Assessment A approved, 28/32 |
| Full catalogue motion | 30 IDP, 35 encrypted-source and 48 application entries; 108 distinct SVGs; fair bags across 24 actors and 12 contexts; static no-JS fallback | Assessment B passed: 900 local/worker routes byte-identical, with six current canonical and three alias live-route hashes verified |
| Optional workload flow | Balanced shuffled none/action 1/action 2/both masks; first scene includes one optional action | 8,000-render catalogue probe passed without overflow |
| Compact landing | Single-line motto, direct first-featured Explore link, upward wheel/swipe to Explore, continuously animated eight-node diagram, and no horizontal diagram/story divider | Assessment A approved; desktop wheel and mobile touch navigation verified live |
| Diagram finish | Active/delivered routes and node overlays use 2px emphasis; idle bases remain lighter; hub base is borderless; 18px text padding | Assessment A approved |
| Responsive shell | Mobile icon/name/purpose face grid with 18px row gutter; stable shared header gutter/padding; language before theme ≤760px | Assessment A approved |
| Loader and keys | Full-viewport centred logo loader; neutral-grey opacity breath; reduced still; Escape closes search dropdown while the diagram retains Escape pause | Assessment A and B approved |
| Explore | Six groups, 55 documents, 85 parts; compact index with real-heading summaries and fewer rules; Help distribution/autocomplete; current cq/cq-sdk guides | Assessment A approved, 34/36; 611 closed reader links; 59-file frozen source; narrow tables scroll at 9rem minimum cells |
| Shared shell | Consistent header, themes and discreet bottom footer; compact Explore menu for larger layouts; outlined text Help control | Assessment A approved |
| Login | Thirty provider choices, account actions and real footer in one document scroll; feedback only after interaction | Assessment A approved, 33/40; no real sign-in or email persistence |
| Service boundary | Eighteen disabled integration points with fail-closed consent/capability gates | Five service tests pass |

The local source freeze is based on `15f368a9f8ac4b45f5d92772c492e863b0a0f5f9`, with SHA-256 `abb8878ee734dbe398f98c15c8abd22de1e3a511e1620d2de9917f4fd0d16b45`. Frozen Markdown and assets are hash-recorded and reproducible after later Git commits. `--refresh-local-source` explicitly replaces the snapshot.

The public frontend is live at `https://circularquay.aldous.info/`; `cq.aldous.info` is a proxied CNAME alias to it. The Worker is deployed at 100% with ASSETS binding only; workers.dev and preview URLs are disabled. No application backend is included in this deployment scope. Authentication/provisioning, configured vendor adapters, approved legal/CMP content and device/assistive-technology qualification remain deployment requirements. Design assessment and engineering provenance are kept in `CANDIDATE-REVIEW.md` and `review.json`.
