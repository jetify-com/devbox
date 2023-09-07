// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package envir

import (
	"os"
	"strconv"
)

func IsDevboxCloud() bool {
	return os.Getenv(DevboxRegion) != ""
}

func IsDevboxShellEnabled() bool {
	inDevboxShell, _ := strconv.ParseBool(os.Getenv(DevboxShellEnabled))
	return inDevboxShell
}

func DoNotTrack() bool {
	// https://consoledonottrack.com/
	doNotTrack, _ := strconv.ParseBool(os.Getenv("DO_NOT_TRACK"))
	return doNotTrack
}

func IsInBrowser() bool { // TODO: a better name
	inBrowser, _ := strconv.ParseBool(os.Getenv("START_WEB_TERMINAL"))
	return inBrowser
}

func IsCI() bool {
	ci, err := strconv.ParseBool(os.Getenv("CI"))
	return ci && err == nil
}

// GetValueOrDefault gets the value of an environment variable.
// If it's empty, it will return the given default value instead.
func GetValueOrDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}

	return val
}
