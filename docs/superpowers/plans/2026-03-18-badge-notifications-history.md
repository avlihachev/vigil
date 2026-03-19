# Badge, Notifications, History — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add tray icon badge (confirm+waiting count), macOS notifications on confirm (with on/off toggle), and a History tab to browse and resume past Claude Code sessions.

**Architecture:** Extend `monitor.Manager` + `App` + tray without new abstractions: `monitor/notifier.go` and `monitor/history_scanner.go` are pure Go and fully testable; `tray/tray_darwin.m` gets one new ObjC function for badge rendering; `app.go` grows a Settings struct (mutex-guarded) and four new Wails-exposed methods; the frontend gets a tab bar in `session-list.ts`, a new `history-list.ts` component, and a settings gear in `status-bar.ts`.

**Tech Stack:** Go 1.21, CGo + Objective-C (Cocoa), Lit 3, TypeScript, Wails v2, osascript for notifications and terminal launch.

**Spec:** `docs/superpowers/specs/2026-03-18-vigil-badge-notifications-history-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `monitor/models.go` | Add `CountNeedsAttention([]Session) int` helper |
| Create | `monitor/notifier.go` | Notifier struct — detect confirm transitions, fire notifications |
| Create | `monitor/notifier_test.go` | Tests for Notifier |
| Create | `monitor/history_scanner.go` | ScanHistory — walk JSONL files, build ProjectHistory groups |
| Create | `monitor/history_scanner_test.go` | Tests for HistoryScanner |
| Modify | `monitor/activity_parser.go` | Extract `parseTokensFromLines` helper for reuse |
| Modify | `tray/tray_darwin.m` | Add `traySetBadge(int count)` ObjC function |
| Modify | `tray/tray_darwin.go` | Add CGo preamble entry + `SetBadge(int)` Go wrapper |
| Modify | `app.go` | Add Settings, Notifier, prevSessions; add 4 new methods |
| Modify | `frontend/src/types.ts` | Add `HistoricalSession`, `ProjectHistory` interfaces |
| Modify | `frontend/src/session-list.ts` | Add Active/History tab bar |
| Create | `frontend/src/history-list.ts` | Collapsible history component |
| Modify | `frontend/src/status-bar.ts` | Add gear icon + notifications toggle overlay |

---

## Task 1: CountNeedsAttention helper + Notifier

### Files:
- Modify: `monitor/models.go`
- Create: `monitor/notifier.go`
- Create: `monitor/notifier_test.go`

- [ ] **Step 1: Write failing tests for Notifier**

Create `monitor/notifier_test.go`:
```go
package monitor

import (
	"sync"
	"testing"
)

type firedEntry struct {
	project string
	status  SessionStatus
}

func newTestNotifier() (*Notifier, *[]firedEntry) {
	fired := &[]firedEntry{}
	n := &Notifier{
		notified: make(map[string]SessionStatus),
		fireFunc: func(projectName string, status SessionStatus) {
			*fired = append(*fired, firedEntry{projectName, status})
		},
	}
	n.notifyConfirm.Store(true)
	n.notifyWaiting.Store(false)
	return n, fired
}

func TestNotifier_FiresOnNewConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	curr := []Session{{SessionID: "s1", ProjectName: "myproject", Status: StatusConfirm}}
	n.Check(nil, curr)
	if len(*fired) != 1 || (*fired)[0].project != "myproject" {
		t.Errorf("expected 1 notification for myproject, got %v", *fired)
	}
}

func TestNotifier_DoesNotRepeatWhileConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	n.Check(nil, curr)
	n.Check(curr, curr) // stays confirm
	if len(*fired) != 1 {
		t.Errorf("expected exactly 1 notification, got %d", len(*fired))
	}
}

func TestNotifier_RefiresAfterLeavingConfirm(t *testing.T) {
	n, fired := newTestNotifier()
	confirm := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	idle := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusIdle}}
	n.Check(nil, confirm)    // fires once
	n.Check(confirm, idle)   // leaves confirm — clears flag
	n.Check(idle, confirm)   // back to confirm — fires again
	if len(*fired) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(*fired))
	}
}

func TestNotifier_SkipsConfirmWhenDisabled(t *testing.T) {
	n, fired := newTestNotifier()
	n.notifyConfirm.Store(false)
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusConfirm}}
	n.Check(nil, curr)
	if len(*fired) != 0 {
		t.Errorf("expected 0 notifications when confirm disabled, got %d", len(*fired))
	}
}

func TestNotifier_FiresOnWaitingWhenEnabled(t *testing.T) {
	n, fired := newTestNotifier()
	n.notifyWaiting.Store(true)
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusWaiting}}
	n.Check(nil, curr)
	if len(*fired) != 1 || (*fired)[0].status != StatusWaiting {
		t.Errorf("expected 1 waiting notification, got %v", *fired)
	}
}

