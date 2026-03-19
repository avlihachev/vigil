# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# dev mode (hot-reload, opens app)
wails dev

# production build
wails build -platform darwin/universal -clean

# build DMG for distribution
bash scripts/build-dmg.sh          # → dist/Vigil.dmg

# regenerate app icon
python3 -m venv /tmp/v && /tmp/v/bin/pip install Pillow -q && /tmp/v/bin/python3 scripts/gen_icon.py

# run Go tests
go test ./monitor/...
go test ./monitor/ -run TestName     # single test
go test ./switcher/...
```

## Architecture

Vigil is a macOS menubar app built with **Wails v2** (Go backend + Lit/TypeScript frontend). It polls Claude Code's local data files every 3 seconds and shows active sessions in a floating popup window.

### Data flow

```
~/.claude/sessions/*.json          → Scanner      → live pids + cwd
~/.claude/ide/*.lock               → IDEDetector  → VSCode / Cursor workspace mapping
~/.claude/projects/**/*.jsonl      → ActivityParser → recent actions, tokens, name, status
```

`monitor.Manager.Collect()` orchestrates these three sources into `[]Session`, which `App.emitSessions()` pushes to the frontend via Wails event `sessions:updated`.

### Package layout

| Package | Purpose |
|---------|---------|
| `monitor/` | All data collection. `Manager` composes Scanner + IDEDetector + ActivityParser. |
| `switcher/` | macOS Accessibility C API + CGo to raise the right IDE/terminal window on click. |
| `tray/` | Native macOS status bar item (Objective-C + CGo). Left click = toggle popup; right/ctrl click = context menu with Quit. |
| `frontend/src/` | Lit web components: `session-list` → `session-card` + `status-bar`. No framework router — single view. |

### Key design decisions

- **No dock icon** — `NSApplicationActivationPolicyAccessory` set in `tray_init` after Wails starts.
- **Window visibility** — uses `NSWindow orderFront/orderOut` (not `NSApp activate/hide`) to avoid macOS stealing focus from the user's terminal.
- **Status detection** — based on JSONL file modification time (idle if >5 min stale) + last assistant message type (text-only → waiting, tool_use pending → confirm).
- **JSONL lookup** — for single-session CWDs: `FindJSONL` tries `sessionId.jsonl` first, falls back to newest `.jsonl`. For sibling sessions (multiple sessions sharing the same CWD): `Collect()` groups by CWD, does exact filename match first, then greedy 1:1 assignment (sessions sorted by `startedAt` desc, JSONLs by `modTime` desc). `ParseAllFromPath` reads each resolved JSONL once to extract actions, name, tokens, and modTime in a single pass.
- **Terminal detection** — walks the process tree (up to 6 levels) via `ps -o ppid=,comm=` to identify the terminal emulator.

### Window switching (`switcher/`)

Uses the **macOS Accessibility C API** directly from CGo (`script_darwin.m`), not `osascript`. This ensures Accessibility permissions are checked against `Vigil.app` itself.

**Matching strategy** — `raise_window()` iterates AX windows and tries two matchers in order:
1. **AXDocument** — contains the CWD as a `file://` URL for terminals (Ghostty, Terminal.app) or the open file path for VSCode. Matched with `hasPrefix` against the session's CWD URL.
2. **AXTitle** — window title substring match against `filepath.Base(cwd)`.

**Per-source behavior:**
| Source | Primary | Fallback |
|--------|---------|----------|
| **Ghostty / terminals** | AX API (doc match by CWD, then title) | `open -a AppName` |
| **VSCode** | AX API first (handles workspaces where title ≠ folder name) | `code /path` CLI + `open -a` |
| **Cursor** | AX API first | `cursor /path` CLI + `open -a` |

**Key learnings:**
- `osascript` is a separate process — Accessibility permission must be granted to it independently, which is bad UX. Running AppleScript via `NSAppleScript` (in-process) checks Vigil.app instead.
- Direct AX C API (`AXUIElementPerformAction` + `AXRaiseAction`) is more reliable than AppleScript `perform action "AXRaise"` and avoids the NSAppleScript overhead.
- Ghostty's `AXDocument` contains the CWD (`file:///Users/foo/`), NOT the TTY path. Window titles show the foreground process name (e.g. "✳ Claude Code"), not the folder.
- VSCode workspace windows may have arbitrary titles (from `.code-workspace` files). AXDocument matching by CWD is more reliable than title matching.
- `code` CLI is not in PATH for macOS GUI apps. Use full paths: `/usr/local/bin/code`, `/Applications/Visual Studio Code.app/.../bin/code`.
- `NSRunningApplication.activateWithOptions:` alone doesn't reliably bring windows to front from an accessory app. Combine with `AXRaiseAction` for the specific window.
- Production builds must be signed with a stable identifier (`codesign --sign - --identifier com.vigil.app`) so Accessibility permissions persist across rebuilds. `wails dev` creates ad-hoc `a.out` signatures that macOS invalidates on each build.
- `PromptAccessibility()` calls `AXIsProcessTrustedWithOptions` with `kAXTrustedCheckOptionPrompt` at startup to show the system dialog.

### Settings & notifications

`status-bar` component (bottom of popup) has a gear toggle that expands a settings panel. Settings are persisted to `~/.vigil/settings.json` and applied immediately.

**Controls (custom pill toggles, not system checkboxes):**
- **Notifications** — desktop notifications for `confirm` and `waiting` states
- **Badge** — menubar icon badge count for `confirm`, `waiting`, `active` states

**Badge rendering** (`tray_darwin.m` `traySetBadge`) — draws a base `◉` symbol + rounded-rect badge overlay with count. Badge color is muted orange (`#f08000`), matching the confirm status color in the UI.

### Wails bindings

`frontend/wailsjs/` is auto-generated by `wails dev` — don't edit manually. The frontend calls `window.go.main.App.GetSessions()` and `window.go.main.App.OpenSession(source, cwd, pid)`. Go methods on `App` struct are exposed automatically.

### tray CGo note

`tray/tray_darwin.m` is Objective-C compiled via CGo. Go callbacks are exported with `//export trayOnClick` and `//export trayOnQuit`. Adding a new callback requires declaration in both `.go` (CGo `extern`) and `.m` (`extern void ...`).
