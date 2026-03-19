package monitor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

	// filter alive and enrich with source
	var alive []Session
	for _, s := range raw {
		if !IsProcessAlive(s.PID) {
			continue
		}
		s.Source = m.ide.GetSource(s.CWD)
		if s.Source == "Terminal" {
			s.Source = terminalForPID(s.PID)
		}
		alive = append(alive, s)
	}

	// group by CWD
	byCWD := map[string][]int{}
	for i, s := range alive {
		byCWD[s.CWD] = append(byCWD[s.CWD], i)
	}

	// resolve JSONL path for each session
	jsonlPaths := make([]string, len(alive))
	for cwd, indices := range byCWD {
		if len(indices) == 1 {
			i := indices[0]
			jsonlPaths[i] = m.activity.FindJSONL(alive[i].SessionID, cwd)
		} else {
			// greedy 1:1 matching: sort sessions by startedAt desc, jsonls by modTime desc
			sorted := make([]int, len(indices))
			copy(sorted, indices)
			sort.Slice(sorted, func(a, b int) bool {
				return alive[sorted[a]].StartedAt > alive[sorted[b]].StartedAt
			})

			jsonls := m.activity.ListProjectJSONLs(cwd)

			// first pass: exact filename match
			used := make(map[string]bool)
			for _, i := range sorted {
				for _, jp := range jsonls {
					if used[jp] {
						continue
					}
					// check exact match by sessionID in filename
					if matchesFilename(jp, alive[i].SessionID) {
						jsonlPaths[i] = jp
						used[jp] = true
						break
					}
				}
			}

			// second pass: greedy assign remaining by mod-time order
			jIdx := 0
			for _, i := range sorted {
				if jsonlPaths[i] != "" {
					continue
				}
				for jIdx < len(jsonls) && used[jsonls[jIdx]] {
					jIdx++
				}
				if jIdx < len(jsonls) {
					jsonlPaths[i] = jsonls[jIdx]
					used[jsonls[jIdx]] = true
					jIdx++
				}
			}
		}
	}

	now := time.Now().UnixMilli()
	result := make([]Session, 0, len(alive))
	for i, s := range alive {
		sa := m.activity.ParseAllFromPath(jsonlPaths[i])
		s.RecentActions = sa.Actions
		s.Name = sa.Name
		s.TokensIn = FormatTokens(sa.TokensIn)
		s.TokensOut = FormatTokens(sa.TokensOut)
		s.Status = DetermineStatus(sa.Actions, sa.FileModMs, now)
		s.Duration = FormatDuration(now - s.StartedAt)
		result = append(result, s)
	}

	// mark sibling sessions sharing the same CWD
	cwdCount := make(map[string]int)
	for _, s := range result {
		cwdCount[s.CWD]++
	}
	if len(result) > 0 {
		idx := make(map[string]int)
		for i := range result {
			cwd := result[i].CWD
			if cwdCount[cwd] > 1 {
				idx[cwd]++
				result[i].Sibling = fmt.Sprintf("%d/%d", idx[cwd], cwdCount[cwd])
			}
		}
	}

	return result
}

// matchesFilename checks if a JSONL path's filename (without extension) matches a sessionID
func matchesFilename(jsonlPath, sessionID string) bool {
	base := filepath.Base(jsonlPath)
	return strings.TrimSuffix(base, ".jsonl") == sessionID
}

func (m *Manager) GetIDESource(cwd string) string {
	src := m.ide.GetSource(cwd)
	if src == "Terminal" {
		return ""
	}
	return src
}
