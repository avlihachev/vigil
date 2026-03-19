package monitor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type HistoricalSession struct {
	SessionID    string `json:"sessionId"`
	Name         string `json:"name"`
	LastActiveAt int64  `json:"lastActiveAt"`
	TokensIn     string `json:"tokensIn"`
	TokensOut    string `json:"tokensOut"`
}

type ProjectHistory struct {
	ProjectName string              `json:"projectName"`
	CWD         string              `json:"cwd"`
	Sessions    []HistoricalSession `json:"sessions"`
}

type HistoryScanner struct {
	baseDir  string
	activity *ActivityParser
}

func NewHistoryScanner(baseDir string) *HistoryScanner {
	return &HistoryScanner{
		baseDir:  baseDir,
		activity: NewActivityParser(baseDir),
	}
}

func (s *HistoryScanner) ScanHistory(activeCWDs []string) []ProjectHistory {
	active := make(map[string]bool, len(activeCWDs))
	for _, cwd := range activeCWDs {
		active[strings.ReplaceAll(cwd, "/", "-")] = true
	}

	projectsDir := filepath.Join(s.baseDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}

	var groups []ProjectHistory
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		encodedCWD := e.Name()
		if active[encodedCWD] {
			continue
		}

		cwd := "/" + strings.ReplaceAll(strings.TrimPrefix(encodedCWD, "-"), "-", "/")

		dirPath := filepath.Join(projectsDir, encodedCWD)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		var sessions []HistoricalSession
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			sessionID := strings.TrimSuffix(f.Name(), ".jsonl")
			jsonlPath := filepath.Join(dirPath, f.Name())

			info, err := f.Info()
			if err != nil {
				continue
			}
			lastActive := info.ModTime().UnixMilli()

			lines, _ := tailFile(jsonlPath, 200)
			tokIn, tokOut := parseTokensFromLines(lines)
			name := s.activity.ParseName(sessionID, cwd)

			sessions = append(sessions, HistoricalSession{
				SessionID:    sessionID,
				Name:         name,
				LastActiveAt: lastActive,
				TokensIn:     FormatTokens(tokIn),
				TokensOut:    FormatTokens(tokOut),
			})
		}

		if len(sessions) == 0 {
			continue
		}

		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].LastActiveAt > sessions[j].LastActiveAt
		})
		if len(sessions) > 5 {
			sessions = sessions[:5]
		}

		groups = append(groups, ProjectHistory{
			ProjectName: filepath.Base(cwd),
			CWD:         cwd,
			Sessions:    sessions,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Sessions) == 0 {
			return false
		}
		if len(groups[j].Sessions) == 0 {
			return true
		}
		return groups[i].Sessions[0].LastActiveAt > groups[j].Sessions[0].LastActiveAt
	})
	if len(groups) > 50 {
		groups = groups[:50]
	}
	return groups
}
