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

var statusPriority = map[SessionStatus]int{
	StatusConfirm: 0,
	StatusActive:  1,
	StatusWaiting: 2,
	StatusIdle:    3,
}

func (m *Manager) Collect() []Session {
	raw, err := m.scanner.ScanSessions()
	if err != nil {
		return nil
	}
	m.ide.Load()

	now := time.Now().UnixMilli()
	var all []Session
	for _, s := range raw {
		if !IsProcessAlive(s.PID) {
			continue
		}
		s.Source = m.ide.GetSource(s.CWD)
		if s.Source == "Terminal" {
			s.Source = terminalForPID(s.PID)
		}
		actions, _ := m.activity.Parse(s.SessionID, s.CWD)
		s.RecentActions = actions
		s.Name = m.activity.ParseName(s.SessionID, s.CWD)
		tokIn, tokOut := m.activity.ParseTokens(s.SessionID, s.CWD)
		s.TokensIn = FormatTokens(tokIn)
		s.TokensOut = FormatTokens(tokOut)
		fileMod := m.activity.LastModifiedMs(s.SessionID, s.CWD)
		s.Status = DetermineStatus(actions, fileMod, now)
		s.Duration = FormatDuration(now - s.StartedAt)
		all = append(all, s)
	}

	// deduplicate by CWD: when multiple sessions share a CWD (resumed sessions),
	// keep the one with the highest-priority status (most active)
	seen := make(map[string]int) // cwd → index in result
	var result []Session
	for _, s := range all {
		if idx, ok := seen[s.CWD]; ok {
			existing := result[idx]
			ep := statusPriority[existing.Status]
			sp := statusPriority[s.Status]
			if sp < ep || (sp == ep && s.StartedAt > existing.StartedAt) {
				result[idx] = s
			}
			continue
		}
		seen[s.CWD] = len(result)
		result = append(result, s)
	}
	return result
}

func (m *Manager) GetIDESource(cwd string) string {
	src := m.ide.GetSource(cwd)
	if src == "Terminal" {
		return ""
	}
	return src
}
