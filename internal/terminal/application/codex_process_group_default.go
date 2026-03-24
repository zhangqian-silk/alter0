//go:build !unix

package application

import "os/exec"

func configureCodexCommand(cmd *exec.Cmd) {
	_ = cmd
}
