// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nim

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "nim.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("*.nim", "*.nimble")
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {

	// TODO inspect .nimble file and check for constraints on the nim version to use
	// 1. `nim >= 1.6`. No problem, install the latest nim package as we do today.
	// 2. `nim == 1.6`. This is hard to do today, because we don't have a facility to install
	//     a specific version of nim package.

	cfg, err := parseNimbleConfig(srcDir)
	if err != nil {
		plan := plansdk.Plan{}
		return plan.WithError(err)
	}

	if len(cfg.bin) != 1 {
		plan := plansdk.Plan{}
		return plan.WithError(
			errors.Errorf(
				"Nim image cannot be built because the NimPlanner supports having exactly"+
					" one binary, but got %d binaries (%v)",
				len(cfg.bin),
				cfg.bin,
			),
		)
	}
	bin := cfg.bin[0]

	return &plansdk.Plan{
		DevPackages:     []string{"nim", "openssl"},
		RuntimePackages: []string{},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "nimble install -l",
		},
		BuildStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    "nimble build -l",
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{fmt.Sprintf("./%s%s", cfg.binDir, bin)},
			Command:    "./" + bin,
		},
	}
}

// https://github.com/nim-lang/nimble#nimble-reference
type nimbleConfig struct {
	binDir string
	bin    []string
}

func parseNimbleConfig(srcDir string) (*nimbleConfig, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}
	paths := a.GlobFiles("*.nimble")
	if len(paths) != 1 {
		return nil, errors.Errorf(
			"expected exactly one .nimble file in %s, but got %d .nimble files",
			srcDir,
			len(paths),
		)
	}
	contents, err := os.ReadFile(paths[0])
	if err != nil {
		return nil, err
	}

	cfg := &nimbleConfig{}
	for _, line := range strings.Split(string(contents), "\n") {
		if isBinLine(line) {
			cfg.bin = parseBinLine(line)
		} else if isBinDirLine(line) {
			cfg.binDir = parseBinDirLine(line)
		}
	}
	return cfg, nil
}

func isBinLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "bin")
}

func isBinDirLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "binDir")
}

func parseBinLine(line string) []string {
	binLocationsRegexStr := ".*bin.+=.*@\\[(.+)]"
	re := regexp.MustCompile(binLocationsRegexStr)
	matching := re.FindAllSubmatch([]byte(line), -1)
	if len(matching) != 1 {
		return nil
	}
	if len(matching[0]) != 2 {
		return nil
	}

	// matching[0][1] is the capturing group within the array-bracket notation of [...]
	// for example, if line is 'bin = @["hello_world", "second_world"]'
	//              then matching[0][1] is '"hello_world", "second_world"'
	parts := strings.Split(string(matching[0][1]), ",")
	for idx, part := range parts {

		parts[idx] = strings.Trim(part, "\" ")
	}
	return parts
}

func parseBinDirLine(line string) string {
	binLocationsRegexStr := ".*binDir.+=.*\"(.*)\""
	re := regexp.MustCompile(binLocationsRegexStr)
	matching := re.FindAllSubmatch([]byte(line), -1)
	if len(matching) != 1 {
		return ""
	}
	if len(matching[0]) != 2 {
		return ""
	}
	return string(matching[0][1])
}
