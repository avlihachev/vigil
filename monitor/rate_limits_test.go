package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRateLimitReader_NativeFile(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "claude", "cache")
	os.MkdirAll(nativeDir, 0755)
	vigilDir := filepath.Join(dir, "vigil")
	os.MkdirAll(vigilDir, 0755)

	data := map[string]interface{}{
		"five_hour":  map[string]interface{}{"used_percentage": 42, "resets_at": 1774026000},
		"seven_day":  map[string]interface{}{"used_percentage": 18, "resets_at": 1774605600},
		"updated_at": "2026-03-20T12:30:00Z",
	}
	b, _ := json.Marshal(data)
	os.WriteFile(filepath.Join(nativeDir, "rate-limits.json"), b, 0644)

	reader := NewRateLimitReader(filepath.Join(dir, "claude"), vigilDir)
	rl := reader.Read()
	if rl == nil {
		t.Fatal("expected rate limits, got nil")
	}
	if !rl.Available {
		t.Error("expected Available=true")
	}
	if rl.FiveHour == nil || rl.FiveHour.UsedPercentage != 42 {
		t.Errorf("expected 5h=42%%, got %+v", rl.FiveHour)
	}
	if rl.SevenDay == nil || rl.SevenDay.UsedPercentage != 18 {
		t.Errorf("expected 7d=18%%, got %+v", rl.SevenDay)
	}
}

func TestRateLimitReader_BridgeFallback(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, "claude")
	os.MkdirAll(filepath.Join(claudeDir, "cache"), 0755)
	vigilDir := filepath.Join(dir, "vigil")
	os.MkdirAll(vigilDir, 0755)

	data := map[string]interface{}{
		"five_hour":  map[string]interface{}{"used_percentage": 60, "resets_at": 1774026000},
		"updated_at": "2026-03-20T12:30:00Z",
	}
	b, _ := json.Marshal(data)
	os.WriteFile(filepath.Join(vigilDir, "rate-limits.json"), b, 0644)

	reader := NewRateLimitReader(claudeDir, vigilDir)
	rl := reader.Read()
	if rl == nil {
		t.Fatal("expected rate limits from bridge, got nil")
	}
	if rl.FiveHour == nil || rl.FiveHour.UsedPercentage != 60 {
		t.Errorf("expected 5h=60%%, got %+v", rl.FiveHour)
	}
}

func TestRateLimitReader_OldFile(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, "claude")
	nativeDir := filepath.Join(claudeDir, "cache")
	os.MkdirAll(nativeDir, 0755)
	vigilDir := filepath.Join(dir, "vigil")
	os.MkdirAll(vigilDir, 0755)

	path := filepath.Join(nativeDir, "rate-limits.json")
	data := map[string]interface{}{
		"five_hour":  map[string]interface{}{"used_percentage": 42, "resets_at": 1774026000},
		"updated_at": "2026-03-20T12:30:00Z",
	}
	b, _ := json.Marshal(data)
	os.WriteFile(path, b, 0644)

	old := time.Now().Add(-10 * time.Minute)
	os.Chtimes(path, old, old)

	reader := NewRateLimitReader(claudeDir, vigilDir)
	rl := reader.Read()
	if rl == nil {
		t.Fatal("expected rate limits even from old file")
	}
	if rl.FiveHour == nil || rl.FiveHour.UsedPercentage != 42 {
		t.Errorf("expected 5h=42%%, got %+v", rl.FiveHour)
	}
}

func TestRateLimitReader_MissingFile(t *testing.T) {
	dir := t.TempDir()
	reader := NewRateLimitReader(filepath.Join(dir, "claude"), filepath.Join(dir, "vigil"))
	rl := reader.Read()
	if rl != nil {
		t.Error("expected nil when no files exist")
	}
}

func TestRateLimitReader_Throttle(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "claude", "cache")
	os.MkdirAll(nativeDir, 0755)
	vigilDir := filepath.Join(dir, "vigil")
	os.MkdirAll(vigilDir, 0755)

	path := filepath.Join(nativeDir, "rate-limits.json")
	data := map[string]interface{}{
		"five_hour":  map[string]interface{}{"used_percentage": 42, "resets_at": 1774026000},
		"updated_at": "2026-03-20T12:30:00Z",
	}
	b, _ := json.Marshal(data)
	os.WriteFile(path, b, 0644)

	reader := NewRateLimitReader(filepath.Join(dir, "claude"), vigilDir)

	rl1 := reader.Read()
	if rl1 == nil {
		t.Fatal("first read should return data")
	}

	// update file with different data
	data["five_hour"] = map[string]interface{}{"used_percentage": 99, "resets_at": 1774026000}
	b, _ = json.Marshal(data)
	os.WriteFile(path, b, 0644)

	// second read within 10s should return cached
	rl2 := reader.Read()
	if rl2.FiveHour.UsedPercentage != 42 {
		t.Error("expected cached value within throttle window")
	}
}
