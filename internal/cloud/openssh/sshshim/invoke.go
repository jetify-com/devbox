// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package sshshim

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/debug"
)

func InvokeSSHOrSCPCommand(args []string) error {
	if !strings.HasSuffix(args[0], "ssh") && !strings.HasSuffix(args[0], "scp") {
		return errors.Errorf("received %s for args[0], but expected ssh or scp", args[0])
	}

	executableName := "ssh"
	if strings.HasSuffix(args[0], "scp") {
		executableName = "scp"
	}

	// We need to look for ssh or scp in PATH. If we directly call "ssh", for example,
	// then we recursively loop into calling the ssh-named symlink that points to devbox.
	executablePath, err := exec.LookPath(executableName)
	if err != nil {
		return errors.WithStack(err)
	}

	// We set executablePath to the first argument in `args` because:
	//
	// https://man7.org/linux/man-pages/man2/execve.2.html
	// argv is an array of pointers to strings passed to the new program
	//       as its command-line arguments.  By convention, the first of these
	//       strings (i.e., argv[0]) should contain the filename associated
	//       with the file being executed.
	args[0] = executablePath
	debug.Log("invoking %s with args: %v", args[0], args)

	// Choose syscall.Exec instead of exec.Cmd so that we preserve the exit code
	// and environment, and the current process is replaced by ssh.
	// Without this, we'd see errors during mutagen sync: the mutagen binary
	// would fail to be copied to Beta, the remote machine.
	return errors.WithStack(syscall.Exec(executablePath, args, os.Environ()))
}
