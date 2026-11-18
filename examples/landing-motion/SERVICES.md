# Landing browser-service scaffold

This production-candidate frontend scaffold has no vendor SDK, CDN script,
network request, identifier, DSN, advertising slot, host target, consent record,
or entitlement. Importing `initialiseBrowserServices({ surface })` produces a
frozen status/API and defaults every service off in `preview` mode. The landing
page must call that source hook explicitly; an adapter is a later, reviewed
handoff, not an implementation claim.

## Guardrails

- `public-landing` is the only currently eligible event surface. `auth-page`
  and `app-account` do not receive diagnostics, advertising, or analytics by
  default.
- A callback can run only with qualified mode, `enabled`, valid public
  configuration, state `independently verified`, a qualified surface, a
  declared consent purpose, and currently granted consent. Withdrawal calls a registered adapter's bounded `stop`
  callback and blocks later events. `dispose()` is idempotent.
- Diagnostics contain only the fixed codes in `service-config.js`. Raw errors,
  stacks, URLs, user/email, configuration, source selectors, credentials and
  unknown properties are never projected to an adapter. Replay is absent.
- The only production origins accepted by the config validator are
  `https://circularquay.aldous.info` and `https://cq.aldous.info`; they remain
  unusable until the same authorized, verified hosting target is identified.
  Local preview never becomes configured merely by running on localhost.

## Inventory and present state

Every service below is `pending applicability`, disabled and unconfigured in
the manifest. Presence is planning inventory only.

| Service | Candidate surface | Missing before any activation |
| --- | --- | --- |
| Sentry technical diagnostics | public landing | CQ project/routing reference, release/sampling/retention/redaction review, consent and qualified surface |
| iubenda consent | public landing, auth page | CQ site registration, licensed policy languages, purpose/vendor and privacy approval |
| AdSense; Google Ads / AdWords | public landing only | suitability, CQ approval/slots or conversion decision, and consent qualification |
| Cloudflare Zaraz | public landing only | CQ zone/tool, coarse-purpose allowlist, consent and surface qualification |
| Support; feedback | public landing, auth page | selected route/provider, owner, consent/privacy review |
| Brevo transactional email | none; conditional and not selected | approved route, explicit provider selection, sender/domain and synthetic-recipient qualification |
| Cloudflare/Netlify hosting; Cloudflare/Netlify identity; database/storage | no browser adapter | verified project/issuer/bindings/custody/roles/capacity and restore evidence |
| Status/monitoring; discovery; engineering | status/discovery may later be public landing | selected provider/property/owner and independent verification |
| PWA; Android | PWA may later be public landing | shipped platform/package, permission, cache, accessibility and policy qualification |

Google Analytics is deliberately absent: it must not be inferred from Zaraz or
Google advertising. No support platform or Brevo selection is implied.

## Adapter handoff

An accountable owner must select the service and enter a reviewed public config
outside this repository. A narrow adapter can then be registered after calling
`initialiseBrowserServices({ surface, config })`. The scaffold accepts only
literal production origins plus syntactically bounded public project/release
identifiers; it does not authenticate accounts or validate entitlement.

An iubenda/Zaraz bridge can call `setConsent({ purpose, granted })` for a
reviewed purpose. No vendor consent state is fabricated here. The integration
must prove prior blocking, purpose-specific acceptance and withdrawal through
network/storage inspection before deployment.

Production remains blocked on actual service IDs/routing references, legal and
consent completion, service/site approval where applicable, selected hosting
target, qualification evidence, CSP review, and owner/operator readback. See
the canonical [release procedure](../../RELEASE.md#product-go-live-and-commercialization-service-provisioning),
[Sentry data filtering](https://github.com/getsentry/sentry-docs/blob/master/platform-includes/configuration/before-send/javascript.mdx),
[iubenda setup](https://www.iubenda.com/en/help/1177-cookie-solution-getting-started/),
[AdSense approval](https://support.google.com/adsense/answer/7584263?hl=en),
and [Zaraz consent management](https://developers.cloudflare.com/zaraz/consent-management/enable-consent-management/).
