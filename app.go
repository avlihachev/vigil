package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"claude-sessions-monitor/monitor"
	"claude-sessions-monitor/switcher"
	"claude-sessions-monitor/tray"

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
	tray.Init("◉", "Claude Sessions Monitor", a.ToggleWindow)

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

func (a *App) OpenSession(source string, cwd string) {
	switcher.ActivateSession(source, cwd)
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
	runtime.WindowShow(a.ctx)
	a.visible = true
}

func (a *App) HideWindow() {
	runtime.WindowHide(a.ctx)
	a.visible = false
}
