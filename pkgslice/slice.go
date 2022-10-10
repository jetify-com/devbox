// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package pkgslice manipulates slices of devbox package names.
package pkgslice

func Unique(s []string) []string {
	deduped := make([]string, 0, len(s))
	seen := make(map[string]bool, len(s))
	for _, str := range s {
		if !seen[str] {
			deduped = append(deduped, str)
		}
		seen[str] = true
	}
	return deduped
}

func Exclude(s []string, elems []string) []string {
	excluded := make(map[string]bool, len(elems))
	for _, ex := range elems {
		excluded[ex] = true
	}

	filtered := make([]string, 0, len(s))
	for _, str := range s {
		if !excluded[str] {
			filtered = append(filtered, str)
		}
	}
	return filtered
}

// returns true when s includes all elements from elems.
func Contains(s []string, elems []string) bool {
	sMap := make(map[string]bool, len(s))
	for _, str := range s {
		sMap[str] = true
	}
	for _, e := range elems {
		if _, ok := sMap[e]; !ok {
			return false
		}
	}
	return true
}
