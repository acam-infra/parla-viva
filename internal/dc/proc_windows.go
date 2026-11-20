//go:build windows

package dc

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func configureSubprocess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
