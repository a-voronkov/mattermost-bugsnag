package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/a-voronkov/mattermost-bugsnag/server/api"
	"github.com/a-voronkov/mattermost-bugsnag/server/scheduler"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

//go:embed assets/bugsnag-icon.png
var bugsnagIconPNG []byte

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
	botUserID     string
}

func main() {
	plugin.ClientMain(&Plugin{})
}

// OnActivate initializes the plugin and starts background routines.
func (p *Plugin) OnActivate() error {
	botUserID, err := p.ensureBot()
	if err != nil {
		p.API.LogError("failed to ensure bot", "err", err.Error())
		return err
	}
	p.botUserID = botUserID
	p.API.LogInfo("Bugsnag bot ready", "bot_user_id", botUserID)

	return p.OnConfigurationChange()
}

// ensureBot creates the bot if it doesn't exist, or returns the existing bot's user ID.
func (p *Plugin) ensureBot() (string, error) {
	botUsername := "bugsnag"

	// Try to get existing bot by username
	existingBot, appErr := p.API.GetBot(botUsername, false)
	if appErr == nil && existingBot != nil {
		p.API.LogDebug("Using existing Bugsnag bot", "bot_user_id", existingBot.UserId)
		// Update profile image for existing bot if needed
		_ = p.setBotProfileImage(existingBot.UserId)
		return existingBot.UserId, nil
	}

	// Check if a user with this username exists (not a bot)
	existingUser, appErr := p.API.GetUserByUsername(botUsername)
	if appErr == nil && existingUser != nil {
		p.API.LogWarn("User with username 'bugsnag' exists but is not a bot, using it anyway", "user_id", existingUser.Id)
		// Still try to set profile image
		_ = p.setBotProfileImage(existingUser.Id)
		return existingUser.Id, nil
	}

	// Bot doesn't exist, create new one
	p.API.LogInfo("Creating new Bugsnag bot")
	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: "Bugsnag",
		Description: "Bot for Bugsnag integration - posts error notifications to channels.",
	}

	createdBot, appErr := p.API.CreateBot(bot)
	if appErr != nil {
		return "", appErr
	}

	// Set bot profile image
	if err := p.setBotProfileImage(createdBot.UserId); err != nil {
		p.API.LogWarn("failed to set bot profile image", "err", err.Error())
		// Don't fail activation if avatar fails
	}

	return createdBot.UserId, nil
}

// setBotProfileImage sets the Bugsnag logo as the bot's avatar.
func (p *Plugin) setBotProfileImage(botUserID string) error {
	// Bugsnag logo as base64-encoded PNG (small 128x128 icon)
	// This is a simplified Bugsnag-style bug icon
	iconData, err := p.getBugsnagIcon()
	if err != nil {
		return err
	}

	appErr := p.API.SetProfileImage(botUserID, iconData)
	if appErr != nil {
		return appErr
	}

	p.API.LogInfo("Set Bugsnag bot profile image")
	return nil
}

// getBugsnagIcon returns the Bugsnag logo PNG bytes.
func (p *Plugin) getBugsnagIcon() ([]byte, error) {
	// Use embedded icon if available
	if len(bugsnagIconPNG) > 0 {
		return bugsnagIconPNG, nil
	}

	// Fallback: generate a simple Bugsnag-colored icon
	p.API.LogDebug("Using fallback generated icon")
	return p.generateFallbackIcon(), nil
}

// generateFallbackIcon creates a simple Bugsnag-colored PNG icon.
func (p *Plugin) generateFallbackIcon() []byte {
	// Create a 128x128 image with Bugsnag brand color (#4949E4 - purple/blue)
	size := 128
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Bugsnag purple color
	bugsnagColor := color.RGBA{R: 73, G: 73, B: 228, A: 255}

	// Fill with Bugsnag color
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bugsnagColor)
		}
	}

	// Add a simple "bug" shape in white (antennae and body outline)
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	// Draw a simplified bug body (oval in center)
	centerX, centerY := size/2, size/2+10
	for y := centerY - 30; y <= centerY+30; y++ {
		for x := centerX - 20; x <= centerX+20; x++ {
			// Ellipse equation
			dx := float64(x-centerX) / 20.0
			dy := float64(y-centerY) / 30.0
			if dx*dx+dy*dy <= 1.0 {
				img.Set(x, y, white)
			}
		}
	}

	// Draw antennae
	for i := 0; i < 15; i++ {
		img.Set(centerX-10+i/2, centerY-30-i, white)
		img.Set(centerX+10-i/2, centerY-30-i, white)
	}

	// Draw legs (3 on each side)
	for leg := 0; leg < 3; leg++ {
		yPos := centerY - 15 + leg*15
		for i := 0; i < 15; i++ {
			img.Set(centerX-20-i, yPos+i/3, white)
			img.Set(centerX+20+i, yPos+i/3, white)
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}

	return buf.Bytes()
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
