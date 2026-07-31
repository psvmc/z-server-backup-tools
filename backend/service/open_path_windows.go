//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openPathInExplorer(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		dir := filepath.Dir(path)
		if dir == "" || dir == "." {
			return err
		}
		return exec.Command("explorer", dir).Start()
	}
	if info.IsDir() {
		return exec.Command("explorer", path).Start()
	}
	return exec.Command("explorer", "/select,", path).Start()
}
