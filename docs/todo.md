# Implementation TODOs for the first workable build

The repository now includes a thin server skeleton and manifest. These steps
should lead to a buildable plugin that can be uploaded to Mattermost.

## Progress tracker

- [x] Wire Mattermost client helpers
  - Implemented `MMClient` wrapper to create posts (with attachments), read
    channels/users, and store/load JSON from KV with optional debug logging.
  - Next: add opinionated helpers for thread updates and richer post updates.

- [x] Configuration and validation
  - Implemented validation for `BugsnagAPIToken` and webhook secret/token and
    surfaced errors via `OnConfigurationChange`.
  - Remaining: persist project/channel mappings and user mappings in KV.

- [ ] Webhook handler
  - Added validation for webhook token/secret from query/header.
  - Implemented project→channel mapping storage (KV), rule-based filtering
    (environment/severity/event type), and error→post mapping so repeated
    webhooks update the same card with a thread note about the event.
  - Optional `channel_id` query override remains for provisional posting while
    admin UI endpoints are still pending.
  - Remaining: normalize Bugsnag payloads, enrich card copy/fields, and plug in
    real project/channel mappings from the admin UI API.

- [ ] Interactive actions
  - Implemented: resolve Mattermost user → Bugsnag user mapping (KV + email
    fallback), log action context, write a thread note on the linked card, and
    call Bugsnag API endpoints for assignment and status changes (resolve/
    ignore) with timeouts.
  - Remaining: refresh the card footer/status after successful Bugsnag calls
    and align status values with the final admin-configurable mapping.

- [ ] Scheduler
  - Pending: track active errors in KV and poll Bugsnag on an interval for
    spikes, last seen, and counts over 1h/24h; update posts accordingly.

- [ ] Webapp admin UI
  - Pending: repurpose starter-template tabs for Connection, Projects &
    Channels, User Mapping, and Notification Rules, plus REST endpoints to serve
    and persist these configs.

- [ ] Packaging
  - Pending: mirror the starter-template `Makefile` to build `server/dist`
    binaries and `webapp/dist/main.js` bundle; ensure `plugin.json` paths align
    for `make plugin`.
