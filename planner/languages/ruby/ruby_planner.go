// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package ruby

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/planner/plansdk"
	"golang.org/x/mod/semver"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

var nixPackages = map[string]string{
	"3.1": "ruby_3_1",
	"3.0": "ruby_3_0",
	"2.7": "ruby",
}

const defaultPkg = "ruby_3_1"

func (p *Planner) Name() string {
	return "ruby.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "Gemfile"))
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	gemfile := filepath.Join(srcDir, "Gemfile")
	v := parseRubyVersion(gemfile)
	pkg, ok := nixPackages[semver.MajorMinor(v)]
	if !ok {
		pkg = defaultPkg
	}

	return &plansdk.ShellPlan{
		DevPackages: []string{
			pkg,
			"gcc",     // for rails
			"gnumake", // for rails
		},
		ShellInitHook: []string{plansdk.WelcomeMessage(
			"It looks like you are developing a Ruby project.\n" +
				"To keep dependencies isolated, it is recommended that you install them in deployment mode, by running:\n" +
				" > bundler config set --local deployment 'true'\n" +
				" > bundler install\n" +
				"And then run your ruby app with bundler. For example:\n" +
				" > bundler exec ruby app.rb")},
	}
}

func (p *Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	gemfile := filepath.Join(srcDir, "Gemfile")
	v := parseRubyVersion(gemfile)
	pkg, ok := nixPackages[semver.MajorMinor(v)]
	if !ok {
		pkg = defaultPkg
	}
	cmd := "bundle exec ruby app.rb"
	if hasRails(gemfile) {
		cmd = "bin/rails server -b 0.0.0.0 -e production"
	}
	return &plansdk.BuildPlan{
		DevPackages: []string{
			pkg,
			"gcc",     // for rails
			"gnumake", // for rails
		},
		RuntimePackages: []string{
			pkg,
		},
		InstallStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command:    "bundle config set --local deployment 'true' && bundle install",
		},
		StartStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command:    cmd,
		},
	}
}

var rubyVersionRegex = regexp.MustCompile(`ruby\s+"(<|>|<=|>=|~>|=|)\s*([\d|\\.]+)"`)

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

func hasRails(gemfile string) bool {
	c, err := os.ReadFile(gemfile)
	if err != nil {
		return false
	}
	return strings.Contains(string(c), "gem \"rails\"")
}
