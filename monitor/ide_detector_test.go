package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIDEDetector_MatchesByCWD(t *testing.T) {
	dir := t.TempDir()
	ideDir := filepath.Join(dir, "ide")
	os.MkdirAll(ideDir, 0755)
	os.WriteFile(
		filepath.Join(ideDir, "12345.lock"),
		[]byte(`{"pid":99,"workspaceFolders":["/tmp/proj"],"ideName":"Visual Studio Code","transport":"ws"}`),
		0644,
	)

	detector := NewIDEDetector(dir)
	detector.Load()

	if s := detector.GetSource("/tmp/proj"); s != "VSCode" {
		t.Errorf("expected VSCode for exact match, got %s", s)
	}
}

func TestIDEDetector_MatchesSubdirectory(t *testing.T) {
	dir := t.TempDir()
	ideDir := filepath.Join(dir, "ide")
	os.MkdirAll(ideDir, 0755)
	os.WriteFile(
		filepath.Join(ideDir, "12345.lock"),
		[]byte(`{"pid":99,"workspaceFolders":["/Users/test/project"],"ideName":"Visual Studio Code"}`),
		0644,
	)

	detector := NewIDEDetector(dir)
	detector.Load()

	if s := detector.GetSource("/Users/test/project/services/api"); s != "VSCode" {
		t.Errorf("expected VSCode for subdirectory, got %s", s)
	}
}

func TestIDEDetector_ReturnsTerminalForUnknownCWD(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "ide"), 0755)

	detector := NewIDEDetector(dir)
	detector.Load()

	if s := detector.GetSource("/tmp/unknown"); s != "Terminal" {
		t.Errorf("expected Terminal, got %s", s)
	}
}
