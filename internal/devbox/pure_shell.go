// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/xdg"
)

// findNixInPATH looks for locations in PATH which nix might exist and
// it returns a slice containing all paths that might contain nix.
// For single-user, and multi-user installation there are default locations
// unless XDG_* env variables are set. So we look for nix in 3 locations
// to see if any of those exist in path.
func findNixInPATH(env map[string]string) ([]string, error) {
	defaultSingleUserNixBin := fmt.Sprintf("%s/.nix-profile/bin", env["HOME"])
	defaultMultiUserNixBin := "/nix/var/nix/profiles/default/bin"
	xdgNixBin := xdg.StateSubpath("/nix/profile/bin")
	pathElements := strings.Split(env["PATH"], ":")
	debug.Log("path elements: %v", pathElements)
	nixBinsInPath := []string{}
	for _, el := range pathElements {
		if el == xdgNixBin ||
			el == defaultSingleUserNixBin ||
			el == defaultMultiUserNixBin {
			nixBinsInPath = append(nixBinsInPath, el)
		}
	}

	if len(nixBinsInPath) == 0 {
		// did not find nix executable in PATH, return error
		return nil, errors.New("could not find any nix executable in PATH. Make sure Nix is installed and in PATH, then try again")
	}
	return nixBinsInPath, nil
}

// Creates a symlink for devbox in .devbox/bin
// so that devbox can be available inside a pure shell
func createDevboxSymlink(d *Devbox) error {
	// Get absolute path for where devbox is called
	devboxPath, err := filepath.Abs(os.Args[0])
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
