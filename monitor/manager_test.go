package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_CollectSessions(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sessions")
	ideDir := filepath.Join(dir, "ide")
	os.MkdirAll(sessDir, 0755)
	os.MkdirAll(ideDir, 0755)

	pid := os.Getpid()
	sessJSON := fmt.Sprintf(`{"pid":%d,"sessionId":"test-sess","cwd":"/tmp/myproject","startedAt":%d}`, pid, time.Now().UnixMilli()-60000)
	os.WriteFile(filepath.Join(sessDir, fmt.Sprintf("%d.json", pid)), []byte(sessJSON), 0644)

	// create JSONL in projects dir (new format)
	projDir := filepath.Join(dir, "projects", "-tmp-myproject")
	os.MkdirAll(projDir, 0755)
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	jsonl := fmt.Sprintf(`{"type":"assistant","timestamp":"%s","sessionId":"test-sess","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","id":"t1","input":{}}]}}`, now)
	os.WriteFile(filepath.Join(projDir, "test-sess.jsonl"), []byte(jsonl+"\n"), 0644)

	mgr := NewManager(dir)
	sessions := mgr.Collect()

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.ProjectName != "myproject" {
		t.Errorf("expected myproject, got %s", s.ProjectName)
	}
	if s.Source != "Terminal" {
		t.Errorf("expected Terminal, got %s", s.Source)
	}
	if s.Status != StatusConfirm {
		t.Errorf("expected confirm, got %s", s.Status)
	}
	// 1 tool_use + 1 confirm (no user response follows)
	if len(s.RecentActions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(s.RecentActions))
	}
	if s.Duration == "" {
		t.Error("expected non-empty duration")
	}
}

func TestManager_SiblingSessionsGetDifferentJSONLs(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sessions")
	ideDir := filepath.Join(dir, "ide")
	os.MkdirAll(sessDir, 0755)
	os.MkdirAll(ideDir, 0755)

	pid := os.Getpid()
	now := time.Now()

	// two sessions sharing the same CWD, different startedAt
	sess1 := fmt.Sprintf(`{"pid":%d,"sessionId":"sess-aaa","cwd":"/tmp/shared","startedAt":%d}`, pid, now.UnixMilli()-120000)
	sess2 := fmt.Sprintf(`{"pid":%d,"sessionId":"sess-bbb","cwd":"/tmp/shared","startedAt":%d}`, pid, now.UnixMilli()-60000)
	os.WriteFile(filepath.Join(sessDir, fmt.Sprintf("%d.json", pid)), []byte(sess1), 0644)
	os.WriteFile(filepath.Join(sessDir, fmt.Sprintf("%d-2.json", pid)), []byte(sess2), 0644)

	projDir := filepath.Join(dir, "projects", "-tmp-shared")
	os.MkdirAll(projDir, 0755)

	ts := now.UTC().Format("2006-01-02T15:04:05.000Z")
	jsonl1 := fmt.Sprintf(`{"type":"assistant","timestamp":"%s","sessionId":"sess-aaa","slug":"older-session","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`, ts) + "\n"
	jsonl2 := fmt.Sprintf(`{"type":"assistant","timestamp":"%s","sessionId":"sess-bbb","slug":"newer-session","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`, ts) + "\n"

	// older JSONL
	f1 := filepath.Join(projDir, "sess-aaa.jsonl")
	os.WriteFile(f1, []byte(jsonl1), 0644)
	os.Chtimes(f1, now.Add(-2*time.Hour), now.Add(-2*time.Hour))

	// newer JSONL
	f2 := filepath.Join(projDir, "sess-bbb.jsonl")
	os.WriteFile(f2, []byte(jsonl2), 0644)

	mgr := NewManager(dir)
	sessions := mgr.Collect()

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	names := map[string]bool{}
	for _, s := range sessions {
		names[s.Name] = true
		if s.Sibling == "" {
			t.Errorf("expected sibling indicator for session %s", s.SessionID)
		}
	}
	if !names["older-session"] || !names["newer-session"] {
		t.Errorf("expected both session names, got %v", names)
	}
}

func TestManager_FilterDeadPIDs(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessDir, 0755)

	os.WriteFile(
		filepath.Join(sessDir, "9999999.json"),
		[]byte(`{"pid":9999999,"sessionId":"dead","cwd":"/tmp/dead","startedAt":1700000000000}`),
		0644,
	)

	mgr := NewManager(dir)
	sessions := mgr.Collect()

	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions (dead PID), got %d", len(sessions))
	}
}
