// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/boxcli/usererr"
)

// https://github.com/NixOS/nixpkgs/tree/nixos-22.05/pkgs/development/interpreters/php
// These seem to change somewhat frequently, so we may need an automated way to
// keep this in sync.
//
// Keep reverse sorted
var supportedPHPVersions = []string{
	"8.1",
	"8.0",
	"7.4",
}

type PHPPlanner struct{}

// PHPPlanner implements interface Planner (compile-time check)
var _ Planner = (*PHPPlanner)(nil)

func (g *PHPPlanner) Name() string {
	return "PHPPlanner"
}

func (g *PHPPlanner) IsRelevant(srcDir string) bool {
	return fileExists(filepath.Join(srcDir, "composer.lock")) ||
		fileExists(filepath.Join(srcDir, "composer.json"))
}

func (g *PHPPlanner) GetPlan(srcDir string) *Plan {
	v := g.version(srcDir)
	plan := &Plan{
		DevPackages: []string{
			fmt.Sprintf("php%s", v.majorMinorConcatenated()),
			fmt.Sprintf("php%sPackages.composer", v.majorMinorConcatenated()),
		},
		RuntimePackages: []string{
			fmt.Sprintf("php%s", v.majorMinorConcatenated()),
			fmt.Sprintf("php%sPackages.composer", v.majorMinorConcatenated()),
		},
	}
	if !fileExists(filepath.Join(srcDir, "public/index.php")) {
		return plan.WithError(usererr.New("Can't build. No public/index.php found."))
	}

	plan.InstallStage = &Stage{
		Command: "composer install --no-dev --no-ansi",
	}
	plan.StartStage = &Stage{
		Command: "php -S 0.0.0.0:8080 -t public",
	}
	return plan
}

func (g *PHPPlanner) version(srcDir string) *version {
	latestVersion, _ := newVersion(supportedPHPVersions[0])
	composerJSONPath := filepath.Join(srcDir, "composer.json")
	content, err := os.ReadFile(composerJSONPath)

	if err != nil {
		return latestVersion
	}

	composerJSON := struct {
		Config struct {
			Platform struct {
				PHP string `json:"php"`
			} `json:"platform"`
		} `json:"config"`
	}{}
	if err := json.Unmarshal(content, &composerJSON); err != nil ||
		composerJSON.Config.Platform.PHP == "" {
		return latestVersion
	}

	version, err := newVersion(composerJSON.Config.Platform.PHP)
	if err != nil {
		return latestVersion
	}

	// Look for major minor match first.
	for _, supportedVersion := range supportedPHPVersions {
		if strings.HasPrefix(supportedVersion, version.majorMinor()) {
			return version
		}
	}

	// If no major minor match, just try to find a major match.
	for _, supportedVersion := range supportedPHPVersions {
		if strings.HasPrefix(supportedVersion, version.major()) {
			return version
		}
	}

	// Old version of php detected. They'll need to make changes regardless, we
	// might as well pick the latest version.
	return latestVersion
}
