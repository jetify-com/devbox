// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package php

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
		Definitions: p.definitions(srcDir, v),
	}
	if !plansdk.FileExists(filepath.Join(srcDir, "public/index.php")) {
		return plan.WithError(usererr.New("Can't build. No public/index.php found."))
	}

	plan.InstallStage = &plansdk.Stage{
		InputFiles: []string{"."},
		Command:    "composer install --no-dev --no-ansi",
	}
	plan.StartStage = &plansdk.Stage{
		InputFiles: []string{"."},
		Command:    "php -S 0.0.0.0:8080 -t public",
	}
	return plan
}

type composerPackages struct {
	Config struct {
		Platform struct {
			PHP string `json:"php"`
		} `json:"platform"`
	} `json:"config"`
	Require map[string]string `json:"require"`
}

func (p *Planner) version(srcDir string) *plansdk.Version {
	latestVersion, _ := plansdk.NewVersion(supportedPHPVersions[0])
	project, err := p.parseComposerPackages(srcDir)

	if err != nil {
		return latestVersion
	}

	composerPHPVersion := project.Require["php"]
	if composerPHPVersion == "" {
		composerPHPVersion = project.Config.Platform.PHP
	}

	if composerPHPVersion == "" {
		return latestVersion
	}

	version, err := plansdk.NewVersion(composerPHPVersion)
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

func (p *Planner) definitions(srcDir string, v *plansdk.Version) []string {
	extensions, err := p.extensions(srcDir)
	if len(extensions) == 0 || err != nil {
		return []string{}
	}
	return []string{
		fmt.Sprintf(
			"php%s = pkgs.php%s.withExtensions ({ enabled, all }: enabled ++ (with all; [ %s ]));",
			v.MajorMinorConcatenated(),
			v.MajorMinorConcatenated(),
			strings.Join(extensions, " "),
		),
	}
}

func (p *Planner) extensions(srcDir string) ([]string, error) {
	project, err := p.parseComposerPackages(srcDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	extensions := []string{}
	for requirement := range project.Require {
		if strings.HasPrefix(requirement, "ext-") {
			name := strings.Split(requirement, "-")[1]
			if name != "" && name != "json" {
				extensions = append(extensions, name)
			}
		}
	}

	return extensions, nil
}

func (p *Planner) parseComposerPackages(srcDir string) (*composerPackages, error) {
	composerJSONPath := filepath.Join(srcDir, "composer.json")
	content, err := os.ReadFile(composerJSONPath)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	project := &composerPackages{}
	return project, errors.WithStack(json.Unmarshal(content, project))
}
