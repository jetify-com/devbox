// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/redact"
)

func ProfileList(writer io.Writer, profilePath string, useJSON bool) (string, error) {
	cmd := command("profile", "list", "--profile", profilePath)
	if useJSON {
		cmd.Args = append(cmd.Args, "--json")
	}
	out, err := cmd.Output(context.TODO())
	if err != nil {
		return "", redact.Errorf("error running \"nix profile list\": %w", err)
	}
	return string(out), nil
}

type ProfileInstallArgs struct {
	Installable string
	Offline     bool
	ProfilePath string
	Writer      io.Writer
}

func ProfileInstall(ctx context.Context, args *ProfileInstallArgs) error {
	defer debug.FunctionTimer().End()
	if !IsInsecureAllowed() && PackageIsInsecure(args.Installable) {
		knownVulnerabilities := PackageKnownVulnerabilities(args.Installable)
		errString := fmt.Sprintf("Package %s is insecure. \n\n", args.Installable)
		if len(knownVulnerabilities) > 0 {
			errString += fmt.Sprintf("Known vulnerabilities: %s \n\n", knownVulnerabilities)
		}
		errString += "To override use `devbox add <pkg> --allow-insecure`"
		return usererr.New(errString)
	}

	cmd := command(
		"profile", "install",
		"--profile", args.ProfilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
		// Using an arbitrary priority to avoid conflicts with other packages.
		// Note that this is not really the priority we care about, since we
		// use the flake.nix to specify the priority.
		"--priority", nextPriority(args.ProfilePath),
	)
	if args.Offline {
		cmd.Args = append(cmd.Args, "--offline")
	}
	cmd.Args = append(cmd.Args, args.Installable)
	cmd.Env = allowUnfreeEnv(os.Environ())

	// If nix profile install runs as tty, the output is much nicer. If we ever
	// need to change this to our own writers, consider that you may need
	// to implement your own nicer output. --print-build-logs flag may be useful.
	cmd.Stdin = os.Stdin
	cmd.Stdout = args.Writer
	cmd.Stderr = args.Writer

	slog.Debug("running command", "cmd", cmd)
	return cmd.Run(ctx)
}

// ProfileRemove removes packages from a profile.
// WARNING, don't use indexes, they are not supported by nix 2.20+
func ProfileRemove(profilePath string, packageNames ...string) error {
	defer debug.FunctionTimer().End()
	cmd := command(
		"profile", "remove",
		"--profile", profilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
	)
	cmd.Args = appendArgs(cmd.Args, packageNames)
	cmd.Env = allowUnfreeEnv(allowInsecureEnv(os.Environ()))
	return cmd.Run(context.TODO())
}

type manifest struct {
	Elements []struct {
		Priority int
	}
}

func readManifest(profilePath string) (manifest, error) {
	data, err := os.ReadFile(filepath.Join(profilePath, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest{}, nil
	}
	if err != nil {
		return manifest{}, err
	}

	type manifestModern struct {
		Elements map[string]struct {
			Priority int `json:"priority"`
		} `json:"elements"`
	}
	var modernMani manifestModern
	if err := json.Unmarshal(data, &modernMani); err == nil {
		// Convert to the result format
		result := manifest{}
		for _, e := range modernMani.Elements {
			result.Elements = append(result.Elements, struct{ Priority int }{e.Priority})
		}
		return result, nil
	}

	type manifestLegacy struct {
		Elements []struct {
			Priority int `json:"priority"`
		} `json:"elements"`
	}
	var legacyMani manifestLegacy
	if err := json.Unmarshal(data, &legacyMani); err != nil {
		return manifest{}, err
	}

	// Convert to the result format
	result := manifest{}
	for _, e := range legacyMani.Elements {
		result.Elements = append(result.Elements, struct{ Priority int }{e.Priority})
	}
	return result, nil
}

const DefaultPriority = 5

func nextPriority(profilePath string) string {
	// error is ignored because it's ok if the file doesn't exist
	m, _ := readManifest(profilePath)
	max := DefaultPriority
	for _, e := range m.Elements {
		if e.Priority > max {
			max = e.Priority
		}
	}
	// Each subsequent package gets a lower priority. This matches how flake.nix
	// behaves
	return fmt.Sprintf("%d", max+1)
}
