# Mattermost Bugsnag Integration Plugin — Technical Specification

## 1. Plugin goals

The Mattermost plugin integrated with Bugsnag must:

1. Connect to Bugsnag via API using a personal API token.
2. Receive error events through a Bugsnag webhook.
3. Post **bug cards** with action buttons into selected Mattermost channels.
4. Keep error information up to date:
   * current status;
   * occurrence counts for the last N minutes/hours;
   * date and time of the most recent occurrence.
5. Maintain **change history** (status changes, reassignments, spikes, etc.) in threads attached to those cards.
6. Support **user mapping** between Bugsnag and Mattermost users:
   * for @mentions of assignees in messages;
   * for setting assignees in Bugsnag when buttons are pressed in Mattermost.
7. Provide a **System Console admin UI**:
   * enter the Bugsnag API token/organization;
   * configure project-to-channel mappings;
   * configure notification rules;
   * configure user mappings.

---

## 2. Architecture

### 2.1. Components

1. **Server plugin (Go)**
   * HTTP endpoint for receiving the Bugsnag webhook: `/plugins/bugsnag/webhook`.
   * HTTP endpoint for processing button presses (interactive actions): `/plugins/bugsnag/actions`.
   * Client for the Bugsnag REST API.
   * Mattermost Plugin API usage:
     * create and update posts;
     * create threads and replies;
     * KV storage (mappings, active errors, configuration).
   * Periodic scheduler (goroutine) to refresh statistics for active errors.

2. **Webapp plugin (JS/TS/React)**
   * Settings page in System Console → Plugins → Bugsnag Integration.
   * (Optional) additional UI elements:
     * channel header button;
     * right-hand side (RHS) panel for viewing error details/filters.

3. **Bugsnag**
   * Webhook to `https://<mattermost-host>/plugins/bugsnag/webhook?token=<secret>`.
   * Data Access API and other public APIs:
     * get organizations/projects/errors;
     * read details of a specific error;
     * change status/assignee (where endpoints permit);
     * fetch users/collaborators (if available).

---

## 3. Data flows

### 3.1. Pairing the plugin with Bugsnag

1. A Mattermost administrator opens System Console → Plugins → Bugsnag Integration.
2. Inputs:
   * Bugsnag API Token;
   * (optional) Organization ID.
3. Clicks **Test connection / Load projects**.
4. The server plugin:
   * calls the Bugsnag API (by token) to fetch organizations/projects;
   * stores the project list (id, name, slug) in KV.

### 3.2. Configuring the webhook in Bugsnag

> The webhook is assumed to be configured manually in Bugsnag UI
> (Project Settings → Integrations → Data forwarding → Webhook).

Steps:

1. The plugin generates a URL such as:
   `https://<mattermost-host>/plugins/bugsnag/webhook?token=<secret>`.
2. The admin UI displays this URL per project.
3. The administrator copies the URL and pastes it when creating/configuring the webhook in Bugsnag.
4. The desired event types are enabled in Bugsnag (new error, spike, frequent error, reopened, etc.).

### 3.3. New bug / new event

1. Bugsnag sends POST to `/plugins/bugsnag/webhook` with the error/event payload.
2. The server plugin:
   * validates the secret `token`;
   * parses the payload: error_id, project_id, status, summary, counts, last_seen, environment, etc.
3. By `project_id`, it reads from KV the list of **Mattermost channels** subscribed to that project:
   * filter by severity;
   * by environment (prod, staging, dev);
   * by event type (new, spike, reopened, etc.).
4. For each matching channel:
   * check whether a post already exists for `error_id` (via KV mapping `errorID+projectID → postID`);
   * if no post:
     * create a **new card** for the error in the channel;
     * store the mapping `errorID+projectID → postID, channelID`;
   * if a post exists:
     * update the card (status, last_seen, counters);
     * add a reply in the thread (change history).

### 3.4. Button press on the card

1. A user clicks a button (e.g., “Assign to me”, “Resolve”, “Ignore”).
2. Mattermost sends POST to `/plugins/bugsnag/actions` with:
   * `user_id` (Mattermost);
   * `context` (action, error_id, project_id, etc.).
3. The server plugin:
   * extracts error data from `context`;
   * fetches Mattermost user data by `user_id` (email, etc.);
   * uses the mapping table to find the corresponding Bugsnag user (by email or ID).
4. Calls the Bugsnag API:
   * change the error status (open / in-progress / fixed / ignored);
   * optionally change the assignee to the mapped Bugsnag user.
