package monitor

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{30_000, "0m"},
		{60_000, "1m"},
		{300_000, "5m"},
		{3_600_000, "1h 0m"},
		{5_400_000, "1h 30m"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestCountBadge(t *testing.T) {
	sessions := []Session{
		{SessionID: "1", Status: StatusConfirm},
		{SessionID: "2", Status: StatusWaiting},
		{SessionID: "3", Status: StatusActive},
		{SessionID: "4", Status: StatusIdle},
	}
	if n := CountBadge(sessions, true, true, false); n != 2 {
		t.Errorf("expected 2 (confirm+waiting), got %d", n)
	}
	if n := CountBadge(sessions, true, false, false); n != 1 {
		t.Errorf("expected 1 (confirm only), got %d", n)
	}
	if n := CountBadge(sessions, true, true, true); n != 3 {
		t.Errorf("expected 3 (confirm+waiting+active), got %d", n)
	}
	if n := CountBadge(sessions, false, false, false); n != 0 {
		t.Errorf("expected 0 (all disabled), got %d", n)
	}
	if n := CountBadge(nil, true, true, true); n != 0 {
		t.Errorf("expected 0 for nil, got %d", n)
	}
}
