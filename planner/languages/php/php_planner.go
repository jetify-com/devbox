// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package php

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/planner/plansdk"
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

type Planner struct{}

// PHPPlanner implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "php.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "composer.lock")) ||
		plansdk.FileExists(filepath.Join(srcDir, "composer.json"))
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	v := p.version(srcDir)
	plan := &plansdk.Plan{
		DevPackages: []string{
			fmt.Sprintf("php%s", v.MajorMinorConcatenated()),
			fmt.Sprintf("php%sPackages.composer", v.MajorMinorConcatenated()),
		},
		RuntimePackages: []string{
			fmt.Sprintf("php%s", v.MajorMinorConcatenated()),
			fmt.Sprintf("php%sPackages.composer", v.MajorMinorConcatenated()),
		},
	}
	if !plansdk.FileExists(filepath.Join(srcDir, "public/index.php")) {
		return plan.WithError(usererr.New("Can't build. No public/index.php found."))
	}

	plan.InstallStage = &plansdk.Stage{
		Command: "composer install --no-dev --no-ansi",
	}
	plan.StartStage = &plansdk.Stage{
		Command: "php -S 0.0.0.0:8080 -t public",
	}
	return plan
}

func (p *Planner) version(srcDir string) *plansdk.Version {
	latestVersion, _ := plansdk.NewVersion(supportedPHPVersions[0])
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

	version, err := plansdk.NewVersion(composerJSON.Config.Platform.PHP)
	if err != nil {
		return latestVersion
	}

	// Look for major minor match first.
	for _, supportedVersion := range supportedPHPVersions {
		if strings.HasPrefix(supportedVersion, version.MajorMinor()) {
			return version
		}
	}

	// If no major minor match, just try to find a major match.
	for _, supportedVersion := range supportedPHPVersions {
		if strings.HasPrefix(supportedVersion, version.Major()) {
			return version
		}
	}

	// Old version of php detected. They'll need to make changes regardless, we
	// might as well pick the latest version.
	return latestVersion
}
