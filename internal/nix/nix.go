// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime/trace"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// or `nix profile install --profile ProfilePath <package...>`
// Instead of using directory, prefer using the devbox.ProfileDir() function that ensures the directory exists.
// Warning: don't use the bins in default/bin, they won't always match bins
// produced by the flakes.nix. Use devbox.NixBins() instead.
const ProfilePath = ".devbox/nix/profile/default"

var ErrPackageNotFound = errors.New("package not found")
var ErrPackageNotInstalled = errors.New("package not installed")

func PkgExists(nixpkgsCommit, pkg string) bool {
	_, found := PkgInfo(nixpkgsCommit, pkg)
	return found
}

type Info struct {
	// attribute key is different in flakes vs legacy so we should only use it
	// if we know exactly which version we are using
	attributeKey string
	NixName      string
	Name         string
	Version      string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.Name, i.Version)
}

func PkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	exactPackage := fmt.Sprintf("%s#%s", FlakeNixpkgs(nixpkgsCommit), pkg)
	if nixpkgsCommit == "" {
		exactPackage = fmt.Sprintf("nixpkgs#%s", pkg)
	}

	cmd := exec.Command("nix", "search", "--json", exactPackage)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	cmd.Stderr = os.Stderr
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil, false /* not found */
	}
	pkgInfo := parseInfo(pkg, out)
	if pkgInfo == nil {
		return nil, false /* not found */
	}
	return pkgInfo, true /* found */
}

func parseInfo(pkg string, data []byte) *Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	for key, result := range results {
		pkgInfo := &Info{
			attributeKey: key,
			NixName:      pkg,
			Name:         result["pname"].(string),
			Version:      result["version"].(string),
		}

		return pkgInfo
	}
	return nil
}

type varsAndFuncs struct {
	Functions map[string]string   // the key is the name, the value is the body.
	Variables map[string]variable // the key is the name.
}
type variable struct {
	Type  string // valid types are var, exported, and array.
	Value any    // can be a string or an array of strings (iff type is array).
}

// PrintDevEnv calls `nix print-dev-env -f <path>` and returns its output. The output contains
// all the environment variables and bash functions required to create a nix shell.
func PrintDevEnv(ctx context.Context, nixShellFilePath, nixFlakesFilePath string) (*varsAndFuncs, error) {
	defer trace.StartRegion(ctx, "nixPrintDevEnv").End()

	cmd := exec.CommandContext(ctx, "nix", "print-dev-env", nixFlakesFilePath)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	cmd.Args = append(cmd.Args, "--json")
	debug.Log("Running print-dev-env cmd: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Command: %s", cmd)
	}

	var vaf varsAndFuncs
	return &vaf, errors.WithStack(json.Unmarshal(out, &vaf))
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
