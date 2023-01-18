// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package impl

import (
	"os/exec"

	"github.com/pkg/errors"
)

type TestDevbox struct {
	// env         string
	// commit_hash string
	// packages    map[string]string
	// ... other information needed to create an example environment for a devbox project
}

func (td *TestDevbox) Info(pkg string, markdown bool) (string, error) {
	cmd := exec.Command("devbox", "info", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(output), nil
}

func Open() *TestDevbox {
	return &TestDevbox{}
}
