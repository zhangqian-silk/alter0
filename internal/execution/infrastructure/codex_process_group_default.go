//go:build !unix

package infrastructure

import "os/exec"

func configureCodexCommand(cmd *exec.Cmd) {
	_ = cmd
}
