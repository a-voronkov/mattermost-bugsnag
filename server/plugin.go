package main

import (
	"net/http"
	"sync/atomic"

	"github.com/mattermost/mattermost-server/v6/plugin"
)

// Plugin implements the Mattermost plugin interface and wires HTTP endpoints for the
// Bugsnag integration. Most logic is stubbed and described in docs/TODO until the
// full implementation is added.
type Plugin struct {
	plugin.MattermostPlugin

	configuration atomic.Pointer[Configuration]
	kvNamespace   string
}

func main() {
	plugin.ClientMain(&Plugin{})
}

// OnConfigurationChange is called when configuration changes are made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration Configuration
	if err := p.API.LoadPluginConfiguration(&configuration); err != nil {
		p.API.LogError("failed to load configuration", "err", err.Error())
		return err
	}

	if err := configuration.Validate(); err != nil {
		p.API.LogWarn("invalid configuration", "err", err.Error())
		return err
	}

	if p.kvNamespace == "" {
		p.kvNamespace = pluginID
		p.API.LogDebug("kv namespace initialized", "namespace", p.kvNamespace)
	}

	cfg := configuration.Clone()
	p.configuration.Store(&cfg)
	p.API.LogInfo("configuration loaded", "org_id", cfg.OrganizationID, "sync_interval", cfg.SyncInterval.String())
	return nil
}

// ServeHTTP routes external HTTP requests to the appropriate handler.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", p.handleWebhook)
	mux.HandleFunc("/actions", p.handleActions)
	mux.ServeHTTP(w, r)
}

// getConfiguration returns the active configuration or a zero-value configuration
// when nothing has been loaded yet.
func (p *Plugin) getConfiguration() Configuration {
	if cfg := p.configuration.Load(); cfg != nil {
		return *cfg
	}
	return Configuration{}
}

func (p *Plugin) kvNS() string {
	if p.kvNamespace == "" {
		return pluginID
	}
	return p.kvNamespace
}

func (p *Plugin) logDebug(msg string, keyValuePairs ...any) {
	p.API.LogDebug(msg, keyValuePairs...)
}

func (p *Plugin) logWarn(msg string, keyValuePairs ...any) {
	p.API.LogWarn(msg, keyValuePairs...)
}

func (p *Plugin) logError(msg string, keyValuePairs ...any) {
	p.API.LogError(msg, keyValuePairs...)
}
