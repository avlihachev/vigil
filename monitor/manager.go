package monitor

import "time"

type Manager struct {
	scanner  *Scanner
	ide      *IDEDetector
	activity *ActivityParser
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		scanner:  NewScanner(baseDir),
		ide:      NewIDEDetector(baseDir),
		activity: NewActivityParser(baseDir),
	}
}

func (m *Manager) Collect() []Session {
	raw, err := m.scanner.ScanSessions()
	if err != nil {
		return nil
	}
	m.ide.Load()

	now := time.Now().UnixMilli()
	var result []Session
	for _, s := range raw {
		if !IsProcessAlive(s.PID) {
			continue
		}
		s.Source = m.ide.GetSource(s.PID)
		actions, _ := m.activity.Parse(s.SessionID)
		s.RecentActions = actions
		s.Status = DetermineStatus(actions, now)
		s.Duration = FormatDuration(now - s.StartedAt)
		result = append(result, s)
	}
	return result
}