5. After Bugsnag succeeds:
   * update the card content (status, assigned to);
   * add a reply in the thread like
     `@mm-user changed status to "In progress" and assigned to @mm-user`;
   * optionally send an ephemeral response to the action initiator.

### 3.5. Periodic sync of statistics

1. The plugin stores a list of active errors (e.g., open/in-progress) in KV:
   * `error_id, project_id, post_id, channel_id, last_synced_at`.
2. A scheduler goroutine runs every N minutes:
   * iterates over active errors;
   * calls the Bugsnag API:
     * status;
     * event counts over the last X minutes/hours;
     * `last_seen`.
3. After updating:
   * modify the card/attachment text;
   * add a thread entry for significant changes (e.g., a sharp spike in events).

---

## 4. Message card format

### 4.1. Visual representation

Example post in a channel:

> :rotating_light: **[BUG]** NullReferenceException in CheckoutController
> Project: `my-backend-api` · Env: `production` · Status: **Open**

Attachment includes:

* Title: concise error summary.
* Link to the error in Bugsnag.
* Text:
  * Environment
  * Severity
  * Release / version
  * Number of affected users
  * Events for 1h / 24h
  * Last seen (UTC or local time)
* Footer: `Bugsnag • <project-name>`

Action buttons:

* `🙋 Assign to me`
* `✅ Resolve`
* `🙈 Ignore`
* `🔗 Open in Bugsnag`

### 4.2. Simplified JSON payload (logic, not final code)

```json
{
  "channel_id": "<channel-id>",
  "message": ":rotating_light: **[BUG]** NullReferenceException in CheckoutController",
  "props": {
    "attachments": [
      {
        "fallback": "Bugsnag error",
        "title": "NullReferenceException: Object reference not set...",
        "title_link": "https://app.bugsnag.com/org/project/errors/ERROR_ID",
        "text": "Env: production | Severity: error | Users: 123 | Events (1h/24h): 5 / 42\nLast seen: 2025-11-28 10:23 UTC",
        "footer": "Bugsnag • my-backend-api",
        "actions": [
          {
            "id": "assign_me",
            "name": "🙋 Assign to me",
            "type": "button",
            "style": "primary",
            "integration": {
              "url": "https://<mm-host>/plugins/bugsnag/actions",
              "context": {
                "action": "assign_me",
                "error_id": "ERROR_ID",
                "project_id": "PROJECT_ID"
              }
            }
          },
          {
            "id": "resolve",
            "name": "✅ Resolve",
            "type": "button",
            "style": "primary",
            "integration": {
              "url": "https://<mm-host>/plugins/bugsnag/actions",
              "context": {
                "action": "resolve",
                "error_id": "ERROR_ID",
                "project_id": "PROJECT_ID"
              }
            }
          },
          {
            "id": "ignore",
            "name": "🙈 Ignore",
            "type": "button",
            "style": "default",
            "integration": {
              "url": "https://<mm-host>/plugins/bugsnag/actions",
              "context": {
                "action": "ignore",
                "error_id": "ERROR_ID",
                "project_id": "PROJECT_ID"
              }
            }
          },
          {
            "id": "open",
            "name": "🔗 Open in Bugsnag",
            "type": "button",
            "style": "link",
            "integration": {
              "url": "https://<mm-host>/plugins/bugsnag/actions",
              "context": {
                "action": "open_in_browser",
                "error_url": "https://app.bugsnag.com/org/project/errors/ERROR_ID"
              }
            }
          }
        ]
      }
    ]
  }
}
```

---

## 5. Admin UI (System Console)

The admin UI is divided into tabs.

### 5.1. **Connection** tab

Fields:

* **Bugsnag API Token** (required).
* **Organization ID** (optional).
* **Test connection** button:
  * call Bugsnag;
  * show status (OK/error) and basic info (e.g., list of organizations).

### 5.2. **Projects & Channels** tab

Capabilities:

* List Bugsnag projects (pulled via API).
* For each project:
  * Mattermost Team (dropdown).
  * Mattermost Channel (dropdown).
  * Environments (multi-select: prod, staging, dev, etc.).
  * Severities (multi-select: error, warning, info).
  * Event types to report (checkboxes: new error, spike, frequent, reopened, etc.).

Example structure:

```go
type ProjectChannelMapping struct {
    ProjectID    string
    ChannelID    string
    Environments []string
    Severities   []string
    Events       []string // "new_error", "spike", "reopened" ...
}
```

**Sync projects from Bugsnag** button:

* refetch projects via API;
* refresh the local list without overwriting existing channel assignments (by ProjectID).

### 5.3. **User Mapping** tab

