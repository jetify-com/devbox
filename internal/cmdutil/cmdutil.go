// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cmdutil

import (
	"os/exec"
	"time"
)

// Exists indicates if the command exists
func Exists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetPathOrDefault gets the path for the given command.
// If it's not found, it will return the given value instead.
func GetPathOrDefault(command string, def string) string {
	path, err := exec.LookPath(command)
	if err != nil {
		path = def
	}

	return path
}

// WithRetry retries the given function for at most retries times.
// You can adjust the wait time in your function.
func WithRetry(retries int, fn func(round int) (time.Duration, error)) error {
	var finalErr error
	for num := 0; num < retries; num++ {
		wait, err := fn(num)
		if err == nil {
			return nil
		}
		finalErr = err
		time.Sleep(wait)
	}

	return finalErr
}
