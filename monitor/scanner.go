package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
)

type Scanner struct {
	baseDir string
}

func NewScanner(baseDir string) *Scanner {
	return &Scanner{baseDir: baseDir}
}

type rawSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
}

func (s *Scanner) ScanSessions() ([]Session, error) {
	pattern := filepath.Join(s.baseDir, "sessions", "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var raw rawSession
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if raw.PID == 0 || raw.SessionID == "" {
			continue
		}

		sess := Session{
			PID:         raw.PID,
			SessionID:   raw.SessionID,
			CWD:         raw.CWD,
			StartedAt:   raw.StartedAt,
			ProjectName: filepath.Base(raw.CWD),
			Source:      "Terminal",
			Status:      StatusIdle,
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func IsProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
