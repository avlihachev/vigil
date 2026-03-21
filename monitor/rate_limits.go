package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RateWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

type RateLimits struct {
	FiveHour  *RateWindow `json:"five_hour,omitempty"`
	SevenDay  *RateWindow `json:"seven_day,omitempty"`
	UpdatedAt string      `json:"updated_at"`
	Available bool        `json:"dataAvailable"`
}

type RateLimitReader struct {
	nativePath string
	bridgeDir  string
	legacyPath string
	cached     *RateLimits
	lastRead   time.Time
	hasRead    bool
	mu         sync.Mutex
}

func NewRateLimitReader(claudeDir, vigilDir string) *RateLimitReader {
	return &RateLimitReader{
		nativePath: filepath.Join(claudeDir, "cache", "rate-limits.json"),
		bridgeDir:  filepath.Join(vigilDir, "rate-limits"),
		legacyPath: filepath.Join(vigilDir, "rate-limits.json"),
	}
}

func (r *RateLimitReader) Read() *RateLimits {
	return r.ReadForSessions(nil)
}

func (r *RateLimitReader) ReadForSessions(activeIDs []string) *RateLimits {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.hasRead && time.Since(r.lastRead) < 10*time.Second {
		return r.cached
	}

	rl := r.tryRead(r.nativePath)
	if rl == nil {
		rl = r.readFreshestForSessions(activeIDs)
	}
	if rl == nil {
		rl = r.readFreshest()
	}
	if rl == nil {
		rl = r.tryRead(r.legacyPath)
	}

	r.cached = rl
	r.lastRead = time.Now()
	r.hasRead = true
	return rl
}

func (r *RateLimitReader) Invalidate() {
	r.mu.Lock()
	r.hasRead = false
	r.mu.Unlock()
}

// readFreshestForSessions returns rate limits from the most recently modified
// per-session file matching one of the active session IDs.
func (r *RateLimitReader) readFreshestForSessions(activeIDs []string) *RateLimits {
	if len(activeIDs) == 0 {
		return nil
	}
	active := make(map[string]bool, len(activeIDs))
	for _, id := range activeIDs {
		active[id] = true
	}

	entries, err := os.ReadDir(r.bridgeDir)
	if err != nil {
		return nil
	}

	var bestPath string
	var bestMod time.Time
	staleThreshold := time.Now().Add(-1 * time.Hour)

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		path := filepath.Join(r.bridgeDir, e.Name())

		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(staleThreshold) {
			os.Remove(path)
			continue
		}

		if !active[name] {
			continue
		}

		if bestPath == "" || info.ModTime().After(bestMod) {
			bestPath = path
			bestMod = info.ModTime()
		}
	}

	if bestPath == "" {
		return nil
	}
	return r.tryRead(bestPath)
}

// readFreshest returns rate limits from the most recently modified per-session file.
func (r *RateLimitReader) readFreshest() *RateLimits {
	entries, err := os.ReadDir(r.bridgeDir)
	if err != nil {
		return nil
	}

	var bestPath string
	var bestMod time.Time
	staleThreshold := time.Now().Add(-1 * time.Hour)

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(r.bridgeDir, e.Name())

		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(staleThreshold) {
			os.Remove(path)
			continue
		}

		if bestPath == "" || info.ModTime().After(bestMod) {
			bestPath = path
			bestMod = info.ModTime()
		}
	}

	if bestPath == "" {
		return nil
	}
	return r.tryRead(bestPath)
}

func (r *RateLimitReader) tryRead(path string) *RateLimits {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var rl RateLimits
	if err := json.Unmarshal(data, &rl); err != nil {
		return nil
	}

	rl.Available = true
	return &rl
}
