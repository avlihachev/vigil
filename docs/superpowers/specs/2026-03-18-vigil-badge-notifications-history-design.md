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

**`tray/tray_darwin.m`**: new `traySetBadge(int count)` function
- Renders the base icon + number string via `NSImage` + `NSString drawAtPoint:`
- Updates `NSStatusBarButton.image`

**`tray/tray_darwin.go`**: declare `traySetBadge` in the CGo preamble alongside other `void tray_*` declarations (NOT with `//export` — it is called from Go into ObjC, not the other way around):
```go
/*
...existing declarations...
void traySetBadge(int count);
*/
import "C"

func SetBadge(count int) {
    C.traySetBadge(C.int(count))
}
```

**`app.go`**: after each `Collect()`, compute count and call `tray.SetBadge(count)`.

---

## Feature 2: Notifications

### Goal
Alert the user when Claude is waiting for tool confirmation so they can switch to the session.

### Trigger
- Session transitions into `confirm` status (was not `confirm` in previous poll)
- A session is NOT re-notified while it stays in `confirm`
- If it transitions to ANY other status (including `idle`) the `notified` flag is cleared, so a future `confirm` re-triggers a notification

### Content
```
Title:  Vigil
Body:   «<ProjectName>» needs confirmation
```

### Implementation

**`monitor/notifier.go`**: `Notifier` struct
```go
type Notifier struct {
    notified map[string]bool // keyed by sessionID
    enabled  atomic.Bool     // use sync/atomic for lock-free read in Check
    mu       sync.Mutex      // guards notified map only
}
```
- `Check(prev, curr []Session)` — detects new `confirm` transitions, fires notification, tracks state
  - Reads `enabled` via `atomic.Bool.Load()` — no mutex needed for this field
  - Acquires `mu` to read/write the `notified` map
  - For each sessionID in `curr` with status `confirm`: if not in `notified` map → fire, add to map
  - For each sessionID in `notified` that is NOT `confirm` in `curr` (including gone or idle) → delete from map
  - Skips all firing if `enabled == false`
- `SetEnabled(v bool)` — calls `atomic.Bool.Store(v)`
- Fires via `exec.Command("osascript", "-e", `display notification "..." with title "Vigil"``)`

**`app.go`**:
- `Notifier` field on `App`, initialized at startup
- Called in `emitSessions()` with previous and current session slices
- `prevSessions []Session` stored as a field on `App`, updated each tick

### Settings

`~/.vigil/settings.json`:
```json
{
  "notifyConfirm": true,
  "notifyWaiting": false,
  "badgeConfirm": true,
  "badgeWaiting": true,
  "badgeActive": false
}
```

Each setting controls which statuses trigger the corresponding feature:
- **Notifications**: `notifyConfirm`, `notifyWaiting` — which statuses fire macOS notifications
- **Badge**: `badgeConfirm`, `badgeWaiting`, `badgeActive` — which statuses count toward the tray badge number

**Concurrency**: settings are read/written from two goroutines (poll goroutine reads; Wails handler goroutine writes). Use a `sync.Mutex` on a `Settings` struct in `app.go`:
```go
type Settings struct {
    NotifyConfirm bool `json:"notifyConfirm"`
    NotifyWaiting bool `json:"notifyWaiting"`
    BadgeConfirm  bool `json:"badgeConfirm"`
    BadgeWaiting  bool `json:"badgeWaiting"`
    BadgeActive   bool `json:"badgeActive"`
}
```

**App methods** (both called by frontend via Wails):
- `GetSettings() Settings` — acquires mutex, returns current settings object
- `UpdateSettings(s Settings)` — acquires mutex, saves to file, propagates to Notifier

Loaded at startup from file; defaults: notifyConfirm=true, rest of notifications off; badgeConfirm+badgeWaiting=true, badgeActive=false.

---

## Feature 3: Session History

### Goal
Browse all past Claude Code sessions (no active PID) grouped by project, and resume them in the right IDE/terminal.

### Data Source
Walk `~/.claude/projects/` directories. Each directory name is an encoded CWD (`strings.ReplaceAll(cwd, "/", "-")`). For each `.jsonl` file in each directory, check if the directory name appears among the active sessions' re-encoded CWDs — compute the encoded form of each active session CWD on the fly with `strings.ReplaceAll(s.CWD, "/", "-")` and build a `map[string]bool` lookup before walking. If the directory name is in that map, skip it (active session); otherwise include it in history.

**CWD recovery**: decode directory name with `"/" + strings.ReplaceAll(encoded, "-", "/")`. This is lossy when the original path contains literal `-` characters; this limitation is accepted as a known edge case. The decoded CWD is used for display and for `ResumeSession`.

### Data Extracted Per Session
| Field | Source |
|-------|--------|
| `sessionId` | filename (without `.jsonl`) |
| `name` | `customTitle` or `slug` from JSONL (reuse `ParseName`) |
| `lastActiveAt` | file modification time |
| `tokensIn/Out` | reuse `ParseTokens` — but scan only last 200 lines (same `tailFile` helper) to bound latency for large files |

### Grouping
- Grouped by decoded CWD
- Groups sorted by most recent `lastActiveAt` descending
- Max 50 projects; max 5 sessions per project (most recent first)

### Resume Action

`App.ResumeSession(cwd, sessionID string)`:
1. Sanitize both `cwd` and `sessionID` before embedding in AppleScript: escape all `"` → `\"` (e.g. `strings.ReplaceAll(s, `"`, `\"`)`). Session IDs from Claude Code are UUID-like and contain no special chars, but CWDs can contain quotes on unusual setups — sanitize both unconditionally.
2. Uses `osascript` to open a new Terminal.app window at `cwd` running `claude --resume <sessionID>`:
```applescript
tell application "Terminal"
    do script "cd \"<escaped-cwd>\" && claude --resume \"<escaped-sessionID>\""
    activate
end tell
```
3. If VS Code or Cursor is detected as open for that `cwd` (via `IDEDetector`), additionally bring the IDE window to the front using the existing `switcher.ActivateSession` — but this is best-effort; the terminal command always runs regardless.

