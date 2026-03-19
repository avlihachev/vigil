package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"vigil/monitor"
	"vigil/switcher"
	"vigil/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	manager *monitor.Manager
	stop    chan struct{}
	visible bool
}

func NewApp() *App {
	homeDir, _ := os.UserHomeDir()
	claudeDir := filepath.Join(homeDir, ".claude")
	return &App{
		manager: monitor.NewManager(claudeDir),
		stop:    make(chan struct{}),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.pollLoop()
	switcher.PromptAccessibility()
	tray.Init("◉", "Vigil", a.ToggleWindow, func() {
		runtime.Quit(a.ctx)
	})

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
		case <-a.stop:
			return
		}
	}
}

func (a *App) emitSessions() {
	sessions := a.manager.Collect()
	runtime.EventsEmit(a.ctx, "sessions:updated", sessions)
}

func (a *App) GetSessions() []monitor.Session {
	return a.manager.Collect()
}

func (a *App) OpenSession(source string, cwd string, pid int) {
	a.HideWindow() // hide popup before switching so it doesn't block focus
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
	// position near top-right of screen (menubar area)
	screens, _ := runtime.ScreenGetAll(a.ctx)
	if len(screens) > 0 {
		primary := screens[0]
		x := primary.Size.Width - 340
		y := 30
		runtime.WindowSetPosition(a.ctx, x, y)
	}
	// orderFront: shows window without activating our app,
	// so the user's previous app (e.g. Ghostty) keeps focus
	tray.ShowPopup()
	a.visible = true
}

func (a *App) HideWindow() {
	// orderOut: hides window without triggering app-deactivation,
	// so macOS doesn't switch focus back to our app's "previous" app
	tray.HidePopup()
	a.visible = false
}
