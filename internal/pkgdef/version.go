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
//     "1.22.x" that is not a pre-release, and "21.0.5" matches
//     "21.0.5+11" (build metadata).
//
// A version is a pre-release when the part before any "+build" suffix
// contains a letter or a hyphen, for example "1.23.0-rc1" or "1.21rc2".
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
		if requested != Latest && !hasVersionPrefix(v, requested) {
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

// hasVersionPrefix reports whether v is a more specific version of prefix:
// "1.22.3" and "1.22+4" have the prefix "1.22", "1.2.3" does not have "1.22".
func hasVersionPrefix(v, prefix string) bool {
	return strings.HasPrefix(v, prefix+".") || strings.HasPrefix(v, prefix+"+")
}

// IsPrerelease reports whether v looks like a pre-release version. Build
// metadata after "+" does not count.
func IsPrerelease(v string) bool {
	core, _, _ := strings.Cut(v, "+")
	if strings.Contains(core, "-") {
		return true
	}
	for _, r := range core {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// CompareVersions orders two version strings. Dot-separated segments are
// compared numerically when both are numbers and lexically otherwise. A
// version that is a prefix of another sorts first ("1.22" < "1.22.1").
// Build metadata after "+" breaks ties between equal versions, so a
// vendor's respin ("21.0.12+101") sorts after the original ("21.0.12+8").
func CompareVersions(a, b string) int {
	aCore, aBuild, _ := strings.Cut(a, "+")
	bCore, bBuild, _ := strings.Cut(b, "+")
	if c := compareDotted(aCore, bCore); c != 0 {
		return c
	}
	return compareDotted(aBuild, bBuild)
}

func compareDotted(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" || b == "" {
		return len(a) - len(b)
	}
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		if c := compareSegment(as[i], bs[i]); c != 0 {
			return c
		}
	}
	return len(as) - len(bs)
}

// MatchesConstraint reports whether version satisfies constraint, a list
// of comparisons separated by spaces or commas that must all hold, such as
// "<11" or ">=2, <4". Operators are <, <=, >, >=, = and !=. A bare version
// matches itself and every more specific version ("1.22" matches
// "1.22.3"). Comparison uses CompareVersions, so "<11" matches "10.9.9"
// but not "11.0.0". An invalid constraint matches nothing.
func MatchesConstraint(version, constraint string) bool {
	terms, err := parseConstraint(constraint)
	if err != nil {
		return false
	}
	for _, t := range terms {
		if !t.matches(version) {
			return false
		}
	}
	return true
}

func validateConstraint(constraint string) error {
	_, err := parseConstraint(constraint)
	return err
}

type constraintTerm struct {
	op      string
	version string
}

func (t constraintTerm) matches(version string) bool {
	cmp := CompareVersions(version, t.version)
	switch t.op {
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "!=":
		return cmp != 0
	case "=":
		return cmp == 0
	default: // bare version
		return version == t.version || hasVersionPrefix(version, t.version)
	}
}

func parseConstraint(constraint string) ([]constraintTerm, error) {
	fields := strings.FieldsFunc(constraint, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) == 0 {
		return nil, errors.New("constraint is empty")
	}
	terms := make([]constraintTerm, 0, len(fields))
	for _, field := range fields {
		term := constraintTerm{}
		for _, op := range []string{"<=", ">=", "!=", "<", ">", "="} {
			if rest, ok := strings.CutPrefix(field, op); ok {
				term.op, field = op, rest
				break
			}
		}
		if field == "" || strings.ContainsAny(field, "<>=!") {
			return nil, fmt.Errorf("invalid term in version constraint %q", constraint)
		}
		term.version = field
		terms = append(terms, term)
	}
	return terms, nil
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
