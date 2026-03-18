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
	if s.Status != StatusActive {
		t.Errorf("expected active, got %s", s.Status)
	}
	if len(s.RecentActions) != 1 {
		t.Errorf("expected 1 action, got %d", len(s.RecentActions))
	}
	if s.Duration == "" {
		t.Error("expected non-empty duration")
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