Note: the resume command runs in a new macOS Terminal window, not in the IDE's integrated terminal, to avoid the complexity of terminal injection.

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
    ProjectName string              `json:"projectName"`
    CWD         string              `json:"cwd"`
    Sessions    []HistoricalSession `json:"sessions"`
}
```

### New App Methods
- `GetHistory() []ProjectHistory`
- `ResumeSession(cwd, sessionID string)`
- `GetSettings() Settings`
- `UpdateSettings(s Settings)`

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
- Groups with > 1 session are collapsed by default; groups with exactly 1 session are expanded
- Click on a session row → `ResumeSession(cwd, sessionId)`
- `GetHistory()` called once when the tab is first opened; no auto-refresh (history is static)

### Settings panel
`status-bar.ts` gains a small gear icon (⚙) on the right. Click shows an inline panel with two sections:

**Notifications** (which statuses fire macOS notifications):
- Needs confirmation (default: on)
- Waiting for input (default: off)

**Badge** (which statuses count toward tray icon badge):
- Needs confirmation (default: on)
- Waiting for input (default: on)
- Active sessions (default: off)

On mount, reads settings via `GetSettings()`. Each checkbox change calls `UpdateSettings(settings)`.

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
  → monitor.Manager.Collect()              → []Session
  → Notifier.Check(prevSessions, curr)     → osascript if new confirm
  → tray.SetBadge(confirmCount + waitingCount)
  → wails event "sessions:updated"         → frontend Active tab

user opens History tab
  → App.GetHistory()                       → walks ~/.claude/projects/**/*.jsonl
  → filters out active cwds
  → groups, sorts, limits (50 proj × 5 sessions)
  → returns []ProjectHistory              → frontend history-list

user clicks history session
  → App.ResumeSession(cwd, sessionId)
  → osascript opens new Terminal.app window, runs claude --resume <id>
  → switcher.ActivateSession for IDE (best-effort)

user changes any setting
  → App.UpdateSettings(settings)          → settingsMu.Lock, write file, apply to Notifier
  → App.GetSettings() Settings            → settingsMu.Lock, read current values
```

---

## Out of Scope

- Cost / pricing calculations
- Per-session statistics panel
- Cross-machine sync
- Injecting commands into VS Code's integrated terminal
