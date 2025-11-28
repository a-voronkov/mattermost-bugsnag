# Mattermost Bugsnag Integration Plugin

Technical specification and draft architecture for a Mattermost plugin integrated with Bugsnag. The project aims to deliver error notifications into Mattermost channels, provide quick status actions, and offer easy administration through the System Console.

## Key capabilities

- Connect to Bugsnag using a personal API token and optional Organization ID.
- Receive error events via webhook and post cards with action buttons into selected Mattermost channels.
- Track statuses, recurrence, and last-occurrence timestamps with updates in threads.
- Map Bugsnag users ↔ Mattermost users for correct mentions and reassignment from cards.
- Admin UI in System Console: connectivity, project-to-channel mapping, notification rules, and user mapping.

## High-level architecture

### Server Plugin (Go)

- Endpoints `/plugins/bugsnag/webhook` for events and `/plugins/bugsnag/actions` for interactive buttons.
- Bugsnag API client for projects, errors, and status/assignee management.
- Mattermost Plugin API for creating/updating posts and storing mappings in KV.
- Planned periodic sync of active errors to keep statistics fresh.

### Webapp Plugin (React)

- System Console settings page with tabs for Connection, Projects & Channels, User Mapping, and Notification Rules.
- Optional UI extensions (channel header button, RHS panel) for browsing error details.

### Bugsnag

- Webhook configured in the Bugsnag UI sends events to the plugin’s public URL with a secret token.
- REST API used to read error/project data and perform actions (status changes, assignee updates).

## Main user flows

1. **Connection**: administrator enters the API token and, if needed, the Organization ID. The plugin fetches organizations/projects and stores them in KV.
2. **Webhook setup**: the admin UI shows a URL like `https://<mm-host>/plugins/bugsnag/webhook?token=<secret>`, which is added in the Bugsnag project settings.
3. **Incoming errors**: webhook events are filtered by configured channels, environments, and event types; new errors create cards, existing ones get metric updates and thread entries.
4. **Interactive actions**: buttons “Assign to me”, “Resolve”, “Ignore”, “Open in Bugsnag” hit the server handler, which maps users and updates the error via the Bugsnag API.
5. **Periodic sync**: active errors are polled at intervals; cards and threads are updated with fresh stats and significant changes.

## Supporting documents

- Full technical description of requirements, data flows, and UI lives in [`initial-plan.md`](initial-plan.md).

## Repository status

- Minimal server plugin scaffold in Go (`server/`):
  - `plugin.go` registers `/webhook` and `/actions` via `ServeHTTP` and loads configuration.
  - `webhook.go` validates tokens, applies project→channel mapping rules (with
    filters for environment/severity/event), stores error→post mappings, and can
    render a provisional card when `channel_id` is supplied in the webhook query.
  - `actions.go` accepts payloads, maps Mattermost users to Bugsnag users (KV +
    email fallback), records action notes in the corresponding error thread, and
    invokes the Bugsnag API client for assignment and status updates.
  - `bugsnag_client.go` is a focused HTTP client for status and assignment
    updates with API token auth.
  - `message_templates.go` contains draft card/action structures.
  - `mm_client.go` wraps Mattermost API calls for posts, KV JSON, users, and channels
    with optional debug logging.
- `plugin.json` defines the plugin manifest, admin settings, and expected build artifacts.
- `docs/` holds sample payloads and the TODO checklist for turning the scaffold into a working build.

Next step: copy the Makefile and webapp from `mattermost-plugin-starter-template`, attach REST endpoints for the admin console, and implement webhook/interactive logic per `docs/todo.md`.
