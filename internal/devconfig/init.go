// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"io/fs"
	"os"
	"path/filepath"

	"go.jetify.com/devbox/internal/devconfig/configfile"
)

func Init(dir string) (*Config, error) {
	// A project may already be configured under an alternate filename
	// (devbox.jsonc). Creating devbox.json next to it would silently take
	// precedence, so treat any recognized config name as "already exists".
	for _, name := range configfile.ValidNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
		}
	}

	file, err := os.OpenFile(
		filepath.Join(dir, configfile.DefaultName),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0o644,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.Remove(file.Name())
		}
	}()

	newConfig := DefaultConfig()
	_, err = file.Write(newConfig.Root.Bytes())
	defer file.Close()
	if err != nil {
		return nil, err
	}
	return newConfig, nil
}
