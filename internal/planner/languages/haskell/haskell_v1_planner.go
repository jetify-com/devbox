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

// TODO: We should also support different GHC versions.
// These can take the format of `haskell.compiler.ghc90` or similar
var ghcRegex = regexp.MustCompile(`^ghc$`)
var ghcVersionRegex = regexp.MustCompile(`^haskell\.compiler\.(.*)$`)

var stackRegex = regexp.MustCompile(`^stack$`)
var cabalRegex = regexp.MustCompile(`^cabal-install$`)

type V2Planner struct {
	userPackages []string
}

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
	return p.getGHCPackage().compilerType != None
}

func (p *V2Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	ghcPackage := p.getGHCPackage()
	haskellPackages := p.getHaskellPackages(ghcPackage)
	definitions := []string{}
	fmt.Print(ghcPackage.compilerType)
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
		DevPackages: []string{"haskell-pkg"},
	}
}

func (p *V2Planner) getGHCPackage() GHCPackage {
	for _, pkg := range p.userPackages {
		if ghcRegex.Match([]byte(pkg)) {
			return GHCPackage{compilerType: Default, pkg: pkg}
		} else if matches := ghcVersionRegex.FindStringSubmatch(pkg); matches != nil {
			return GHCPackage{compilerType: Versioned, pkg: matches[1]}
		}
	}
	return GHCPackage{compilerType: None, pkg: ""}
}

func (p *V2Planner) getHaskellPackages(ghcPackage GHCPackage) []string {
	var haskellPackages []string
	for _, pkg := range p.userPackages {
		switch ghcPackage.compilerType {
		case Default:
			if stackRegex.Match([]byte(pkg)) || cabalRegex.Match([]byte(pkg)) {
				haskellPackages = append(haskellPackages, pkg)
			} else if haskellPackageRegex.Match([]byte(pkg)) {
				haskellPackages = append(haskellPackages, strings.Split(pkg, ".")[1])
			}
		case Versioned:
			if matches := haskellPackageVersionRegex.FindAllStringSubmatch(pkg, -1); matches != nil {
				fmt.Println(matches)
				if matches[0][1] == ghcPackage.pkg {
					haskellPackages = append(haskellPackages, matches[0][2])
				}
			}
		}
	}
	return haskellPackages
}
