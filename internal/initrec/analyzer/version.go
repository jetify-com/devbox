// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package analyzer

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Version handles very simple numeric semver versions (e.g. "1.2.3")
type Version string

func NewVersion(v string) (*Version, error) {
	ver := Version(v)
	if ver.Exact() == "" {
		return nil, errors.New("invalid version")
	}
	return &ver, nil
}

func (v Version) parts() []string {
	// This regex allows starting versions with ^ or >=
	// It ignored anything after a comma (including the comma)
	// Maybe consider using https://github.com/aquasecurity/go-pep440-version
	// or equivalent
	r := regexp.MustCompile(`^(?:\^|>=)?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(?:,.*)?$`)
	groups := r.FindStringSubmatch(string(v))
	if len(groups) > 0 {
		return groups[1:]
	}
	return nil
}

func (v Version) Exact() string {
	return strings.Join(v.parts(), "")
}

func (v Version) Major() string {
	parts := v.parts()
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (v Version) MajorMinor() string {
	parts := v.parts()
	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 1 {
		return strings.Join(parts[:2], "")
	}
	return parts[0]
}

func (v Version) MajorMinorConcatenated() string {
	return strings.ReplaceAll(v.MajorMinor(), ".", "")
}
