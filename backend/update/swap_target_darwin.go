//go:build darwin

package update

import (
	"os"
	"path/filepath"
	"strings"
)

func swapTarget(exe string) string {
	parts := strings.Split(filepath.Clean(exe), string(os.PathSeparator))
	for i, part := range parts {
		if strings.HasSuffix(part, ".app") {
			return string(os.PathSeparator) + filepath.Join(parts[1:i+1]...)
		}
	}
	return exe
}
