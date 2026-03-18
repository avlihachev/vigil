# Vigil: Badge, Notifications, History — Design Spec

**Date:** 2026-03-18
**Status:** Approved

## Overview

Three new features added to Vigil, the macOS menubar monitor for Claude Code sessions:

1. **Tray badge** — count of sessions needing attention (`confirm` + `waiting`)
2. **Notifications** — macOS notification when a session enters `confirm` status; can be disabled
3. **Session history** — browse and resume past Claude Code sessions, grouped by project

## Architecture

All three features extend the existing `monitor.Manager` + `App` + tray pattern without new abstractions. Settings persisted in `~/.vigil/settings.json`.

---

## Feature 1: Tray Badge

### Goal
Show how many sessions need attention directly on the tray icon so the user doesn't need to open the popup.

### Logic
- Badge count = number of sessions with status `confirm` OR `waiting`
- Count 0 → clean icon (no badge)
- Count > 0 → number drawn over icon

### Implementation
- `tray/tray_darwin.m`: new `traySetBadge(int count)` function
  - Renders the base icon + number string via `NSImage` + `NSString drawAtPoint:`
  - Updates `NSStatusBarButton.image`
- `tray/tray_darwin.go`: exports `SetBadge(count int)` Go wrapper
- `app.go`: after each `Collect()`, compute count and call `tray.SetBadge(count)`

---

## Feature 2: Notifications

### Goal
Alert the user when Claude is waiting for tool confirmation so they can switch to the session.

### Trigger
- Session transitions into `confirm` status (was not `confirm` in previous poll)
- Same session is NOT re-notified until it leaves and re-enters `confirm`

### Content
```
Title:  Vigil
Body:   «<ProjectName>» needs confirmation
```

### Implementation
- `monitor/notifier.go`: `Notifier` struct
  - Holds `notified map[string]bool` (keyed by sessionID)
  - `Check(prev, curr []Session)` — detects new `confirm` transitions, fires notification, tracks state
  - Fires via `osascript -e 'display notification "..." with title "Vigil"'`
- `app.go`: `Notifier` initialized at startup; called in `emitSessions()` with previous/current session snapshots

### Settings
- `~/.vigil/settings.json`: `{ "notificationsEnabled": true }`
- `App.SetNotificationsEnabled(enabled bool)` — writes file, updates in-memory flag
- `Notifier.Check()` skips if disabled
- Loaded at startup; defaults to `true` if file absent

---

## Feature 3: Session History

### Goal
Browse all past Claude Code sessions (no active PID) grouped by project, and resume them in the right IDE/terminal.

### Data Source
`~/.claude/projects/**/*.jsonl` — all JSONL files whose `cwd` does not belong to any currently active session.

### Data Extracted Per Session
| Field | Source |
|-------|--------|
| `sessionId` | filename (without `.jsonl`) |
| `name` | `customTitle` or `slug` from JSONL (reuse `ParseName`) |
| `lastActiveAt` | file modification time |
| `tokensIn/Out` | reuse `ParseTokens` |

### Grouping
- Grouped by decoded `cwd` path (project folder)
- Sorted by most recent `lastActiveAt` descending
- Max 50 projects shown; max 5 sessions per project

### Resume Action
`App.ResumeSession(cwd, sessionID string)`:
1. Checks if VSCode/Cursor is open with that `cwd` (reuse `IDEDetector`)
2. If yes: opens a terminal in that IDE window, runs `claude --resume <sessionID>`
3. If no: opens the system default terminal at `cwd`, runs `claude --resume <sessionID>`
4. Uses existing `switcher` package for IDE activation

### New Go Types
```go
// monitor/history_scanner.go
type HistoricalSession struct {
    SessionID    string `json:"sessionId"`
    Name         string `json:"name"`
    LastActiveAt int64  `json:"lastActiveAt"`
    TokensIn     string `json:"tokensIn"`
    TokensOut    string `json:"tokensOut"`
}

type ProjectHistory struct {
    ProjectName string             `json:"projectName"`
    CWD         string             `json:"cwd"`
    Sessions    []HistoricalSession `json:"sessions"`
}
```

### New App Methods
- `GetHistory() []ProjectHistory`
- `ResumeSession(cwd, sessionID string)`
- `SetNotificationsEnabled(enabled bool)`

---

## Frontend

### Tabs
`session-list.ts` gains a tab bar at the top:
```
[ Active (2) ]  [ History ]
```
Active tab shows existing session list. History tab shows `<history-list>` component.

### `history-list.ts` (new component)
Collapsible project groups:
```
▼ my-project                        ~/Projects/my-project
   ├ fix auth bug          2d ago    ↑120k ↓45k
   └ add tests             5d ago    ↑80k ↓30k

▶ other-project            ...
```
- Groups collapsed by default if > 1 session; expanded if 1 session
- Click on a session row → `ResumeSession(cwd, sessionId)`

### Settings toggle
`status-bar.ts` gains a small gear icon (⚙) on the right. Click shows an inline overlay with a single toggle: `Notifications for confirmations`. State read/written via `SetNotificationsEnabled()`.

### New Frontend Types
```ts
interface HistoricalSession {
  sessionId: string;
  name: string;
  lastActiveAt: number;
  tokensIn: string;
  tokensOut: string;
}
interface ProjectHistory {
  projectName: string;
  cwd: string;
  sessions: HistoricalSession[];
}
```

---

## Data Flow Summary

```
poll tick (3s)
  → monitor.Manager.Collect()         → []Session
  → Notifier.Check(prev, curr)        → osascript if new confirm
  → tray.SetBadge(confirmCount + waitingCount)
  → wails event "sessions:updated"    → frontend Active tab

user opens History tab
  → App.GetHistory()                  → scans ~/.claude/projects/**/*.jsonl
  → filters out active cwds
  → groups, sorts, limits
  → returns []ProjectHistory          → frontend history-list

user clicks history session
  → App.ResumeSession(cwd, sessionId)
  → IDEDetector checks open IDEs
  → switcher opens terminal + claude --resume <id>
```

---

## Out of Scope

- Cost / pricing calculations
- Per-session statistics panel
- Cross-machine sync
