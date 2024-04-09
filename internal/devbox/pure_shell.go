// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

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
