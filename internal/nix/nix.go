// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// or `nix profile install --profile ProfilePath <package...>`
// Instead of using directory, prefer using the devbox.ProfileDir() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

var ErrPackageNotFound = errors.New("package not found")
var ErrPackageNotInstalled = errors.New("package not installed")

func PkgExists(nixpkgsCommit, pkg, projectDir string) (bool, error) {
	input := InputFromString(pkg, projectDir)
	if input.IsFlake() {
		return input.validateExists()
	}
	return PkgInfo(nixpkgsCommit, pkg) != nil, nil
}

type Info struct {
	// attribute key is different in flakes vs legacy so we should only use it
	// if we know exactly which version we are using
	attributeKey string
	PName        string
	Version      string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.PName, i.Version)
}

func PkgInfo(nixpkgsCommit, pkg string) *Info {
	exactPackage := fmt.Sprintf("%s#%s", FlakeNixpkgs(nixpkgsCommit), pkg)
	if nixpkgsCommit == "" {
		exactPackage = fmt.Sprintf("nixpkgs#%s", pkg)
	}

	results := search(exactPackage)
	if len(results) == 0 {
		return nil
	}
	// we should only have one result
	return lo.Values(results)[0]
}

func search(url string) map[string]*Info {
	cmd := exec.Command("nix", "search", "--json", url)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	cmd.Stderr = os.Stderr
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil
	}
	return parseSearchResults(out)
}

func parseSearchResults(data []byte) map[string]*Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	infos := map[string]*Info{}
	for key, result := range results {
		infos[key] = &Info{
			attributeKey: key,
			PName:        result["pname"].(string),
			Version:      result["version"].(string),
		}

	}
	return infos
}

type printDevEnvOut struct {
	Variables map[string]variable // the key is the name.
}

type variable struct {
	Type  string // valid types are var, exported, and array.
	Value any    // can be a string or an array of strings (iff type is array).
}

type PrintDevEnvArgs struct {
	FlakesFilePath       string
	PrintDevEnvCachePath string
	UsePrintDevEnvCache  bool
}

// PrintDevEnv calls `nix print-dev-env -f <path>` and returns its output. The output contains
// all the environment variables and bash functions required to create a nix shell.
func PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*printDevEnvOut, error) {
	defer trace.StartRegion(ctx, "nixPrintDevEnv").End()

	var data []byte
	var err error
	var out printDevEnvOut

	if args.UsePrintDevEnvCache {
		data, err = os.ReadFile(args.PrintDevEnvCachePath)
		if err == nil {
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, errors.WithStack(err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
	}

	if len(data) == 0 {
		cmd := exec.CommandContext(ctx, "nix", "print-dev-env", args.FlakesFilePath)
		cmd.Args = append(cmd.Args, ExperimentalFlags()...)
		cmd.Args = append(cmd.Args, "--json")
		debug.Log("Running print-dev-env cmd: %s\n", cmd)
		data, err = cmd.Output()
		if err != nil {
			return nil, errors.Wrapf(err, "Command: %s", cmd)
		}

		if err := json.Unmarshal(data, &out); err != nil {
			return nil, errors.WithStack(err)
		}

		if err = savePrintDevEnvCache(args.PrintDevEnvCachePath, out); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &out, nil
}

func PrintDevEnvCacheHash(profileDir string) (string, error) {
	return cuecfg.FileHash(
		filepath.Join(profileDir, ".devbox", ".nix-print-dev-env-cache"),
	)
}

func savePrintDevEnvCache(path string, out printDevEnvOut) error {
	data, err := json.Marshal(out)
	if err != nil {
		return errors.WithStack(err)
	}

	_ = os.WriteFile(path, data, 0644)
	return nil
}

// FlakeNixpkgs returns a flakes-compatible reference to the nixpkgs registry.
// TODO savil. Ensure this works with the nixed cache service.
func FlakeNixpkgs(commit string) string {
	// Using nixpkgs/<commit> means:
	// The nixpkgs entry in the flake registry, with its Git revision overridden to a specific value.
	return "nixpkgs/" + commit
}

func ExperimentalFlags() []string {
	return []string{
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", "nix-command flakes",
	}
}

// Warning: be careful using the bins in default/bin, they won't always match bins
// produced by the flakes.nix. Use devbox.NixBins() instead.
func ProfileBinPath(projectDir string) string {
	return filepath.Join(projectDir, ProfilePath, "bin")
}
