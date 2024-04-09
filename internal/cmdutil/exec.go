// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cmdutil

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

func CommandTTY(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// CommandTTYWithBuffer returns a command with stdin, stdout, and stderr
// and a buffer that contains stdout and stderr combined.
func CommandTTYWithBuffer(
	name string,
	arg ...string,
) (*exec.Cmd, *bytes.Buffer) {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin

	errBuf := bytes.NewBuffer(nil)
	outBuf := bytes.NewBuffer(nil)
	cmd.Stderr = io.MultiWriter(os.Stderr, errBuf)
	cmd.Stdout = io.MultiWriter(os.Stdout, outBuf)
	outBuf.Write(errBuf.Bytes())
	return cmd, outBuf
}
