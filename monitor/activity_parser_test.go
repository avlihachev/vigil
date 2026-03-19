package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActivityParser_ParsesToolUseFromJSONL(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-tmp-myproject")
	os.MkdirAll(projDir, 0755)

	jsonl := `{"type":"assistant","timestamp":"2026-03-18T10:00:02.000Z","sessionId":"sess-1","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","id":"t1","input":{}}]}}
{"type":"assistant","timestamp":"2026-03-18T10:00:03.000Z","sessionId":"sess-1","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","id":"t2","input":{}}]}}
{"type":"assistant","timestamp":"2026-03-18T10:00:04.000Z","sessionId":"sess-1","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","id":"t3","input":{}}]}}
{"type":"assistant","timestamp":"2026-03-18T10:00:05.000Z","sessionId":"sess-1","message":{"role":"assistant","content":[{"type":"tool_use","name":"Grep","id":"t4","input":{}}]}}
`
	os.WriteFile(filepath.Join(projDir, "sess-1.jsonl"), []byte(jsonl), 0644)

	parser := NewActivityParser(dir)
	actions, err := parser.Parse("sess-1", "/tmp/myproject")
	if err != nil {
		t.Fatal(err)
	}
	// 4 tool actions + 1 confirm (last assistant had tool_use with no result)
	if len(actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(actions))
	}
	if actions[0].Type != ActionRead {
		t.Errorf("expected Read, got %s", actions[0].Type)
	}
	if actions[1].Type != ActionEdit {
		t.Errorf("expected Edit, got %s", actions[1].Type)
	}
	if actions[2].Type != ActionBash {
		t.Errorf("expected Bash, got %s", actions[2].Type)
	}
	if actions[3].Type != ActionSearch {
		t.Errorf("expected Search (Grep), got %s", actions[3].Type)
	}
	if actions[4].Type != ActionConfirm {
		t.Errorf("expected Confirm, got %s", actions[4].Type)
	}
}

func TestActivityParser_LimitsTo5Actions(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-tmp-proj")
	os.MkdirAll(projDir, 0755)

	jsonl := ""
	for i := 0; i < 10; i++ {
		jsonl += `{"type":"assistant","timestamp":"2026-03-18T10:00:01.000Z","sessionId":"s","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","id":"t","input":{}}]}}` + "\n"
	}
	os.WriteFile(filepath.Join(projDir, "s.jsonl"), []byte(jsonl), 0644)

	parser := NewActivityParser(dir)
	actions, _ := parser.Parse("s", "/tmp/proj")
	if len(actions) != 5 {
		t.Fatalf("expected max 5, got %d", len(actions))
	}
}

func TestActivityParser_ReturnsEmptyForMissingFile(t *testing.T) {
	parser := NewActivityParser(t.TempDir())
	actions, err := parser.Parse("nonexistent", "/tmp/missing")
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
	if s := DetermineStatus(actions, now-10_000, now); s != StatusActive {
		t.Errorf("expected active, got %s", s)
	}
}

func TestDetermineStatus_Idle(t *testing.T) {
	now := time.Now().UnixMilli()
	actions := []Action{{Type: ActionEdit, Timestamp: now - 600_000}}
	// file modified long ago
	if s := DetermineStatus(actions, now-600_000, now); s != StatusIdle {
		t.Errorf("expected idle, got %s", s)
	}
}

func TestDetermineStatus_Waiting(t *testing.T) {
	now := time.Now().UnixMilli()
	actions := []Action{{Type: ActionWaiting, Timestamp: now - 5_000}}
	if s := DetermineStatus(actions, now-5_000, now); s != StatusWaiting {
		t.Errorf("expected waiting, got %s", s)
	}
}

func TestDetermineStatus_EmptyActions(t *testing.T) {
	now := time.Now().UnixMilli()
	if s := DetermineStatus(nil, 0, now); s != StatusIdle {
		t.Errorf("expected idle for empty, got %s", s)
	}
}

func TestParseAllFromPath(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-tmp-proj")
	os.MkdirAll(projDir, 0755)

	jsonl := `{"type":"assistant","timestamp":"2026-03-18T10:00:01.000Z","sessionId":"s1","slug":"my-session","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","id":"t1","input":{}}],"usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"user","sessionId":"s1","toolUseResult":"ok"}
{"type":"assistant","timestamp":"2026-03-18T10:00:02.000Z","sessionId":"s1","message":{"role":"assistant","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":200,"output_tokens":100}}}
`
	path := filepath.Join(projDir, "s1.jsonl")
	os.WriteFile(path, []byte(jsonl), 0644)

	parser := NewActivityParser(dir)
	sa := parser.ParseAllFromPath(path)

	if sa.Name != "my-session" {
		t.Errorf("expected name 'my-session', got %q", sa.Name)
	}
	if sa.TokensIn != 300 {
		t.Errorf("expected 300 input tokens, got %d", sa.TokensIn)
	}
	if sa.TokensOut != 150 {
		t.Errorf("expected 150 output tokens, got %d", sa.TokensOut)
	}
	if sa.FileModMs == 0 {
		t.Error("expected non-zero FileModMs")
	}
	// 1 Read action + 1 waiting (last assistant was text-only)
	if len(sa.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(sa.Actions))
	}
	if sa.Actions[0].Type != ActionRead {
		t.Errorf("expected Read, got %s", sa.Actions[0].Type)
	}
	if sa.Actions[1].Type != ActionWaiting {
		t.Errorf("expected Waiting, got %s", sa.Actions[1].Type)
	}
}

func TestListProjectJSONLs(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-tmp-proj")
	os.MkdirAll(projDir, 0755)

	// create files with different mod times
	f1 := filepath.Join(projDir, "older.jsonl")
	f2 := filepath.Join(projDir, "newer.jsonl")
	os.WriteFile(f1, []byte(`{"type":"assistant"}`+"\n"), 0644)
	os.Chtimes(f1, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
	os.WriteFile(f2, []byte(`{"type":"assistant"}`+"\n"), 0644)

	parser := NewActivityParser(dir)
	paths := parser.ListProjectJSONLs("/tmp/proj")

	if len(paths) != 2 {
		t.Fatalf("expected 2 jsonls, got %d", len(paths))
	}
	if filepath.Base(paths[0]) != "newer.jsonl" {
		t.Errorf("expected newest first, got %s", filepath.Base(paths[0]))
	}
	if filepath.Base(paths[1]) != "older.jsonl" {
		t.Errorf("expected oldest second, got %s", filepath.Base(paths[1]))
	}
}
