// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package shell detects the user's default shell and configures it to run in
// Devbox.
package shell

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type name string

const (
	shUnknown name = ""
	shBash    name = "bash"
	shZsh     name = "zsh"
	shKsh     name = "ksh"
	shPosix   name = "posix"
)

// Shell configures a user's shell to run in Devbox.
type Shell struct {
	name           name
	path           string
	initFile       string
	devboxInitFile string
}

// Detect attempts to determine the user's default shell.
func Detect() (*Shell, error) {
	path := os.Getenv("SHELL")
	if path == "" {
		return nil, errors.New("unable to detect the current shell")
	}

	sh := &Shell{path: filepath.Clean(path)}
	base := filepath.Base(path)
	// Login shell
	if base[0] == '-' {
		base = base[1:]
	}
	switch base {
	case "bash":
		sh.name = shBash
		sh.initFile = rcfilePath(".bashrc")
	case "zsh":
		sh.name = shZsh
		sh.initFile = rcfilePath(".zshrc")
	case "ksh":
		sh.name = shKsh
		sh.initFile = rcfilePath(".kshrc")
	case "dash", "ash", "sh":
		sh.name = shPosix
		sh.initFile = os.Getenv("ENV")

		// Just make up a name if there isn't already an init file set
		// so we have somewhere to put a new one.
		if sh.initFile == "" {
			sh.initFile = ".shinit"
		}
	default:
		sh.name = shUnknown
	}
	return sh, nil
}

// rcfilePath returns the absolute path for an rcfile, which is usually in the
// user's home directory. It doesn't guarantee that the file exists.
func rcfilePath(basename string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, basename)
}

// SetInit configures the shell to run a script at startup. The script runs
// after the user's usual init files. The script's environment will contain an
// ORIGINAL_PATH environment variable, which will bet set to the PATH before
// the user's init files have had a chance to modify it.
func (s *Shell) SetInit(script string) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil
	}

	initFile, _ := os.ReadFile(s.initFile)
	initFile = bytes.TrimSpace(initFile)
	if len(initFile) > 0 {
		initFile = append(initFile, '\n', '\n')
	}

	buf := bytes.NewBuffer(initFile)
	buf.WriteString(`

# Begin Devbox Shell Hook

`)
	buf.WriteString(script)
	buf.WriteString(`

# End Devbox Shell Hook
`)

	// We need a temp dir (as opposed to a temp file) because zsh uses
	// ZDOTDIR to point to a new directory containing the .zshrc.
	tmp, err := os.MkdirTemp("", "devbox")
	if err != nil {
		return fmt.Errorf("create temp dir for shell init file: %v", err)
	}
	devboxInitFile := filepath.Join(tmp, filepath.Base(s.initFile))
	if err := os.WriteFile(devboxInitFile, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("write to shell init file: %v", err)
	}
	s.devboxInitFile = devboxInitFile
	return nil
}

// ExecCommand is a command that replaces the current shell with s.
func (s *Shell) ExecCommand() string {
	if s.devboxInitFile == "" {
		return "exec " + s.path
	}

	switch s.name {
	case shBash:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" %s --rcfile "%s"`,
			os.Getenv("PATH"), s.path, s.devboxInitFile)
	case shZsh:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" ZDOTDIR="%s" %s`,
			os.Getenv("PATH"), filepath.Dir(s.devboxInitFile), s.path)
	case shKsh, shPosix:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" ENV="%s" %s `,
			os.Getenv("PATH"), s.devboxInitFile, s.path)
	default:
		return "exec " + s.path
	}
}
