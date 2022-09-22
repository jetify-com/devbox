// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package elixir

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

var NoElixirVersionSetErr = errors.New("No version set in mix.exs")

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

func getAvailableVersions(versionMap map[string]string) []string {
	keys := make([]string, 0, len(versionMap))
	for k := range versionMap {
		keys = append(keys, k)
	}
	return keys
}

const defaultPkg = "elixir" // Default to the Nix Default

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
		RuntimePackages: []string{
			"systemd",
			"ncurses",
		},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"mix.exs"},
			Command: strings.TrimSpace(`
			mix local.hex --force && \
			mix local.rebar --force && \
			mix deps.get --only-prod`),
		},
		BuildStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
			Command: strings.TrimSpace(`
			MIX_ENV=prod mix compile && \
			MIX_ENV=prod mix release --overwrite`),
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{fmt.Sprintf("_build/prod/rel/%s", elixirProject.name)},
			Command:    fmt.Sprintf(`bin/%s start`, elixirProject.name),
		},
	}
}

func getElixirProject(srcDir string) (*ElixirProject, error) {
	mixPath := filepath.Join(srcDir, "mix.exs")
	mixContents, err := os.ReadFile(mixPath)
	if err != nil {
		return nil, errors.Errorf("Unable to read your mix.exs file. Failed with %s", err)
	}
	elixirPackage, err := getElixirPackage(string(mixContents))
	if err != nil {
		return nil, err
	}
	appname, err := getElixirAppName(string(mixContents))
	if err != nil {
		return nil, err
	}

	return &ElixirProject{
		name:          appname,
		elixirPackage: elixirPackage,
	}, nil
}

func getElixirPackage(mixContents string) (string, error) {
	elixirVersion, err := parseElixirVersion(mixContents)
	if errors.Is(err, NoElixirVersionSetErr) {
		log.Printf("No Elixir version specified in your mix.exs. Using default Nix version 1.13")
		return defaultPkg, nil
	}
	v, ok := versionMap[elixirVersion]
	if ok {
		log.Printf("Using Elixir Package: %s", elixirVersion)
		return v, nil
	} else {
		return "", errors.Errorf("Could not find a Nix package for Elixir that matched your required version. You requested: %s. Available versions: %s", elixirVersion, strings.Join(getAvailableVersions(versionMap), ", "))
	}
}

func parseElixirVersion(mixContents string) (string, error) {
	r := regexp.MustCompile(`(?:elixir: "\D*)([0-9].[0-9]*)`)
	match := r.FindStringSubmatch(mixContents)
	if len(match) < 1 {
		return "", NoElixirVersionSetErr
	} else {
		return match[1], nil
	}
}

func getElixirAppName(mixContents string) (string, error) {
	r := regexp.MustCompile(`(?:app: )(?:\:)([a-z\_]*)`)
	match := r.FindStringSubmatch(mixContents)
	if len(match) <= 1 {
		return "", errors.New("Unable to parse an app name from your mix.exs")
	} else {
		log.Printf("Detected app name: %s", match[1])
		return match[1], nil
	}
}
