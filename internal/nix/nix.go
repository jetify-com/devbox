// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"

	"go.jetpack.io/devbox/internal/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// or `nix profile install --profile ProfilePath <package...>`
// Instead of using directory, prefer using the devbox.ProfileDir() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

type PrintDevEnvOut struct {
	Variables map[string]Variable // the key is the name.
}

type Variable struct {
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
func (*Nix) PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*PrintDevEnvOut, error) {
	defer trace.StartRegion(ctx, "nixPrintDevEnv").End()

	var data []byte
	var err error
	var out PrintDevEnvOut

	if args.UsePrintDevEnvCache {
		data, err = os.ReadFile(args.PrintDevEnvCachePath)
		if err == nil {
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, errors.WithStack(err)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
	}

	if len(data) == 0 {
		cmd := exec.CommandContext(
			ctx,
			"nix", "print-dev-env",
			args.FlakesFilePath,
		)
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

func savePrintDevEnvCache(path string, out PrintDevEnvOut) error {
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
	options := []string{"nix-command", "flakes"}
	if featureflag.RemoveNixpkgs.Enabled() {
		options = append(options, "fetch-closure")
	}
	return []string{
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", strings.Join(options, " "),
	}
}

var cachedSystem string

func System() (string, error) {
	// For Savil to debug "remove nixpkgs" feature. The Search api lacks x86-darwin info.
	// So, I need to fake that I am x86-linux and inspect the output in generated devbox.lock
	// and flake.nix files.
	override := os.Getenv("__DEVBOX_NIX_SYSTEM")
	if override != "" {
		return override, nil
	}

	if cachedSystem == "" {
		cmd := exec.Command(
			"nix", "eval", "--impure", "--raw", "--expr", "builtins.currentSystem",
		)
		cmd.Args = append(cmd.Args, ExperimentalFlags()...)
		out, err := cmd.Output()
		if err != nil {
			return "", errors.WithStack(err)
		}
		cachedSystem = string(out)
	}
	return cachedSystem, nil
}

// Warning: be careful using the bins in default/bin, they won't always match bins
// produced by the flakes.nix. Use devbox.NixBins() instead.
func ProfileBinPath(projectDir string) string {
	return filepath.Join(projectDir, ProfilePath, "bin")
}
