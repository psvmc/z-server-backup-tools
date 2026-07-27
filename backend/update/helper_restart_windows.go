//go:build windows

package update

import (
	"os/exec"
	"syscall"
)

func ApplyVisibleDetachAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const (
		detachedProcess    = 0x00000008
		createNewProcGroup = 0x00000200
	)
	cmd.SysProcAttr.CreationFlags |= detachedProcess | createNewProcGroup
}

func ApplyDetachAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const (
		detachedProcess    = 0x00000008
		createNoWindow     = 0x08000000
		createNewProcGroup = 0x00000200
	)
	cmd.SysProcAttr.CreationFlags |= detachedProcess | createNoWindow | createNewProcGroup
	cmd.SysProcAttr.HideWindow = true
}

func applyDetachAttrs(cmd *exec.Cmd) {
	ApplyDetachAttrs(cmd)
}
