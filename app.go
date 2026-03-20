package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vigil/monitor"
	"vigil/switcher"
	"vigil/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Settings struct {
	NotifyConfirm   bool   `json:"notifyConfirm"`
	NotifyWaiting   bool   `json:"notifyWaiting"`
	BadgeConfirm    bool   `json:"badgeConfirm"`
	BadgeWaiting    bool   `json:"badgeWaiting"`
	BadgeActive     bool   `json:"badgeActive"`
	ShowRateLimits  bool   `json:"showRateLimits"`
	LastUpdateCheck string `json:"lastUpdateCheck,omitempty"`
}

type App struct {
	ctx          context.Context
	manager      *monitor.Manager
	history      *monitor.HistoryScanner
	notifier     *monitor.Notifier
	updater      *monitor.Updater
	updateInfo   *monitor.UpdateInfo
	rateLimits   *monitor.RateLimitReader
	stop         chan struct{}
	visible      bool
	prevSessions []monitor.Session
	settingsMu   sync.Mutex
	settings     Settings
	settingsPath string
	claudeDir    string
	vigilDir     string
}

func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	claudeDir := filepath.Join(homeDir, ".claude")
	vigilDir := filepath.Join(homeDir, ".vigil")
	os.MkdirAll(vigilDir, 0755)

	app := &App{
		manager:      monitor.NewManager(claudeDir),
		history:      monitor.NewHistoryScanner(claudeDir),
		notifier:     monitor.NewNotifier(),
		rateLimits:   monitor.NewRateLimitReader(claudeDir, vigilDir),
		stop:         make(chan struct{}),
		settingsPath: filepath.Join(vigilDir, "settings.json"),
		claudeDir:    claudeDir,
		vigilDir:     vigilDir,
	}
	app.updater = monitor.NewUpdater(Version, "https://api.github.com/repos/avlihachev/vigil/releases/latest")
	app.loadSettings()
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.pollLoop()
	switcher.PromptAccessibility()
	tray.Init("◉", "Vigil", a.ToggleWindow, func() {
		runtime.Quit(a.ctx)
	})
	if a.needsUpdateCheck() {
		go a.checkForUpdate()
	}

	// hide window when it loses focus
	runtime.EventsOn(a.ctx, "window:blur", func(optionalData ...interface{}) {
		a.HideWindow()
	})
}

func (a *App) shutdown(ctx context.Context) {
	close(a.stop)
	tray.Remove()
}

func (a *App) pollLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	a.emitSessions()
	for {
		select {
		case <-ticker.C:
			a.emitSessions()
			if a.updateInfo == nil && a.needsUpdateCheck() {
				go a.checkForUpdate()
			}
		case <-a.stop:
			return
		}
	}
}

func (a *App) emitSessions() {
	sessions := a.manager.Collect()
	a.notifier.Check(a.prevSessions, sessions)

	a.settingsMu.Lock()
	badge := monitor.CountBadge(sessions, a.settings.BadgeConfirm, a.settings.BadgeWaiting, a.settings.BadgeActive)
	a.settingsMu.Unlock()
	tray.SetBadge(badge)

	a.prevSessions = sessions
	runtime.EventsEmit(a.ctx, "sessions:updated", sessions)

	a.settingsMu.Lock()
	showRL := a.settings.ShowRateLimits
	a.settingsMu.Unlock()
	if showRL {
		rl := a.rateLimits.Read()
		runtime.EventsEmit(a.ctx, "ratelimits:updated", rl)
	} else {
		runtime.EventsEmit(a.ctx, "ratelimits:updated", nil)
	}
}

func (a *App) GetSessions() []monitor.Session {
	return a.manager.Collect()
}

func (a *App) OpenSession(source string, cwd string, pid int) {
	a.HideWindow()
	switcher.ActivateSession(source, cwd, pid)
}

func (a *App) ToggleWindow() {
	if a.visible {
		a.HideWindow()
	} else {
		a.ShowWindow()
	}
}

func (a *App) ShowWindow() {
	screens, _ := runtime.ScreenGetAll(a.ctx)
	if len(screens) > 0 {
		primary := screens[0]
		x := primary.Size.Width - 340
		y := 30
		runtime.WindowSetPosition(a.ctx, x, y)
	}
	tray.ShowPopup()
	a.visible = true
}

func (a *App) HideWindow() {
	tray.HidePopup()
	a.visible = false
}

func (a *App) GetHistory() []monitor.ProjectHistory {
	active := a.manager.Collect()
	cwds := make([]string, 0, len(active))
	for _, s := range active {
		cwds = append(cwds, s.CWD)
	}
	return a.history.ScanHistory(cwds)
}

func (a *App) ResumeSession(cwd string, sessionID string) {
	escapedCWD := strings.ReplaceAll(cwd, `"`, `\"`)
	escapedID := strings.ReplaceAll(sessionID, `"`, `\"`)
	script := fmt.Sprintf(
		`tell application "Terminal"
	do script "cd \"%s\" && claude --resume \"%s\""
	activate
end tell`, escapedCWD, escapedID)
	exec.Command("osascript", "-e", script).Run()

	source := a.manager.GetIDESource(cwd)
	if source != "" {
		switcher.ActivateSession(source, cwd, 0)
	}
}

func (a *App) checkForUpdate() {
	info, err := a.updater.Check()

	// save timestamp regardless of result to avoid re-checking every 3s
	a.settingsMu.Lock()
	a.settings.LastUpdateCheck = time.Now().Format(time.RFC3339)
	data, _ := json.MarshalIndent(a.settings, "", "  ")
	path := a.settingsPath
	a.settingsMu.Unlock()
	os.WriteFile(path, data, 0644)

	if err != nil || info == nil {
		return
	}
	a.updateInfo = info
	runtime.EventsEmit(a.ctx, "update:available", info)
}

func (a *App) needsUpdateCheck() bool {
	a.settingsMu.Lock()
	last := a.settings.LastUpdateCheck
	a.settingsMu.Unlock()

	if last == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return true
	}
	return time.Since(t) > 7*24*time.Hour
}

func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) GetRateLimits() *monitor.RateLimits {
	return a.rateLimits.Read()
}

func (a *App) EnableRateLimits() error {
	if err := monitor.InstallBridge(a.vigilDir); err != nil {
		return err
	}
	return monitor.EnableStatusLine(a.claudeDir, a.vigilDir)
}

func (a *App) DisableRateLimits() {
	monitor.DisableStatusLine(a.claudeDir)
}

func (a *App) IsBridgeInstalled() bool {
	return monitor.IsBridgeInstalled(a.claudeDir, a.vigilDir)
}

func (a *App) GetSettings() Settings {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.settings
}

func (a *App) UpdateSettings(s Settings) {
	a.settingsMu.Lock()
	a.settings = s
	data, _ := json.MarshalIndent(a.settings, "", "  ")
	path := a.settingsPath
	a.settingsMu.Unlock()

	os.WriteFile(path, data, 0644)
	a.applySettings()
}

func defaultSettings() Settings {
	return Settings{
		NotifyConfirm: true,
		NotifyWaiting: false,
		BadgeConfirm:  true,
		BadgeWaiting:  true,
		BadgeActive:   false,
	}
}

func (a *App) loadSettings() {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	a.settings = defaultSettings()
	data, err := os.ReadFile(a.settingsPath)
	if err == nil {
		json.Unmarshal(data, &a.settings)
	}
	a.applySettings()
}

func (a *App) applySettings() {
	a.notifier.SetNotifyConfirm(a.settings.NotifyConfirm)
	a.notifier.SetNotifyWaiting(a.settings.NotifyWaiting)
}
