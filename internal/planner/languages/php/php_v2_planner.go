// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package php

import (
	"fmt"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/planner/plansdk"
)

var composerPackageRegex = regexp.MustCompile(`^php\d\dPackages.composer$`)

type V2Planner struct {
	userPackages []string
}

// PHPV2Planner implements interface PlannerForPackages (compile-time check)
var _ plansdk.PlannerForPackages = (*V2Planner)(nil)

func (p *V2Planner) Name() string {
	return "php.v2.Planner"
}

func (p *V2Planner) IsRelevant(srcDir string) bool {
	return false
}

func (p *V2Planner) IsRelevantForPackages(packages []string) bool {
	p.userPackages = packages
	return p.getPHPPackage() != "" && len(p.getExtensions()) > 0
}

func (p *V2Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	phpPackage := p.getPHPPackage()
	definitions := []string{
		fmt.Sprintf(
			"%s = pkgs.%s.withExtensions ({ enabled, all }: enabled ++ (with all; [ %s ]));",
			phpPackage,
			phpPackage,
			strings.Join(p.getExtensions(), " "),
		),
	}

	if composerPackage := p.getComposerPackage(); composerPackage != "" {
		definitions = append(
			definitions,
			fmt.Sprintf("%s = %s.packages.composer;", composerPackage, phpPackage),
		)
	}

	return &plansdk.ShellPlan{Definitions: definitions}
}

func (p *V2Planner) GetBuildPlan(srcDir string) *plansdk.BuildPlan {
	return nil
}

func (p *V2Planner) getPHPPackage() string {
	regexp := regexp.MustCompile(`^php[0-9]*$`)
	for _, pkg := range p.userPackages {
		if regexp.Match([]byte(pkg)) {
			return pkg
		}
	}
	return ""
}

func (p *V2Planner) getComposerPackage() string {
	for _, pkg := range p.userPackages {
		if composerPackageRegex.Match([]byte(pkg)) {
			return pkg
		}
	}
	return ""
}

func (p *V2Planner) getExtensions() []string {
	regexp := regexp.MustCompile(`^php[0-9]*Extensions\..+$`)
	var extensions []string
	for _, pkg := range p.userPackages {
		if regexp.Match([]byte(pkg)) {
			extensions = append(extensions, strings.Split(pkg, ".")[1])
		}
	}
	return extensions
}
