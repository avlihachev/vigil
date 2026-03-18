package monitor

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type ActivityParser struct {
	baseDir string
}

func NewActivityParser(baseDir string) *ActivityParser {
	return &ActivityParser{baseDir: baseDir}
}

var (
	reToolCall   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[DEBUG\]\s+executePreToolHooks called for tool:\s+(\w+)`)
	reIdlePrompt = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[DEBUG\]\s+Getting matching hook commands for Notification with query:\s+idle_prompt`)
)

var toolTypeMap = map[string]ActionType{
	"Edit":      ActionEdit,
	"Write":     ActionEdit,
	"Read":      ActionRead,
	"Bash":      ActionBash,
	"Grep":      ActionSearch,
	"Glob":      ActionSearch,
	"Agent":     ActionBash,
	"WebSearch": ActionSearch,
	"WebFetch":  ActionRead,
}

func (p *ActivityParser) Parse(sessionID string) ([]Action, error) {
	path := filepath.Join(p.baseDir, "debug", sessionID+".txt")
	lines, err := tailFile(path, 80)
	if err != nil {
		return nil, nil
	}

	var actions []Action
	for _, line := range lines {
		if m := reIdlePrompt.FindStringSubmatch(line); m != nil {
			ts := parseTimestamp(m[1])
			actions = append(actions, Action{
				Type:      ActionWaiting,
				Target:    "waiting for input",
				Timestamp: ts,
			})
			continue
		}
		if m := reToolCall.FindStringSubmatch(line); m != nil {
			ts := parseTimestamp(m[1])
			toolName := m[2]
			actionType, ok := toolTypeMap[toolName]
			if !ok {
				actionType = ActionBash
			}
			actions = append(actions, Action{
				Type:      actionType,
				Target:    toolName,
				Timestamp: ts,
			})
		}
	}

	if len(actions) > 5 {
		actions = actions[len(actions)-5:]
	}
	return actions, nil
}

func parseTimestamp(s string) int64 {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func DetermineStatus(actions []Action, nowMs int64) SessionStatus {
	if len(actions) == 0 {
		return StatusIdle
	}
	last := actions[len(actions)-1]
	if last.Type == ActionWaiting {
		return StatusWaiting
	}
	if nowMs-last.Timestamp < 30_000 {
		return StatusActive
	}
	return StatusIdle
}
