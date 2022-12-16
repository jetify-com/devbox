// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package dotnet

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Project struct {
	PropertyGroup struct {
		TargetFramework string `xml:"TargetFramework,omitempty"`
	} `xml:"PropertyGroup,omitempty"`
}

const CSharpExtension = "csproj"
const FSharpExtension = "fsproj"

// The .Net Planner supports C# and F# languages.
type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "dotnet.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return false
	}
	isRelevant := a.HasAnyFile(
		fmt.Sprintf("*.%s", CSharpExtension),
		fmt.Sprintf("*.%s", FSharpExtension),
	)
	return isRelevant
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	proj, err := project(srcDir)
	if err != nil {
		return &plansdk.ShellPlan{}
	}
	dotNetPkg, err := dotNetNixPackage(proj)
	if err != nil {
		return &plansdk.ShellPlan{}
	}
	return &plansdk.ShellPlan{
		DevPackages: []string{dotNetPkg},
	}
}

func project(srcDir string) (*Project, error) {
	a, err := plansdk.NewAnalyzer(srcDir)
	if err != nil {
		// We should log that an error has occurred.
		return nil, err
	}
	paths := a.GlobFiles(
		fmt.Sprintf("*.%s", CSharpExtension),
		fmt.Sprintf("*.%s", FSharpExtension),
	)
	if len(paths) < 1 {
		return nil, errors.Errorf(
			"expected to find a %s or %s file in directory %s",
			CSharpExtension,
			FSharpExtension,
			srcDir,
		)
	}
	projectFilePath := paths[0]

	proj := &Project{}
	err = cuecfg.ParseFileWithExtension(projectFilePath, ".xml", proj)
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
