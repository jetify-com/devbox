// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"errors"
	"fmt"
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

// MatchVersion picks the version from available that best satisfies
// requested, following the rules in the spec:
//
//   - "latest" picks the highest non-pre-release version.
//   - An exact match wins when present.
//   - Otherwise requested is a prefix: "1.22" matches the highest
//     "1.22.x" that is not a pre-release.
//
// A version is a pre-release when it contains a letter or a hyphen, for
// example "1.23.0-rc1" or "1.21rc2".
func MatchVersion(available []string, requested string) (string, error) {
	if requested == "" {
		requested = Latest
	}
	if slices.Contains(available, requested) {
		return requested, nil
	}

	best := ""
	for _, v := range available {
		if v == "" || IsPrerelease(v) {
			continue
		}
		if requested != Latest && !strings.HasPrefix(v, requested+".") {
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
