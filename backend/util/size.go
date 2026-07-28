package util

import "fmt"

// FormatBytes formats a size for human-readable logs (binary SI).
func FormatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div := int64(unit)
	exp := 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	val := float64(n) / float64(div)
	suffix := []string{"KiB", "MiB", "GiB", "TiB"}[exp]
	return fmt.Sprintf("%.2f %s", val, suffix)
}
