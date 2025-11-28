// Package kvkeys provides centralized KV store key constants for the plugin.
// This package has no dependencies to avoid circular imports.
package kvkeys

// PluginID is the unique identifier for this plugin.
const PluginID = "com.mattermost.bugsnag"

// KV store key constants.
const (
	// ProjectChannelMappings stores the project-to-channel routing rules.
	ProjectChannelMappings = "bugsnag:project-channel-mappings"

	// UserMappings stores the Bugsnag-to-Mattermost user mappings.
	UserMappings = "bugsnag:user-mappings"

	// ActiveErrors stores the list of active errors being tracked.
	ActiveErrors = "bugsnag:active-errors"

	// ErrorPostPrefix is the prefix for error-to-post mapping keys.
	ErrorPostPrefix = "bugsnag:error-post:"
)
