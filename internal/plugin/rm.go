// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func Remove(projectDir string, pkgs []string) error {
	for _, pkg := range pkgs {
		if err := os.RemoveAll(filepath.Join(projectDir, VirtenvPath, pkg)); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func RemoveInvalidSymlinks(projectDir string) error {
	binPath := filepath.Join(projectDir, VirtenvBinPath)
	if _, err := os.Stat(binPath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	dirEntry, err := os.ReadDir(binPath)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range dirEntry {
		_, err := os.Stat(filepath.Join(projectDir, VirtenvBinPath, entry.Name()))
		if errors.Is(err, fs.ErrNotExist) {
			os.Remove(filepath.Join(projectDir, VirtenvBinPath, entry.Name()))
		}
	}
	return nil
}
