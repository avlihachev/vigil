package monitor

import "fmt"

type ActionType string

const (
	ActionEdit    ActionType = "edit"
	ActionRead    ActionType = "read"
	ActionBash    ActionType = "bash"
	ActionSearch  ActionType = "search"
	ActionWaiting ActionType = "waiting"
)

type Action struct {
	Type      ActionType `json:"type"`
	Target    string     `json:"target"`
	Result    string     `json:"result,omitempty"`
	Timestamp int64      `json:"timestamp"`
}

type SessionStatus string

const (
	StatusActive  SessionStatus = "active"
	StatusWaiting SessionStatus = "waiting"
	StatusIdle    SessionStatus = "idle"
)

type Session struct {
	PID           int           `json:"pid"`
	SessionID     string        `json:"sessionId"`
	CWD           string        `json:"cwd"`
	StartedAt     int64         `json:"startedAt"`
	Source        string        `json:"source"`
	ProjectName   string        `json:"projectName"`
	Status        SessionStatus `json:"status"`
	Duration      string        `json:"duration"`
	RecentActions []Action      `json:"recentActions"`
}

func FormatDuration(ms int64) string {
	total := ms / 1000
	hours := total / 3600
	minutes := (total % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
