package main

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/api"
	"github.com/a-voronkov/mattermost-bugsnag/server/scheduler"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the Mattermost plugin interface and wires HTTP endpoints for the
// Bugsnag integration. Most logic is stubbed and described in docs/TODO until the
// full implementation is added.
type Plugin struct {
	plugin.MattermostPlugin

	configuration atomic.Pointer[Configuration]
	kvNamespace   string
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

	p.configuration.Store(&configuration)
	p.API.LogInfo("configuration loaded", "org_id", configuration.OrganizationID, "sync_interval_sec", configuration.SyncIntervalSec)
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

	p.syncRunner = scheduler.NewRunner(p.API, cfg.EnableDebugLog, func() string {
		return p.getConfiguration().BugsnagAPIToken
	})
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

func (p *Plugin) getAPIHandler() http.Handler {
	if p.apiHandler == nil {
		p.apiHandler = api.NewRouter(api.Config{
			TokenProvider: func() string {
				cfg := p.getConfiguration()
				return cfg.BugsnagAPIToken
			},
			OrgIDProvider: func() string {
				cfg := p.getConfiguration()
				return cfg.OrganizationID
			},
			KVStore: &pluginKVAdapter{api: p.API, namespace: p.kvNS()},
		})
	}

	return p.apiHandler
}

// pluginKVAdapter adapts plugin.API to api.KVStore interface.
type pluginKVAdapter struct {
	api       plugin.API
	namespace string
}

func (a *pluginKVAdapter) Get(key string) ([]byte, error) {
	data, appErr := a.api.KVGet(a.namespace + ":" + key)
	if appErr != nil {
		return nil, appErr
	}
	return data, nil
}

func (a *pluginKVAdapter) Set(key string, value []byte) error {
	appErr := a.api.KVSet(a.namespace+":"+key, value)
	if appErr != nil {
		return appErr
	}
	return nil
}
