// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package haskell

import (
	"fmt"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/planner/plansdk"
)

var haskellPackageRegex = regexp.MustCompile(`^haskellPackages.[a-zA-Z0-9\-\_\.]+$`)

// TODO: We should also support different GHC versions.
// These can take the format of `haskell.compiler.ghc90` or similar
var ghcRegex = regexp.MustCompile(`^ghc$`)

var stackRegex = regexp.MustCompile(`^stack$`)
var cabalRegex = regexp.MustCompile(`^cabal-install$`)

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
	return p.getGHCPackage() != "" && len(p.getHaskellPackages()) > 0
}

func (p *V2Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	ghcPackage := p.getGHCPackage()
	definitions := []string{
		fmt.Sprintf(
			"%s = pkgs.haskellPackages.ghcWithPackages (ps: with ps; [ %s ]);",
			ghcPackage,
			strings.Join(p.getHaskellPackages(), " "),
		),
	}

	return &plansdk.ShellPlan{Definitions: definitions}
}

func (p *V2Planner) getGHCPackage() string {
	for _, pkg := range p.userPackages {
		if ghcRegex.Match([]byte(pkg)) {
			return pkg
		}
	}
	return ""
}

func (p *V2Planner) getHaskellPackages() []string {
	var haskellPackages []string
	for _, pkg := range p.userPackages {
		if haskellPackageRegex.Match([]byte(pkg)) {
			haskellPackages = append(haskellPackages, strings.Split(pkg, ".")[1])
		} else if stackRegex.Match([]byte(pkg)) || cabalRegex.Match([]byte(pkg)) {
			haskellPackages = append(haskellPackages, pkg)
		}
	}
	return haskellPackages
}
