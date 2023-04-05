// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package haskell

import (
	"fmt"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/planner/plansdk"
)

var haskellPackageRegex = regexp.MustCompile(`^haskellPackages.[a-zA-Z0-9\-\_\.]*$`)

// Use matchgroups to get the compiler, then the package name
// Compiler will be in Mathches[0][1] and package name in Matches[1][2]
var haskellPackageVersionRegex = regexp.MustCompile(`^haskell\.packages\.(.*)\.(.*)$`)

var ghcRegex = regexp.MustCompile(`^ghc$`)
var ghcVersionRegex = regexp.MustCompile(`^haskell\.compiler\.(.*)$`)

var stackRegex = regexp.MustCompile(`^stack$`)
var cabalRegex = regexp.MustCompile(`^cabal-install$`)

type V2Planner struct {
	userPackages []string
}

// Create types and struct that tells us which GHC compiler we're using

type CompilerType string

const (
	Default   CompilerType = "default"
	Versioned CompilerType = "versioned"
	None      CompilerType = ""
)

type GHCPackage struct {
	compilerType CompilerType
	pkg          string
}

var _ plansdk.PlannerForPackages = (*V2Planner)(nil)

func (p *V2Planner) Name() string {
	return "haskell.v1.Planner"
}

func (p *V2Planner) IsRelevant(srcDir string) bool {
	return false
}

func (p *V2Planner) IsRelevantForPackages(packages []string) bool {
	p.userPackages = packages
	_, index := p.getGHCPackage()
	return index != -1
}

func (p *V2Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	ghcPackage, index := p.getGHCPackage()
	// Remove the ghc package from the list of user packages
	p.userPackages = append(p.userPackages[:index], p.userPackages[index+1:]...)
	haskellPackages := p.getHaskellPackages(ghcPackage)
	definitions := []string{}
	// Create the haskell-pkg definition based on the compiler type
	switch ghcPackage.compilerType {
	case Default:
		definitions = []string{
			fmt.Sprintf(
				"haskell-pkg = pkgs.haskellPackages.ghcWithPackages (ps: with ps; [ %s ]);",
				strings.Join(haskellPackages, " "),
			),
		}
	case Versioned:
		definitions = []string{
			fmt.Sprintf(
				"haskell-pkg = pkgs.haskell.packages.%s.ghcWithPackages (ps: with ps; [ %s ]);",
				ghcPackage.pkg,
				strings.Join(haskellPackages, " "),
			),
		}
	}
	return &plansdk.ShellPlan{
		Definitions: definitions,
		// Prepend "haskell-pkg" to the list of user packages
		DevPackages: append([]string{"haskell-pkg"}, p.userPackages...),
	}
}

func (p *V2Planner) getGHCPackage() (GHCPackage, int) {
	// packages := p.userPackages
	for index, pkg := range p.userPackages {
		if ghcRegex.Match([]byte(pkg)) {
			return GHCPackage{compilerType: Default, pkg: pkg}, index
		} else if matches := ghcVersionRegex.FindStringSubmatch(pkg); matches != nil {
			return GHCPackage{compilerType: Versioned, pkg: matches[1]}, index
		}
	}
	return GHCPackage{compilerType: None, pkg: ""}, -1
}

func (p *V2Planner) getHaskellPackages(ghcPackage GHCPackage) []string {
	var haskellPackages []string
	var filteredPackages []string
	for _, pkg := range p.userPackages {
		switch ghcPackage.compilerType {
		case Default:
			if stackRegex.Match([]byte(pkg)) || cabalRegex.Match([]byte(pkg)) {
				haskellPackages = append(haskellPackages, pkg)
			} else if haskellPackageRegex.Match([]byte(pkg)) {
				haskellPackages = append(haskellPackages, strings.Split(pkg, ".")[1])
			} else {
				filteredPackages = append(filteredPackages, pkg)
			}
		case Versioned:
			if matches := haskellPackageVersionRegex.FindAllStringSubmatch(pkg, -1); matches != nil {
				if matches[0][1] == ghcPackage.pkg {
					haskellPackages = append(haskellPackages, matches[0][2])
				}
			} else {
				filteredPackages = append(filteredPackages, pkg)
			}
		}
	}
	//  Remove the packages that we've already added to the haskellPackages
	// list from the userPackages list
	p.userPackages = filteredPackages
	return haskellPackages
}
