package zipbak

import (
	"fmt"
	"strings"
)

// SanitizePartPrefix strips characters invalid in zip file names.
func SanitizePartPrefix(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// PartZipName builds the archive file name for a given serial.
func PartZipName(prefix string, serial int) string {
	return fmt.Sprintf("%spart-%06d.zip", SanitizePartPrefix(prefix), serial)
}

// IsBackupPartName reports whether name matches the backup part pattern.
func IsBackupPartName(name, prefix string) bool {
	lower := strings.ToLower(name)
	expected := strings.ToLower(SanitizePartPrefix(prefix) + "part-")
	return strings.HasPrefix(lower, expected) && strings.HasSuffix(lower, ".zip")
}
