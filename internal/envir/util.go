// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package envir

import (
	"os"
	"slices"
	"strconv"
	"strings"
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

// MapToPairs creates a slice of environment variable "key=value" pairs from a
// map.
func MapToPairs(m map[string]string) []string {
	pairs := make([]string, len(m))
	i := 0
	for k, v := range m {
		pairs[i] = k + "=" + v
		i++
	}
	slices.Sort(pairs) // for reproducibility
	return pairs
}

// PairsToMap creates a map from a slice of "key=value" environment variable
// pairs. Note that maps are not ordered, which can affect the final variable
// values when pairs contains duplicate keys.
func PairsToMap(pairs []string) map[string]string {
	vars := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		vars[k] = v
	}
	return vars
}
