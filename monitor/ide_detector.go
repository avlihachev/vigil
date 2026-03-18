package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type IDEDetector struct {
	baseDir string
	// maps workspace folder path → IDE name
	cwdToIDE map[string]string
}

type rawIDELock struct {
	PID              int      `json:"pid"`
	IDEName          string   `json:"ideName"`
	WorkspaceFolders []string `json:"workspaceFolders"`
}

func NewIDEDetector(baseDir string) *IDEDetector {
	return &IDEDetector{
		baseDir:  baseDir,
		cwdToIDE: make(map[string]string),
	}
}

func (d *IDEDetector) Load() error {
	d.cwdToIDE = make(map[string]string)
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
		if name == "" {
			continue
		}
		for _, folder := range raw.WorkspaceFolders {
			d.cwdToIDE[folder] = name
		}
	}
	return nil
}

// GetSource matches session CWD against workspace folders
func (d *IDEDetector) GetSource(cwd string) string {
	if name, ok := d.cwdToIDE[cwd]; ok {
		return name
	}
	// check if cwd is a subdirectory of a workspace folder
	for folder, name := range d.cwdToIDE {
		if strings.HasPrefix(cwd, folder+"/") {
			return name
		}
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
