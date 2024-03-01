// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// findNixInPATH looks for locations in PATH which nix might exist and
// it returns the path that contains nix.
func findNixInPATH() (string, error) {
	path, err := exec.LookPath("nix")
	if err != nil {
		if errors.Is(err, exec.ErrDot) {
			err = nil
			workingDirectory, err := os.Getwd()
			if err != nil {
				return "", errors.New("could not find any nix executable in PATH. Make sure Nix is installed and in PATH, then try again")
			}
			path = workingDirectory
		}
		if err != nil {
			return "", errors.New("could not find any nix executable in PATH. Make sure Nix is installed and in PATH, then try again")
		}
	} else {
		path = filepath.Dir(path)
	}

	debug.Log("found nix in PATH: %s", path)
	return path, nil
}

// Creates a symlink for devbox in .devbox/bin
// so that devbox can be available inside a pure shell
func createDevboxSymlink(d *Devbox) error {
	// Get absolute path for where devbox is called
	devboxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "failed to create devbox symlink. Devbox command won't be available inside the shell")
	}
	// ensure .devbox/bin directory exists
	binPath := dotdevboxBinPath(d)
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		return errors.WithStack(err)
	}
	// Create a symlink between devbox and .devbox/bin
	err = os.Symlink(devboxPath, filepath.Join(binPath, "devbox"))
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.Wrap(err, "failed to create devbox symlink. Devbox command won't be available inside the shell")
	}
	return nil
}

func dotdevboxBinPath(d *Devbox) string {
	return filepath.Join(d.ProjectDir(), ".devbox/bin")
}
