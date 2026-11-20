//go:build !windows

package notify

import "os/exec"

func configureHiddenCmd(cmd *exec.Cmd) {}
