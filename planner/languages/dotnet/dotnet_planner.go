// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package dotnet

import (
	"fmt"
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

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	plan, err := p.getPlan(srcDir)
	if err != nil {
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

		// TODO replace dotNetPkg to reduce runtime image size
		//
		// Including dotNetPkg results in the image size being large (~700MB for csharp_10-dotnet_6 testdata project)
		// To reduce size, I tried compiling a I tried compiling a "self-contained executable" as explained in
		// https://docs.microsoft.com/en-us/dotnet/core/deploying/ by doing `dotnet publish -r <RID>`.
		// This resulted in some errors:
		// Error #1. An error for missing `libstdc++`. Adding nix pkg `libstdcxx5` didn't help.
		// Adding `gcc` resolved it (but results in image size being 300MB)
		// Error #2. An error for missing `libicu`. Adding nix pkg `icu` didn't help. TODO need to resolve this issue.
		RuntimePackages: []string{dotNetPkg},

		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			// --packages stores the downloaded packages in a local directory called nuget-packages
			// Otherwise, the default location is ~/.nuget/packages,
			// which is hard to copy over into StartStage
			Command: "dotnet restore --packages nuget-packages",
		},

		BuildStage: &plansdk.Stage{

			// TODO modify this command to reduce image size
			//
			// Useful references for improving this publish command to reduce image size:
			// dotnet publish -r linux-x64 -p:PublishSingleFile:true
			// - for dotnet publish options: https://docs.microsoft.com/en-us/dotnet/core/tools/dotnet-publish
			// - for -r options: https://docs.microsoft.com/en-us/dotnet/core/rid-catalog
			// - for publishing a single file: https://docs.microsoft.com/en-us/dotnet/core/deploying/single-file/overview?tabs=cli
			Command: "dotnet publish -c Publish --no-restore",
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			// TODO to invoke single-executable: ./bin/Debug/net6.0/linux-64/publish/<projectName>
			Command: "dotnet run -c Publish --no-build",
		},
	}, nil
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
