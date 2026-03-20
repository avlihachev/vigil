package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.hasRead && time.Since(r.lastRead) < 10*time.Second {
		return r.cached
	}

	rl := r.tryRead(r.nativePath)
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

// newerWindow picks the better of two windows: newer resets_at wins, then higher percentage
func newerWindow(a, b *RateWindow) *RateWindow {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if b.ResetsAt > a.ResetsAt {
		return b
	}
	if b.ResetsAt == a.ResetsAt && b.UsedPercentage > a.UsedPercentage {
		return b
	}
	return a
}

func (r *RateLimitReader) readFreshest() *RateLimits {
	entries, err := os.ReadDir(r.bridgeDir)
	if err != nil {
		return nil
	}

	var bestFive, bestSeven *RateWindow
	var bestUpdated string
	found := false
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

		rl := r.tryRead(path)
		if rl == nil {
			continue
		}

		bestFive = newerWindow(bestFive, rl.FiveHour)
		bestSeven = newerWindow(bestSeven, rl.SevenDay)
		if rl.UpdatedAt > bestUpdated {
			bestUpdated = rl.UpdatedAt
		}
		found = true
	}

	if !found {
		return nil
	}
	return &RateLimits{
		FiveHour:  bestFive,
		SevenDay:  bestSeven,
		UpdatedAt: bestUpdated,
		Available: true,
	}
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
