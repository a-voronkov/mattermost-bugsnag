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
- [x] **Scheduler** — `server/scheduler/` package for periodic sync with real Bugsnag API
- [x] **API endpoints** — `server/api/` package with test, projects, organizations, user-mappings, channel-rules
- [x] **Code quality cleanup** — consolidated clients, centralized constants in `kvkeys` package
- [x] **Refresh card** — update post status after successful resolve/ignore/assign actions

- [x] **Webapp admin UI**
  - [x] Connection tab — test Bugsnag API connection
  - [x] Projects & Channels tab — project→channel mapping
  - [x] User Mapping tab — Mattermost↔Bugsnag user mapping
  - [x] Notification Rules tab — environment/severity/event filters

- [x] **Packaging**
  - [x] Makefile for building server binaries and webapp bundle
  - [x] package.json and webpack.config.js for webapp

- [x] **Documentation**
  - [x] Add CONTRIBUTING.md
  - [x] Add deployment guide
  - [x] Add integration tests

## Pending

- [ ] **End-to-end testing** — test full flow with real Mattermost and Bugsnag instances
- [ ] **CI/CD** — GitHub Actions for build, test, and release