func TestNotifier_SkipsWaitingWhenDisabled(t *testing.T) {
	n, fired := newTestNotifier()
	// notifyWaiting defaults to false
	curr := []Session{{SessionID: "s1", ProjectName: "p", Status: StatusWaiting}}
	n.Check(nil, curr)
	if len(*fired) != 0 {
		t.Errorf("expected 0 waiting notifications when disabled, got %d", len(*fired))
	}
}

func TestNotifier_ConcurrentConfig(t *testing.T) {
	n, _ := newTestNotifier()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v bool) {
			defer wg.Done()
			n.SetNotifyConfirm(v)
			n.SetNotifyWaiting(v)
		}(i%2 == 0)
	}
	wg.Wait() // must not race
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/ -run TestNotifier -v
```
Expected: compile error — `Notifier` not defined.

- [ ] **Step 3: Add CountBadge to models.go**

Add at end of `monitor/models.go`:
```go
// CountBadge returns the number of sessions matching the enabled badge statuses.
func CountBadge(sessions []Session, confirm, waiting, active bool) int {
	n := 0
	for _, s := range sessions {
		switch s.Status {
		case StatusConfirm:
			if confirm { n++ }
		case StatusWaiting:
			if waiting { n++ }
		case StatusActive:
			if active { n++ }
		}
	}
	return n
}
```

- [ ] **Step 4: Implement Notifier**

Create `monitor/notifier.go`:
```go
package monitor

import (
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Notifier tracks session status transitions and fires macOS notifications
// for configurable statuses (confirm, waiting).
type Notifier struct {
	mu            sync.Mutex
	notified      map[string]SessionStatus
	notifyConfirm atomic.Bool
	notifyWaiting atomic.Bool
	fireFunc      func(projectName string, status SessionStatus) // injectable for tests
}

func NewNotifier() *Notifier {
	n := &Notifier{notified: make(map[string]SessionStatus)}
	n.notifyConfirm.Store(true)
	n.notifyWaiting.Store(false)
	return n
}

func (n *Notifier) SetNotifyConfirm(v bool) { n.notifyConfirm.Store(v) }
func (n *Notifier) SetNotifyWaiting(v bool) { n.notifyWaiting.Store(v) }

func (n *Notifier) shouldNotify(status SessionStatus) bool {
	switch status {
	case StatusConfirm:
		return n.notifyConfirm.Load()
	case StatusWaiting:
		return n.notifyWaiting.Load()
	}
	return false
}

// Check compares prev and curr snapshots, firing a notification when a session
// enters an enabled status for the first time. Clears the flag when it leaves.
func (n *Notifier) Check(prev, curr []Session) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currMap := make(map[string]Session, len(curr))
	for _, s := range curr {
		currMap[s.SessionID] = s
	}

	// clear flag for sessions whose status changed or disappeared
	for id, prevStatus := range n.notified {
		s, ok := currMap[id]
		if !ok || s.Status != prevStatus {
			delete(n.notified, id)
		}
	}

	// fire for new notify-worthy sessions
	for _, s := range curr {
		if !n.shouldNotify(s.Status) {
			continue
		}
		if _, already := n.notified[s.SessionID]; already {
			continue
		}
		n.notified[s.SessionID] = s.Status
		if n.fireFunc != nil {
			n.fireFunc(s.ProjectName, s.Status)
		} else {
			fireNotification(s.ProjectName, s.Status)
		}
	}
}

var statusMessages = map[SessionStatus]string{
	StatusConfirm: "needs confirmation",
	StatusWaiting: "waiting for input",
}

func fireNotification(projectName string, status SessionStatus) {
	text := statusMessages[status]
	if text == "" {
		text = string(status)
	}
	msg := fmt.Sprintf(`display notification "%s %s" with title "Vigil"`, projectName, text)
	exec.Command("osascript", "-e", msg).Run()
}
```

- [ ] **Step 5: Run tests to confirm they pass**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/ -run TestNotifier -v
```
Expected: all 5 pass.

- [ ] **Step 6: Add CountNeedsAttention test to models_test.go**

Add to `monitor/models_test.go`:
```go
func TestCountBadge(t *testing.T) {
	sessions := []Session{
		{SessionID: "1", Status: StatusConfirm},
		{SessionID: "2", Status: StatusWaiting},
		{SessionID: "3", Status: StatusActive},
		{SessionID: "4", Status: StatusIdle},
	}
	// confirm + waiting only
	if n := CountBadge(sessions, true, true, false); n != 2 {
		t.Errorf("expected 2 (confirm+waiting), got %d", n)
	}
	// confirm only
	if n := CountBadge(sessions, true, false, false); n != 1 {
		t.Errorf("expected 1 (confirm only), got %d", n)
	}
	// all three
	if n := CountBadge(sessions, true, true, true); n != 3 {
		t.Errorf("expected 3 (confirm+waiting+active), got %d", n)
	}
	// nothing enabled
	if n := CountBadge(sessions, false, false, false); n != 0 {
		t.Errorf("expected 0 (all disabled), got %d", n)
	}
	if n := CountBadge(nil, true, true, true); n != 0 {
		t.Errorf("expected 0 for nil, got %d", n)
	}
}
```

