// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagenbox

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const shimDirPath = ".config/devbox/ssh/shims"

func ShimDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	shimDir := filepath.Join(home, shimDirPath)
	return shimDir, nil
}
