//go:build windows

package notify

import (
	"os/exec"
	"syscall"
)

func configureHiddenCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