Purpose: connect Bugsnag users to Mattermost users.

UI:

* **Load Bugsnag users** button:
  * fetch list of Bugsnag users/collaborators (id, name, email).
* Table:
  * Bugsnag Name
  * Bugsnag Email
  * Bugsnag User ID (if present)
  * Mattermost User (dropdown: Mattermost users)
  * Auto-select the Mattermost user by email (pre-select, editable).

Structure:

```go
type UserMapping struct {
    BugsnagUserID string // or empty if mapping is email-only
    BugsnagEmail  string
    MMUserID      string
}
```

Usage:

* When rendering a card:
  * if the error is assigned to a Bugsnag user with email X → find mapping → display `Assigned to @mm-user`.
* When clicking “Assign to me”:
  * Mattermost user → their email → find Bugsnag user → call Bugsnag API for reassignment.

### 5.4. **Notification Rules & Advanced** tab

Parameters:

* Time windows for statistics:
  * Events window (e.g., 1h / 24h).
* Spike parameters (optional):
  * e.g., “treat as spike if events in 10 minutes > X”.
* Periodic sync interval:
  * interval in minutes (e.g., 5/10/15).
* Security:
  * Secret token for webhook (in query or header).
* Logging:
  * enable/disable debug logs.

---

## 6. Development steps (checklist)

1. **Basic plugin scaffold**
   * Clone `mattermost-plugin-starter-template`.
   * Rename to `mattermost-plugin-bugsnag`.
   * Update manifest (Plugin ID, name, description, routes).

2. **Server plugin (Go)**
   * Define `Configuration` structure.
   * Implement:
     * `OnConfigurationChange` — read/validate configuration;
     * routes `/webhook` and `/actions`.
   * Initialize KV storage.

3. **Bugsnag client**
   * Module with functions:
     * `GetOrganizations()`
     * `GetProjects(orgID)`
     * `GetErrors(projectID, filters)`
     * `GetError(errorID)`
     * `GetUsers(orgID)` (if available)
     * `UpdateErrorStatus(errorID, status)`
     * `AssignError(errorID, assignee)`
   * Data types: Error, Project, User, Status, etc.

4. **Webhook handler (`/webhook`)**
   * Accept and validate request (token).
   * Parse Bugsnag payload.
   * Determine project and channels by mapping.
   * Create/update error card.
   * Update KV: `errorID+projectID → postID`.

5. **Actions handler (`/actions`)**
   * Parse interactive actions:
     * `context.action`, `context.error_id`, `context.project_id`.
   * Get Mattermost user info (`user_id`).
   * Map Bugsnag ⇄ MM user.
   * Call Bugsnag API (status, assignee).
   * Update card and add thread entry.

6. **Scheduler (periodic sync)**
   * Start a goroutine with ticker at plugin start.
   * Every N minutes:
     * read list of active errors;
     * call Bugsnag API:
       * status;
       * event counts over the last X minutes/hours;
       * `last_seen`.
   * After update:
     * modify card/attachment text;
     * add thread entries for significant changes.

7. **Webapp plugin (React) — settings UI**
   * Implement tabs:
     * Connection;
     * Projects & Channels;
     * User Mapping;
     * Notification Rules & Advanced.
   * REST endpoints on the server to fetch/save configuration and mappings.

8. **Logging and error handling**
   * Log Bugsnag API errors, invalid payloads, and Mattermost API failures.
   * Optional fallback messages in threads for failed operations.

9. **Documentation / README**
   * Describe the plugin and capabilities.
   * Build and install instructions.
   * Configuration guidance for Bugsnag connectivity and webhooks.
   * Example usage scenarios.

---

## 7. Quick setup guide (for admins)

1. Install and enable the plugin in Mattermost.
2. In System Console → Plugins → Bugsnag Integration:
   * enter the Bugsnag API Token;
   * specify Organization ID (if needed);
   * click **Test connection**.
3. On the **Projects & Channels** tab:
   * click **Sync projects from Bugsnag**;
   * choose Team + Channel for desired projects;
   * set filters for environment / severity / events;
   * save.
4. On the **User Mapping** tab:
   * click **Load Bugsnag users**;
   * map Bugsnag users to Mattermost users;
   * save.
5. In Bugsnag (UI) for each project:
   * Project Settings → Integrations → Webhook/Data forwarding;
   * add a webhook with the URL from the plugin settings;
   * choose events to send.
6. Verification:
   * generate a test error in Bugsnag;
   * ensure the corresponding card appears in the configured Mattermost channel;
   * test buttons (assign, resolve, ignore).
