package sshshim

import (
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

func InvokeSSHCommand(sshArgs []string) error {

	cmd := exec.Command("ssh", sshArgs...)
	debug.Log("executing command: %s\n", cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return errors.WithStack(err)
}
