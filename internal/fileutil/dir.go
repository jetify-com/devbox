// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
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
	f, err := os.Stat(dir)
	if err != nil {
		return errors.WithStack(err)
	}
	mode := f.Mode()

	if err := os.RemoveAll(dir); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(os.MkdirAll(dir, mode))
}

func CreateDevboxTempDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", "devbox")
	return tmpDir, errors.WithStack(err)
}
