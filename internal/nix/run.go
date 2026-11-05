// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/cmdutil"
)

func RunScript(projectDir, cmdWithArgs string, env map[string]string) error {
	if cmdWithArgs == "" {
		return errors.New("attempted to run an empty command or script")
	}

	envPairs := []string{}
	for k, v := range env {
		envPairs = append(envPairs, fmt.Sprintf("%s=%s", k, v))
	}

	// Try to find sh in the PATH, if not, default to a well known absolute path.
	shPath := cmdutil.GetPathOrDefault("sh", "/bin/sh")
	cmd := exec.Command(shPath, "-c", cmdWithArgs)
	cmd.Env = envPairs
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Debug("executing script", "cmd", cmd.Args)
	// Report error as exec error when executing scripts.
	return usererr.NewExecError(runWaitingOnInterrupt(cmd))
}

// runWaitingOnInterrupt runs cmd. If the user presses Ctrl-C (or Ctrl-\),
// devbox stays alive until cmd exits, so that the shell prompt only comes back
// once the command has finished shutting down, the same as running the command
// directly or with `npm run`.
func runWaitingOnInterrupt(cmd *exec.Cmd) error {
	// Subscribe before starting the command so that devbox can't be killed
	// between starting it and waiting for it. Leave ignored signals alone:
	// Notify would stop ignoring them, in devbox and in the command (e.g. in a
	// background job of a non-interactive shell).
	signals := make(chan os.Signal, 1)
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGQUIT} {
		if !signal.Ignored(sig) {
			signal.Notify(signals, sig)
		}
	}
	defer signal.Stop(signals)

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case sig := <-signals:
				handleInterrupt(sig)
			case <-done:
				return
			}
		}
	}()
	return cmd.Wait()
}

func handleInterrupt(sig os.Signal) {
	// The terminal sends Ctrl-C and Ctrl-\ to its whole foreground process
	// group, which includes the command, so it's enough to wait for it.
	if inForegroundProcessGroup() {
		slog.Debug("waiting for command to exit after interrupt", "signal", sig)
		return
	}
	// The signal came from somewhere else (e.g. kill -INT <devbox pid>), and
	// the command didn't get it. Exit the way devbox did before it handled
	// interrupts.
	signal.Reset(sig)
	_ = syscall.Kill(os.Getpid(), sig.(syscall.Signal))
}

// inForegroundProcessGroup reports whether devbox (and so the command it
// started, which shares its process group) is in the foreground process group
// of its controlling terminal.
func inForegroundProcessGroup() bool {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false // no controlling terminal
	}
	defer tty.Close()
	pgrp, err := unix.IoctlGetInt(int(tty.Fd()), unix.TIOCGPGRP)
	return err == nil && pgrp == unix.Getpgrp()
}
