package monitor

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ActivityParser struct {
	baseDir string
}

func NewActivityParser(baseDir string) *ActivityParser {
	return &ActivityParser{baseDir: baseDir}
}

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
	"Skill":     ActionBash,
}

// jsonlEntry is a partial parse of the project JSONL format
type jsonlEntry struct {
	Type          string          `json:"type"`
	Timestamp     string          `json:"timestamp"`
	SessionID     string          `json:"sessionId"`
	Slug          string          `json:"slug"`
	CustomTitle   string          `json:"customTitle"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
	Message       *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type contentBlock struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// SessionActivity holds all parsed data from a single JSONL file
type SessionActivity struct {
	Actions   []Action
	Name      string
	TokensIn  int64
	TokensOut int64
	FileModMs int64
}

// ParseAllFromPath reads a JSONL file once and extracts actions, name, and tokens
func (p *ActivityParser) ParseAllFromPath(jsonlPath string) SessionActivity {
	var sa SessionActivity
	if jsonlPath == "" {
		return sa
	}

	info, err := os.Stat(jsonlPath)
	if err != nil {
		return sa
	}
	sa.FileModMs = info.ModTime().UnixMilli()

	lines, err := readAllLines(jsonlPath)
	if err != nil {
		return sa
	}

	// single pass: extract tokens, name, slug
	var slug string
	for _, line := range lines {
		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Type == "assistant" && entry.Message != nil && entry.Message.Usage != nil {
			sa.TokensIn += entry.Message.Usage.InputTokens
			sa.TokensOut += entry.Message.Usage.OutputTokens
		}
		if entry.Type == "custom-title" && entry.CustomTitle != "" {
			sa.Name = entry.CustomTitle
		}
		if slug == "" && entry.Slug != "" {
			slug = entry.Slug
		}
	}
	if sa.Name == "" {
		sa.Name = slug
	}

	// extract actions from tail
	tail := lines
	if len(tail) > 200 {
		tail = tail[len(tail)-200:]
	}
	sa.Actions = parseActionsFromLines(tail)

	return sa
}

func (p *ActivityParser) Parse(sessionID string, cwd string) ([]Action, error) {
	jsonlPath := p.FindJSONL(sessionID, cwd)
	if jsonlPath == "" {
		return nil, nil
	}

	lines, err := tailFile(jsonlPath, 200)
	if err != nil {
		return nil, nil
	}

	return parseActionsFromLines(lines), nil
}

func parseActionsFromLines(lines []string) []Action {
	var actions []Action
	lastAssistantWasTextOnly := false
	lastAssistantHadToolUse := false
	gotUserAfterLastAssistant := false
	var lastAssistantTs int64

	for _, line := range lines {
		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		ts := parseTimestamp(entry.Timestamp)
		if ts == 0 {
			continue
		}

		if entry.Message != nil && entry.Message.Role == "assistant" {
			var blocks []contentBlock
			if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
				continue
			}
			hasToolUse := false
			for _, b := range blocks {
				if b.Type == "tool_use" && b.Name != "" {
					hasToolUse = true
					actionType, ok := toolTypeMap[b.Name]
					if !ok {
						actionType = ActionBash
					}
					actions = append(actions, Action{
						Type:      actionType,
						Target:    b.Name,
						Timestamp: ts,
					})
				}
			}
			lastAssistantHadToolUse = hasToolUse
			lastAssistantWasTextOnly = !hasToolUse
			gotUserAfterLastAssistant = false
			lastAssistantTs = ts
		}

		if entry.Type == "user" && len(entry.ToolUseResult) > 0 {
			gotUserAfterLastAssistant = true
		}
		if entry.Type == "user" && entry.Message != nil && len(entry.ToolUseResult) == 0 {
			gotUserAfterLastAssistant = true
			lastAssistantWasTextOnly = false
			lastAssistantHadToolUse = false
		}
	}

	if lastAssistantHadToolUse && !gotUserAfterLastAssistant && lastAssistantTs > 0 {
		actions = append(actions, Action{
			Type:      ActionConfirm,
			Target:    "needs confirmation",
			Timestamp: lastAssistantTs,
		})
	}

	if lastAssistantWasTextOnly && lastAssistantTs > 0 {
		actions = append(actions, Action{
			Type:      ActionWaiting,
			Target:    "waiting for input",
			Timestamp: lastAssistantTs,
		})
	}

	if len(actions) > 5 {
		actions = actions[len(actions)-5:]
	}
	return actions
}

func parseTokensFromLines(lines []string) (in, out int64) {
	for _, line := range lines {
		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Type == "assistant" && entry.Message != nil && entry.Message.Usage != nil {
			in += entry.Message.Usage.InputTokens
			out += entry.Message.Usage.OutputTokens
		}
	}
	return
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, nil
}

// ParseName returns the session display name: last custom title if set, otherwise slug.
func (p *ActivityParser) ParseName(sessionID string, cwd string) string {
	jsonlPath := p.FindJSONL(sessionID, cwd)
	if jsonlPath == "" {
		return ""
	}
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var slug, name string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var entry jsonlEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Type == "custom-title" && entry.CustomTitle != "" {
			name = entry.CustomTitle
		}
		if slug == "" && entry.Slug != "" {
			slug = entry.Slug
		}
	}
	if name != "" {
		return name
	}
	return slug
}

// FindJSONL locates the JSONL file for a session: exact filename match first, then newest fallback.
func (p *ActivityParser) FindJSONL(sessionID string, cwd string) string {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(p.baseDir, "projects", encoded)

	// try exact match first (new sessions)
	exact := filepath.Join(dir, sessionID+".jsonl")
	if _, err := os.Stat(exact); err == nil {
		return exact
	}

	// fallback to newest JSONL in the project dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var newest string
	var newestMod int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime().UnixMilli(); t > newestMod {
			newestMod = t
			newest = filepath.Join(dir, e.Name())
		}
	}
	return newest
}

// ListProjectJSONLs returns all .jsonl paths in the project dir for a CWD, sorted by modTime desc.
func (p *ActivityParser) ListProjectJSONLs(cwd string) []string {
	encoded := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(p.baseDir, "projects", encoded)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	type jsonlFile struct {
		path  string
		modMs int64
	}
	var files []jsonlFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, jsonlFile{
			path:  filepath.Join(dir, e.Name()),
			modMs: info.ModTime().UnixMilli(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modMs > files[j].modMs
	})

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	return paths
}

func parseTimestamp(s string) int64 {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", s)
	if err != nil {
		// try without milliseconds
		t, err = time.Parse("2006-01-02T15:04:05Z", s)
		if err != nil {
			return 0
		}
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
	// increase buffer for large JSONL lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// DetermineStatus uses file modification time for freshness, actions for state
func DetermineStatus(actions []Action, fileModifiedMs int64, nowMs int64) SessionStatus {
	recentlyModified := fileModifiedMs > 0 && (nowMs-fileModifiedMs) < 300_000

	if !recentlyModified {
		return StatusIdle
	}

	if len(actions) > 0 {
		switch actions[len(actions)-1].Type {
		case ActionWaiting:
			return StatusWaiting
		case ActionConfirm:
			return StatusConfirm
		}
	}
	return StatusActive
}
