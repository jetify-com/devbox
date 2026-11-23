// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Latest is the version constraint that resolves to the newest stable
// version.
const Latest = "latest"

// ErrVersionNotFound is returned when no available version satisfies the
// requested constraint.
var ErrVersionNotFound = errors.New("version not found")

// versionRegexp is what every version must look like. Versions become path
// components of the install directory and are rendered into templates
// (URLs, post_install hooks), so they may not contain "/", whitespace or
// shell metacharacters.
var versionRegexp = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]*$`)

// IsValidVersion reports whether v is an acceptable version string. Version
// sources drop versions that are not, and requested versions must be.
func IsValidVersion(v string) bool {
	return versionRegexp.MatchString(v)
}

// MatchVersion picks the version from available that best satisfies
// requested, following the rules in the spec:
//
//   - "latest" picks the highest non-pre-release version.
//   - Otherwise the candidates are an exact match, if present, and every
//     version that has requested as a prefix ("1.22" matches "1.22.x")
//     and is not a pre-release. The highest candidate wins, so "1.20"
//     picks "1.20.14" even when the vendor names 1.20.0 "1.20".
//
// A version is a pre-release when it contains a letter or a hyphen, for
// example "1.23.0-rc1" or "1.21rc2". Pre-releases only match exactly.
// Versions that are not IsValidVersion are ignored.
func MatchVersion(available []string, requested string) (string, error) {
	if requested == "" {
		requested = Latest
	}
	if !IsValidVersion(requested) {
		return "", fmt.Errorf("%w: invalid version %q", ErrVersionNotFound, requested)
	}

	best := ""
	for _, v := range available {
		if !IsValidVersion(v) {
			continue
		}
		exact := requested != Latest && v == requested
		if !exact && IsPrerelease(v) {
			continue
		}
		if requested != Latest && !exact && !strings.HasPrefix(v, requested+".") {
			continue
		}
		if best == "" || CompareVersions(v, best) > 0 {
			best = v
		}
	}
	if best == "" {
		return "", fmt.Errorf("%w: no version matches %q (available: %s)",
			ErrVersionNotFound, requested, summarize(available))
	}
	return best, nil
}

// IsPrerelease reports whether v looks like a pre-release version.
func IsPrerelease(v string) bool {
	if strings.Contains(v, "-") {
		return true
	}
	for _, r := range v {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// CompareVersions orders two version strings. Dot-separated segments are
// compared numerically when both are numbers and lexically otherwise. A
// version that is a prefix of another sorts first ("1.22" < "1.22.1").
func CompareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		if c := compareSegment(as[i], bs[i]); c != 0 {
			return c
		}
	}
	return len(as) - len(bs)
}

func compareSegment(a, b string) int {
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return an - bn
	}
	return strings.Compare(a, b)
}

// SortVersions sorts versions from newest to oldest.
func SortVersions(versions []string) {
	slices.SortStableFunc(versions, func(a, b string) int {
		return CompareVersions(b, a)
	})
}

func summarize(versions []string) string {
	if len(versions) == 0 {
		return "none"
	}
	sorted := slices.Clone(versions)
	SortVersions(sorted)
	const limit = 5
	if len(sorted) > limit {
		return strings.Join(sorted[:limit], ", ") + fmt.Sprintf(", and %d more", len(sorted)-limit)
	}
	return strings.Join(sorted, ", ")
}
