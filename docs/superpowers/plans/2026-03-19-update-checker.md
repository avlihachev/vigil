# Update Checker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Notify users when a new Vigil version is available on GitHub Releases, with a visual indicator on the settings gear and a download banner in the settings panel.

**Architecture:** Go backend checks GitHub API at startup and weekly, emits a Wails event when an update is found. Frontend shows an orange dot on the gear icon and a banner with download link inside the settings panel.

**Tech Stack:** Go (net/http, encoding/json), Wails v2 events, Lit web components, GitHub Releases API.

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `version.go` | Create | `Version` constant, `App.GetVersion()` |
| `monitor/updater.go` | Create | `Updater` struct, `Check()`, semver comparison, HTTP call |
| `monitor/updater_test.go` | Create | Tests with `httptest` mock server |
| `app.go` | Modify | Wire updater into startup + pollLoop, add `OpenURL()`, add `LastUpdateCheck` to Settings |
| `frontend/src/types.ts` | Modify | Add `UpdateInfo` interface, add `lastUpdateCheck` to Settings |
| `frontend/src/status-bar.ts` | Modify | Listen for `update:available` event, gear dot indicator, update banner |

---

### Task 1: Version constant and GetVersion

**Files:**
- Create: `version.go`

- [ ] **Step 1: Create version.go**

```go
package main

const Version = "0.1.1"

func (a *App) GetVersion() string {
	return Version
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: compiles with no errors (ld warning is ok)

- [ ] **Step 3: Commit**

```bash
git add version.go
git commit -m "feat: add Version constant and GetVersion binding"
```

---

### Task 2: Updater with semver comparison and HTTP check

**Files:**
- Create: `monitor/updater.go`
- Create: `monitor/updater_test.go`

- [ ] **Step 1: Write failing tests**

Create `monitor/updater_test.go`:

```go
package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	if !newerVersion("0.2.0", "0.1.1") {
		t.Error("0.2.0 should be newer than 0.1.1")
	}
	if newerVersion("0.1.1", "0.1.1") {
		t.Error("same version should not be newer")
	}
	if newerVersion("0.1.0", "0.1.1") {
		t.Error("0.1.0 should not be newer than 0.1.1")
	}
	if !newerVersion("1.0.0", "0.9.9") {
		t.Error("1.0.0 should be newer than 0.9.9")
	}
	if !newerVersion("0.1.2", "0.1.1") {
		t.Error("0.1.2 should be newer than 0.1.1")
	}
}

func TestUpdaterCheck_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v0.2.0",
			"html_url": "https://github.com/test/repo/releases/tag/v0.2.0",
		})
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected update info, got nil")
	}
	if info.Version != "0.2.0" {
		t.Errorf("expected version 0.2.0, got %s", info.Version)
	}
	if info.DownloadURL != "https://github.com/test/repo/releases/tag/v0.2.0" {
		t.Errorf("unexpected download URL: %s", info.DownloadURL)
	}
}

func TestUpdaterCheck_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v0.1.1",
			"html_url": "https://github.com/test/repo/releases/tag/v0.1.1",
		})
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil (up to date), got %+v", info)
	}
}

