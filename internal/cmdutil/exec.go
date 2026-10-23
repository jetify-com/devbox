// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cmdutil

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
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

	// stdout and stderr are copied by separate goroutines, so guard the
	// shared buffer with a mutex.
	outBuf := bytes.NewBuffer(nil)
	w := &syncWriter{w: outBuf}
	cmd.Stderr = io.MultiWriter(os.Stderr, w)
	cmd.Stdout = io.MultiWriter(os.Stdout, w)
	return cmd, outBuf
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
