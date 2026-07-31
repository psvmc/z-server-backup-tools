//go:build !windows

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
	if _, err := os.Stat(path); err != nil {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			path = dir
		} else {
			return err
		}
	}
	openCmd := "xdg-open"
	if execPath, err := exec.LookPath("open"); err == nil {
		openCmd = execPath
	}
	return exec.Command(openCmd, path).Start()
}
