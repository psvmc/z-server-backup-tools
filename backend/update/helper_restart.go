package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Environment keys for the Wails updater helper protocol. Must stay in sync
// with github.com/wailsapp/wails/v3/pkg/updater.
const (
	envHelperMode   = "WAILS_UPDATER_HELPER"
	envHelperTarget = "WAILS_UPDATER_HELPER_TARGET"
	envHelperNew    = "WAILS_UPDATER_HELPER_NEW"
	envHelperPID    = "WAILS_UPDATER_HELPER_PID"
	envHelperLog    = "WAILS_UPDATER_HELPER_LOG"
)

// ApplyStagedUpdate relocates the downloaded artifact next to the running
// executable and spawns the Wails helper process to swap binaries in place.
func ApplyStagedUpdate(stagedPath string) error {
	if stagedPath == "" {
		return fmt.Errorf("staged update path is empty")
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	localPath, err := RelocateBesideExecutable(self, stagedPath)
	if err != nil {
		return err
	}

	target := swapTarget(self)
	logPath := filepath.Join(os.TempDir(), fmt.Sprintf("wails-update-%d.log", os.Getpid()))
	return spawnUpdaterHelper(self, target, localPath, os.Getpid(), logPath)
}

func spawnUpdaterHelper(self, target, newPath string, pid int, logPath string) error {
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(),
		envHelperMode+"=1",
		envHelperTarget+"="+target,
		envHelperNew+"="+newPath,
		envHelperPID+"="+strconv.Itoa(pid),
		envHelperLog+"="+logPath,
	)
	applyDetachAttrs(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn updater helper: %w", err)
	}
	return nil
}
