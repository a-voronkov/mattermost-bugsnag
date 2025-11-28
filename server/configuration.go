package main

import (
	"fmt"
	"strings"
	"time"
)

// Configuration collects the server-side settings supplied via System Console.
// These fields intentionally mirror the "settings_schema" keys in plugin.json.
type Configuration struct {
	BugsnagAPIToken string
	OrganizationID  string
	WebhookSecret   string
	WebhookToken    string
	EnableDebugLog  bool
	SyncInterval    time.Duration
	SyncIntervalSec int
}

// Clone returns a shallow copy. Useful when we start adding mutable slices/maps.
func (c *Configuration) Clone() Configuration {
	return *c
}

// Validate ensures required fields are provided and defaults are set.
// Webhook token validation deliberately accepts either WebhookToken or
// WebhookSecret to avoid breaking existing deployments when the name changes.
func (c *Configuration) Validate() error {
	missing := []string{}

	if c.SyncInterval <= 0 {
		c.SyncInterval = 5 * time.Minute
	}

	if strings.TrimSpace(c.BugsnagAPIToken) == "" {
		missing = append(missing, "Bugsnag API Token")
	}

	if strings.TrimSpace(c.WebhookToken) == "" && strings.TrimSpace(c.WebhookSecret) == "" {
		missing = append(missing, "Webhook Token/Secret")
	}

	if c.SyncIntervalSec <= 0 {
		c.SyncIntervalSec = 300
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}
