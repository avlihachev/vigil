package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type IDEDetector struct {
	baseDir  string
	pidToIDE map[int]string
}

type rawIDELock struct {
	PID     int    `json:"pid"`
	IDEName string `json:"ideName"`
}

func NewIDEDetector(baseDir string) *IDEDetector {
	return &IDEDetector{
		baseDir:  baseDir,
		pidToIDE: make(map[int]string),
	}
}

func (d *IDEDetector) Load() error {
	d.pidToIDE = make(map[int]string)
	pattern := filepath.Join(d.baseDir, "ide", "*.lock")
	files, _ := filepath.Glob(pattern)

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var raw rawIDELock
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		name := normalizeIDEName(raw.IDEName)
		if raw.PID != 0 && name != "" {
			d.pidToIDE[raw.PID] = name
		}
	}
	return nil
}

func (d *IDEDetector) GetSource(pid int) string {
	if name, ok := d.pidToIDE[pid]; ok {
		return name
	}
	return "Terminal"
}

func normalizeIDEName(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "visual studio code") || strings.Contains(lower, "vscode") {
		return "VSCode"
	}
	if strings.Contains(lower, "cursor") {
		return "Cursor"
	}
	if name != "" {
		return name
	}
	return ""
}
