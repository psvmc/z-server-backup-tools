package util

import "fmt"

// FormatDuration formats milliseconds as m:ss or h:mm:ss (same as frontend formatDuration).
func FormatDuration(ms int64) string {
	if ms < 0 {
		return "--:--"
	}
	totalSec := ms / 1000
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
