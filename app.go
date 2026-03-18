package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"claude-sessions-monitor/monitor"

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
}

func (a *App) shutdown(ctx context.Context) {
	close(a.stop)
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

func (a *App) ToggleWindow() {
	if a.visible {
		runtime.WindowHide(a.ctx)
		a.visible = false
	} else {
		runtime.WindowShow(a.ctx)
		a.visible = true
	}
}
