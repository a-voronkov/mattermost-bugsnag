# Implementation TODOs for the first workable build

The repository now includes a thin server skeleton and manifest. These steps
should lead to a buildable plugin that can be uploaded to Mattermost.

1. **Wire Mattermost client helpers**
   - Use `pluginapi.NewClient` to wrap `p.API` and provide typed helpers for KV,
     posts, and users.
   - Add convenience methods for channel lookup and post updates.

2. **Configuration and validation**
   - Validate that `BugsnagAPIToken` and `WebhookSecret`/`WebhookToken` are
     present; expose inline admin settings errors when missing.
   - Persist project/channel mappings and user mappings in KV.

3. **Webhook handler**
   - Validate the token/secret in query/header.
   - Normalize Bugsnag payload into internal structs; drop events that do not
     match configured environment/severity/event-type filters.
   - Create or update a channel post with the `cardAttachment` template; store
     `errorID+projectID → postID` in KV and append thread updates on changes.

4. **Interactive actions**
   - Map `user_id` to a Bugsnag user via stored mappings; fall back to email
     matching.
   - Call Bugsnag API to update status/assignee; reflect the result in the post
     and thread.

5. **Scheduler**
   - Track active errors in KV and poll Bugsnag on an interval for spikes, last
     seen, and counts over 1h/24h; update posts accordingly.

6. **Webapp admin UI**
   - Repurpose the starter template tabs for Connection, Projects & Channels,
     User Mapping, and Notification Rules.
   - Implement REST endpoints in the server to serve and persist these configs.

7. **Packaging**
   - Mirror the starter template `Makefile` to build `server/dist` binaries and
     `webapp/dist/main.js` bundle.
   - Ensure `plugin.json` paths align with build outputs for `make plugin`.
