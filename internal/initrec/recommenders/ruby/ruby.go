// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package ruby

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"

	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"golang.org/x/mod/semver"
)

type Recommender struct {
	SrcDir string
}

// implements interface Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

var nixPackages = map[string]string{
	"3.1": "ruby_3_1",
	"3.0": "ruby_3_0",
	"2.7": "ruby",
}

const defaultPkg = "ruby_3_1"

var rubyVersionRegex = regexp.MustCompile(`ruby\s+"(<|>|<=|>=|~>|=|)\s*([\d|\\.]+)"`)

func (r *Recommender) IsRelevant() bool {
	return plansdk.FileExists(filepath.Join(r.SrcDir, "Gemfile"))
}

func (r *Recommender) Packages() []string {
	gemfile := filepath.Join(r.SrcDir, "Gemfile")
	v := parseRubyVersion(gemfile)
	pkg, ok := nixPackages[semver.MajorMinor(v)]
	if !ok {
		pkg = defaultPkg
	}
	return []string{
		pkg,
		"gcc",     // for rails
		"gnumake", // for rails
	}
}

func parseRubyVersion(gemfile string) string {
	f, err := os.Open(gemfile)
	if err != nil {
		return ""
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		matches := rubyVersionRegex.FindStringSubmatch(line)
		if matches != nil {
			// TODO: return and use comparator as well.
			return matches[2]
		}
	}
	if err := s.Err(); err != nil {
		return ""
	}
	return "" // not found
}
