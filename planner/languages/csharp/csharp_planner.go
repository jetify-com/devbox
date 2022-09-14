// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package csharp

import (
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

type Project struct {
	PropertyGroup struct {
		TargetFramework string `xml:"TargetFramework,omitempty"`
	} `xml:"PropertyGroup,omitempty"`
}

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "csharp.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	return a.HasAnyFile("*.csproj")
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
		// Added this Printf because `devbox shell` was silently swallowing this error.
		// TODO savil. Have `devbox shell` error out or print it instead.
		// fmt.Printf("error in getPlan: %s\n", err)
		plan = &plansdk.Plan{}
		plan.WithError(err)
	}
	return plan
}

func (p *Planner) getPlan(srcDir string) (*plansdk.Plan, error) {

	proj, err := project(srcDir)
	if err != nil {
		return nil, err
	}
	dotNetPkg, err := dotNetNixPackage(proj)
	if err != nil {
		return nil, err
	}

	return &plansdk.Plan{
		DevPackages: []string{dotNetPkg},
	}, nil
}

func project(srcDir string) (*Project, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}
	paths := a.GlobFiles("*.csproj")
	if len(paths) < 1 {
		return nil, errors.Errorf("expected to find a .csproj file in directory %s", srcDir)
	}
	projectFilePath := paths[0]

	proj := &Project{}
	err = cuecfg.ParseFile(projectFilePath, proj)
	return proj, err
}

// The TargetFramework is more complicated than below, but I'm picking out what
// seem to be the common uses.
// https://docs.microsoft.com/en-us/dotnet/standard/frameworks
func dotNetNixPackage(proj *Project) (string, error) {
	if proj.PropertyGroup.TargetFramework == "" {
		return "", errors.New("Did not find Dot Net Framework in .csproj")
	}

	if strings.HasPrefix(proj.PropertyGroup.TargetFramework, "net7") { // for net7.x
		return "dotnet-sdk_7", nil
	}
	if strings.HasPrefix(proj.PropertyGroup.TargetFramework, "net6") { // for net6.x
		return "dotnet-sdk", nil
	}
	if strings.HasPrefix(proj.PropertyGroup.TargetFramework, "net5") { // for net5.x
		return "dotnet-sdk_5", nil
	}
	// NOTE: there is in fact NO dot-net_4. Reference: https://docs.microsoft.com/en-us/dotnet/core/whats-new/dotnet-5
	if strings.HasPrefix(proj.PropertyGroup.TargetFramework, "netcoreapp3") {
		return "dotnet-sdk_3", nil
	}
	return "", errors.Errorf("Unrecognized DotNet Framework version: %s", proj.PropertyGroup.TargetFramework)
}
