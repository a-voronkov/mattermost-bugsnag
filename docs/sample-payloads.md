# Sample payloads and message drafts

These samples capture the minimum data we expect to use in the first webhook and
interactive action flows. They map 1:1 to the placeholder structs in `server/`.

## Webhook payload (Bugsnag → plugin)

```json
{
  "event": "error",
  "error_id": "abcd1234efgh5678",
  "project_id": "my-project",
  "summary": "NullReferenceException in CheckoutController",
  "environment": "production",
  "severity": "error",
  "last_seen": "2025-11-28T10:23:00Z",
  "counts": {
    "users": 123,
    "events_1h": 5,
    "events_24h": 42
  }
}
```

## Interactive action payload (Mattermost → plugin)

```json
{
  "user_id": "mm-user-id",
  "context": {
    "action": "assign_me",
    "error_id": "abcd1234efgh5678",
    "project_id": "my-project",
    "error_url": "https://app.bugsnag.com/org/project/errors/abcd1234efgh5678"
  }
}
```

## Draft Mattermost card

```json
{
  "channel_id": "<channel-id>",
  "message": ":rotating_light: **[BUG]** NullReferenceException in CheckoutController",
  "props": {
    "attachments": [
      {
        "fallback": "Bugsnag error",
        "title": "NullReferenceException: Object reference not set...",
        "title_link": "https://app.bugsnag.com/org/project/errors/abcd1234efgh5678",
        "text": "Env: production | Severity: error | Users: 123 | Events (1h/24h): 5 / 42\nLast seen: 2025-11-28 10:23 UTC",
        "footer": "Bugsnag • my-backend-api",
        "actions": [
          {"id": "assign_me", "name": "🙋 Assign to me", "type": "button", "style": "primary"},
          {"id": "resolve", "name": "✅ Resolve", "type": "button", "style": "primary"},
          {"id": "ignore", "name": "🙈 Ignore", "type": "button", "style": "default"},
          {"id": "open", "name": "🔗 Open in Bugsnag", "type": "button", "style": "link"}
        ]
      }
    ]
  }
}
```
