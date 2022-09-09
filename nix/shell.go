// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

//go:embed shellrc.tmpl
var shellrcText string
var shellrcTmpl = template.Must(template.New("shellrc").Parse(shellrcText))

type name string

const (
	shUnknown name = ""
	shBash    name = "bash"
	shZsh     name = "zsh"
	shKsh     name = "ksh"
	shPosix   name = "posix"
)

// Shell configures a user's shell to run in Devbox. Its zero value is a
// fallback shell that launches a regular Nix shell.
type Shell struct {
	name            name
	binPath         string
	userShellrcPath string

	// UserInitHook contains commands that will run at shell startup.
	UserInitHook string
}

// DetectShell attempts to determine the user's default shell.
func DetectShell() (*Shell, error) {
	path := os.Getenv("SHELL")
	if path == "" {
		return nil, errors.New("unable to detect the current shell")
	}

	sh := &Shell{binPath: filepath.Clean(path)}
	base := filepath.Base(path)
	// Login shell
	if base[0] == '-' {
		base = base[1:]
	}
	switch base {
	case "bash":
		sh.name = shBash
		sh.userShellrcPath = rcfilePath(".bashrc")
	case "zsh":
		sh.name = shZsh
		sh.userShellrcPath = rcfilePath(".zshrc")
	case "ksh":
		sh.name = shKsh
		sh.userShellrcPath = rcfilePath(".kshrc")
	case "dash", "ash", "sh":
		sh.name = shPosix
		sh.userShellrcPath = os.Getenv("ENV")

		// Just make up a name if there isn't already an init file set
		// so we have somewhere to put a new one.
		if sh.userShellrcPath == "" {
			sh.userShellrcPath = ".shinit"
		}
	default:
		sh.name = shUnknown
	}
	debug.Log("Detected shell: %s", sh.binPath)
	debug.Log("Recognized shell as: %s", sh.binPath)
	debug.Log("Looking for user's shell init file at: %s", sh.userShellrcPath)
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

func (s *Shell) Run(nixPath string) error {
	// Launch a fallback shell if we couldn't find the path to the user's
	// default shell.
	if s.binPath == "" {
		cmd := exec.Command("nix-shell", nixPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		debug.Log("Unrecognized user shell, falling back to: %v", cmd.Args)
		return errors.WithStack(cmd.Run())
	}

	cmd := exec.Command("nix-shell", nixPath)
	cmd.Args = append(cmd.Args, "--pure", "--command", s.execCommand())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	debug.Log("Executing nix-shell command: %v", cmd.Args)
	return errors.WithStack(cmd.Run())
}

// execCommand is a command that replaces the current shell with s.
func (s *Shell) execCommand() string {
	shellrc, err := writeDevboxShellrc(s.userShellrcPath, s.UserInitHook)
	if err != nil {
		debug.Log("Failed to write devbox shellrc: %v", err)
		return "exec " + s.binPath
	}

	switch s.name {
	case shBash:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" %s --rcfile "%s"`,
			os.Getenv("PATH"), s.binPath, shellrc)
	case shZsh:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" ZDOTDIR="%s" %s`,
			os.Getenv("PATH"), filepath.Dir(shellrc), s.binPath)
	case shKsh, shPosix:
		return fmt.Sprintf(`exec /usr/bin/env ORIGINAL_PATH="%s" ENV="%s" %s `,
			os.Getenv("PATH"), shellrc, s.binPath)
	default:
		return "exec " + s.binPath
	}
}

func writeDevboxShellrc(userShellrcPath string, userHook string) (path string, err error) {
	// We need a temp dir (as opposed to a temp file) because zsh uses
	// ZDOTDIR to point to a new directory containing the .zshrc.
	tmp, err := os.MkdirTemp("", "devbox")
	if err != nil {
		return "", fmt.Errorf("create temp dir for shell init file: %v", err)
	}

	// This is a best-effort to include the user's existing shellrc. If we
	// can't read it, then just omit it from the devbox shellrc.
	userShellrc, err := os.ReadFile(userShellrcPath)
	if err != nil {
		userShellrc = []byte{}
	}

	path = filepath.Join(tmp, filepath.Base(userShellrcPath))
	shellrcf, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("write to shell init file: %v", err)
	}
	defer func() {
		cerr := shellrcf.Close()
		if err == nil {
			err = cerr
		}
	}()

	err = shellrcTmpl.Execute(shellrcf, struct {
		OriginalInit     string
		OriginalInitPath string
		UserHook         string
	}{
		OriginalInit:     string(bytes.TrimSpace(userShellrc)),
		OriginalInitPath: filepath.Clean(userShellrcPath),
		UserHook:         strings.TrimSpace(userHook),
	})
	if err != nil {
		return "", fmt.Errorf("execute shellrc template: %v", err)
	}

	debug.Log("Wrote devbox shellrc to: %s", path)
	return path, nil
}
