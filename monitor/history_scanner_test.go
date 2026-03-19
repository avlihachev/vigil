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