- [ ] **Step 7: Run all monitor tests**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/... -v
```
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add monitor/notifier.go monitor/notifier_test.go monitor/models.go monitor/models_test.go
git commit -m "feat: add Notifier with configurable statuses and CountBadge helper"
```

---

## Task 2: History Scanner

### Files:
- Modify: `monitor/activity_parser.go` — extract `parseTokensFromLines` helper
- Create: `monitor/history_scanner.go`
- Create: `monitor/history_scanner_test.go`

- [ ] **Step 1: Write failing tests**

Create `monitor/history_scanner_test.go`:
```go
package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeJSONL(t *testing.T, dir, encodedCWD, sessionID, slug string, tokensIn int) {
	t.Helper()
	projDir := filepath.Join(dir, "projects", encodedCWD)
	os.MkdirAll(projDir, 0755)
	ts := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	line := fmt.Sprintf(
		`{"type":"assistant","timestamp":"%s","sessionId":"%s","slug":"%s","message":{"role":"assistant","content":[],"usage":{"input_tokens":%d,"output_tokens":10}}}`,
		ts, sessionID, slug, tokensIn,
	)
	os.WriteFile(filepath.Join(projDir, sessionID+".jsonl"), []byte(line+"\n"), 0644)
}

func TestHistoryScanner_ReturnsHistoricalSessions(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, dir, "-tmp-proj", "sess-1", "my-task", 1000)

	scanner := NewHistoryScanner(dir)
	result := scanner.ScanHistory(nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 project, got %d", len(result))
	}
	p := result[0]
	if p.CWD != "/tmp/proj" {
		t.Errorf("expected CWD /tmp/proj, got %s", p.CWD)
	}
	if p.ProjectName != "proj" {
		t.Errorf("expected ProjectName proj, got %s", p.ProjectName)
	}
	if len(p.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(p.Sessions))
	}
	if p.Sessions[0].SessionID != "sess-1" {
		t.Errorf("expected sess-1, got %s", p.Sessions[0].SessionID)
	}
	if p.Sessions[0].Name != "my-task" {
		t.Errorf("expected slug my-task, got %s", p.Sessions[0].Name)
	}
}

func TestHistoryScanner_ExcludesActiveSessions(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, dir, "-tmp-active", "sess-active", "active", 100)
	writeJSONL(t, dir, "-tmp-history", "sess-old", "old-task", 200)

	scanner := NewHistoryScanner(dir)
	result := scanner.ScanHistory([]string{"/tmp/active"})

	if len(result) != 1 {
		t.Fatalf("expected 1 project (active excluded), got %d", len(result))
	}
	if result[0].CWD != "/tmp/history" {
		t.Errorf("unexpected CWD: %s", result[0].CWD)
	}
}

func TestHistoryScanner_GroupsByProject(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, dir, "-tmp-proj", "sess-1", "task-1", 100)
	writeJSONL(t, dir, "-tmp-proj", "sess-2", "task-2", 200)

	scanner := NewHistoryScanner(dir)
	result := scanner.ScanHistory(nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 project group, got %d", len(result))
	}
	if len(result[0].Sessions) != 2 {
		t.Errorf("expected 2 sessions in group, got %d", len(result[0].Sessions))
	}
}

func TestHistoryScanner_LimitsTo5SessionsPerProject(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 8; i++ {
		writeJSONL(t, dir, "-tmp-proj", fmt.Sprintf("sess-%d", i), fmt.Sprintf("task-%d", i), 100)
	}

	scanner := NewHistoryScanner(dir)
	result := scanner.ScanHistory(nil)

	if len(result[0].Sessions) != 5 {
		t.Errorf("expected max 5 sessions, got %d", len(result[0].Sessions))
	}
}

func TestHistoryScanner_EmptyDir(t *testing.T) {
	scanner := NewHistoryScanner(t.TempDir())
	result := scanner.ScanHistory(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results for empty dir, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/ -run TestHistoryScanner -v
```
Expected: compile error — `HistoryScanner` not defined.

- [ ] **Step 3: Extract parseTokensFromLines from activity_parser.go**

In `monitor/activity_parser.go`, replace the body of `ParseTokens` with a call to the new helper. Add after the `ParseName` function:

```go
// parseTokensFromLines sums input and output tokens from pre-read lines.
func parseTokensFromLines(lines []string) (in, out int64) {
	for _, line := range lines {
		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Type == "assistant" && entry.Message != nil && entry.Message.Usage != nil {
			in += entry.Message.Usage.InputTokens
			out += entry.Message.Usage.OutputTokens
		}
	}
	return
}
```

Replace the body of the existing `ParseTokens` method:
```go
func (p *ActivityParser) ParseTokens(sessionID string, cwd string) (in, out int64) {
	jsonlPath := p.findJSONL(sessionID, cwd)
	if jsonlPath == "" {
		return
	}
	lines, err := readAllLines(jsonlPath)
	if err != nil {
		return
	}
	return parseTokensFromLines(lines)
}
```

Add the `readAllLines` helper below `tailFile`:
```go
func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, nil
}
```

