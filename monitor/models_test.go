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
