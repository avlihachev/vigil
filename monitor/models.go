package monitor

import "fmt"

type ActionType string

const (
	ActionEdit    ActionType = "edit"
	ActionRead    ActionType = "read"
	ActionBash    ActionType = "bash"
	ActionSearch  ActionType = "search"
	ActionWaiting ActionType = "waiting" // finished, waiting for user
	ActionConfirm ActionType = "confirm" // waiting for tool approval
)

type Action struct {
	Type      ActionType `json:"type"`
	Target    string     `json:"target"`
	Result    string     `json:"result,omitempty"`
	Timestamp int64      `json:"timestamp"`
}

type SessionStatus string

const (
	StatusActive       SessionStatus = "active"
	StatusWaiting      SessionStatus = "waiting"       // finished, waiting for user message
	StatusConfirm      SessionStatus = "confirm"        // waiting for tool approval
	StatusIdle         SessionStatus = "idle"
)

type Session struct {
	PID           int           `json:"pid"`
	SessionID     string        `json:"sessionId"`
	CWD           string        `json:"cwd"`
	StartedAt     int64         `json:"startedAt"`
	Source        string        `json:"source"`
	ProjectName   string        `json:"projectName"`
	Name          string        `json:"name"`
	Status        SessionStatus `json:"status"`
	Duration      string        `json:"duration"`
	TokensIn      string        `json:"tokensIn"`
	TokensOut     string        `json:"tokensOut"`
	RecentActions []Action      `json:"recentActions"`
	Sibling       string        `json:"sibling,omitempty"`
}

func FormatTokens(n int64) string {
	switch {
	case n == 0:
		return ""
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
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

func CountBadge(sessions []Session, confirm, waiting, active bool) int {
	n := 0
	for _, s := range sessions {
		switch s.Status {
		case StatusConfirm:
			if confirm {
				n++
			}
		case StatusWaiting:
			if waiting {
				n++
			}
		case StatusActive:
			if active {
				n++
			}
		}
	}
	return n
}
