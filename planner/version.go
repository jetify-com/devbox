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
	r := regexp.MustCompile("^\\^?([0-9]+)(\\.[0-9]+)?(\\.[0-9]+)?$")
	return r.FindStringSubmatch(string(v))
}

func (v version) exact() string {
	parts := v.parts()
	if len(parts) > 0 {
		return strings.Join(parts[1:], "")
	}
	return ""
}

func (v version) majorMinorConcatenated() string {
	parts := v.parts()
	if len(parts) > 0 && len(parts) < 3 {
		return strings.ReplaceAll(strings.Join(parts[1:], ""), ".", "")
	} else if len(parts) > 0 {
		return strings.ReplaceAll(strings.Join(parts[1:3], ""), ".", "")
	}
	return ""
}
