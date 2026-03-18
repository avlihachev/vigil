package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActivityParser_ParsesToolCalls(t *testing.T) {
	dir := t.TempDir()
	debugDir := filepath.Join(dir, "debug")
	os.MkdirAll(debugDir, 0755)

	logContent := "2026-03-18T10:00:01.000Z [DEBUG] Stream started - received first chunk\n" +
		"2026-03-18T10:00:02.000Z [DEBUG] executePreToolHooks called for tool: Read\n" +
		"2026-03-18T10:00:03.000Z [DEBUG] executePreToolHooks called for tool: Edit\n" +
		"2026-03-18T10:00:04.000Z [DEBUG] executePreToolHooks called for tool: Bash\n" +
		"2026-03-18T10:00:05.000Z [DEBUG] executePreToolHooks called for tool: Grep\n" +
		"2026-03-18T10:00:06.000Z [DEBUG] executePreToolHooks called for tool: Glob\n"
	os.WriteFile(filepath.Join(debugDir, "sess-123.txt"), []byte(logContent), 0644)

	parser := NewActivityParser(dir)
	actions, err := parser.Parse("sess-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(actions))
	}
	if actions[0].Type != ActionRead {
		t.Errorf("expected first action Read, got %s", actions[0].Type)
	}
	if actions[1].Type != ActionEdit {
		t.Errorf("expected second action Edit, got %s", actions[1].Type)
	}
	if actions[2].Type != ActionBash {
		t.Errorf("expected third action Bash, got %s", actions[2].Type)
	}
	if actions[3].Type != ActionSearch {
		t.Errorf("expected Grep mapped to search, got %s", actions[3].Type)
	}
	if actions[4].Type != ActionSearch {
		t.Errorf("expected Glob mapped to search, got %s", actions[4].Type)
	}
}

func TestActivityParser_LimitsTo5Actions(t *testing.T) {
	dir := t.TempDir()
	debugDir := filepath.Join(dir, "debug")
	os.MkdirAll(debugDir, 0755)

	logContent := ""
	for i := 0; i < 10; i++ {
		logContent += "2026-03-18T10:00:01.000Z [DEBUG] executePreToolHooks called for tool: Read\n"
	}
	os.WriteFile(filepath.Join(debugDir, "many.txt"), []byte(logContent), 0644)

	parser := NewActivityParser(dir)
	actions, _ := parser.Parse("many")
	if len(actions) != 5 {
		t.Fatalf("expected max 5 actions, got %d", len(actions))
	}
}

func TestActivityParser_DetectsIdlePrompt(t *testing.T) {
	dir := t.TempDir()
	debugDir := filepath.Join(dir, "debug")
	os.MkdirAll(debugDir, 0755)

	logContent := "2026-03-18T10:00:01.000Z [DEBUG] executePreToolHooks called for tool: Read\n" +
		"2026-03-18T10:00:02.000Z [DEBUG] Getting matching hook commands for Notification with query: idle_prompt\n"
	os.WriteFile(filepath.Join(debugDir, "sess-456.txt"), []byte(logContent), 0644)

	parser := NewActivityParser(dir)
	actions, _ := parser.Parse("sess-456")

	last := actions[len(actions)-1]
	if last.Type != ActionWaiting {
		t.Errorf("expected last action waiting, got %s", last.Type)
	}
}

func TestActivityParser_ReturnsEmptyForMissingFile(t *testing.T) {
	parser := NewActivityParser(t.TempDir())
	actions, err := parser.Parse("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}

func TestDetermineStatus_Active(t *testing.T) {
	now := time.Now().UnixMilli()
	actions := []Action{{Type: ActionEdit, Timestamp: now - 10_000}}
	if s := DetermineStatus(actions, now); s != StatusActive {
		t.Errorf("expected active, got %s", s)
	}
}

func TestDetermineStatus_Idle(t *testing.T) {
	now := time.Now().UnixMilli()
	actions := []Action{{Type: ActionEdit, Timestamp: now - 60_000}}
	if s := DetermineStatus(actions, now); s != StatusIdle {
		t.Errorf("expected idle, got %s", s)
	}
}

func TestDetermineStatus_Waiting(t *testing.T) {
	now := time.Now().UnixMilli()
	actions := []Action{{Type: ActionWaiting, Timestamp: now - 5_000}}
	if s := DetermineStatus(actions, now); s != StatusWaiting {
		t.Errorf("expected waiting, got %s", s)
	}
}

func TestDetermineStatus_EmptyActions(t *testing.T) {
	now := time.Now().UnixMilli()
	if s := DetermineStatus(nil, now); s != StatusIdle {
		t.Errorf("expected idle for empty actions, got %s", s)
	}
}
