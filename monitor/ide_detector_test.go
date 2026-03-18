package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIDEDetector_MatchesSessionToIDE(t *testing.T) {
	dir := t.TempDir()
	ideDir := filepath.Join(dir, "ide")
	os.MkdirAll(ideDir, 0755)
	os.WriteFile(
		filepath.Join(ideDir, "12345.lock"),
		[]byte(`{"pid":12345,"workspaceFolders":["/tmp/proj"],"ideName":"Visual Studio Code","transport":"ws"}`),
		0644,
	)

	detector := NewIDEDetector(dir)
	err := detector.Load()
	if err != nil {
		t.Fatal(err)
	}

	source := detector.GetSource(12345)
	if source != "VSCode" {
		t.Errorf("expected VSCode, got %s", source)
	}
}

func TestIDEDetector_ReturnsTerminalForUnknownPID(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "ide"), 0755)

	detector := NewIDEDetector(dir)
	detector.Load()

	source := detector.GetSource(99999)
	if source != "Terminal" {
		t.Errorf("expected Terminal, got %s", source)
	}
}
