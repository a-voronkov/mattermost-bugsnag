package main

import (
	"net/http"
	"sync"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mattermost-bugsnag/plugin/server/api"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost-bugsnag/plugin/server/scheduler"
)

// Plugin implements the Mattermost plugin interface and wires HTTP endpoints for the
// Bugsnag integration. Most logic is stubbed and described in docs/TODO until the
// full implementation is added.
type Plugin struct {
	plugin.MattermostPlugin

	configuration atomic.Pointer[Configuration]
	syncMu        sync.Mutex
	syncRunner    *scheduler.Runner
	apiHandler    http.Handler
}

func main() {
	plugin.ClientMain(&Plugin{})
}

// OnActivate initializes the plugin and starts background routines.
func (p *Plugin) OnActivate() error {
	return p.OnConfigurationChange()
}

// OnConfigurationChange is called when configuration changes are made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration Configuration
	if err := p.API.LoadPluginConfiguration(&configuration); err != nil {
		return err
	}

	if err := configuration.Validate(); err != nil {
		return err
	}

	p.configuration.Store(&configuration)
	p.restartSyncRoutine(configuration)
	return nil
}

// OnDeactivate stops background work when the plugin is disabled.
func (p *Plugin) OnDeactivate() error {
	p.stopSyncRoutine()
	return nil
}

// Close stops background work when the server is shutting down.
func (p *Plugin) Close() {
	p.stopSyncRoutine()
}

func (p *Plugin) restartSyncRoutine(cfg Configuration) {
	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	p.stopSyncRoutineLocked()

	interval := time.Duration(cfg.SyncIntervalSec) * time.Second
	if interval <= 0 {
		return
	}

	p.syncRunner = scheduler.NewRunner(p.API, cfg.EnableDebugLog)
	p.syncRunner.Start(interval)
}

func (p *Plugin) stopSyncRoutine() {
	p.syncMu.Lock()
	defer p.syncMu.Unlock()
	p.stopSyncRoutineLocked()
}

func (p *Plugin) stopSyncRoutineLocked() {
	if p.syncRunner != nil {
		p.syncRunner.Stop()
		p.syncRunner = nil
	}
}

// ServeHTTP routes external HTTP requests to the appropriate handler.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/webhook":
		p.handleWebhook(w, r)
		return
	case "/actions":
		p.handleActions(w, r)
		return
	default:
		if strings.HasPrefix(r.URL.Path, "/api/") {
			p.getAPIHandler().ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}
}

// getConfiguration returns the active configuration or a zero-value configuration
// when nothing has been loaded yet.
func (p *Plugin) getConfiguration() Configuration {
	if cfg := p.configuration.Load(); cfg != nil {
		return *cfg
	}
	return Configuration{}
}

func (p *Plugin) getAPIHandler() http.Handler {
	if p.apiHandler == nil {
		p.apiHandler = api.NewHandler(func() string {
			cfg := p.getConfiguration()
			return cfg.BugsnagAPIToken
		})
	}

	return p.apiHandler
}
