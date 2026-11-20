//go:build !windows

package dc

import "os/exec"

func configureSubprocess(cmd *exec.Cmd) {}
