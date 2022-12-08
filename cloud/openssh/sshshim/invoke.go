package sshshim

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

func InvokeSSHCommand() error {

	// We need to look for ssh in PATH. If we directly call "ssh", then we recursively
	// loop into calling the ssh-named symlink that points to devbox.
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return errors.WithStack(err)
	}

	// We set sshPath to the first argument in `args` because:
	//
	// https://man7.org/linux/man-pages/man2/execve.2.html
	// argv is an array of pointers to strings passed to the new program
	//       as its command-line arguments.  By convention, the first of these
	//       strings (i.e., argv[0]) should contain the filename associated
	//       with the file being executed.
	args := os.Args
	args[0] = sshPath
	debug.Log("invoking ssh with args: %v", args)

	// Choose syscall.Exec instead of exec.Cmd so that we preserve the exit code
	// and environment, and the current process is replaced by ssh.
	// Without this, we'd see errors during mutagen sync: the mutagen binary
	// would fail to be copied to Beta, the remote machine.
	return errors.WithStack(syscall.Exec(sshPath, args, os.Environ()))
}
