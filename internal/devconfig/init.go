// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"errors"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/devconfig/configfile"
)

func Init(dir string) (*Config, error) {
	file, err := os.OpenFile(
		filepath.Join(dir, configfile.DefaultName),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0o644,
	)
	if errors.Is(err, os.ErrExist) {
		return nil, err
	}
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