func TestUpdaterCheck_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err == nil {
		t.Error("expected error on 500 response")
	}
	if info != nil {
		t.Error("expected nil info on error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./monitor/ -run TestNewer -v && go test ./monitor/ -run TestUpdater -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement updater**

Create `monitor/updater.go`:

```go
package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type UpdateInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadURL"`
}

type Updater struct {
	currentVersion string
	apiURL         string
	client         *http.Client
}

// NewUpdater creates an updater. apiURL is the full URL to the releases/latest endpoint
// (or a test server URL).
func NewUpdater(currentVersion, apiURL string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		apiURL:         apiURL,
		client:         &http.Client{Timeout: 10 * time.Second},
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func (u *Updater) Check() (*UpdateInfo, error) {
	resp, err := u.client.Get(u.apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	remote := strings.TrimPrefix(rel.TagName, "v")
	if !newerVersion(remote, u.currentVersion) {
		return nil, nil
	}

	return &UpdateInfo{
		Version:     remote,
		DownloadURL: rel.HTMLURL,
	}, nil
}

// newerVersion returns true if a is newer than b (simple X.Y.Z comparison)
func newerVersion(a, b string) bool {
	ap := parseVersion(a)
	bp := parseVersion(b)
	for i := 0; i < 3; i++ {
		if ap[i] > bp[i] {
			return true
		}
		if ap[i] < bp[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		result[i], _ = strconv.Atoi(p)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./monitor/ -run "TestNewer|TestUpdater" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add monitor/updater.go monitor/updater_test.go
git commit -m "feat: add Updater with GitHub Releases API check and semver comparison"
```

---

### Task 3: Wire updater into App startup and pollLoop

**Files:**
- Modify: `app.go` (Settings struct, NewApp, startup, pollLoop)

- [ ] **Step 1: Add `LastUpdateCheck` to Settings struct and `updater` + `updateInfo` to App**

In `app.go`, add to `Settings`:

```go
type Settings struct {
	NotifyConfirm   bool   `json:"notifyConfirm"`
	NotifyWaiting   bool   `json:"notifyWaiting"`
	BadgeConfirm    bool   `json:"badgeConfirm"`
	BadgeWaiting    bool   `json:"badgeWaiting"`
	BadgeActive     bool   `json:"badgeActive"`
	LastUpdateCheck string `json:"lastUpdateCheck,omitempty"`
}
```

Add to `App` struct:

```go
updater    *monitor.Updater
updateInfo *monitor.UpdateInfo
```

In `NewApp()`, after creating the app, add:

```go
app.updater = monitor.NewUpdater(Version, "https://api.github.com/repos/avlihachev/vigil/releases/latest")
```

- [ ] **Step 2: Add update check method and OpenURL**

Add to `app.go`:

```go
func (a *App) checkForUpdate() {
	info, err := a.updater.Check()
	if err != nil || info == nil {
		return
	}
	a.updateInfo = info
	runtime.EventsEmit(a.ctx, "update:available", info)

	a.settingsMu.Lock()
	a.settings.LastUpdateCheck = time.Now().Format(time.RFC3339)
	data, _ := json.MarshalIndent(a.settings, "", "  ")
	path := a.settingsPath
	a.settingsMu.Unlock()
	os.WriteFile(path, data, 0644)
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
```

- [ ] **Step 3: Call update check at startup**

In `startup()`, after `tray.Init(...)`, add:

```go
if a.needsUpdateCheck() {
	go a.checkForUpdate()
}
```

- [ ] **Step 4: Add periodic check to pollLoop**

In `pollLoop()`, inside the `case <-ticker.C:` block, after `a.emitSessions()`:

```go
if a.updateInfo != nil {
	// already found an update, re-emit for newly opened windows
} else if a.needsUpdateCheck() {
	go a.checkForUpdate()
}
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: compiles with no errors

- [ ] **Step 6: Commit**

```bash
git add app.go
git commit -m "feat: wire updater into startup and poll loop with weekly check"
```

---

### Task 4: Frontend — UpdateInfo type and status-bar indicator

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/status-bar.ts`

- [ ] **Step 1: Add UpdateInfo to types.ts**

Add to `frontend/src/types.ts`:

```typescript
export interface UpdateInfo {
  version: string;
  downloadURL: string;
}
```

- [ ] **Step 2: Add update state and event listener to status-bar.ts**

Add import of `UpdateInfo` and new state:

```typescript
import type { Settings, UpdateInfo } from './types';
```

Add state property:

```typescript
@state() private updateInfo: UpdateInfo | null = null;
```

In `connectedCallback()`, add Wails event listener:

```typescript
// @ts-ignore
window.runtime?.EventsOn('update:available', (info: UpdateInfo) => {
  this.updateInfo = info;
});
```

- [ ] **Step 3: Add gear dot and update banner styles**

Add to the `static styles` block:

```css
.gear-wrap {
  position: relative;
  display: inline-block;
}
.update-dot {
  position: absolute;
  top: -1px;
  right: -1px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f08000;
}
.update-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(240, 128, 0, 0.1);
  border-top: 1px solid rgba(240, 128, 0, 0.2);
  font-size: 12px;
  color: #f0a050;
}
.update-banner a {
  color: #f08000;
  text-decoration: none;
  font-weight: 500;
  cursor: pointer;
}
.update-banner a:hover {
  text-decoration: underline;
}
```

- [ ] **Step 4: Update render method**

Replace the gear span with a wrapper that includes the dot:

```html
<span class="gear-wrap">
  <span class="gear" @click=${this._toggleSettings}>&#x2699;</span>
  ${this.updateInfo ? html`<span class="update-dot"></span>` : ''}
</span>
```

Add the update banner at the top of the settings panel (inside the `${this.showSettings ? html\`...\` : ''}` block, before the Notifications section):

```html
${this.updateInfo ? html`
  <div class="update-banner">
    <span>Vigil ${this.updateInfo.version} available</span>
    <a @click=${() => this._openUpdate()}>Download</a>
  </div>
` : ''}
```

- [ ] **Step 5: Add _openUpdate method**

```typescript
private _openUpdate() {
  if (this.updateInfo) {
    // @ts-ignore
    window.go?.main?.App?.OpenURL(this.updateInfo.downloadURL);
  }
}
```

- [ ] **Step 6: Verify build**

Run: `cd frontend && npm run build`
Expected: compiles with no errors

- [ ] **Step 7: Commit**

```bash
git add frontend/src/types.ts frontend/src/status-bar.ts
git commit -m "feat: show update indicator on gear icon and download banner in settings"
```

---

### Task 5: Build, test, and release

**Files:** none new

- [ ] **Step 1: Full build**

Run: `go build ./... && cd frontend && npm run build && cd ..`
Expected: no errors

- [ ] **Step 2: Run all Go tests**

Run: `go test ./monitor/... ./switcher/...`
Expected: all PASS

- [ ] **Step 3: Build DMG**

Run: `bash scripts/build-dmg.sh`
Expected: `dist/Vigil.dmg` created

- [ ] **Step 4: Manual test**

Install DMG, launch Vigil. Verify:
- No crash on startup (update check runs in background)
- If update is available: orange dot on gear, banner in settings panel
- "Download" link opens browser to GitHub release page

- [ ] **Step 5: Commit any fixes and push**

```bash
git push
```
