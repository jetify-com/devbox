// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/cmdutil"
)

func CopyAll(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range entries {
		cmd := cmdutil.CommandTTY("cp", "-rf", filepath.Join(src, entry.Name()), dst)
		if err := cmd.Run(); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func ClearDir(dir string) error {
	// if the dir doesn't exist, use default filemode 0755 to create it
	// if the dir exists, use its own filemode to re-create it
	var mode os.FileMode
	f, err := os.Stat(dir)
	if err == nil {
		mode = f.Mode()
	} else if errors.Is(err, fs.ErrNotExist) {
		mode = 0o755
	} else {
		return errors.WithStack(err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(os.MkdirAll(dir, mode))
}

func CreateDevboxTempDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "devbox")
	return tmpDir, errors.WithStack(err)
}
