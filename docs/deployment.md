# Deployment Guide

This guide covers deploying the Mattermost Bugsnag plugin to a production Mattermost server.

## Requirements

### Mattermost Server

- **Version**: 9.6.0 or higher
- **Plugin uploads enabled**: `PluginSettings.EnableUploads = true`
- **Network access**: Server must be reachable from Bugsnag for webhooks

### Bugsnag

- **API Token**: Personal or Organization API token with read/write access
- **Webhook configuration**: Ability to add webhook URLs to projects

## Installation

### Option 1: System Console UI

1. Download the plugin bundle (`mattermost-bugsnag.tar.gz`)
2. Navigate to **System Console → Plugin Management → Management**
3. Click **Upload Plugin** and select the bundle
4. Click **Enable** on the Bugsnag Integration plugin

### Option 2: mmctl CLI

```bash
# Upload the plugin
mmctl plugin add mattermost-bugsnag.tar.gz

# Enable the plugin
mmctl plugin enable com.mattermost.bugsnag
```

### Option 3: Direct File Deployment

Copy the extracted plugin to the Mattermost plugins directory:

```bash
# Extract plugin
tar -xzf mattermost-bugsnag.tar.gz

# Copy to plugins directory
cp -r mattermost-bugsnag /opt/mattermost/plugins/

# Restart Mattermost or enable via API
```

## Configuration

### Plugin Settings

Configure via **System Console → Plugins → Bugsnag Integration**:

| Setting | Description | Required |
|---------|-------------|----------|
| **Bugsnag API Token** | Personal API token for Bugsnag API access | Yes |
| **Organization ID** | Limit to specific Bugsnag organization | No |
| **Webhook Secret** | Shared secret for webhook validation | Recommended |
| **Webhook Token** | Query parameter token for webhook URL | Optional |
| **Enable Debug Log** | Verbose logging for troubleshooting | No |
| **Sync Interval** | Polling interval for error updates (seconds) | No (default: 300) |

### Getting a Bugsnag API Token

1. Log in to [Bugsnag](https://app.bugsnag.com)
2. Navigate to **Settings → My Account → Personal Auth Tokens**
3. Create a new token with appropriate scopes
4. Copy and paste into plugin settings

## Webhook Setup

### Webhook URL Format

```
https://<your-mattermost-host>/plugins/bugsnag/webhook?token=<webhook-token>
```

**Example**:
```
https://mattermost.company.com/plugins/bugsnag/webhook?token=abc123secret
```

### Configure in Bugsnag

1. Open your Bugsnag project
2. Navigate to **Settings → Integrations → Webhooks**
3. Add a new webhook with the URL from above
4. Select events to trigger (new errors, error spikes, etc.)
5. Save and test the webhook

### Webhook Events

The plugin handles these Bugsnag webhook events:

- `error` — New error occurred
- `errorStateChanged` — Error status changed (resolved, reopened)
- `errorAssignmentChanged` — Error assigned/unassigned
- `spike` — Error spike detected

## Channel Mapping

Errors are routed to channels based on project configuration stored in KV:

```json
{
  "project_id": "bugsnag-project-id",
  "channel_id": "mattermost-channel-id",
  "environments": ["production"],
  "severities": ["error", "warning"],
  "events": ["error", "spike"]
}
```

Configure via the admin API or upcoming System Console UI.

## User Mapping

Map Bugsnag users to Mattermost users for mentions and assignments:

```json
{
  "bugsnag_user_id": "mattermost_user_id"
}
```

Users can also be matched by email address automatically.

## Security Considerations

### Webhook Token

Always use a webhook token in production:
- Generate a strong random token
- Configure in both plugin settings and Bugsnag webhook URL
- Rotate periodically

### API Token Scope

Use minimal required scopes for the Bugsnag API token:
- Read access to projects and errors
- Write access for status/assignment updates

### Network Security

- Use HTTPS for all webhook traffic
- Consider IP allowlisting for Bugsnag IPs if your firewall supports it
- The plugin validates webhook tokens before processing

## Troubleshooting

### Plugin Not Loading

1. Check Mattermost logs: `tail -f /opt/mattermost/logs/mattermost.log`
2. Verify plugin is enabled in System Console
3. Ensure `EnableUploads` is true in config

### Webhooks Not Received

1. Enable debug logging in plugin settings
2. Verify webhook URL is correct
3. Check network connectivity from Bugsnag to Mattermost
4. Verify webhook token matches

### Actions Not Working

1. Verify Bugsnag API token is valid
2. Check user mapping configuration
3. Review plugin logs for API errors

## Monitoring

### Logs

Plugin logs are written to Mattermost server logs at these levels:

- **INFO**: Configuration changes, sync cycles
- **WARN**: Invalid requests, missing mappings
- **ERROR**: API failures, webhook errors
- **DEBUG**: Request details (when enabled)

### Health Check

Verify plugin is responding:

```bash
curl -i https://your-mattermost/plugins/bugsnag/api/health
```

## Upgrading

1. Download new plugin version
2. Upload via System Console or mmctl
3. Plugin will be updated automatically
4. Previous configuration is preserved