- [ ] **Step 4: Verify existing tests still pass**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/... -v
```
Expected: all existing tests pass.

- [ ] **Step 5: Implement HistoryScanner**

Create `monitor/history_scanner.go`:
```go
package monitor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

type HistoryScanner struct {
	baseDir  string
	activity *ActivityParser
}

func NewHistoryScanner(baseDir string) *HistoryScanner {
	return &HistoryScanner{
		baseDir:  baseDir,
		activity: NewActivityParser(baseDir),
	}
}

// ScanHistory returns historical project groups, excluding any CWDs in activeCWDs.
// Results are sorted by most recent session, capped at 50 projects × 5 sessions.
func (s *HistoryScanner) ScanHistory(activeCWDs []string) []ProjectHistory {
	// build lookup of encoded active CWDs
	active := make(map[string]bool, len(activeCWDs))
	for _, cwd := range activeCWDs {
		active[strings.ReplaceAll(cwd, "/", "-")] = true
	}

	projectsDir := filepath.Join(s.baseDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}

	var groups []ProjectHistory
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		encodedCWD := e.Name()
		if active[encodedCWD] {
			continue
		}

		// decode CWD: "-Users-john-project" → "/Users/john/project"
		// lossy for paths with literal '-', accepted limitation
		cwd := "/" + strings.ReplaceAll(strings.TrimPrefix(encodedCWD, "-"), "-", "/")

		dirPath := filepath.Join(projectsDir, encodedCWD)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		var sessions []HistoricalSession
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			sessionID := strings.TrimSuffix(f.Name(), ".jsonl")
			jsonlPath := filepath.Join(dirPath, f.Name())

			info, err := f.Info()
			if err != nil {
				continue
			}
			lastActive := info.ModTime().UnixMilli()

			lines, _ := tailFile(jsonlPath, 200)
			tokIn, tokOut := parseTokensFromLines(lines)
			name := s.activity.ParseName(sessionID, cwd)

			sessions = append(sessions, HistoricalSession{
				SessionID:    sessionID,
				Name:         name,
				LastActiveAt: lastActive,
				TokensIn:     FormatTokens(tokIn),
				TokensOut:    FormatTokens(tokOut),
			})
		}

		if len(sessions) == 0 {
			continue
		}

		// sort by most recent first
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].LastActiveAt > sessions[j].LastActiveAt
		})
		if len(sessions) > 5 {
			sessions = sessions[:5]
		}

		groups = append(groups, ProjectHistory{
			ProjectName: filepath.Base(cwd),
			CWD:         cwd,
			Sessions:    sessions,
		})
	}

	// sort groups by most recent session
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Sessions) == 0 {
			return false
		}
		if len(groups[j].Sessions) == 0 {
			return true
		}
		return groups[i].Sessions[0].LastActiveAt > groups[j].Sessions[0].LastActiveAt
	})
	if len(groups) > 50 {
		groups = groups[:50]
	}
	return groups
}
```

- [ ] **Step 6: Run all history scanner tests**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/ -run TestHistoryScanner -v
```
Expected: all 5 pass.

- [ ] **Step 7: Run all monitor tests**

```bash
cd /Users/lihachev/Projects/vigil && go test ./monitor/... -v
```
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add monitor/history_scanner.go monitor/history_scanner_test.go monitor/activity_parser.go
git commit -m "feat: add HistoryScanner and parseTokensFromLines refactor"
```

---

## Task 3: Tray Badge (CGo + ObjC)

### Files:
- Modify: `tray/tray_darwin.m`
- Modify: `tray/tray_darwin.go`

No automated tests possible for CGo/ObjC rendering; verify by building.

- [ ] **Step 1: Add traySetBadge to tray_darwin.m**

Add this function at the end of `tray/tray_darwin.m`, before the final newline:

```objc
void traySetBadge(int count) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;

        if (count <= 0) {
            // restore original title text; clear any image
            statusItem.button.image = nil;
            statusItem.button.title = @"◉";
            return;
        }

        // render a small colored square with the count number as the button image
        NSSize size = NSMakeSize(22, 18);
        NSImage *img = [[NSImage alloc] initWithSize:size];
        [img lockFocus];

        // draw the base title character so the icon keeps its normal look
        NSString *base = @"◉";
        NSDictionary *baseAttrs = @{
            NSFontAttributeName: [NSFont systemFontOfSize:13],
            NSForegroundColorAttributeName: [NSColor labelColor],
        };
        [base drawAtPoint:NSMakePoint(0, 2) withAttributes:baseAttrs];

        // draw badge in top-right corner
        NSString *label = [NSString stringWithFormat:@"%d", count > 9 ? 9 : count];
        NSDictionary *attrs = @{
            NSFontAttributeName: [NSFont boldSystemFontOfSize:8],
            NSForegroundColorAttributeName: [NSColor whiteColor],
        };
        NSSize textSize = [label sizeWithAttributes:attrs];
        CGFloat badgeW = MAX(textSize.width + 4, 11);
        NSRect badgeRect = NSMakeRect(size.width - badgeW, size.height - 11, badgeW, 11);

        [[NSColor colorWithRed:0.94 green:0.27 blue:0.27 alpha:1.0] setFill];
        NSBezierPath *path = [NSBezierPath bezierPathWithRoundedRect:badgeRect xRadius:3 yRadius:3];
        [path fill];

        NSPoint textPoint = NSMakePoint(
            badgeRect.origin.x + (badgeW - textSize.width) / 2,
            badgeRect.origin.y + (11 - textSize.height) / 2
        );
        [label drawAtPoint:textPoint withAttributes:attrs];

        [img unlockFocus];
        [img setTemplate:NO];
        statusItem.button.image = img;
        statusItem.button.title = @"";
    });
}
```

- [ ] **Step 2: Add CGo declaration and Go wrapper to tray_darwin.go**

In `tray/tray_darwin.go`, add `void traySetBadge(int count);` to the CGo preamble and add the Go wrapper:

The CGo preamble (the `/* ... */` block before `import "C"`) should become:
```go
/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdlib.h>

