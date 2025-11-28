package main

import "github.com/a-voronkov/mattermost-bugsnag/server/kvkeys"

// PluginID is the unique identifier for this plugin.
const PluginID = kvkeys.PluginID

// For backwards compatibility with internal references.
const pluginID = PluginID

// Re-export KV key constants for use within the main package.
const (
	KVKeyProjectChannelMappings = kvkeys.ProjectChannelMappings
	KVKeyUserMappings           = kvkeys.UserMappings
	KVKeyActiveErrors           = kvkeys.ActiveErrors
	KVKeyErrorPostPrefix        = kvkeys.ErrorPostPrefix
)
