# Implementation TODOs

The repository includes a working server implementation with modular packages.

## Completed

- [x] **Mattermost client helpers** — `MMClient` wrapper for posts, KV, users, channels
- [x] **Configuration and validation** — API token, webhook secret, project/channel/user mappings in KV
- [x] **Webhook handler** — token validation, project→channel routing with filters, error→post mapping
- [x] **Interactive actions** — user mapping, thread notes, Bugsnag API calls for assign/resolve/ignore
- [x] **Bugsnag API client** — `server/bugsnag/` package with GetProjects, UpdateErrorStatus, AssignError
- [x] **Post formatter** — `server/formatter/` package for error cards with action buttons
- [x] **Store abstraction** — `server/store/` package with KVStore interface
- [x] **Scheduler** — `server/scheduler/` package for periodic sync
- [x] **API endpoints** — `server/api/` package (test endpoint)
- [x] **Code quality cleanup** — consolidated clients, centralized constants in `kvkeys` package

## In Progress

- [ ] Refresh card footer/status after successful Bugsnag API calls
- [ ] Integrate scheduler with real Bugsnag API (currently uses mock)

## Pending

- [ ] **Webapp admin UI**
  - React components for Connection, Projects & Channels, User Mapping, Notification Rules tabs
  - Expand REST endpoints in `server/api/` to serve and persist configs

- [ ] **Packaging**
  - Add `Makefile` for building server binaries and webapp bundle
  - Ensure `plugin.json` paths align for `make plugin`

- [ ] **Documentation**
  - Add CONTRIBUTING.md
  - Add deployment guide
  - Add integration tests
