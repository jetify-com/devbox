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
	shellrc, err := writeDevboxShellrc(s.userShellrcPath, s.UserInitHook, os.Environ())
	if err != nil {
		debug.Log("Failed to write devbox shellrc: %v", err)
		return "exec " + s.binPath
	}

	switch s.name {
	case shBash:
		return fmt.Sprintf(`exec %s --rcfile "%s"`, s.binPath, shellrc)
	case shZsh:
		return fmt.Sprintf(`exec /usr/bin/env ZDOTDIR="%s" %s`, filepath.Dir(shellrc), s.binPath)
	case shKsh, shPosix:
		return fmt.Sprintf(`exec /usr/bin/env ENV="%s" %s`, shellrc, s.binPath)
	default:
		return "exec " + s.binPath
	}
}

func writeDevboxShellrc(userShellrcPath string, userHook string, env []string) (path string, err error) {
	if userShellrcPath == "" {
		// If this happens, then there's a bug with how we detect shells
		// and their shellrc paths. If the shell is unknown or we can't
		// determine the shellrc path, then we should launch a fallback
		// shell instead.
		panic("writeDevboxShellrc called with an empty user shellrc path; use the fallback shell instead")
	}

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

	var envPath []string
	for _, kv := range env {
		key, val, _ := strings.Cut(kv, "=")
		if key == "PATH" {
			envPath = filepath.SplitList(val)
			break
		}
	}

	// If the user already has a shellrc file, then give the devbox shellrc
	// file the same name. Otherwise, use an arbitrary name of "shellrc".
	shellrcName := "shellrc"
	if userShellrcPath != "" {
		shellrcName = filepath.Base(userShellrcPath)
	}
	path = filepath.Join(tmp, shellrcName)
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
		Paths            []string
		OriginalInit     string
		OriginalInitPath string
		UserHook         string
	}{
		Paths:            envPath,
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
