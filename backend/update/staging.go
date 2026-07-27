package update

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// RelocateBesideExecutable copies the artifact next to the running executable.
func RelocateBesideExecutable(exePath, artifactPath string) (string, error) {
	return relocateNextToExecutable(exePath, artifactPath)
}

// relocateNextToExecutable copies the staged update artifact into the same
// directory as the running executable so the Wails helper can os.Rename it
// into place. os.Rename fails on Windows when source and target live on
// different drives (Temp is often on C: while the app is on D:).
func relocateNextToExecutable(exePath, stagedPath string) (string, error) {
	info, err := os.Stat(stagedPath)
	if err != nil {
		return "", fmt.Errorf("stat staged artifact: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	stagingDir, err := prepareStagingDir(exeDir)
	if err != nil {
		return "", err
	}

	dest := filepath.Join(stagingDir, filepath.Base(stagedPath))
	if err := os.RemoveAll(dest); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", fmt.Errorf("clear staging dest: %w", err)
	}

	if info.IsDir() {
		if err := copyTree(stagedPath, dest, info.Mode()); err != nil {
			_ = os.RemoveAll(stagingDir)
			return "", err
		}
	} else if err := copyFile(stagedPath, dest, info.Mode()); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", err
	}

	return dest, nil
}

func prepareStagingDir(exeDir string) (string, error) {
	primary := filepath.Join(exeDir, fmt.Sprintf(".z-server-backup-tools-update-%d", os.Getpid()))
	if err := os.MkdirAll(primary, 0o755); err == nil {
		return primary, nil
	} else if !isAccessDenied(err) {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	fallback := filepath.Join(writableStagingBase(exeDir), fmt.Sprintf("z-server-backup-tools-update-%d", os.Getpid()))
	if err := os.MkdirAll(fallback, 0o755); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}
	return fallback, nil
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access is denied") || strings.Contains(msg, "permission denied")
}

func writableStagingBase(nearPath string) string {
	volume := filepath.VolumeName(nearPath)
	if temp := os.TempDir(); volume == "" || filepath.VolumeName(temp) == volume {
		return filepath.Join(temp, "z-server-backup-tools")
	}
	if cache, err := os.UserCacheDir(); err == nil {
		if volume == "" || filepath.VolumeName(cache) == volume {
			return filepath.Join(cache, "z-server-backup-tools", "updates")
		}
	}
	return filepath.Join(os.TempDir(), "z-server-backup-tools")
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open staged file: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create local staging file: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy staged file: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("sync staged file: %w", err)
	}
	return out.Close()
}

func copyTree(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(dst, mode); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read staged dir: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		switch {
		case entryInfo.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, dstPath); err != nil {
				return err
			}
		case entryInfo.IsDir():
			if err := copyTree(srcPath, dstPath, entryInfo.Mode()); err != nil {
				return err
			}
		default:
			if err := copyFile(srcPath, dstPath, entryInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}
