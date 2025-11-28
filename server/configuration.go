package main

// Configuration collects the server-side settings supplied via System Console.
// These fields intentionally mirror the "settings_schema" keys in plugin.json.
type Configuration struct {
	BugsnagAPIToken string
	OrganizationID  string
	WebhookSecret   string
	WebhookToken    string
	EnableDebugLog  bool
}

// Clone returns a shallow copy. Useful when we start adding mutable slices/maps.
func (c *Configuration) Clone() Configuration {
	return *c
}