void tray_init(const char* title, const char* tooltip);
void tray_set_title(const char* title);
void tray_remove(void);
void tray_show_popup(void);
void tray_hide_popup(void);
void traySetBadge(int count);

// callbacks implemented in Go
extern void trayOnClick();
extern void trayOnQuit();
*/
```

Add this function after `HidePopup()`:
```go
func SetBadge(count int) {
	C.traySetBadge(C.int(count))
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /Users/lihachev/Projects/vigil && go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add tray/tray_darwin.m tray/tray_darwin.go
git commit -m "feat: add traySetBadge for tray icon badge rendering"
```

---

## Task 4: Settings + App Wiring

### Files:
- Modify: `app.go`

- [ ] **Step 1: Rewrite app.go with all new features**

Replace the contents of `app.go`:

```go
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
	NotifyConfirm bool `json:"notifyConfirm"`
	NotifyWaiting bool `json:"notifyWaiting"`
	BadgeConfirm  bool `json:"badgeConfirm"`
	BadgeWaiting  bool `json:"badgeWaiting"`
	BadgeActive   bool `json:"badgeActive"`
}

type App struct {
	ctx          context.Context
	manager      *monitor.Manager
	history      *monitor.HistoryScanner
	notifier     *monitor.Notifier
	stop         chan struct{}
	visible      bool
	prevSessions []monitor.Session
	settingsMu   sync.Mutex
	settings     Settings
	settingsPath string
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
		stop:         make(chan struct{}),
		settingsPath: filepath.Join(vigilDir, "settings.json"),
	}
	app.loadSettings()
	return app
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

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.pollLoop()
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
	a.notifier.Check(a.prevSessions, sessions)

	a.settingsMu.Lock()
	badge := monitor.CountBadge(sessions, a.settings.BadgeConfirm, a.settings.BadgeWaiting, a.settings.BadgeActive)
	a.settingsMu.Unlock()
	tray.SetBadge(badge)

	a.prevSessions = sessions
	runtime.EventsEmit(a.ctx, "sessions:updated", sessions)
}

func (a *App) GetSessions() []monitor.Session {
	return a.manager.Collect()
}

func (a *App) OpenSession(source string, cwd string) {
	a.HideWindow()
	switcher.ActivateSession(source, cwd)
}

// GetHistory returns historical sessions grouped by project, excluding active CWDs.
func (a *App) GetHistory() []monitor.ProjectHistory {
	active := a.manager.Collect()
	cwds := make([]string, 0, len(active))
	for _, s := range active {
		cwds = append(cwds, s.CWD)
	}
	return a.history.ScanHistory(cwds)
}

// ResumeSession opens a new Terminal.app window at cwd and runs claude --resume <sessionID>.
// If an IDE is detected for that cwd, it is also brought to the front (best-effort).
func (a *App) ResumeSession(cwd string, sessionID string) {
	escapedCWD := strings.ReplaceAll(cwd, `"`, `\"`)
	escapedID := strings.ReplaceAll(sessionID, `"`, `\"`)
	script := fmt.Sprintf(
		`tell application "Terminal"
	do script "cd \"%s\" && claude --resume \"%s\""
	activate
end tell`, escapedCWD, escapedID)
	exec.Command("osascript", "-e", script).Run()

	// best-effort: also raise IDE window if open for this project
	source := a.manager.GetIDESource(cwd)
	if source != "" && source != "Terminal" {
		switcher.ActivateSession(source, cwd)
	}
}

// GetSettings returns the current settings for the frontend.
func (a *App) GetSettings() Settings {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.settings
}

