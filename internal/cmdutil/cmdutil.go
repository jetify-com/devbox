// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cmdutil

import (
	"os/exec"
)

// Exists indicates if the command exists
func Exists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetPathOrDefault gets the path for the given command.
// If it's not found, it will return the given value instead.
func GetPathOrDefault(command, def string) string {
	path, err := exec.LookPath(command)
	if err != nil {
		path = def
	}

	return path
}
