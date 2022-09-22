// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package elixir

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

var versionMap = map[string]string{
	"1.9":  "elixir_1_9",
	"1.10": "elixir_1_10",
	"1.11": "elixir_1_11",
	"1.12": "elixir_1_12",
	"1.13": "elixir",
	"1.14": "elixir_1_14",
}

type ElixirProject struct {
	name          string
	elixirPackage string
}

const defaultPkg = "elixir_1_14" // Default to the latest

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "elixir.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	mixPath := filepath.Join(srcDir, "mix.exs")
	return plansdk.FileExists(mixPath)
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	elixirProject, err := getElixirProject(srcDir)
	if err != nil {
		log.Fatal(err)
	}
	return &plansdk.Plan{
		DevPackages: []string{
			elixirProject.elixirPackage,
		},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"mix.eks"},
			Command:    "mix deps.get --only-prod",
		},
		BuildStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command:    "MIX_ENV=prod mix compile && MIX_ENV=prod mix release",
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{fmt.Sprintf("_build/prod/rel/%s", elixirProject.name)},
			Command:    fmt.Sprintf("bin/%s start", elixirProject.name),
		},
	}
}

func getElixirProject(srcDir string) (ElixirProject, error) {
	mixPath := filepath.Join(srcDir, "mix.exs")
	elixirPackage, err := getElixirPackage(mixPath)
	if err != nil {
		log.Fatal(err)
	}
	appname, err := getElixirAppName(mixPath)
	if err != nil {
		log.Fatal(err)
	}

	return ElixirProject{
		name:          appname,
		elixirPackage: elixirPackage,
	}, nil
}

func getElixirPackage(mixPath string) (string, error) {
	elixirVersion := parseElixirVersion(mixPath)
	v, ok := versionMap[elixirVersion]
	if ok {
		return v, nil
	} else {
		return "", errors.New("Could not find a Nix package for Elixir that matched your required version")
	}
}

func parseElixirVersion(mixPath string) string {
	contents, err := os.ReadFile(mixPath)
	if err != nil {
		return ""
	}
	r := regexp.MustCompile(`(?:^elixir: "\\D*)(\\d\.\\d*)`)
	match := r.FindStringSubmatch(string(contents))
	if len(match) != 1 {
		return ""
	} else {
		return match[0]
	}
}

func getElixirAppName(mixPath string) (string, error) {
	contents, err := os.ReadFile(mixPath)
	if err != nil {
		return "", errors.New("Unable to read your mix.exs file")
	}
	r := regexp.MustCompile(`(?:^app: )(?:\:)([a-z\_]*)`)
	match := r.FindStringSubmatch(string(contents))
	if len(match) != 1 {
		return "", errors.New("Unable to parse an app name from your mix.exs")
	} else {
		return match[0], nil
	}
}
