package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RateWindow struct {
	UsedPercentage int   `json:"used_percentage"`
	ResetsAt       int64 `json:"resets_at"`
}

type RateLimits struct {
	FiveHour  *RateWindow `json:"five_hour,omitempty"`
	SevenDay  *RateWindow `json:"seven_day,omitempty"`
	UpdatedAt string      `json:"updated_at"`
	Available bool        `json:"dataAvailable"`
}

type RateLimitReader struct {
	nativePath string
	bridgePath string
	cached     *RateLimits
	lastRead   time.Time
	hasRead    bool
	mu         sync.Mutex
}

func NewRateLimitReader(claudeDir, vigilDir string) *RateLimitReader {
	return &RateLimitReader{
		nativePath: filepath.Join(claudeDir, "cache", "rate-limits.json"),
		bridgePath: filepath.Join(vigilDir, "rate-limits.json"),
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
		rl = r.tryRead(r.bridgePath)
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

func (r *RateLimitReader) tryRead(path string) *RateLimits {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// stale if file older than 5 minutes
	if time.Since(info.ModTime()) > 5*time.Minute {
		return nil
	}

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
