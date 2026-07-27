//go:build linux

package service

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

func (s *UpdateService) applyPlatformUpdate(ctx context.Context) error {
	if err := s.app.Updater.DownloadAndInstall(ctx); err != nil {
		return fmt.Errorf("???????: %w", err)
	}

	packagePath := s.app.Updater.DownloadedPath()
	if packagePath == "" {
		return fmt.Errorf("??????????")
	}

	if err := installLinuxPackage(ctx, packagePath); err != nil {
		return err
	}

	s.app.Quit()
	return nil
}

func installLinuxPackage(ctx context.Context, packagePath string) error {
	ext := filepath.Ext(packagePath)
	switch ext {
	case ".deb":
		return runLinuxInstaller(ctx, []string{"dpkg", "-i", packagePath}, packagePath)
	case ".rpm":
		return runLinuxInstaller(ctx, []string{"rpm", "-U", packagePath}, packagePath)
	default:
		return openPackageWithDesktop(packagePath)
	}
}

func runLinuxInstaller(ctx context.Context, args []string, packagePath string) error {
	if path, err := exec.LookPath("pkexec"); err == nil {
		cmd := exec.CommandContext(ctx, path, args...)
		detachCommand(cmd)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	if path, err := exec.LookPath("sudo"); err == nil {
		cmd := exec.CommandContext(ctx, path, args...)
		detachCommand(cmd)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	return openPackageWithDesktop(packagePath)
}

func openPackageWithDesktop(packagePath string) error {
	if path, err := exec.LookPath("xdg-open"); err == nil {
		cmd := exec.Command(path, packagePath)
		detachCommand(cmd)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("???????????????: %s", packagePath)
}

func detachCommand(cmd *exec.Cmd) {
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
}
