// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Handles very simple numeric semver versions (e.g. "1.2.3")
type version string

func newVersion(v string) (*version, error) {
	ver := version(v)
	if ver.exact() == "" {
		return nil, errors.New("invalid version")
	}
	return &ver, nil
}

func (v version) parts() []string {
	// This regex allows starting versions with ^ or >=
	// It ignored anything after a comma (including the comma)
	// Maybe consider using https://github.com/aquasecurity/go-pep440-version
	// or equivalent
	r := regexp.MustCompile(`^(?:\^|>=)?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(?:,.*)?$`)
	groups := r.FindStringSubmatch(string(v))
	if len(groups) > 0 {
		return groups[1:]
	}
	return []string{}
}

func (v version) exact() string {
	parts := v.parts()
	if len(parts) > 0 {
		return strings.Join(parts, "")
	}
	return ""
}

func (v version) major() string {
	parts := v.parts()
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (v version) majorMinor() string {
	parts := v.parts()
	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 1 {
		return strings.Join(parts[:2], "")
	}
	return parts[0]
}

func (v version) majorMinorConcatenated() string {
	return strings.ReplaceAll(v.majorMinor(), ".", "")
}
