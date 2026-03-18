package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanSessions_ReadsSessionFiles(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessDir, 0755)
	os.WriteFile(
		filepath.Join(sessDir, "12345.json"),
		[]byte(`{"pid":12345,"sessionId":"aaa-bbb","cwd":"/tmp/test","startedAt":1700000000000}`),
		0644,
	)

	scanner := NewScanner(dir)
	sessions, err := scanner.ScanSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].SessionID != "aaa-bbb" {
		t.Errorf("expected sessionId aaa-bbb, got %s", sessions[0].SessionID)
	}
	if sessions[0].ProjectName != "test" {
		t.Errorf("expected projectName test, got %s", sessions[0].ProjectName)
	}
}

func TestScanSessions_SkipsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	sessDir := filepath.Join(dir, "sessions")
	os.MkdirAll(sessDir, 0755)
	os.WriteFile(filepath.Join(sessDir, "bad.json"), []byte(`not json`), 0644)

	scanner := NewScanner(dir)
	sessions, err := scanner.ScanSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	if !IsProcessAlive(os.Getpid()) {
		t.Error("current process should be alive")
	}
}

func TestIsProcessAlive_DeadProcess(t *testing.T) {
	if IsProcessAlive(9999999) {
		t.Error("PID 9999999 should not be alive")
	}
}