// UpdateSettings saves the settings and applies them immediately.
func (a *App) UpdateSettings(s Settings) {
	a.settingsMu.Lock()
	a.settings = s
	data, _ := json.MarshalIndent(a.settings, "", "  ")
	path := a.settingsPath
	a.settingsMu.Unlock()

	os.WriteFile(path, data, 0644)
	a.applySettings()
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
```

Note: `ResumeSession` calls `a.manager.GetIDESource(cwd)` — add this method in the next step.

- [ ] **Step 2: Add GetIDESource to monitor/manager.go**

Add at the end of `monitor/manager.go`:
```go
// GetIDESource returns the IDE source (e.g. "VSCode", "Cursor") for a given CWD,
// or empty string if not found. Uses already-loaded IDE data — does NOT call Load()
// to avoid racing with the concurrent Collect() call that also calls Load().
func (m *Manager) GetIDESource(cwd string) string {
	src := m.ide.GetSource(cwd)
	if src == "Terminal" {
		return ""
	}
	return src
}
```

- [ ] **Step 3: Build to verify compilation**

```bash
cd /Users/lihachev/Projects/vigil && go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add app.go monitor/manager.go
git commit -m "feat: wire Notifier, configurable badge, Settings, GetHistory, ResumeSession into App"
```

---

## Task 5: Frontend Types

### Files:
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Add HistoricalSession and ProjectHistory to types.ts**

Append to `frontend/src/types.ts`:
```typescript
export interface HistoricalSession {
  sessionId: string;
  name: string;
  lastActiveAt: number;
  tokensIn: string;
  tokensOut: string;
}

export interface ProjectHistory {
  projectName: string;
  cwd: string;
  sessions: HistoricalSession[];
}

export interface Settings {
  notifyConfirm: boolean;
  notifyWaiting: boolean;
  badgeConfirm: boolean;
  badgeWaiting: boolean;
  badgeActive: boolean;
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add frontend/src/types.ts
git commit -m "feat: add HistoricalSession and ProjectHistory frontend types"
```

---

## Task 6: History List Component

### Files:
- Create: `frontend/src/history-list.ts`

- [ ] **Step 1: Create history-list.ts**

Create `frontend/src/history-list.ts`:
```typescript
import { LitElement, html, css } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type { ProjectHistory, HistoricalSession } from './types';

@customElement('history-list')
export class HistoryList extends LitElement {
  @state() private groups: ProjectHistory[] = [];
  @state() private loading = true;
  @state() private expanded = new Set<string>();

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      flex: 1;
      overflow-y: auto;
    }
    :host::-webkit-scrollbar { width: 6px; }
    :host::-webkit-scrollbar-thumb {
      background: rgba(128,128,128,0.3);
      border-radius: 3px;
    }
    .empty {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: #8b949e;
      font-size: 13px;
    }
    .group {
      border-bottom: 1px solid rgba(255,255,255,0.06);
    }
    .group-header {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      cursor: pointer;
      transition: background 0.15s;
    }
    .group-header:hover { background: rgba(255,255,255,0.04); }
    .chevron {
      font-size: 10px;
      color: #6e7681;
      width: 10px;
      flex-shrink: 0;
    }
    .group-name {
      font-size: 12px;
      font-weight: 600;
      color: #e6edf3;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .group-path {
      font-size: 10px;
      color: #6e7681;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      max-width: 140px;
    }
    .sessions { }
    .session-row {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 5px 12px 5px 28px;
      cursor: pointer;
      transition: background 0.15s;
    }
    .session-row:hover { background: rgba(255,255,255,0.04); }
    .tree-char {
      color: #30363d;
      font-family: monospace;
      font-size: 11px;
      width: 10px;
      flex-shrink: 0;
    }
    .session-name {
      font-size: 11px;
      color: #8b949e;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-style: italic;
    }
    .session-age {
      font-size: 10px;
      color: #6e7681;
      white-space: nowrap;
      flex-shrink: 0;
    }
    .session-tokens {
      display: flex;
      gap: 3px;
      flex-shrink: 0;
    }
    .tok-in  { font-size: 10px; color: #6e8cff; }
    .tok-out { font-size: 10px; color: #3fb950; }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    window.go?.main?.App?.GetHistory().then((groups: ProjectHistory[]) => {
      this.groups = groups || [];
      // expand groups with only 1 session by default
      this.groups.forEach(g => {
        if (g.sessions.length === 1) this.expanded.add(g.cwd);
      });
      this.loading = false;
    });
  }

  private _toggle(cwd: string) {
    const next = new Set(this.expanded);
    if (next.has(cwd)) next.delete(cwd);
    else next.add(cwd);
    this.expanded = next;
  }

  private _resume(cwd: string, sessionId: string) {
    // @ts-ignore
    window.go?.main?.App?.ResumeSession(cwd, sessionId);
  }

  private _age(ms: number): string {
    const diff = Date.now() - ms;
    const mins = Math.floor(diff / 60_000);
    if (mins < 60) return `${mins}m`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h`;
    return `${Math.floor(hrs / 24)}d`;
  }

  private _shortPath(p: string): string {
    return p.replace(/^\/Users\/[^/]+/, '~');
  }

  render() {
    if (this.loading) return html`<div class="empty">Loading…</div>`;
    if (!this.groups.length) return html`<div class="empty">No history</div>`;

    return html`${this.groups.map(g => {
      const open = this.expanded.has(g.cwd);
      return html`
        <div class="group">
          <div class="group-header" @click=${() => this._toggle(g.cwd)}>
            <span class="chevron">${open ? '▼' : '▶'}</span>
            <span class="group-name">${g.projectName}</span>
            <span class="group-path">${this._shortPath(g.cwd)}</span>
          </div>
          ${open ? html`
            <div class="sessions">
              ${g.sessions.map((s: HistoricalSession, i: number) => {
                const isLast = i === g.sessions.length - 1;
                return html`
                  <div class="session-row" @click=${() => this._resume(g.cwd, s.sessionId)}>
                    <span class="tree-char">${isLast ? '└' : '├'}</span>
                    <span class="session-name">${s.name || s.sessionId}</span>
                    <span class="session-age">${this._age(s.lastActiveAt)}</span>
                    ${s.tokensIn || s.tokensOut ? html`
                      <div class="session-tokens">
                        ${s.tokensIn  ? html`<span class="tok-in">↑${s.tokensIn}</span>`  : ''}
                        ${s.tokensOut ? html`<span class="tok-out">↓${s.tokensOut}</span>` : ''}
                      </div>
                    ` : ''}
                  </div>
                `;
              })}
            </div>
          ` : ''}
        </div>
      `;
    })}`;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add frontend/src/history-list.ts
git commit -m "feat: add history-list Lit component"
```

---

## Task 7: Tabs in session-list.ts

### Files:
- Modify: `frontend/src/session-list.ts`

- [ ] **Step 1: Add tab bar and History tab to session-list.ts**

Replace the contents of `frontend/src/session-list.ts`:
```typescript
import { LitElement, html, css } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import type { Session } from './types';
import './session-card';
import './status-bar';
import './history-list';

const STATUS_ORDER: Record<string, number> = { confirm: 0, active: 1, waiting: 2, idle: 3 };

type Tab = 'active' | 'history';

@customElement('session-list')
export class SessionList extends LitElement {
  @state() private sessions: Session[] = [];
  @state() private tab: Tab = 'active';

  static styles = css`
    :host {
      display: flex;
      flex-direction: column;
      height: 100vh;
      overflow: hidden;
    }
    .tabs {
      display: flex;
      border-bottom: 1px solid rgba(255,255,255,0.08);
      flex-shrink: 0;
    }
    .tab {
      flex: 1;
      padding: 7px 0;
      text-align: center;
      font-size: 12px;
      color: #8b949e;
      cursor: pointer;
      transition: color 0.15s, border-bottom 0.15s;
      border-bottom: 2px solid transparent;
      user-select: none;
    }
    .tab.active-tab {
      color: #e6edf3;
      border-bottom: 2px solid #58a6ff;
    }
    .tab:hover:not(.active-tab) { color: #c9d1d9; }
    .list {
      flex: 1;
      overflow-y: auto;
    }
    .list::-webkit-scrollbar { width: 6px; }
    .list::-webkit-scrollbar-thumb {
      background: rgba(128,128,128,0.3);
      border-radius: 3px;
    }
    .empty {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: var(--text-secondary, #8b949e);
      font-size: 13px;
    }
    history-list {
      flex: 1;
      overflow: hidden;
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    if (window.go?.main?.App?.GetSessions) {
      // @ts-ignore
      window.go.main.App.GetSessions().then((s: Session[]) => {
        this.sessions = s || [];
      });
    }
    // @ts-ignore
    if (window.runtime?.EventsOn) {
      // @ts-ignore
      window.runtime.EventsOn('sessions:updated', (sessions: Session[]) => {
        this.sessions = sessions || [];
      });
    }
  }

  private _activeLabel() {
    const count = this.sessions.filter(s => s.status !== 'idle').length;
    return count > 0 ? `Active (${count})` : 'Active';
  }

  render() {
    const sorted = [...this.sessions].sort((a, b) => {
      const d = (STATUS_ORDER[a.status] ?? 9) - (STATUS_ORDER[b.status] ?? 9);
      if (d !== 0) return d;
      return b.startedAt - a.startedAt;
    });

    return html`
      <div class="tabs">
        <div class="tab ${this.tab === 'active' ? 'active-tab' : ''}"
             @click=${() => { this.tab = 'active'; }}>
          ${this._activeLabel()}
        </div>
        <div class="tab ${this.tab === 'history' ? 'active-tab' : ''}"
             @click=${() => { this.tab = 'history'; }}>
          History
        </div>
      </div>
      ${this.tab === 'active' ? html`
        <div class="list">
          ${sorted.length === 0
            ? html`<div class="empty">No active sessions</div>`
            : sorted.map(s => html`<session-card .session=${s}></session-card>`)}
        </div>
        <status-bar .count=${sorted.length}></status-bar>
      ` : html`
        <history-list></history-list>
      `}
    `;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add frontend/src/session-list.ts
git commit -m "feat: add Active/History tab bar to session-list"
```

---

## Task 8: Settings Toggle in Status Bar

### Files:
- Modify: `frontend/src/status-bar.ts`

- [ ] **Step 1: Rewrite status-bar.ts with gear icon and notifications toggle**

Replace the contents of `frontend/src/status-bar.ts`:
```typescript
import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
import type { Settings } from './types';

@customElement('status-bar')
export class StatusBar extends LitElement {
  @property({ type: Number }) count = 0;
  @state() private showSettings = false;
  @state() private settings: Settings = {
    notifyConfirm: true, notifyWaiting: false,
    badgeConfirm: true, badgeWaiting: true, badgeActive: false,
  };

  static styles = css`
    :host {
      display: block;
      border-top: 1px solid var(--border, rgba(255,255,255,0.08));
      flex-shrink: 0;
    }
    .bar {
      display: flex;
      align-items: center;
      padding: 8px 12px;
      font-size: 12px;
      color: var(--text-secondary, #8b949e);
    }
    .bar-label { flex: 1; }
    .gear {
      cursor: pointer;
      font-size: 13px;
      color: #6e7681;
      padding: 2px 4px;
      border-radius: 4px;
      transition: color 0.15s, background 0.15s;
      user-select: none;
    }
    .gear:hover { color: #c9d1d9; background: rgba(255,255,255,0.06); }
    .settings-panel {
      border-top: 1px solid rgba(255,255,255,0.06);
      padding: 10px 14px;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .section-label {
      font-size: 10px;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: #6e7681;
    }
    .setting-row {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: #8b949e;
      cursor: pointer;
      padding-left: 2px;
    }
    .setting-row:hover { color: #c9d1d9; }
    input[type="checkbox"] { cursor: pointer; accent-color: #58a6ff; }
  `;

  connectedCallback() {
    super.connectedCallback();
    // @ts-ignore
    window.go?.main?.App?.GetSettings().then((s: Settings) => {
      if (s) this.settings = s;
    });
  }

  private _toggleSettings() {
    this.showSettings = !this.showSettings;
  }

  private _update(key: keyof Settings, e: Event) {
    const checked = (e.target as HTMLInputElement).checked;
    this.settings = { ...this.settings, [key]: checked };
    // @ts-ignore
    window.go?.main?.App?.UpdateSettings(this.settings);
  }

  render() {
    const label = this.count === 1 ? 'session' : 'sessions';
    const s = this.settings;
    return html`
      <div class="bar">
        <span class="bar-label">${this.count} active ${label}</span>
        <span class="gear" @click=${this._toggleSettings}>⚙</span>
      </div>
      ${this.showSettings ? html`
        <div class="settings-panel">
          <span class="section-label">Notifications</span>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.notifyConfirm}
                   @change=${(e: Event) => this._update('notifyConfirm', e)} />
            Needs confirmation
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.notifyWaiting}
                   @change=${(e: Event) => this._update('notifyWaiting', e)} />
            Waiting for input
          </label>
          <span class="section-label">Badge</span>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeConfirm}
                   @change=${(e: Event) => this._update('badgeConfirm', e)} />
            Needs confirmation
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeWaiting}
                   @change=${(e: Event) => this._update('badgeWaiting', e)} />
            Waiting for input
          </label>
          <label class="setting-row">
            <input type="checkbox" .checked=${s.badgeActive}
                   @change=${(e: Event) => this._update('badgeActive', e)} />
            Active sessions
          </label>
        </div>
      ` : ''}
    `;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/lihachev/Projects/vigil && git add frontend/src/status-bar.ts
git commit -m "feat: add gear settings toggle with notifications checkbox to status-bar"
```

---

## Task 9: Final Integration Verification

- [ ] **Step 1: Run all Go tests**

```bash
cd /Users/lihachev/Projects/vigil && go test ./...
```
Expected: all pass, 0 failures.

- [ ] **Step 2: Build the full app**

```bash
cd /Users/lihachev/Projects/vigil && wails build -platform darwin/universal -clean 2>&1 | tail -5
```
Expected: `Build successful` with no errors.

- [ ] **Step 3: Manual smoke test checklist**

Launch with `wails dev` and verify:
- [ ] Tray icon shows a red badge number when any session is in `confirm` or `waiting` state
- [ ] Badge disappears when all sessions are `active` or `idle`
- [ ] A macOS notification appears when a session enters `confirm` (test by observing the notification center)
- [ ] Gear icon appears in status bar; clicking it shows/hides settings panel
- [ ] Settings panel shows two sections: Notifications (2 checkboxes) and Badge (3 checkboxes)
- [ ] Unchecking "Needs confirmation" under Notifications → no confirm notifications fire
- [ ] Checking "Waiting for input" under Notifications → waiting notifications fire
- [ ] Unchecking badge checkboxes → badge count changes accordingly
- [ ] Settings persist after app restart (`~/.vigil/settings.json`)
- [ ] Switching to History tab shows past sessions grouped by project
- [ ] Collapsing and expanding groups works
- [ ] Clicking a history session opens Terminal.app at the correct path with `claude --resume <id>`

- [ ] **Step 4: Final commit**

```bash
cd /Users/lihachev/Projects/vigil && git add -A && git status
# confirm only expected files are staged, then:
git commit -m "feat: complete badge, notifications, and history features"
```
