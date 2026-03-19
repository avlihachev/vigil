# Update Checker Design

## Goal

Notify users when a new Vigil version is available on GitHub Releases. Check at startup and weekly. Show an indicator on the settings gear + a banner inside the settings panel with a download link.

## Architecture

### Version constant

New file `version.go` in the root package:

```go
package main

const Version = "0.1.1"
```

Exposed to frontend via `App.GetVersion() string`.

### Updater (`monitor/updater.go`)

```go
type UpdateInfo struct {
    Version     string `json:"version"`
    DownloadURL string `json:"downloadURL"`
}

type Updater struct {
    currentVersion string
    repoOwner      string
    repoName       string
}
```

**`Check() (*UpdateInfo, error)`** — calls `https://api.github.com/repos/{owner}/{repo}/releases/latest`, parses the `tag_name` (strips leading `v`), compares with `currentVersion` using simple semver string comparison. If remote is newer, returns `UpdateInfo` with the release's `html_url` as `DownloadURL`. Returns `nil` if up to date or on error (fail silently — update check should never break the app).

### Scheduling

**Last check timestamp** — stored in `~/.vigil/settings.json` as `lastUpdateCheck` (RFC3339 string). Added to the `Settings` struct but not exposed in the settings UI.

**Startup check** — in `App.startup()`, after init, spawn a goroutine that:
1. Reads `lastUpdateCheck` from settings
2. If never checked or >7 days ago: calls `updater.Check()`
3. If update available: emits Wails event `update:available` with `UpdateInfo`
4. Saves current time as `lastUpdateCheck`

**Periodic check** — in the existing `pollLoop` (runs every 3s), add a lightweight check: if `time.Since(lastUpdateCheck) > 7*24h`, trigger the update check. This handles long-running app instances.

### Frontend display

**Wails event**: `update:available` with payload `{version: string, downloadURL: string}`.

**`status-bar.ts` changes:**

1. New state: `updateAvailable: UpdateInfo | null`
2. Listen for `update:available` event in `connectedCallback()`
3. **Gear indicator**: orange dot overlay on the gear icon when `updateAvailable` is set
4. **Settings panel banner**: at the top of the settings panel, show a styled banner:
   - Text: "Vigil {version} available"
   - "Download" button that calls `App.OpenURL(downloadURL)` to open in browser
   - Muted orange styling consistent with the confirm status color

**`App.OpenURL(url string)`** — new Go method, calls `runtime.BrowserOpenURL(a.ctx, url)`.

### Semver comparison

Simple string split on `.`, compare major/minor/patch as integers. No need for a library — Vigil uses basic `X.Y.Z` tags.

### Error handling

- Network errors: silently ignored, retry next cycle
- Invalid JSON / missing tag: silently ignored
- Rate limiting (GitHub allows 60 req/h unauthenticated): not a concern with weekly checks

## Files to create/modify

| File | Action |
|------|--------|
| `version.go` | Create — `Version` constant + `App.GetVersion()` |
| `monitor/updater.go` | Create — `Updater` struct + `Check()` |
| `monitor/updater_test.go` | Create — tests with mock HTTP |
| `app.go` | Modify — add `Updater` field, startup check, periodic check in pollLoop, `OpenURL()` method |
| `frontend/src/status-bar.ts` | Modify — listen for event, gear dot, update banner |
| `frontend/src/types.ts` | Modify — add `UpdateInfo` interface |

## Not in scope

- Auto-download or auto-install
- Checking for pre-release versions
- User-configurable check interval
- Skipping specific versions
